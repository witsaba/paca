package telemetry_test

import (
	"testing"

	"github.com/witsaba/paca/packages/platform/telemetry"
)

// environment is the environment of a service, as the tests see it.
//
// The platform reads no environment variable: the service hands it the
// reader, and the reader is the one that knows where the variables live.
type environment map[string]string

// lookup reads one variable of the environment.
func (e environment) lookup(name string) (string, bool) {
	value, ok := e[name]
	return value, ok
}

func TestLoadTracingKeepsTheTracingOffWithoutACollector(t *testing.T) {
	tracing, err := telemetry.LoadTracing(environment{}.lookup)
	if err != nil {
		t.Fatalf("LoadTracing() = %v", err)
	}

	if tracing.Enabled {
		t.Fatal("the tracing is on, want it off without a collector")
	}
	if tracing.Protocol != telemetry.DefaultProtocol {
		t.Fatalf("protocol = %q, want %q", tracing.Protocol, telemetry.DefaultProtocol)
	}
	if tracing.Sampler != telemetry.DefaultSampler {
		t.Fatalf("sampler = %q, want %q", tracing.Sampler, telemetry.DefaultSampler)
	}
}

func TestLoadTracingTurnsTheTracingOnWithACollector(t *testing.T) {
	tracing, err := telemetry.LoadTracing(environment{
		telemetry.EnvEndpoint: "collector:4317",
	}.lookup)
	if err != nil {
		t.Fatalf("LoadTracing() = %v", err)
	}

	if !tracing.Enabled {
		t.Fatal("the tracing is off, want it on with a collector")
	}
	if tracing.Endpoint != "collector:4317" {
		t.Fatalf("endpoint = %q, want %q", tracing.Endpoint, "collector:4317")
	}
}

func TestLoadTracingTurnsTheTracingOffWhenTheSpecificationDoes(t *testing.T) {
	tracing, err := telemetry.LoadTracing(environment{
		telemetry.EnvEndpoint: "collector:4317",
		telemetry.EnvDisabled: "true",
	}.lookup)
	if err != nil {
		t.Fatalf("LoadTracing() = %v", err)
	}

	if tracing.Enabled {
		t.Fatal("the tracing is on, want it off")
	}
}

func TestLoadTracingReadsTheSettingsOfTheExporter(t *testing.T) {
	tracing, err := telemetry.LoadTracing(environment{
		telemetry.EnvEndpoint:    "http://collector:4318",
		telemetry.EnvProtocol:    "http/protobuf",
		telemetry.EnvInsecure:    "true",
		telemetry.EnvSampler:     telemetry.SamplerParentBasedTraceIDRatio,
		telemetry.EnvSamplerArg:  "0.25",
		telemetry.EnvServiceName: "paca-canary",
	}.lookup)
	if err != nil {
		t.Fatalf("LoadTracing() = %v", err)
	}

	if tracing.Protocol != "http/protobuf" {
		t.Fatalf("protocol = %q, want %q", tracing.Protocol, "http/protobuf")
	}
	if !tracing.Insecure {
		t.Fatal("the exporter is not insecure, want it insecure")
	}
	if tracing.Sampler != telemetry.SamplerParentBasedTraceIDRatio {
		t.Fatalf("sampler = %q, want %q",
			tracing.Sampler, telemetry.SamplerParentBasedTraceIDRatio)
	}
	if tracing.SamplerArg != "0.25" {
		t.Fatalf("sampler argument = %q, want %q", tracing.SamplerArg, "0.25")
	}
	if tracing.ServiceName != "paca-canary" {
		t.Fatalf("service name = %q, want %q", tracing.ServiceName, "paca-canary")
	}
}

func TestLoadTracingRejectsTheSettingsThatWouldNotWork(t *testing.T) {
	tests := []struct {
		name    string
		env     environment
		message string
	}{
		{
			name: "a protocol that does not exist",
			env: environment{
				telemetry.EnvEndpoint: "collector:4317",
				telemetry.EnvProtocol: "smoke signals",
			},
			message: `OTEL_EXPORTER_OTLP_PROTOCOL: "smoke signals" is not a protocol`,
		},
		{
			name: "an address that is not an address",
			env: environment{
				telemetry.EnvEndpoint: "collector",
			},
			message: `OTEL_EXPORTER_OTLP_ENDPOINT: "collector" is not an address`,
		},
		{
			name: "an address with a scheme the protocol does not take",
			env: environment{
				telemetry.EnvEndpoint: "http://collector:4317",
			},
			message: `OTEL_EXPORTER_OTLP_ENDPOINT: "http://collector:4317" has a scheme, ` +
				`which the grpc protocol does not take`,
		},
		{
			name: "a sampler that does not exist",
			env: environment{
				telemetry.EnvEndpoint: "collector:4317",
				telemetry.EnvSampler:  "whenever",
			},
			message: `OTEL_TRACES_SAMPLER: "whenever" is not a sampler`,
		},
		{
			name: "a ratio that is not a number",
			env: environment{
				telemetry.EnvEndpoint:   "collector:4317",
				telemetry.EnvSamplerArg: "half",
			},
			message: `OTEL_TRACES_SAMPLER_ARG: "half" is not a ratio`,
		},
		{
			name: "a ratio out of bounds",
			env: environment{
				telemetry.EnvEndpoint:   "collector:4317",
				telemetry.EnvSamplerArg: "1.5",
			},
			message: `OTEL_TRACES_SAMPLER_ARG: "1.5" is not a ratio between zero and one`,
		},
		{
			name: "a boolean that is not a boolean",
			env: environment{
				telemetry.EnvEndpoint: "collector:4317",
				telemetry.EnvInsecure: "maybe",
			},
			message: `OTEL_EXPORTER_OTLP_INSECURE: "maybe" is not a boolean`,
		},
		{
			name: "a boolean of the specification that is not a boolean",
			env: environment{
				telemetry.EnvEndpoint: "collector:4317",
				telemetry.EnvDisabled: "perhaps",
			},
			message: `OTEL_SDK_DISABLED: "perhaps" is not a boolean`,
		},
		{
			name: "an empty address",
			env: environment{
				telemetry.EnvEndpoint: " ",
			},
			message: "OTEL_EXPORTER_OTLP_ENDPOINT: the address cannot be empty",
		},
		{
			name: "an empty name",
			env: environment{
				telemetry.EnvEndpoint:    "collector:4317",
				telemetry.EnvServiceName: " ",
			},
			message: "OTEL_SERVICE_NAME: the name cannot be empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := telemetry.LoadTracing(test.env.lookup)
			if err == nil {
				t.Fatalf("LoadTracing() = nil, want the error %q", test.message)
			}
			if err.Error() != test.message {
				t.Fatalf("LoadTracing() = %q, want %q", err.Error(), test.message)
			}
		})
	}
}

func TestTheValidationOfTheTracingRejectsTheSettingsThatWouldNotWork(t *testing.T) {
	tracing := telemetry.TracingConfig{
		Protocol: telemetry.DefaultProtocol,
		Sampler:  telemetry.DefaultSampler,
	}

	if err := tracing.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	tracing.Protocol = "smoke signals"

	err := tracing.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want an error about the protocol")
	}
	want := `OTEL_EXPORTER_OTLP_PROTOCOL: "smoke signals" is not a protocol`
	if err.Error() != want {
		t.Fatalf("Validate() = %q, want %q", err.Error(), want)
	}
}

func TestTheOptionsOfTheTelemetryComeFromTheConfigurationOfTheTracing(t *testing.T) {
	tracing := telemetry.TracingConfig{
		Enabled:    true,
		Endpoint:   "collector:4317",
		Protocol:   telemetry.ProtocolHTTP,
		Insecure:   true,
		Sampler:    telemetry.SamplerTraceIDRatio,
		SamplerArg: "0.5",
	}

	options := telemetry.OptionsOf(tracing, "paca-mcp-server", "1.2.3")

	if !options.Enabled {
		t.Error("the telemetry is off, want it on")
	}
	if options.ServiceName != "paca-mcp-server" {
		t.Errorf("service name = %q, want %q", options.ServiceName, "paca-mcp-server")
	}
	if options.ServiceVersion != "1.2.3" {
		t.Errorf("service version = %q, want %q", options.ServiceVersion, "1.2.3")
	}
	if options.Endpoint != "collector:4317" {
		t.Errorf("endpoint = %q, want %q", options.Endpoint, "collector:4317")
	}
	if options.Protocol != telemetry.ProtocolHTTP {
		t.Errorf("protocol = %q, want %q", options.Protocol, telemetry.ProtocolHTTP)
	}
	if !options.Insecure {
		t.Error("the exporter is not insecure, want it insecure")
	}
	if options.Sampler != telemetry.SamplerTraceIDRatio {
		t.Errorf("sampler = %q, want %q", options.Sampler, telemetry.SamplerTraceIDRatio)
	}
	if options.SamplerArg != "0.5" {
		t.Errorf("sampler argument = %q, want %q", options.SamplerArg, "0.5")
	}
}

func TestTheNameOfTheTelemetryIsTheNameOfTheServiceWhenTheEnvironmentNamesNone(t *testing.T) {
	tracing := telemetry.TracingConfig{}

	name := tracing.ResolveServiceName("paca-db-admin")

	if name != "paca-db-admin" {
		t.Fatalf("name = %q, want %q", name, "paca-db-admin")
	}
}

func TestTheNameOfTheTelemetryIsTheNameOfTheEnvironmentWhenItNamesOne(t *testing.T) {
	tracing := telemetry.TracingConfig{ServiceName: "renamed-by-the-operator"}

	name := tracing.ResolveServiceName("paca-db-admin")

	if name != "renamed-by-the-operator" {
		t.Fatalf("name = %q, want %q", name, "renamed-by-the-operator")
	}
}
