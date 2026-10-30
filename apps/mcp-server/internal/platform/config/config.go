// Package config reads and validates the configuration of the MCP server.
//
// Every setting comes from the environment, so the same binary runs in a
// container, on a laptop and in a test without a configuration file. The
// defaults are the safe ones: the server listens on the loopback interface
// unless it is told otherwise.
//
// The database is named by one of two ways, and the first one wins:
//
//  1. `PACA_DATABASE_URL` — the whole connection string.
//  2. The credentials of the PostgreSQL stack — `POSTGRES_USER`,
//     `POSTGRES_PASSWORD` and `POSTGRES_DB`, which the service loads with
//     the `env_file` of its compose — with `PACA_DATABASE_HOST` (default
//     `postgres`) and `PACA_DATABASE_PORT` (default `5432`).
//
// Without either of them the service runs without a database.
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

// The environment variables the server reads.
const (
	// EnvHost is the interface the server binds to.
	EnvHost = "PACA_MCP_HOST"
	// EnvPort is the port the server listens on.
	EnvPort = "PACA_MCP_PORT"
	// EnvShutdownTimeout is how long the server waits for the requests in
	// flight when it is asked to stop.
	EnvShutdownTimeout = "PACA_MCP_SHUTDOWN_TIMEOUT"
	// EnvCheckTimeout bounds how long a single health check may take.
	EnvCheckTimeout = "PACA_MCP_CHECK_TIMEOUT"
	// EnvMaxRequestBytes is the largest body the MCP transport accepts.
	EnvMaxRequestBytes = "PACA_MCP_MAX_REQUEST_BYTES"
)

// The environment variables of the database. They are the ones of the core,
// and every service of the monorepository reads them the same way.
const (
	// EnvDatabaseURL is the connection string of the database.
	EnvDatabaseURL = "PACA_DATABASE_URL"
	// EnvDatabaseMaxConns is the number of connections the pool may open at
	// the same time.
	EnvDatabaseMaxConns = "PACA_DATABASE_MAX_CONNS"
	// EnvDatabaseConnectTimeout bounds the attempt to open a connection.
	EnvDatabaseConnectTimeout = "PACA_DATABASE_CONNECT_TIMEOUT"
	// EnvDatabaseHost is the host the database answers on. It is only
	// read when the connection string is assembled from the credentials
	// of the PostgreSQL stack.
	EnvDatabaseHost = "PACA_DATABASE_HOST"
	// EnvDatabasePort is the port the database answers on. It is only
	// read when the connection string is assembled from the credentials
	// of the PostgreSQL stack.
	EnvDatabasePort = "PACA_DATABASE_PORT"
)

// The environment variables of the PostgreSQL stack. They are the ones of
// the image of PostgreSQL, and they are the credentials that live in
// `db/postgres/.env`: the stack that owns them is the only one that
// writes them, and every service that talks to the database loads them
// with `env_file`.
const (
	// EnvPostgresUser is the role that owns the application database.
	EnvPostgresUser = "POSTGRES_USER"
	// EnvPostgresPassword is the password of the role.
	EnvPostgresPassword = "POSTGRES_PASSWORD" //nolint:gosec // the name of the environment variable, not a credential.
	// EnvPostgresDB is the application database.
	EnvPostgresDB = "POSTGRES_DB"
)

// The defaults of the settings.
const (
	// DefaultHost binds the loopback interface only. A container overrides
	// it, because the other containers of the stack reach the server
	// through the interface of the container.
	DefaultHost = "127.0.0.1"
	// DefaultPort is the port reserved for the MCP server of the
	// monorepository.
	DefaultPort = 8081
	// DefaultShutdownTimeout is the grace period of the shutdown.
	DefaultShutdownTimeout = 10 * time.Second
	// DefaultCheckTimeout bounds a single health check.
	DefaultCheckTimeout = 2 * time.Second
	// DefaultMaxRequestBytes is the largest body the MCP transport accepts.
	DefaultMaxRequestBytes int64 = 4 << 20
	// DefaultDatabaseMaxConns is the number of connections the pool of the
	// service holds. The service needs few: it talks to the database for
	// one tool at a time.
	DefaultDatabaseMaxConns int32 = 4
	// DefaultDatabaseConnectTimeout bounds the attempt to reach the
	// database.
	DefaultDatabaseConnectTimeout = 5 * time.Second
	// DefaultDatabaseHost is the host of the database of the
	// monorepository: the container of the PostgreSQL stack.
	DefaultDatabaseHost = "postgres"
	// DefaultDatabasePort is the port the database of the monorepository
	// answers on.
	DefaultDatabasePort = 5432
	// ServiceName is the name the MCP clients see.
	ServiceName = "paca-mcp-server"
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
	// MaxRequestBytes is the largest body the MCP transport accepts.
	MaxRequestBytes int64
	// LogLevel is the verbosity of the logs.
	LogLevel slog.Level
	// ServiceVersion is the version reported to the MCP clients.
	ServiceVersion string
	// Database holds the settings of the database.
	Database DatabaseConfig
	// Tracing holds the settings of the tracing.
	Tracing telemetry.TracingConfig
}

// DatabaseConfig holds the settings of the database.
//
// The database is optional: a service that does not name one starts without
// it. The first tool that reads the database makes it required.
type DatabaseConfig struct {
	// URL is the connection string of the database. It is empty when the
	// service runs without a database.
	URL string
	// MaxConns is the number of connections the pool may open at the same
	// time.
	MaxConns int32
	// ConnectTimeout bounds the attempt to open a connection.
	ConnectTimeout time.Duration
	// CredentialsIncomplete says that the environment carries some of the
	// credentials of the database and not all of them. The service runs
	// without a database in that case, as it always did, and it says so in
	// its log: a service that stops because a variable of the environment
	// is half set is a service that stops for a reason nobody asked for.
	CredentialsIncomplete bool
}

// Addr returns the address the server listens on.
func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// Load reads the configuration from the environment and validates it.
//
// An invalid value is an error, never a silent fallback: a server that starts
// with the wrong port or without its limits is a server that nobody asked
// for.
func Load() (Config, error) {
	cfg := Config{
		Host:            DefaultHost,
		Port:            DefaultPort,
		ShutdownTimeout: DefaultShutdownTimeout,
		CheckTimeout:    DefaultCheckTimeout,
		MaxRequestBytes: DefaultMaxRequestBytes,
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

	if raw, ok := os.LookupEnv(EnvMaxRequestBytes); ok {
		size, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %q is not a number of bytes", EnvMaxRequestBytes, raw)
		}
		cfg.MaxRequestBytes = size
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

// loadDatabase reads the settings of the database from the environment.
//
// The database is optional. A service that does not name it runs without
// one, and says nothing about it in its health report.
//
// The connection string is resolved in this order:
//
//  1. `PACA_DATABASE_URL` — the whole connection string. When it is set, it
//     wins, and nothing else is read: a hosted database, or a URL built by
//     hand, needs nothing else.
//  2. Otherwise the connection string is assembled from the credentials of
//     the PostgreSQL stack (`POSTGRES_USER`, `POSTGRES_PASSWORD` and
//     `POSTGRES_DB`, loaded from `db/postgres/.env` with `env_file`) and
//     from `PACA_DATABASE_HOST` and `PACA_DATABASE_PORT`.
//
// Credentials that are half there are ignored: the service runs without a
// database, the way it ran before the credentials of the PostgreSQL stack
// reached it, and it says so in its log. A database that the service tries
// to reach with an incomplete password fails later, in the middle of a
// request; a service that says the credentials are incomplete fails at the
// start, in the log.
func loadDatabase() (DatabaseConfig, error) {
	database := DatabaseConfig{
		MaxConns:       DefaultDatabaseMaxConns,
		ConnectTimeout: DefaultDatabaseConnectTimeout,
	}

	if raw, ok := os.LookupEnv(EnvDatabaseURL); ok {
		connection := strings.TrimSpace(raw)
		if connection == "" {
			return DatabaseConfig{}, fmt.Errorf("%s: the connection string cannot be empty", EnvDatabaseURL)
		}
		database.URL = connection
	} else if err := loadDatabaseFromCredentials(&database); err != nil {
		return DatabaseConfig{}, err
	}

	if raw, ok := os.LookupEnv(EnvDatabaseMaxConns); ok {
		connections, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
		if err != nil {
			return DatabaseConfig{}, fmt.Errorf("%s: %q is not a number of connections", EnvDatabaseMaxConns, raw)
		}
		database.MaxConns = int32(connections)
	}

	if raw, ok := os.LookupEnv(EnvDatabaseConnectTimeout); ok {
		timeout, err := env.Duration(EnvDatabaseConnectTimeout, raw)
		if err != nil {
			return DatabaseConfig{}, err
		}
		database.ConnectTimeout = timeout
	}

	if err := database.validate(); err != nil {
		return DatabaseConfig{}, err
	}

	return database, nil
}

// loadDatabaseFromCredentials assembles the connection string of the
// database from the credentials of the PostgreSQL stack.
//
// The credentials travel to the service through the `env_file` of the
// compose of the service, because the stack that owns them is the only one
// that writes them.
//
// When no credential is there at all, the service has no database: that is
// the behaviour of a service that runs alone. When some of them are there
// and the others are not, the service still has no database, and it says so.
func loadDatabaseFromCredentials(database *DatabaseConfig) error {
	user, hasUser := os.LookupEnv(EnvPostgresUser)
	password, hasPassword := os.LookupEnv(EnvPostgresPassword)
	name, hasDatabase := os.LookupEnv(EnvPostgresDB)

	if !hasUser && !hasPassword && !hasDatabase {
		return nil
	}

	if !hasUser || strings.TrimSpace(user) == "" ||
		!hasPassword || password == "" ||
		!hasDatabase || strings.TrimSpace(name) == "" {
		database.CredentialsIncomplete = true
		return nil
	}

	host, err := loadDatabaseHost()
	if err != nil {
		return err
	}
	port, err := loadDatabasePort()
	if err != nil {
		return err
	}

	database.URL = dbconfig.ConnectionString(dbconfig.Credentials{
		User:     strings.TrimSpace(user),
		Password: password,
		Database: strings.TrimSpace(name),
		Host:     host,
		Port:     strconv.Itoa(port),
	})

	return nil
}

// loadDatabaseHost reads the host the database answers on.
func loadDatabaseHost() (string, error) {
	raw, ok := os.LookupEnv(EnvDatabaseHost)
	if !ok {
		return DefaultDatabaseHost, nil
	}

	host := strings.TrimSpace(raw)
	if host == "" {
		return "", fmt.Errorf("%s: the host cannot be empty", EnvDatabaseHost)
	}

	return host, nil
}

// loadDatabasePort reads the port the database answers on.
func loadDatabasePort() (int, error) {
	raw, ok := os.LookupEnv(EnvDatabasePort)
	if !ok {
		return DefaultDatabasePort, nil
	}

	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a port number", EnvDatabasePort, raw)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("%s: %d is not a valid port", EnvDatabasePort, port)
	}

	return port, nil
}

// validate rejects the settings of the database that would not work.
//
// The settings are always valid, whether the service names a database or
// not: a number of connections that makes no sense is an error on its own.
func (d DatabaseConfig) validate() error {
	if d.MaxConns < 1 {
		return fmt.Errorf("%s: the pool must hold at least one connection, got %d", EnvDatabaseMaxConns, d.MaxConns)
	}
	if d.ConnectTimeout <= 0 {
		return fmt.Errorf("%s: the timeout must be positive", EnvDatabaseConnectTimeout)
	}
	return nil
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
	if c.MaxRequestBytes <= 0 {
		return fmt.Errorf("%s: the limit must be positive", EnvMaxRequestBytes)
	}
	return nil
}
