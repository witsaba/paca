package telemetry

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/witsaba/paca/packages/platform/env"
)

// The environment variables of the tracing. They are the ones of the
// OpenTelemetry specification, so an operator who knows the specification
// knows how to turn the tracing on and off.
const (
	// EnvDisabled turns the tracing off.
	EnvDisabled = "OTEL_SDK_DISABLED"
	// EnvEndpoint is the address of the collector that receives the
	// traces.
	EnvEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
	// EnvProtocol is the protocol the exporter speaks to the
	// collector.
	EnvProtocol = "OTEL_EXPORTER_OTLP_PROTOCOL"
	// EnvInsecure tells whether the exporter speaks to the collector
	// without TLS.
	EnvInsecure = "OTEL_EXPORTER_OTLP_INSECURE"
	// EnvSampler is the sampler that decides which traces are kept.
	EnvSampler = "OTEL_TRACES_SAMPLER"
	// EnvSamplerArg is the argument of the sampler: the ratio of the
	// traces that are kept.
	EnvSamplerArg = "OTEL_TRACES_SAMPLER_ARG"
	// EnvServiceName is the name the telemetry carries. It overrides
	// the name of the service when it is set.
	EnvServiceName = "OTEL_SERVICE_NAME"
)

// The defaults of the tracing.
const (
	// DefaultProtocol is the protocol the exporter speaks to the
	// collector. It is the protocol the collector of the monorepository
	// listens on.
	DefaultProtocol = ProtocolGRPC
	// DefaultSampler keeps every trace. The traces of a service that
	// is not under load are few, and a trace that is missing cannot be
	// recovered.
	DefaultSampler = SamplerParentBasedAlwaysOn
)

// TracingConfig holds the settings of the tracing.
type TracingConfig struct {
	// Enabled tells whether the service traces. A service without a
	// collector does not trace: there is nowhere to send the traces.
	Enabled bool
	// ServiceName is the name the telemetry carries. It is empty when the
	// name of the service is the name of the telemetry.
	ServiceName string
	// Endpoint is the address of the collector that receives the traces.
	Endpoint string
	// Protocol is the protocol the exporter speaks to the collector.
	Protocol string
	// Insecure tells whether the exporter speaks to the collector without
	// TLS.
	Insecure bool
	// Sampler is the name of the sampler that decides which traces are
	// kept.
	Sampler string
	// SamplerArg is the argument of the sampler: the ratio of the traces
	// that are kept.
	SamplerArg string
}

// OptionsOf builds the options of the telemetry from the configuration of a
// service.
//
// The configuration of the tracing is the same in every service of the
// monorepository, and the options of the telemetry have the same shape
// everywhere: the service hands its configuration, its name and its version
// over, and the platform builds what the setup needs. A service that writes
// the options itself writes them twice.
func OptionsOf(config TracingConfig, serviceName, serviceVersion string) Options {
	return Options{
		Enabled:        config.Enabled,
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Endpoint:       config.Endpoint,
		Protocol:       config.Protocol,
		Insecure:       config.Insecure,
		Sampler:        config.Sampler,
		SamplerArg:     config.SamplerArg,
	}
}

// ResolveServiceName returns the name the telemetry carries.
//
// It is the name of the service, unless the environment of the
// OpenTelemetry specification names it otherwise: an operator that renames
// the service in the collector sees the traces under that name.
func (t TracingConfig) ResolveServiceName(serviceName string) string {
	if t.ServiceName != "" {
		return t.ServiceName
	}
	return serviceName
}

// LoadTracing reads the settings of the tracing from the environment of the
// service.
//
// The tracing is on when a collector is configured and the specification
// does not turn it off. A service without a collector does not trace: there
// is nowhere to send the traces, and a service that tries anyway only fills
// its own log with connection errors.
//
// The platform reads no environment variable: the service hands the reader
// of its own environment over, and the reader is the one that knows where
// the variables live.
//
// A setting that would not work is an error, never a silent fallback.
func LoadTracing(lookup env.Lookup) (TracingConfig, error) {
	tracing := TracingConfig{
		Protocol: DefaultProtocol,
		Sampler:  DefaultSampler,
	}

	if raw, ok := lookup(EnvEndpoint); ok {
		endpoint := strings.TrimSpace(raw)
		if endpoint == "" {
			return TracingConfig{}, fmt.Errorf("%s: the address cannot be empty", EnvEndpoint)
		}
		tracing.Endpoint = endpoint
	}

	if raw, ok := lookup(EnvProtocol); ok {
		tracing.Protocol = strings.TrimSpace(raw)
	}

	if raw, ok := lookup(EnvInsecure); ok {
		insecure, err := parseBool(EnvInsecure, raw)
		if err != nil {
			return TracingConfig{}, err
		}
		tracing.Insecure = insecure
	}

	if raw, ok := lookup(EnvSampler); ok {
		tracing.Sampler = strings.TrimSpace(raw)
	}

	if raw, ok := lookup(EnvSamplerArg); ok {
		tracing.SamplerArg = strings.TrimSpace(raw)
	}

	if raw, ok := lookup(EnvServiceName); ok {
		name := strings.TrimSpace(raw)
		if name == "" {
			return TracingConfig{}, fmt.Errorf("%s: the name cannot be empty", EnvServiceName)
		}
		tracing.ServiceName = name
	}

	disabled := false
	if raw, ok := lookup(EnvDisabled); ok {
		value, err := parseBool(EnvDisabled, raw)
		if err != nil {
			return TracingConfig{}, err
		}
		disabled = value
	}

	tracing.Enabled = !disabled && tracing.Endpoint != ""

	if err := tracing.Validate(); err != nil {
		return TracingConfig{}, err
	}

	return tracing, nil
}

// Validate rejects the tracing settings that would not work.
func (t TracingConfig) Validate() error {
	switch t.Protocol {
	case ProtocolGRPC, ProtocolHTTP:
	default:
		return fmt.Errorf("%s: %q is not a protocol", EnvProtocol, t.Protocol)
	}

	if t.Endpoint != "" {
		if err := validateEndpoint(t.Endpoint, t.Protocol); err != nil {
			return err
		}
	}

	switch t.Sampler {
	case SamplerAlwaysOn, SamplerAlwaysOff, SamplerParentBasedAlwaysOn:
	case SamplerTraceIDRatio, SamplerParentBasedTraceIDRatio:
	default:
		return fmt.Errorf("%s: %q is not a sampler", EnvSampler, t.Sampler)
	}

	// A ratio that cannot be read is an error even when the sampler does
	// not take one: a setting that is silently ignored is a setting that
	// nobody read.
	if t.Sampler == SamplerTraceIDRatio || t.Sampler == SamplerParentBasedTraceIDRatio ||
		t.SamplerArg != "" {
		if _, err := parseRatio(EnvSamplerArg, t.SamplerArg); err != nil {
			return err
		}
	}

	return nil
}

// validateEndpoint checks the address of the collector.
//
// The exporter takes the host and the port of the collector. The protocol
// over HTTP also takes a full address.
func validateEndpoint(endpoint, protocol string) error {
	if strings.Contains(endpoint, "://") {
		if protocol != ProtocolHTTP {
			return fmt.Errorf("%s: %q has a scheme, which the %s protocol does not take",
				EnvEndpoint, endpoint, protocol)
		}
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return fmt.Errorf("%s: %q is not an address", EnvEndpoint, endpoint)
		}
		if parsed.Host == "" {
			return fmt.Errorf("%s: %q has no host", EnvEndpoint, endpoint)
		}
		return nil
	}

	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("%s: %q is not an address", EnvEndpoint, endpoint)
	}
	if host == "" || port == "" {
		return fmt.Errorf("%s: %q has no host or no port", EnvEndpoint, endpoint)
	}

	return nil
}

// parseBool reads a boolean from the environment.
func parseBool(name, raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("%s: %q is not a boolean", name, raw)
	}
}

// parseRatio reads the ratio of the traces that are kept.
//
// The name is the one the error carries, so an operator knows which setting
// to fix: the name of the variable when the ratio comes from the
// environment, and the name of the setting when it comes from the options of
// the telemetry.
func parseRatio(name, raw string) (float64, error) {
	ratio, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a ratio", name, raw)
	}
	if ratio < 0 || ratio > 1 {
		return 0, fmt.Errorf("%s: %q is not a ratio between zero and one", name, raw)
	}
	return ratio, nil
}
