// Package telemetry builds the OpenTelemetry providers of the service.
//
// The SDK lives here and nowhere else: the exporter, the sampler, the
// resource and the providers are built in this package, and the rest of the
// service only carries the context. The domain, the application and the
// ports never hear of OpenTelemetry; the composition root and the adapters
// know the type of the tracer provider, and nothing more.
//
// The providers are registered in the global OpenTelemetry registry on
// purpose. The instrumentation of the transport and the bridge of the logger
// read them from that registry, which is the contract every OpenTelemetry
// library expects.
//
// The configuration of the tracing lives here as well: the names of the
// variables, the reading of them and the rules they must obey. The package
// reads no environment variable: a service hands it the reader of its own
// environment.
package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// The protocols the exporter speaks to the collector.
const (
	// ProtocolGRPC is the OTLP over gRPC. It is the default: it is the
	// protocol the collector of the monorepository listens on.
	ProtocolGRPC = "grpc"
	// ProtocolHTTP is the OTLP over HTTP with a protobuf body.
	ProtocolHTTP = "http/protobuf"
)

// The samplers the service understands.
const (
	// SamplerAlwaysOn keeps every trace.
	SamplerAlwaysOn = "always_on"
	// SamplerAlwaysOff drops every trace.
	SamplerAlwaysOff = "always_off"
	// SamplerTraceIDRatio keeps the ratio of the traces, decided by the
	// identifiers of the traces themselves.
	SamplerTraceIDRatio = "traceidratio"
	// SamplerParentBasedAlwaysOn keeps every trace, and keeps whole the
	// traces of the callers that sampled them.
	SamplerParentBasedAlwaysOn = "parentbased_always_on"
	// SamplerParentBasedTraceIDRatio keeps the ratio of the traces, and
	// keeps whole the traces of the callers that sampled them.
	SamplerParentBasedTraceIDRatio = "parentbased_traceidratio"
)

// batchTimeout is how long the exporter waits before it sends a batch. A
// batch is smaller and cheaper than one span per request, and five seconds
// is short enough for a trace to show up while a developer waits for it.
const batchTimeout = 5 * time.Second

// samplerArgName is the name the error of the ratio of the traces carries
// when the telemetry builds the sampler itself.
//
// The ratio comes from the options of the caller, not from the environment of
// a service, and the error names the setting it belongs to.
const samplerArgName = "the sampler argument"

// Options configures the providers of the service.
type Options struct {
	// Enabled tells whether the service traces. A service that does not
	// trace keeps the no-op providers, and its logs carry no trace.
	Enabled bool
	// ServiceName is the name the telemetry carries.
	ServiceName string
	// ServiceVersion is the version the telemetry carries.
	ServiceVersion string
	// Endpoint is the address of the collector that receives the traces.
	Endpoint string
	// Protocol is the protocol the exporter speaks.
	Protocol string
	// Insecure tells whether the exporter talks to the collector without
	// TLS. The internal network of the monorepository is isolated and its
	// traffic never leaves the host, which is where the plain transport
	// belongs.
	Insecure bool
	// Sampler is the name of the sampler, one of the Sampler constants.
	Sampler string
	// SamplerArg is the argument of the sampler: the ratio of the traces
	// that are kept, for the samplers that take one.
	SamplerArg string
	// Exporter overrides the exporter of the other options. The tests use
	// it to read the spans in memory; when it is nil, the exporter is
	// built from the endpoint and the protocol.
	Exporter sdktrace.SpanExporter
}

// Providers holds the OpenTelemetry providers of the service.
type Providers struct {
	// TracerProvider creates the spans of the service.
	TracerProvider trace.TracerProvider
	// Propagator carries the trace context between the services.
	Propagator propagation.TextMapPropagator

	// shutdown flushes the spans that are still in the batcher. It is nil
	// when the service does not trace.
	shutdown func(context.Context) error
}

// Setup builds the providers of the service.
//
// The providers are registered in the global OpenTelemetry registry, so the
// instrumentation of the transport and the bridge of the logger use them
// without knowing them.
//
// A service that does not trace gets the no-op providers: no span is ever
// created, and no log record carries a trace identifier.
func Setup(ctx context.Context, opts Options) (*Providers, error) {
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	if !opts.Enabled {
		provider := noop.NewTracerProvider()
		otel.SetTracerProvider(provider)
		otel.SetTextMapPropagator(propagator)
		return &Providers{TracerProvider: provider, Propagator: propagator}, nil
	}

	exporter := opts.Exporter
	if exporter == nil {
		built, err := newExporter(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("build the span exporter: %w", err)
		}
		exporter = built
	}

	res, err := newResource(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("build the resource: %w", err)
	}

	sampler, err := newSampler(opts)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(batchTimeout)),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagator)

	return &Providers{
		TracerProvider: provider,
		Propagator:     propagator,
		shutdown:       provider.Shutdown,
	}, nil
}

// Shutdown flushes the spans that are still in the batcher.
//
// The service must call it before it stops, otherwise the last spans of the
// process are lost: the batcher holds them until it sends them.
func (p *Providers) Shutdown(ctx context.Context) error {
	if p == nil || p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx)
}

// newExporter builds the exporter that sends the spans to the collector.
func newExporter(ctx context.Context, opts Options) (sdktrace.SpanExporter, error) {
	switch opts.Protocol {
	case ProtocolHTTP:
		exporterOptions := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(opts.Endpoint),
		}
		if opts.Insecure {
			exporterOptions = append(exporterOptions, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, exporterOptions...)

	case ProtocolGRPC, "":
		exporterOptions := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(opts.Endpoint),
		}
		if opts.Insecure {
			exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
		}
		return otlptracegrpc.New(ctx, exporterOptions...)

	default:
		return nil, fmt.Errorf("%q is not a protocol", opts.Protocol)
	}
}

// newResource builds the resource every span of the service carries.
//
// The resource answers "who produced this telemetry": the name and the
// version of the service. The rest of the attributes of the deployment
// come from the environment of the OpenTelemetry specification.
func newResource(ctx context.Context, opts Options) (*resource.Resource, error) {
	attributes := []attribute.KeyValue{
		semconv.ServiceName(opts.ServiceName),
		semconv.ServiceVersion(opts.ServiceVersion),
	}

	// The environment comes first, so the attributes of the options win
	// over the ones of the environment.
	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(attributes...),
	)
}

// newSampler builds the sampler of the service.
//
// The decision of the caller wins: a trace that the caller sampled is kept
// whole, even when the service drops the traces of its own. A trace that
// was cut in half is worse than no trace at all.
func newSampler(opts Options) (sdktrace.Sampler, error) {
	switch opts.Sampler {
	case SamplerAlwaysOn:
		return sdktrace.AlwaysSample(), nil

	case SamplerAlwaysOff:
		return sdktrace.NeverSample(), nil

	case SamplerTraceIDRatio:
		ratio, err := parseRatio(samplerArgName, opts.SamplerArg)
		if err != nil {
			return nil, err
		}
		return sdktrace.TraceIDRatioBased(ratio), nil

	case SamplerParentBasedTraceIDRatio:
		ratio, err := parseRatio(samplerArgName, opts.SamplerArg)
		if err != nil {
			return nil, err
		}
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio)), nil

	case SamplerParentBasedAlwaysOn, "":
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil

	default:
		return nil, fmt.Errorf("%q is not a sampler", opts.Sampler)
	}
}
