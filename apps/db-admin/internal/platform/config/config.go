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
	"os"
	"strconv"
	"strings"
	"time"

	dbconfig "github.com/witsaba/paca/packages/database/config"
	"github.com/witsaba/paca/packages/platform/env"
	"github.com/witsaba/paca/packages/platform/logging"
	"github.com/witsaba/paca/packages/platform/telemetry"
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

// The environment variables of the database that belong to this service.
//
// The names of the database itself — the connection string, the host, the
// port and the credentials of the PostgreSQL stack — live in the shared
// platform (`packages/platform/env`): every service of the monorepository
// reads them the same way.
const (
	// EnvDatabaseWaitTimeout is how long the writer waits for the
	// database to answer when it starts.
	EnvDatabaseWaitTimeout = "PACA_DB_ADMIN_DATABASE_WAIT_TIMEOUT"
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
	Tracing telemetry.TracingConfig
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

	tracing, err := telemetry.LoadTracing(os.LookupEnv)
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

	// An empty value is no value: a connection string that is set to
	// nothing is a connection string that is not set, and the credentials
	// of the PostgreSQL stack are read instead.
	if connection, ok := env.Value(os.LookupEnv, env.DatabaseURL); ok {
		return DatabaseConfig{URL: connection, WaitTimeout: waitTimeout}, nil
	}

	user, hasUser := env.Value(os.LookupEnv, env.PostgresUser)
	// The password is never trimmed: a password may hold a space, and a
	// password that is trimmed is a password that is wrong.
	password, hasPassword := env.Secret(os.LookupEnv, env.PostgresPassword)
	database, hasDatabase := env.Value(os.LookupEnv, env.PostgresDB)

	if !hasUser || !hasPassword || !hasDatabase {
		return DatabaseConfig{}, fmt.Errorf(
			"%s is not set and %s, %s and %s are not complete: "+
				"the service is the writer of the schema and it cannot run without a database",
			env.DatabaseURL, env.PostgresUser, env.PostgresPassword, env.PostgresDB)
	}

	host := env.DefaultDatabaseHost
	if value, ok := env.Value(os.LookupEnv, env.DatabaseHost); ok {
		host = value
	}

	port := env.DefaultDatabasePort
	if raw, ok := env.Value(os.LookupEnv, env.DatabasePort); ok {
		number, err := strconv.Atoi(raw)
		if err != nil {
			return DatabaseConfig{}, fmt.Errorf("%s: %q is not a port number", env.DatabasePort, raw)
		}
		if number < 1 || number > 65535 {
			return DatabaseConfig{}, fmt.Errorf("%s: %d is not a valid port", env.DatabasePort, number)
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
		return fmt.Errorf("%s: the connection string of the database is empty", env.DatabaseURL)
	}
	if c.Database.WaitTimeout <= 0 {
		return fmt.Errorf("%s: the timeout must be positive", EnvDatabaseWaitTimeout)
	}
	return nil
}
