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
