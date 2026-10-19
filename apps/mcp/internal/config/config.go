// Package config parses and validates the configuration of the paca-mcp
// server.
//
// The configuration is the whole contract of the binary: the address it
// listens on, the material that protects it and the database it serves. The
// validation is the part that matters: a server that listens on an address
// other than loopback is exposed to the network, so it refuses to start
// unless it has both a bearer token and TLS material.
//
// The package holds no business rule of PACA and no I/O of its own.
package config

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
)

// The defaults of the server, as ADR 0001 (C1) defines them.
const (
	// DefaultBind is the address the server listens on when none is given.
	DefaultBind = "127.0.0.1"
	// DefaultPort is the port the server listens on when none is given.
	DefaultPort = 7333
)

// The reasons the server refuses to start. They are exported so that a
// caller can tell one refusal from another with errors.Is.
var (
	// ErrInvalidPort is returned when the port is outside 1-65535.
	ErrInvalidPort = errors.New("invalid port")
	// ErrMissingDatabase is returned when no database file was given.
	ErrMissingDatabase = errors.New("missing database path")
	// ErrRequireAuthWithoutToken is returned when authentication is required
	// but no token was given.
	ErrRequireAuthWithoutToken = errors.New("--require-auth needs --auth-token")
	// ErrMissingToken is returned when the server listens on a non-loopback
	// address and no bearer token was given.
	ErrMissingToken = errors.New("missing bearer token for a non-loopback bind")
	// ErrMissingTLS is returned when the server listens on a non-loopback
	// address and the TLS material is incomplete.
	ErrMissingTLS = errors.New("missing TLS material for a non-loopback bind")
)

// Config is the configuration of the server.
//
// The zero value is not a usable configuration: the address and the port
// carry their defaults when they are parsed, and the validation rejects the
// rest.
type Config struct {
	// Bind is the address the server listens on.
	Bind string
	// Port is the TCP port the server listens on.
	Port int
	// AuthToken is the bearer token the MCP surface demands. An empty token
	// means that the MCP surface is open, which is only allowed on loopback.
	AuthToken string
	// TLSCert is the path of the certificate file.
	TLSCert string
	// TLSKey is the path of the private key file.
	TLSKey string
	// DBPath is the path of the SQLite file the server serves.
	DBPath string
	// RequireAuth makes the absence of a token a startup failure, even on
	// loopback.
	RequireAuth bool
}

// Loopback reports whether the server listens on a loopback address.
//
// A server bound to a loopback address is reachable from the machine itself
// only, so it is the one case where the transport may be plain HTTP and the
// token optional. The name "localhost" is a loopback address too, and so is
// every address of 127.0.0.0/8 and of ::1. The wildcard address 0.0.0.0 is
// not: it listens on every interface, network included.
func (c Config) Loopback() bool {
	host := strings.TrimSpace(c.Bind)
	if host == "" {
		host = DefaultBind
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// TLSEnabled reports whether the server serves TLS.
//
// TLS is on when the certificate and the key are both given. The validation
// refuses a non-loopback bind without that material, so the only way to
// serve plain HTTP to the network is a configuration that never starts.
func (c Config) TLSEnabled() bool {
	return c.TLSCert != "" && c.TLSKey != ""
}

// Validate reports whether the configuration may start a server.
//
// The rules are the ones of ADR 0001 (C1 and C2): a loopback address may run
// plain HTTP and may run without a token, and a non-loopback address must
// have both the token and the TLS material. The error names the reason and
// never carries the token.
func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("%w: %d is outside 1-65535", ErrInvalidPort, c.Port)
	}
	if strings.TrimSpace(c.DBPath) == "" {
		return fmt.Errorf("%w: --db is required", ErrMissingDatabase)
	}
	if c.RequireAuth && c.AuthToken == "" {
		return ErrRequireAuthWithoutToken
	}
	if c.Loopback() {
		return nil
	}
	if c.AuthToken == "" {
		return fmt.Errorf("%w: --bind %s is not a loopback address", ErrMissingToken, c.Bind)
	}
	if !c.TLSEnabled() {
		return fmt.Errorf("%w: --bind %s is not a loopback address", ErrMissingTLS, c.Bind)
	}
	return nil
}

// Parse reads the configuration from the arguments of the binary.
//
// The flags are the ones of ADR 0001 (C1): --bind, --port, --auth-token,
// --tls-cert, --tls-key, --db and --require-auth. The defaults of the
// address and the port are applied, and the result is validated, so a
// configuration that would expose the server to the network never returns
// from this function.
func Parse(args []string) (Config, error) {
	cfg := Config{
		Bind: DefaultBind,
		Port: DefaultPort,
	}

	cmd := &cobra.Command{
		Use:           "paca-mcp",
		Short:         "PACA, a personal finance manager for the Colombian market",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(*cobra.Command, []string) error { return nil },
		Args:          cobra.NoArgs,
	}
	flags := cmd.Flags()
	flags.StringVar(&cfg.Bind, "bind", cfg.Bind, "address the server listens on")
	flags.IntVar(&cfg.Port, "port", cfg.Port, "TCP port the server listens on")
	flags.StringVar(&cfg.AuthToken, "auth-token", "", "bearer token the MCP surface demands")
	flags.StringVar(&cfg.TLSCert, "tls-cert", "", "path of the TLS certificate file")
	flags.StringVar(&cfg.TLSKey, "tls-key", "", "path of the TLS private key file")
	flags.StringVar(&cfg.DBPath, "db", "", "path of the SQLite file the server serves")
	flags.BoolVar(&cfg.RequireAuth, "require-auth", false, "refuse to start without a bearer token")

	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
