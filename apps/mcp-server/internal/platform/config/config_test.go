package config_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/platform/config"
)

// clearEnvironment removes the settings of the environment, so the test sees
// the defaults of the server.
func clearEnvironment(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		config.EnvHost,
		config.EnvPort,
		config.EnvShutdownTimeout,
		config.EnvCheckTimeout,
		config.EnvMaxRequestBytes,
		config.EnvLogLevel,
		config.EnvServiceVersion,
		config.EnvDatabaseURL,
		config.EnvDatabaseMaxConns,
		config.EnvDatabaseConnectTimeout,
		config.EnvOtelDisabled,
		config.EnvOtelEndpoint,
		config.EnvOtelProtocol,
		config.EnvOtelInsecure,
		config.EnvOtelSampler,
		config.EnvOtelSamplerArg,
		config.EnvOtelServiceName,
	} {
		if value, ok := os.LookupEnv(name); ok {
			if err := os.Unsetenv(name); err != nil {
				t.Fatalf("unset %s: %v", name, err)
			}
			t.Cleanup(func() {
				_ = os.Setenv(name, value)
			})
		}
	}
}

func TestLoadUsesTheSafeDefaults(t *testing.T) {
	clearEnvironment(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Host != config.DefaultHost {
		t.Fatalf("host = %q, want %q", cfg.Host, config.DefaultHost)
	}
	if cfg.Port != config.DefaultPort {
		t.Fatalf("port = %d, want %d", cfg.Port, config.DefaultPort)
	}
	if cfg.Addr() != "127.0.0.1:8081" {
		t.Fatalf("addr = %q, want %q", cfg.Addr(), "127.0.0.1:8081")
	}
	if cfg.ShutdownTimeout != config.DefaultShutdownTimeout {
		t.Fatalf("shutdown timeout = %s, want %s", cfg.ShutdownTimeout, config.DefaultShutdownTimeout)
	}
	if cfg.CheckTimeout != config.DefaultCheckTimeout {
		t.Fatalf("check timeout = %s, want %s", cfg.CheckTimeout, config.DefaultCheckTimeout)
	}
	if cfg.MaxRequestBytes != config.DefaultMaxRequestBytes {
		t.Fatalf("max request bytes = %d, want %d", cfg.MaxRequestBytes, config.DefaultMaxRequestBytes)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("log level = %s, want %s", cfg.LogLevel, slog.LevelInfo)
	}
}

func TestLoadReadsTheEnvironment(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvHost, "0.0.0.0")
	t.Setenv(config.EnvPort, "9000")
	t.Setenv(config.EnvShutdownTimeout, "30s")
	t.Setenv(config.EnvCheckTimeout, "5s")
	t.Setenv(config.EnvMaxRequestBytes, "1024")
	t.Setenv(config.EnvLogLevel, "debug")
	t.Setenv(config.EnvServiceVersion, "1.2.3")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Addr() != "0.0.0.0:9000" {
		t.Fatalf("addr = %q, want %q", cfg.Addr(), "0.0.0.0:9000")
	}
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Fatalf("shutdown timeout = %s, want %s", cfg.ShutdownTimeout, 30*time.Second)
	}
	if cfg.CheckTimeout != 5*time.Second {
		t.Fatalf("check timeout = %s, want %s", cfg.CheckTimeout, 5*time.Second)
	}
	if cfg.MaxRequestBytes != 1024 {
		t.Fatalf("max request bytes = %d, want %d", cfg.MaxRequestBytes, 1024)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("log level = %s, want %s", cfg.LogLevel, slog.LevelDebug)
	}
	if cfg.ServiceVersion != "1.2.3" {
		t.Fatalf("version = %q, want %q", cfg.ServiceVersion, "1.2.3")
	}
}

func TestLoadReadsTheSettingsOfTheDatabase(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvDatabaseURL, "postgres://paca:paca@postgres:5432/paca?sslmode=disable")
	t.Setenv(config.EnvDatabaseMaxConns, "8")
	t.Setenv(config.EnvDatabaseConnectTimeout, "3s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Database.URL != "postgres://paca:paca@postgres:5432/paca?sslmode=disable" {
		t.Fatalf("database url = %q, want the connection string of the environment", cfg.Database.URL)
	}
	if cfg.Database.MaxConns != 8 {
		t.Fatalf("database max conns = %d, want %d", cfg.Database.MaxConns, 8)
	}
	if cfg.Database.ConnectTimeout != 3*time.Second {
		t.Fatalf("database connect timeout = %s, want %s", cfg.Database.ConnectTimeout, 3*time.Second)
	}
}

func TestLoadRunsWithoutADatabase(t *testing.T) {
	clearEnvironment(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Database.URL != "" {
		t.Fatalf("database url = %q, want it to be empty", cfg.Database.URL)
	}
	if cfg.Database.MaxConns != config.DefaultDatabaseMaxConns {
		t.Fatalf("database max conns = %d, want %d", cfg.Database.MaxConns, config.DefaultDatabaseMaxConns)
	}
	if cfg.Database.ConnectTimeout != config.DefaultDatabaseConnectTimeout {
		t.Fatalf("database connect timeout = %s, want %s",
			cfg.Database.ConnectTimeout, config.DefaultDatabaseConnectTimeout)
	}
}

func TestLoadRejectsAnInvalidSetting(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "a port that is not a number", key: config.EnvPort, value: "eighty"},
		{name: "a port out of range", key: config.EnvPort, value: "70000"},
		{name: "a port that is zero", key: config.EnvPort, value: "0"},
		{name: "an empty host", key: config.EnvHost, value: "  "},
		{name: "a timeout that is not a duration", key: config.EnvShutdownTimeout, value: "soon"},
		{name: "a negative timeout", key: config.EnvCheckTimeout, value: "-1s"},
		{name: "a body limit that is not a number", key: config.EnvMaxRequestBytes, value: "lots"},
		{name: "a body limit that is zero", key: config.EnvMaxRequestBytes, value: "0"},
		{name: "a log level that does not exist", key: config.EnvLogLevel, value: "loud"},
		{name: "an empty version", key: config.EnvServiceVersion, value: " "},
		{name: "an empty connection string", key: config.EnvDatabaseURL, value: " "},
		{name: "a number of connections that is not a number", key: config.EnvDatabaseMaxConns, value: "many"},
		{name: "a number of connections that is zero", key: config.EnvDatabaseMaxConns, value: "0"},
		{name: "a negative number of connections", key: config.EnvDatabaseMaxConns, value: "-1"},
		{name: "a negative timeout to the database", key: config.EnvDatabaseConnectTimeout, value: "-1s"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearEnvironment(t)
			t.Setenv(test.key, test.value)

			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() with %s=%q = nil, want an error", test.key, test.value)
			}
		})
	}
}

func TestLoadKeepsTheTracingOffWithoutACollector(t *testing.T) {
	clearEnvironment(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Tracing.Enabled {
		t.Fatal("the tracing is on, want it off without a collector")
	}
	if cfg.Tracing.Protocol != config.DefaultOtelProtocol {
		t.Fatalf("protocol = %q, want %q", cfg.Tracing.Protocol, config.DefaultOtelProtocol)
	}
	if cfg.Tracing.Sampler != config.DefaultOtelSampler {
		t.Fatalf("sampler = %q, want %q", cfg.Tracing.Sampler, config.DefaultOtelSampler)
	}
}

func TestLoadTurnsTheTracingOnWithACollector(t *testing.T) {
	clearEnvironment(t)
	t.Setenv(config.EnvOtelEndpoint, "jaeger:4317")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if !cfg.Tracing.Enabled {
		t.Fatal("the tracing is off, want it on with a collector")
	}
	if cfg.Tracing.Endpoint != "jaeger:4317" {
		t.Fatalf("endpoint = %q, want %q", cfg.Tracing.Endpoint, "jaeger:4317")
	}
}

func TestLoadTurnsTheTracingOffWhenTheSpecificationDoes(t *testing.T) {
	clearEnvironment(t)
	t.Setenv(config.EnvOtelEndpoint, "jaeger:4317")
	t.Setenv(config.EnvOtelDisabled, "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Tracing.Enabled {
		t.Fatal("the tracing is on, want it off")
	}
}

func TestLoadReadsTheSettingsOfTheExporter(t *testing.T) {
	clearEnvironment(t)
	t.Setenv(config.EnvOtelEndpoint, "http://jaeger:4318")
	t.Setenv(config.EnvOtelProtocol, "http/protobuf")
	t.Setenv(config.EnvOtelInsecure, "true")
	t.Setenv(config.EnvOtelSampler, config.SamplerParentBasedTraceIDRatio)
	t.Setenv(config.EnvOtelSamplerArg, "0.25")
	t.Setenv(config.EnvOtelServiceName, "paca-mcp-server-canary")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Tracing.Protocol != "http/protobuf" {
		t.Fatalf("protocol = %q, want %q", cfg.Tracing.Protocol, "http/protobuf")
	}
	if !cfg.Tracing.Insecure {
		t.Fatal("the exporter is not insecure, want it insecure")
	}
	if cfg.Tracing.Sampler != config.SamplerParentBasedTraceIDRatio {
		t.Fatalf("sampler = %q, want %q",
			cfg.Tracing.Sampler, config.SamplerParentBasedTraceIDRatio)
	}
	if cfg.Tracing.SamplerArg != "0.25" {
		t.Fatalf("sampler argument = %q, want %q", cfg.Tracing.SamplerArg, "0.25")
	}
	if cfg.Tracing.ServiceName != "paca-mcp-server-canary" {
		t.Fatalf("service name = %q, want %q",
			cfg.Tracing.ServiceName, "paca-mcp-server-canary")
	}
}

func TestLoadRejectsTheTracingSettingsThatWouldNotWork(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "a protocol that does not exist", key: config.EnvOtelProtocol, value: "smoke signals"},
		{name: "an address that is not an address", key: config.EnvOtelEndpoint, value: "jaeger"},
		{name: "an address with a scheme the protocol does not take",
			key: config.EnvOtelEndpoint, value: "http://jaeger:4317"},
		{name: "a sampler that does not exist", key: config.EnvOtelSampler, value: "whenever"},
		{name: "a ratio that is not a number", key: config.EnvOtelSamplerArg, value: "half"},
		{name: "a ratio out of bounds", key: config.EnvOtelSamplerArg, value: "1.5"},
		{name: "a boolean that is not a boolean", key: config.EnvOtelInsecure, value: "maybe"},
		{name: "an empty name", key: config.EnvOtelServiceName, value: " "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearEnvironment(t)
			t.Setenv(config.EnvOtelEndpoint, "jaeger:4317")
			t.Setenv(test.key, test.value)

			if _, err := config.Load(); err == nil {
				t.Fatalf("Load() with %s=%q = nil, want an error", test.key, test.value)
			}
		})
	}
}
