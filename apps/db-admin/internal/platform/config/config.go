// Package config reads and validates the configuration of the database admin
// service.
//
// Every setting comes from the environment, so the same binary runs in a
// container, on a laptop and in a test without a configuration file. The
// defaults are the safe ones: the server listens on the loopback interface
// unless it is told otherwise.
//
// The database is the one thing the service cannot do without: it is the
// writer of the schema, so a service without a database has nothing to do.
// The rules of where the credentials come from live here, and nowhere else:
// the core reads no environment variable.
package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	dbconfig "github.com/witsaba/paca/packages/database/config"
	"github.com/witsaba/paca/packages/platform/env"
	"github.com/witsaba/paca/packages/platform/logging"
)

// The environment variables the service reads.
const (
	// EnvHost is the interface the server binds to.
	EnvHost = "PACA_DB_ADMIN_HOST"
	// EnvPort is the port the server listens on.
	EnvPort = "PACA_DB_ADMIN_PORT"
	// EnvShutdownTimeout is how long the server waits for the requests in
	// flight when it is asked to stop.
	EnvShutdownTimeout = "PACA_DB_ADMIN_SHUTDOWN_TIMEOUT"
	// EnvCheckTimeout bounds how long a single health check may take.
	EnvCheckTimeout = "PACA_DB_ADMIN_CHECK_TIMEOUT"
)

// The environment variables of the database.
//
// The connection string is named in two ways, and the first one wins: the
// whole connection string, or the credentials of the PostgreSQL stack plus
// the host and the port of it.
const (
	// EnvDatabaseURL is the whole connection string of the database. When
	// it is set, it wins, and nothing else is read.
	EnvDatabaseURL = "PACA_DATABASE_URL"
	// EnvDatabaseHost is the host of the database, when the connection
	// string is assembled from the credentials.
	EnvDatabaseHost = "PACA_DATABASE_HOST"
	// EnvDatabasePort is the port of the database, when the connection
	// string is assembled from the credentials.
	EnvDatabasePort = "PACA_DATABASE_PORT"
	// EnvDatabaseWaitTimeout is how long the writer waits for the
	// database to answer when it starts.
	EnvDatabaseWaitTimeout = "PACA_DB_ADMIN_DATABASE_WAIT_TIMEOUT"
	// EnvPostgresUser is the user of the database. It is the variable the
	// stack of PostgreSQL writes its credentials to.
	EnvPostgresUser = "POSTGRES_USER"
	// EnvPostgresPassword is the password of the user of the database.
	EnvPostgresPassword = "POSTGRES_PASSWORD" //nolint:gosec // the name of the environment variable, not a credential.
	// EnvPostgresDB is the name of the database.
	EnvPostgresDB = "POSTGRES_DB"
)

// The environment variables of the tracing. They are the ones of the
// OpenTelemetry specification, so an operator who knows the specification
// knows how to turn the tracing on and off.
const (
	// EnvOtelDisabled turns the tracing off.
	EnvOtelDisabled = "OTEL_SDK_DISABLED"
	// EnvOtelEndpoint is the address of the collector that receives the
	// traces.
	EnvOtelEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
	// EnvOtelProtocol is the protocol the exporter speaks to the
	// collector.
	EnvOtelProtocol = "OTEL_EXPORTER_OTLP_PROTOCOL"
	// EnvOtelInsecure tells whether the exporter speaks to the collector
	// without TLS.
	EnvOtelInsecure = "OTEL_EXPORTER_OTLP_INSECURE"
	// EnvOtelSampler is the sampler that decides which traces are kept.
	EnvOtelSampler = "OTEL_TRACES_SAMPLER"
	// EnvOtelSamplerArg is the argument of the sampler: the ratio of the
	// traces that are kept.
	EnvOtelSamplerArg = "OTEL_TRACES_SAMPLER_ARG"
	// EnvOtelServiceName is the name the telemetry carries. It overrides
	// the name of the service when it is set.
	EnvOtelServiceName = "OTEL_SERVICE_NAME"
)

// The defaults of the settings.
const (
	// DefaultHost binds the loopback interface only. A container overrides
	// it, because the other containers of the stack reach the server
	// through the interface of the container.
	DefaultHost = "127.0.0.1"
	// DefaultPort is the port reserved for the database admin service of
	// the monorepository.
	DefaultPort = 8082
	// DefaultShutdownTimeout is the grace period of the shutdown.
	DefaultShutdownTimeout = 10 * time.Second
	// DefaultCheckTimeout bounds a single health check.
	DefaultCheckTimeout = 2 * time.Second
	// DefaultDatabaseHost is the name of the container of the database of
	// the monorepository.
	DefaultDatabaseHost = "postgres"
	// DefaultDatabasePort is the port of the database.
	DefaultDatabasePort = "5432"
	// DefaultDatabaseWaitTimeout is how long the writer waits for the
	// database to answer before it gives up.
	//
	// The database of the monorepository accepts the connections a couple
	// of seconds after its container starts, and the writer starts at the
	// same moment as the database. A minute is long enough for the
	// database to be ready, and short enough for a person to notice that
	// something is wrong.
	DefaultDatabaseWaitTimeout = 60 * time.Second
	// ServiceName is the name the service reports in its health report.
	ServiceName = "paca-db-admin"
	// DefaultOtelProtocol is the protocol the exporter speaks to the
	// collector. It is the protocol the collector of the monorepository
	// listens on.
	DefaultOtelProtocol = "grpc"
	// DefaultOtelSampler keeps every trace. The traces of a service that
	// is not under load are few, and a trace that is missing cannot be
	// recovered.
	DefaultOtelSampler = "parentbased_always_on"
)

// The samplers the service understands. They are the ones of the
// OpenTelemetry specification.
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

// Config is the validated configuration of the service.
type Config struct {
	// Host is the interface the server binds to.
	Host string
	// Port is the port the server listens on.
	Port int
	// ShutdownTimeout is the grace period of the shutdown.
	ShutdownTimeout time.Duration
	// CheckTimeout bounds how long a single health check may take.
	CheckTimeout time.Duration
	// LogLevel is the verbosity of the logs.
	LogLevel slog.Level
	// ServiceVersion is the version reported in the health report.
	ServiceVersion string
	// Database holds the settings of the database.
	Database DatabaseConfig
	// Tracing holds the settings of the tracing.
	Tracing TracingConfig
}

// DatabaseConfig holds the settings of the database.
//
// The database is required: the service is the writer of the schema, and a
// writer without a database has nothing to write.
type DatabaseConfig struct {
	// URL is the connection string of the database.
	URL string
	// WaitTimeout is how long the writer waits for the database to answer
	// when it starts.
	WaitTimeout time.Duration
}

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

// Addr returns the address the server listens on.
func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// Load reads the configuration from the environment and validates it.
//
// An invalid value is an error, never a silent fallback: a service that
// starts with the wrong port, or without its database, is a service that
// nobody asked for.
func Load() (Config, error) {
	cfg := Config{
		Host:            DefaultHost,
		Port:            DefaultPort,
		ShutdownTimeout: DefaultShutdownTimeout,
		CheckTimeout:    DefaultCheckTimeout,
		LogLevel:        logging.DefaultLevel,
	}

	if raw, ok := os.LookupEnv(EnvHost); ok {
		host := strings.TrimSpace(raw)
		if host == "" {
			return Config{}, fmt.Errorf("%s: the host cannot be empty", EnvHost)
		}
		cfg.Host = host
	}

	if raw, ok := os.LookupEnv(EnvPort); ok {
		port, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return Config{}, fmt.Errorf("%s: %q is not a port number", EnvPort, raw)
		}
		cfg.Port = port
	}

	if raw, ok := os.LookupEnv(EnvShutdownTimeout); ok {
		timeout, err := env.Duration(EnvShutdownTimeout, raw)
		if err != nil {
			return Config{}, err
		}
		cfg.ShutdownTimeout = timeout
	}

	if raw, ok := os.LookupEnv(EnvCheckTimeout); ok {
		timeout, err := env.Duration(EnvCheckTimeout, raw)
		if err != nil {
			return Config{}, err
		}
		cfg.CheckTimeout = timeout
	}

	if raw, ok := os.LookupEnv(logging.Level); ok {
		level, err := logging.ParseLevel(raw)
		if err != nil {
			return Config{}, err
		}
		cfg.LogLevel = level
	}

	if raw, ok := os.LookupEnv(env.ServiceVersion); ok {
		version := strings.TrimSpace(raw)
		if version == "" {
			return Config{}, fmt.Errorf("%s: the version cannot be empty", env.ServiceVersion)
		}
		cfg.ServiceVersion = version
	}

	database, err := loadDatabase()
	if err != nil {
		return Config{}, err
	}
	cfg.Database = database

	tracing, err := loadTracing()
	if err != nil {
		return Config{}, err
	}
	cfg.Tracing = tracing

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// LoadPort reads only the port of the server.
//
// The health check of the container asks the running service whether it is
// alive, and it needs the port and nothing else: a service that cannot
// reach its database still answers its health endpoint, and the answer is
// what the check is after.
func LoadPort() (int, error) {
	raw, ok := os.LookupEnv(EnvPort)
	if !ok {
		return DefaultPort, nil
	}

	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a port number", EnvPort, raw)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("%s: %d is not a valid port", EnvPort, port)
	}

	return port, nil
}

// loadDatabase reads the settings of the database from the environment.
//
// The database is named in two ways, and the first one wins:
//
//  1. the whole connection string, in PACA_DATABASE_URL. When it is set,
//     nothing else is read: a hosted database, or a URL built by hand, needs
//     nothing else.
//
//  2. otherwise the connection string is assembled from the credentials of
//     the PostgreSQL stack — POSTGRES_USER, POSTGRES_PASSWORD and
//     POSTGRES_DB, which the compose file loads with env_file — and the
//     host and the port of it.
//
// A service without a database has nothing to do: it is the writer of the
// schema, so it refuses to start.
func loadDatabase() (DatabaseConfig, error) {
	waitTimeout := DefaultDatabaseWaitTimeout
	if raw, ok := os.LookupEnv(EnvDatabaseWaitTimeout); ok {
		timeout, err := env.Duration(EnvDatabaseWaitTimeout, raw)
		if err != nil {
			return DatabaseConfig{}, err
		}
		waitTimeout = timeout
	}

	if raw, ok := os.LookupEnv(EnvDatabaseURL); ok {
		connection := strings.TrimSpace(raw)
		if connection == "" {
			return DatabaseConfig{}, fmt.Errorf("%s: the connection string cannot be empty", EnvDatabaseURL)
		}
		return DatabaseConfig{URL: connection, WaitTimeout: waitTimeout}, nil
	}

	user := strings.TrimSpace(os.Getenv(EnvPostgresUser))
	// The password is never trimmed: a password may hold a space, and a
	// password that is trimmed is a password that is wrong.
	password := os.Getenv(EnvPostgresPassword)
	database := strings.TrimSpace(os.Getenv(EnvPostgresDB))

	if user == "" || password == "" || database == "" {
		return DatabaseConfig{}, fmt.Errorf(
			"%s is not set and %s, %s and %s are not complete: "+
				"the service is the writer of the schema and it cannot run without a database",
			EnvDatabaseURL, EnvPostgresUser, EnvPostgresPassword, EnvPostgresDB)
	}

	host := DefaultDatabaseHost
	if raw, ok := os.LookupEnv(EnvDatabaseHost); ok {
		host = strings.TrimSpace(raw)
		if host == "" {
			return DatabaseConfig{}, fmt.Errorf("%s: the host cannot be empty", EnvDatabaseHost)
		}
	}

	port := DefaultDatabasePort
	if raw, ok := os.LookupEnv(EnvDatabasePort); ok {
		number, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return DatabaseConfig{}, fmt.Errorf("%s: %q is not a port number", EnvDatabasePort, raw)
		}
		if number < 1 || number > 65535 {
			return DatabaseConfig{}, fmt.Errorf("%s: %d is not a valid port", EnvDatabasePort, number)
		}
		port = strconv.Itoa(number)
	}

	return DatabaseConfig{
		URL: dbconfig.ConnectionString(dbconfig.Credentials{
			User:     user,
			Password: password,
			Database: database,
			Host:     host,
			Port:     port,
		}),
		WaitTimeout: waitTimeout,
	}, nil
}

// loadTracing reads the settings of the tracing from the environment.
//
// The tracing is on when a collector is configured and the specification
// does not turn it off. A service without a collector does not trace: there
// is nowhere to send the traces, and a service that tries anyway only fills
// its own log with connection errors.
func loadTracing() (TracingConfig, error) {
	tracing := TracingConfig{
		Protocol: DefaultOtelProtocol,
		Sampler:  DefaultOtelSampler,
	}

	if raw, ok := os.LookupEnv(EnvOtelEndpoint); ok {
		endpoint := strings.TrimSpace(raw)
		if endpoint == "" {
			return TracingConfig{}, fmt.Errorf("%s: the address cannot be empty", EnvOtelEndpoint)
		}
		tracing.Endpoint = endpoint
	}

	if raw, ok := os.LookupEnv(EnvOtelProtocol); ok {
		tracing.Protocol = strings.TrimSpace(raw)
	}

	if raw, ok := os.LookupEnv(EnvOtelInsecure); ok {
		insecure, err := parseBool(EnvOtelInsecure, raw)
		if err != nil {
			return TracingConfig{}, err
		}
		tracing.Insecure = insecure
	}

	if raw, ok := os.LookupEnv(EnvOtelSampler); ok {
		tracing.Sampler = strings.TrimSpace(raw)
	}

	if raw, ok := os.LookupEnv(EnvOtelSamplerArg); ok {
		tracing.SamplerArg = strings.TrimSpace(raw)
	}

	if raw, ok := os.LookupEnv(EnvOtelServiceName); ok {
		name := strings.TrimSpace(raw)
		if name == "" {
			return TracingConfig{}, fmt.Errorf("%s: the name cannot be empty", EnvOtelServiceName)
		}
		tracing.ServiceName = name
	}

	disabled := false
	if raw, ok := os.LookupEnv(EnvOtelDisabled); ok {
		value, err := parseBool(EnvOtelDisabled, raw)
		if err != nil {
			return TracingConfig{}, err
		}
		disabled = value
	}

	tracing.Enabled = !disabled && tracing.Endpoint != ""

	if err := tracing.validate(); err != nil {
		return TracingConfig{}, err
	}

	return tracing, nil
}

// validate rejects the configurations that would make the server unsafe or
// useless.
func (c Config) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("%s: %d is not a valid port", EnvPort, c.Port)
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("%s: the timeout must be positive", EnvShutdownTimeout)
	}
	if c.CheckTimeout <= 0 {
		return fmt.Errorf("%s: the timeout must be positive", EnvCheckTimeout)
	}
	if c.Database.URL == "" {
		return fmt.Errorf("%s: the connection string of the database is empty", EnvDatabaseURL)
	}
	if c.Database.WaitTimeout <= 0 {
		return fmt.Errorf("%s: the timeout must be positive", EnvDatabaseWaitTimeout)
	}
	return nil
}

// validate rejects the tracing settings that would not work.
func (t TracingConfig) validate() error {
	switch t.Protocol {
	case "grpc", "http/protobuf":
	default:
		return fmt.Errorf("%s: %q is not a protocol", EnvOtelProtocol, t.Protocol)
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
		return fmt.Errorf("%s: %q is not a sampler", EnvOtelSampler, t.Sampler)
	}

	// A ratio that cannot be read is an error even when the sampler does
	// not take one: a setting that is silently ignored is a setting that
	// nobody read.
	if t.Sampler == SamplerTraceIDRatio || t.Sampler == SamplerParentBasedTraceIDRatio ||
		t.SamplerArg != "" {
		if _, err := parseRatio(EnvOtelSamplerArg, t.SamplerArg); err != nil {
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
		if protocol != "http/protobuf" {
			return fmt.Errorf("%s: %q has a scheme, which the %s protocol does not take",
				EnvOtelEndpoint, endpoint, protocol)
		}
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return fmt.Errorf("%s: %q is not an address", EnvOtelEndpoint, endpoint)
		}
		if parsed.Host == "" {
			return fmt.Errorf("%s: %q has no host", EnvOtelEndpoint, endpoint)
		}
		return nil
	}

	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("%s: %q is not an address", EnvOtelEndpoint, endpoint)
	}
	if host == "" || port == "" {
		return fmt.Errorf("%s: %q has no host or no port", EnvOtelEndpoint, endpoint)
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
