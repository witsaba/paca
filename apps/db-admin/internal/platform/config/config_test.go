package config_test

import (
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/platform/config"
	"github.com/witsaba/paca/packages/platform/env"
	"github.com/witsaba/paca/packages/platform/logging"
	"github.com/witsaba/paca/packages/platform/telemetry"
)

// clearEnvironment removes the settings of the environment, so the test sees
// the defaults of the service.
func clearEnvironment(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		config.EnvHost,
		config.EnvPort,
		config.EnvShutdownTimeout,
		config.EnvCheckTimeout,
		config.EnvDatabaseWaitTimeout,
		logging.Level,
		env.ServiceVersion,
		config.EnvDatabaseURL,
		config.EnvDatabaseHost,
		config.EnvDatabasePort,
		config.EnvPostgresUser,
		config.EnvPostgresPassword,
		config.EnvPostgresDB,
		telemetry.EnvDisabled,
		telemetry.EnvEndpoint,
		telemetry.EnvProtocol,
		telemetry.EnvInsecure,
		telemetry.EnvSampler,
		telemetry.EnvSamplerArg,
		telemetry.EnvServiceName,
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

// setCredentials names a database the way the stack of PostgreSQL does.
func setCredentials(t *testing.T) {
	t.Helper()

	t.Setenv(config.EnvPostgresUser, "paca")
	t.Setenv(config.EnvPostgresPassword, "secret")
	t.Setenv(config.EnvPostgresDB, "paca")
}

func TestLoadUsesTheSafeDefaults(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

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
	if cfg.Addr() != "127.0.0.1:8082" {
		t.Fatalf("addr = %q, want %q", cfg.Addr(), "127.0.0.1:8082")
	}
	if cfg.ShutdownTimeout != config.DefaultShutdownTimeout {
		t.Fatalf("shutdown timeout = %s, want %s", cfg.ShutdownTimeout, config.DefaultShutdownTimeout)
	}
	if cfg.CheckTimeout != config.DefaultCheckTimeout {
		t.Fatalf("check timeout = %s, want %s", cfg.CheckTimeout, config.DefaultCheckTimeout)
	}
	if cfg.Database.WaitTimeout != config.DefaultDatabaseWaitTimeout {
		t.Fatalf("database wait timeout = %s, want %s", cfg.Database.WaitTimeout, config.DefaultDatabaseWaitTimeout)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("log level = %s, want %s", cfg.LogLevel, slog.LevelInfo)
	}
	if cfg.Tracing.Enabled {
		t.Fatal("the service traces, want no tracing without a collector")
	}
}

func TestLoadReadsTheEnvironment(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvHost, "0.0.0.0")
	t.Setenv(config.EnvPort, "9000")
	t.Setenv(config.EnvShutdownTimeout, "30s")
	t.Setenv(config.EnvCheckTimeout, "5s")
	t.Setenv(logging.Level, "debug")
	t.Setenv(env.ServiceVersion, "1.2.3")

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
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("log level = %s, want %s", cfg.LogLevel, slog.LevelDebug)
	}
	if cfg.ServiceVersion != "1.2.3" {
		t.Fatalf("version = %q, want %q", cfg.ServiceVersion, "1.2.3")
	}
}

func TestLoadUsesTheConnectionStringAsItIs(t *testing.T) {
	clearEnvironment(t)

	const connection = "postgres://paca:secret@elsewhere:5432/paca?sslmode=disable"
	t.Setenv(config.EnvDatabaseURL, connection)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Database.URL != connection {
		t.Fatalf("database url = %q, want %q", cfg.Database.URL, connection)
	}
}

func TestLoadUsesTheConnectionStringOverTheCredentials(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvDatabaseURL, "postgres://paca:secret@elsewhere:5432/paca?sslmode=disable")
	t.Setenv(config.EnvPostgresUser, "other")
	t.Setenv(config.EnvPostgresPassword, "other-secret")
	t.Setenv(config.EnvPostgresDB, "other")
	t.Setenv(config.EnvDatabaseHost, "other-host")
	t.Setenv(config.EnvDatabasePort, "6543")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	want := "postgres://paca:secret@elsewhere:5432/paca?sslmode=disable"
	if cfg.Database.URL != want {
		t.Fatalf("database url = %q, want %q", cfg.Database.URL, want)
	}
}

func TestLoadReadsNothingElseWhenTheConnectionStringIsSet(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvDatabaseURL, "postgres://paca:secret@elsewhere:5432/paca?sslmode=disable")
	t.Setenv(config.EnvDatabasePort, "not-a-port")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	want := "postgres://paca:secret@elsewhere:5432/paca?sslmode=disable"
	if cfg.Database.URL != want {
		t.Fatalf("database url = %q, want %q", cfg.Database.URL, want)
	}
}

func TestLoadAssemblesTheConnectionStringFromTheCredentials(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	want := "postgres://paca:secret@postgres:5432/paca?sslmode=disable"
	if cfg.Database.URL != want {
		t.Fatalf("database url = %q, want %q", cfg.Database.URL, want)
	}
}

func TestLoadReadsTheHostAndThePortOfTheDatabase(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvDatabaseHost, "database.internal")
	t.Setenv(config.EnvDatabasePort, "6543")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	want := "postgres://paca:secret@database.internal:6543/paca?sslmode=disable"
	if cfg.Database.URL != want {
		t.Fatalf("database url = %q, want %q", cfg.Database.URL, want)
	}
}

func TestLoadEscapesThePasswordOfTheDatabase(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvPostgresUser, "paca")
	t.Setenv(config.EnvPostgresPassword, "p@ss word")
	t.Setenv(config.EnvPostgresDB, "paca")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if !strings.Contains(cfg.Database.URL, "p%40ss%20word") {
		t.Fatalf("database url = %q, want the password to be escaped", cfg.Database.URL)
	}
}

func TestLoadKeepsTheSpacesOfThePassword(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvPostgresUser, "paca")
	t.Setenv(config.EnvPostgresPassword, " s3cr3t ")
	t.Setenv(config.EnvPostgresDB, "paca")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if !strings.Contains(cfg.Database.URL, "s3cr3t%20") {
		t.Fatalf("database url = %q, want the spaces of the password to be kept", cfg.Database.URL)
	}

	parsed, err := url.Parse(cfg.Database.URL)
	if err != nil {
		t.Fatalf("parse %q: %v", cfg.Database.URL, err)
	}
	password, ok := parsed.User.Password()
	if !ok {
		t.Fatalf("database url %q carries no password", cfg.Database.URL)
	}
	if password != " s3cr3t " {
		t.Fatalf("password = %q, want %q", password, " s3cr3t ")
	}
}

func TestLoadFailsWithoutADatabase(t *testing.T) {
	clearEnvironment(t)

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the database")
	}
}

func TestLoadFailsWhenTheCredentialsAreIncomplete(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvPostgresUser, "paca")
	t.Setenv(config.EnvPostgresDB, "paca")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the password")
	}
}

func TestLoadFailsWhenTheConnectionStringIsEmpty(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvDatabaseURL, "   ")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the connection string")
	}
}

func TestLoadFailsWhenTheHostOfTheDatabaseIsEmpty(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvDatabaseHost, "  ")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the host of the database")
	}
}

func TestLoadFailsWhenThePortOfTheDatabaseIsNotANumber(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvDatabasePort, "the port of the database")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the port of the database")
	}
}

func TestLoadFailsWhenThePortOfTheDatabaseIsOutOfBounds(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvDatabasePort, "70000")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the port of the database")
	}
}

func TestLoadFailsWithAnInvalidHost(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvHost, "  ")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the host")
	}
}

func TestLoadFailsWithAnInvalidPort(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvPort, "eighty")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the port")
	}
}

func TestLoadFailsWithAnInvalidTimeout(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvCheckTimeout, "soon")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the timeout of the checks")
	}
}

func TestLoadReadsTheWaitTimeoutOfTheDatabase(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvDatabaseWaitTimeout, "90s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Database.WaitTimeout != 90*time.Second {
		t.Fatalf("database wait timeout = %s, want %s", cfg.Database.WaitTimeout, 90*time.Second)
	}
}

func TestLoadFailsWithAnInvalidWaitTimeoutOfTheDatabase(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvDatabaseWaitTimeout, "soon")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the timeout of the wait")
	}
}

func TestLoadFailsWithANegativeWaitTimeoutOfTheDatabase(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvDatabaseWaitTimeout, "-1s")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the timeout of the wait")
	}
}

func TestLoadFailsWithANegativeTimeout(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(config.EnvShutdownTimeout, "-1s")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the timeout of the shutdown")
	}
}

func TestLoadFailsWithAnInvalidLogLevel(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(logging.Level, "shout")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the level of the logs")
	}
}

func TestLoadFailsWithAnEmptyVersion(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(env.ServiceVersion, "  ")

	if _, err := config.Load(); err == nil {
		t.Fatal("Load() = nil, want an error about the version")
	}
}

func TestLoadReadsTheTracing(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(telemetry.EnvEndpoint, "collector:4317")
	t.Setenv(telemetry.EnvProtocol, "grpc")
	t.Setenv(telemetry.EnvInsecure, "true")
	t.Setenv(telemetry.EnvSampler, telemetry.SamplerAlwaysOn)
	t.Setenv(telemetry.EnvSamplerArg, "0.5")
	t.Setenv(telemetry.EnvServiceName, "paca-db-admin-test")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if !cfg.Tracing.Enabled {
		t.Fatal("the service does not trace, want it to trace")
	}
	if cfg.Tracing.Endpoint != "collector:4317" {
		t.Fatalf("collector = %q, want %q", cfg.Tracing.Endpoint, "collector:4317")
	}
	if !cfg.Tracing.Insecure {
		t.Fatal("the exporter wants TLS, want it to be insecure")
	}
	if cfg.Tracing.Sampler != telemetry.SamplerAlwaysOn {
		t.Fatalf("sampler = %q, want %q", cfg.Tracing.Sampler, telemetry.SamplerAlwaysOn)
	}
	if cfg.Tracing.SamplerArg != "0.5" {
		t.Fatalf("sampler arg = %q, want %q", cfg.Tracing.SamplerArg, "0.5")
	}
	if cfg.Tracing.ServiceName != "paca-db-admin-test" {
		t.Fatalf("name = %q, want %q", cfg.Tracing.ServiceName, "paca-db-admin-test")
	}
}

func TestLoadTurnsTheTracingOff(t *testing.T) {
	clearEnvironment(t)
	setCredentials(t)

	t.Setenv(telemetry.EnvEndpoint, "collector:4317")
	t.Setenv(telemetry.EnvDisabled, "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.Tracing.Enabled {
		t.Fatal("the service traces, want the tracing to be off")
	}
}

func TestLoadRejectsAnInvalidTracing(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		message string
	}{
		{
			name:    "a protocol that does not exist",
			key:     telemetry.EnvProtocol,
			value:   "smoke signals",
			message: `OTEL_EXPORTER_OTLP_PROTOCOL: "smoke signals" is not a protocol`,
		},
		{
			name:    "a sampler that does not exist",
			key:     telemetry.EnvSampler,
			value:   "whenever",
			message: `OTEL_TRACES_SAMPLER: "whenever" is not a sampler`,
		},
		{
			name:    "a ratio out of bounds",
			key:     telemetry.EnvSamplerArg,
			value:   "1.5",
			message: `OTEL_TRACES_SAMPLER_ARG: "1.5" is not a ratio between zero and one`,
		},
		{
			name:    "a collector that is not an address",
			key:     telemetry.EnvEndpoint,
			value:   "collector",
			message: `OTEL_EXPORTER_OTLP_ENDPOINT: "collector" is not an address`,
		},
		{
			name:  "a collector that has a scheme",
			key:   telemetry.EnvEndpoint,
			value: "http://collector:4317",
			message: `OTEL_EXPORTER_OTLP_ENDPOINT: "http://collector:4317" has a scheme, ` +
				`which the grpc protocol does not take`,
		},
		{
			name:    "an empty collector",
			key:     telemetry.EnvEndpoint,
			value:   "  ",
			message: "OTEL_EXPORTER_OTLP_ENDPOINT: the address cannot be empty",
		},
		{
			name:    "an empty name of the tracing",
			key:     telemetry.EnvServiceName,
			value:   "  ",
			message: "OTEL_SERVICE_NAME: the name cannot be empty",
		},
		{
			name:    "an invalid boolean",
			key:     telemetry.EnvInsecure,
			value:   "perhaps",
			message: `OTEL_EXPORTER_OTLP_INSECURE: "perhaps" is not a boolean`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearEnvironment(t)
			setCredentials(t)

			t.Setenv(telemetry.EnvEndpoint, "collector:4317")
			t.Setenv(test.key, test.value)

			_, err := config.Load()
			if err == nil {
				t.Fatalf("Load() with %s=%q = nil, want an error", test.key, test.value)
			}
			if err.Error() != test.message {
				t.Fatalf("Load() with %s=%q = %q, want %q",
					test.key, test.value, err.Error(), test.message)
			}
		})
	}
}

func TestLoadPortReadsThePortOfTheServer(t *testing.T) {
	clearEnvironment(t)

	port, err := config.LoadPort()
	if err != nil {
		t.Fatalf("LoadPort() = %v", err)
	}
	if port != config.DefaultPort {
		t.Fatalf("port = %d, want %d", port, config.DefaultPort)
	}

	t.Setenv(config.EnvPort, "9000")

	port, err = config.LoadPort()
	if err != nil {
		t.Fatalf("LoadPort() = %v", err)
	}
	if port != 9000 {
		t.Fatalf("port = %d, want %d", port, 9000)
	}
}

func TestLoadPortNeedsNoDatabase(t *testing.T) {
	clearEnvironment(t)

	// The health check of the container asks the service whether it is
	// alive: it needs the port, and nothing else.
	if _, err := config.LoadPort(); err != nil {
		t.Fatalf("LoadPort() = %v", err)
	}
}

func TestLoadPortFailsWithAnInvalidPort(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvPort, "eighty")

	if _, err := config.LoadPort(); err == nil {
		t.Fatal("LoadPort() = nil, want an error about the port")
	}
}

func TestLoadPortFailsWithAPortOutOfBounds(t *testing.T) {
	clearEnvironment(t)

	t.Setenv(config.EnvPort, "70000")

	if _, err := config.LoadPort(); err == nil {
		t.Fatal("LoadPort() = nil, want an error about the port")
	}
}
