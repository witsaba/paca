// Package config reads and validates the configuration of the MCP server.
//
// Every setting comes from the environment, so the same binary runs in a
// container, on a laptop and in a test without a configuration file. The
// defaults are the safe ones: the server listens on the loopback interface
// unless it is told otherwise.
package config

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
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
	// EnvLogLevel is the verbosity of the logs.
	EnvLogLevel = "PACA_LOG_LEVEL"
	// EnvServiceVersion is the version reported to the MCP clients.
	EnvServiceVersion = "PACA_SERVICE_VERSION"
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
	// DefaultLogLevel is the verbosity of the logs.
	DefaultLogLevel = slog.LevelInfo
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
		LogLevel:        DefaultLogLevel,
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
		timeout, err := parseDuration(EnvShutdownTimeout, raw)
		if err != nil {
			return Config{}, err
		}
		cfg.ShutdownTimeout = timeout
	}

	if raw, ok := os.LookupEnv(EnvCheckTimeout); ok {
		timeout, err := parseDuration(EnvCheckTimeout, raw)
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

	if raw, ok := os.LookupEnv(EnvLogLevel); ok {
		level, err := parseLogLevel(raw)
		if err != nil {
			return Config{}, err
		}
		cfg.LogLevel = level
	}

	if raw, ok := os.LookupEnv(EnvServiceVersion); ok {
		version := strings.TrimSpace(raw)
		if version == "" {
			return Config{}, fmt.Errorf("%s: the version cannot be empty", EnvServiceVersion)
		}
		cfg.ServiceVersion = version
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
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

// parseDuration reads a duration from the environment.
func parseDuration(name, raw string) (time.Duration, error) {
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a duration", name, raw)
	}
	return value, nil
}

// parseLogLevel reads the verbosity of the logs from the environment.
func parseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("%s: %q is not a log level", EnvLogLevel, raw)
	}
}
