package telemetry_test

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/packages/platform/telemetry"
)

// The tests of this package are not parallel: they register the providers
// in the global OpenTelemetry registry, which belongs to the process.

// recordingExporter keeps the spans it receives.
//
// The exporter of the SDK clears the spans it holds when it shuts down,
// which would hide the flush that the shutdown of the service performs.
type recordingExporter struct {
	mutex sync.Mutex
	spans tracetest.SpanStubs
}

// ExportSpans keeps the spans.
func (e *recordingExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.spans = append(e.spans, tracetest.SpanStubsFromReadOnlySpans(spans)...)
	return nil
}

// Shutdown keeps the spans.
func (e *recordingExporter) Shutdown(context.Context) error {
	return nil
}

// Spans returns the spans the exporter received.
func (e *recordingExporter) Spans() tracetest.SpanStubs {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return e.spans
}

// flush forces the spans of the provider out of the batcher.
func flush(t *testing.T, providers *telemetry.Providers) {
	t.Helper()

	provider, ok := providers.TracerProvider.(*sdktrace.TracerProvider)
	if !ok {
		t.Fatalf("the tracer provider is a %T, want the one of the SDK", providers.TracerProvider)
	}
	if err := provider.ForceFlush(context.Background()); err != nil {
		t.Fatalf("force flush: %v", err)
	}
}

func TestTheSetupTracesTheService(t *testing.T) {
	exporter := &recordingExporter{}

	providers, err := telemetry.Setup(context.Background(), telemetry.Options{
		Enabled:        true,
		ServiceName:    "paca-mcp-server",
		ServiceVersion: "1.2.3",
		Sampler:        telemetry.SamplerParentBasedAlwaysOn,
		Exporter:       exporter,
	})
	if err != nil {
		t.Fatalf("Setup() = %v", err)
	}

	_, span := providers.TracerProvider.Tracer("test").Start(context.Background(), "a span")
	span.End()
	flush(t, providers)

	spans := exporter.Spans()
	if len(spans) != 1 {
		t.Fatalf("the exporter holds %d spans, want one", len(spans))
	}
	if spans[0].Name != "a span" {
		t.Fatalf("span name = %q, want %q", spans[0].Name, "a span")
	}
}

func TestTheSetupNamesTheServiceInTheResource(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment.name=test")

	exporter := &recordingExporter{}

	providers, err := telemetry.Setup(context.Background(), telemetry.Options{
		Enabled:        true,
		ServiceName:    "paca-mcp-server",
		ServiceVersion: "1.2.3",
		Exporter:       exporter,
	})
	if err != nil {
		t.Fatalf("Setup() = %v", err)
	}

	_, span := providers.TracerProvider.Tracer("test").Start(context.Background(), "a span")
	span.End()
	flush(t, providers)

	spans := exporter.Spans()
	if len(spans) != 1 {
		t.Fatalf("the exporter holds %d spans, want one", len(spans))
	}

	attributes := spans[0].Resource.Set().ToSlice()
	if len(attributes) == 0 {
		t.Fatal("the resource holds no attribute, want the name of the service")
	}

	found := map[attribute.Key]string{}
	for _, key := range attributes {
		found[key.Key] = key.Value.AsString()
	}

	if found["service.name"] != "paca-mcp-server" {
		t.Fatalf("service.name = %q, want %q", found["service.name"], "paca-mcp-server")
	}
	if found["service.version"] != "1.2.3" {
		t.Fatalf("service.version = %q, want %q", found["service.version"], "1.2.3")
	}
	if found["deployment.environment.name"] != "test" {
		t.Fatalf("deployment.environment.name = %q, want %q",
			found["deployment.environment.name"], "test")
	}
}

func TestTheSetupWithoutTracingCreatesNoSpan(t *testing.T) {
	providers, err := telemetry.Setup(context.Background(), telemetry.Options{
		Enabled:  false,
		Exporter: &recordingExporter{},
	})
	if err != nil {
		t.Fatalf("Setup() = %v", err)
	}

	_, span := providers.TracerProvider.Tracer("test").Start(context.Background(), "a span")
	defer span.End()

	if span.IsRecording() {
		t.Fatal("the span is recorded, want no span at all")
	}
	if span.SpanContext().IsValid() {
		t.Fatal("the span context is valid, want no trace to point at")
	}
}

func TestTheSetupRejectsAnUnknownProtocol(t *testing.T) {
	_, err := telemetry.Setup(context.Background(), telemetry.Options{
		Enabled:  true,
		Endpoint: "collector:4317",
		Protocol: "smoke signals",
	})
	if err == nil {
		t.Fatal("Setup() = nil, want an error about the protocol")
	}
}

func TestTheSetupRejectsAnUnknownSampler(t *testing.T) {
	_, err := telemetry.Setup(context.Background(), telemetry.Options{
		Enabled:  true,
		Endpoint: "collector:4317",
		Sampler:  "whenever",
	})
	if err == nil {
		t.Fatal("Setup() = nil, want an error about the sampler")
	}
}

func TestTheSetupRejectsARatioOutOfBounds(t *testing.T) {
	_, err := telemetry.Setup(context.Background(), telemetry.Options{
		Enabled:    true,
		Endpoint:   "collector:4317",
		Sampler:    telemetry.SamplerTraceIDRatio,
		SamplerArg: "1.5",
	})
	if err == nil {
		t.Fatal("Setup() = nil, want an error about the ratio")
	}
}

func TestTheSetupKeepsTheRatioOfTheTraces(t *testing.T) {
	exporter := &recordingExporter{}

	providers, err := telemetry.Setup(context.Background(), telemetry.Options{
		Enabled:    true,
		Endpoint:   "collector:4317",
		Sampler:    telemetry.SamplerTraceIDRatio,
		SamplerArg: "0",
		Exporter:   exporter,
	})
	if err != nil {
		t.Fatalf("Setup() = %v", err)
	}

	_, span := providers.TracerProvider.Tracer("test").Start(context.Background(), "a span")
	span.End()
	flush(t, providers)

	if spans := exporter.Spans(); len(spans) != 0 {
		t.Fatalf("the exporter holds %d spans, want none", len(spans))
	}
}

func TestTheSetupBuildsTheExporterOfTheCollector(t *testing.T) {
	protocols := []string{telemetry.ProtocolGRPC, telemetry.ProtocolHTTP}

	for _, protocol := range protocols {
		t.Run(protocol, func(t *testing.T) {
			providers, err := telemetry.Setup(context.Background(), telemetry.Options{
				Enabled:  true,
				Endpoint: "collector:4317",
				Protocol: protocol,
				Insecure: true,
				Sampler:  telemetry.SamplerParentBasedAlwaysOn,
			})
			if err != nil {
				t.Fatalf("Setup() = %v", err)
			}
			if providers.TracerProvider == nil {
				t.Fatal("the service has no tracer provider")
			}

			if err := providers.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() = %v", err)
			}
		})
	}
}

func TestTheShutdownFlushesTheSpans(t *testing.T) {
	exporter := &recordingExporter{}

	providers, err := telemetry.Setup(context.Background(), telemetry.Options{
		Enabled:  true,
		Exporter: exporter,
	})
	if err != nil {
		t.Fatalf("Setup() = %v", err)
	}

	_, span := providers.TracerProvider.Tracer("test").Start(context.Background(), "a span")
	span.End()

	if err := providers.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() = %v", err)
	}

	if spans := exporter.Spans(); len(spans) != 1 {
		t.Fatalf("the exporter holds %d spans, want the one of the shutdown", len(spans))
	}
}

func TestTheShutdownOfAServiceThatDoesNotTraceIsQuiet(t *testing.T) {
	providers, err := telemetry.Setup(context.Background(), telemetry.Options{Enabled: false})
	if err != nil {
		t.Fatalf("Setup() = %v", err)
	}

	if err := providers.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() = %v", err)
	}
}

func TestThePropagatorCarriesTheTraceContext(t *testing.T) {
	if _, err := telemetry.Setup(context.Background(), telemetry.Options{Enabled: false}); err != nil {
		t.Fatalf("Setup() = %v", err)
	}

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), spanContext)

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	if _, ok := carrier["traceparent"]; !ok {
		t.Fatalf("the carrier holds %v, want a trace context", carrier)
	}
}

func TestStoppingTheTelemetryOfAServiceThatDoesNotTraceIsClean(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	providers, err := telemetry.Setup(context.Background(), telemetry.Options{})
	if err != nil {
		t.Fatalf("Setup() = %v", err)
	}

	telemetry.Stop(logger, providers)

	if logs.Len() != 0 {
		t.Fatalf("the log of the shutdown = %q, want nothing", logs.String())
	}
}
