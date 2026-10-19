package config_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/witsaba/paca/apps/mcp/internal/config"
)

// dbPath is the path of a database file that does not have to exist for the
// configuration to be valid.
func dbPath(t *testing.T) string {
	t.Helper()

	return filepath.Join(t.TempDir(), "paca.db")
}

func TestParseDefaults(t *testing.T) {
	cfg, err := config.Parse([]string{"--db", dbPath(t)})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if cfg.Bind != config.DefaultBind {
		t.Errorf("bind = %q, want %q", cfg.Bind, config.DefaultBind)
	}
	if cfg.Port != config.DefaultPort {
		t.Errorf("port = %d, want %d", cfg.Port, config.DefaultPort)
	}
	if cfg.AuthToken != "" {
		t.Errorf("auth token = %q, want empty", cfg.AuthToken)
	}
	if cfg.TLSCert != "" || cfg.TLSKey != "" {
		t.Errorf("tls = %q/%q, want empty", cfg.TLSCert, cfg.TLSKey)
	}
	if cfg.RequireAuth {
		t.Error("require auth = true, want false")
	}
}

func TestParseReadsEveryFlag(t *testing.T) {
	path := dbPath(t)
	cert := filepath.Join(t.TempDir(), "cert.pem")
	key := filepath.Join(t.TempDir(), "key.pem")

	cfg, err := config.Parse([]string{
		"--bind", "192.168.1.20",
		"--port", "9000",
		"--auth-token", "s3cret",
		"--tls-cert", cert,
		"--tls-key", key,
		"--db", path,
		"--require-auth",
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if cfg.Bind != "192.168.1.20" {
		t.Errorf("bind = %q, want %q", cfg.Bind, "192.168.1.20")
	}
	if cfg.Port != 9000 {
		t.Errorf("port = %d, want %d", cfg.Port, 9000)
	}
	if cfg.AuthToken != "s3cret" {
		t.Errorf("auth token = %q, want %q", cfg.AuthToken, "s3cret")
	}
	if cfg.TLSCert != cert || cfg.TLSKey != key {
		t.Errorf("tls = %q/%q, want %q/%q", cfg.TLSCert, cfg.TLSKey, cert, key)
	}
	if cfg.DBPath != path {
		t.Errorf("database = %q, want %q", cfg.DBPath, path)
	}
	if !cfg.RequireAuth {
		t.Error("require auth = false, want true")
	}
}

func TestParseRefuses(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr bool
		want    error
	}{
		{
			name:    "unknown flag",
			args:    []string{"--db", dbPath(t), "--nope"},
			wantErr: true,
		},
		{
			name:    "port below the range",
			args:    []string{"--db", dbPath(t), "--port", "0"},
			wantErr: true,
			want:    config.ErrInvalidPort,
		},
		{
			name:    "port above the range",
			args:    []string{"--db", dbPath(t), "--port", "70000"},
			wantErr: true,
			want:    config.ErrInvalidPort,
		},
		{
			name:    "no database",
			args:    []string{},
			wantErr: true,
			want:    config.ErrMissingDatabase,
		},
		{
			name:    "required auth without a token",
			args:    []string{"--db", dbPath(t), "--require-auth"},
			wantErr: true,
			want:    config.ErrRequireAuthWithoutToken,
		},
		{
			name:    "non loopback bind without a token",
			args:    []string{"--db", dbPath(t), "--bind", "192.168.1.20"},
			wantErr: true,
			want:    config.ErrMissingToken,
		},
		{
			name:    "wildcard bind without a token",
			args:    []string{"--db", dbPath(t), "--bind", "0.0.0.0"},
			wantErr: true,
			want:    config.ErrMissingToken,
		},
		{
			name:    "non loopback bind without TLS",
			args:    []string{"--db", dbPath(t), "--bind", "192.168.1.20", "--auth-token", "s3cret"},
			wantErr: true,
			want:    config.ErrMissingTLS,
		},
		{
			name: "non loopback bind without the key",
			args: []string{
				"--db", dbPath(t),
				"--bind", "192.168.1.20",
				"--auth-token", "s3cret",
				"--tls-cert", filepath.Join(t.TempDir(), "cert.pem"),
			},
			wantErr: true,
			want:    config.ErrMissingTLS,
		},
		{
			name: "non loopback bind with a token and TLS",
			args: []string{
				"--db", dbPath(t),
				"--bind", "192.168.1.20",
				"--auth-token", "s3cret",
				"--tls-cert", filepath.Join(t.TempDir(), "cert.pem"),
				"--tls-key", filepath.Join(t.TempDir(), "key.pem"),
			},
		},
		{
			name: "loopback bind with a token and no TLS",
			args: []string{"--db", dbPath(t), "--auth-token", "s3cret"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.Parse(tc.args)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("Parse: unexpected error %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Parse: no error, want %v", tc.want)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("Parse: error = %v, want %v", err, tc.want)
			}
			if cfg != (config.Config{}) {
				t.Errorf("Parse: configuration = %+v, want the zero value on a refusal", cfg)
			}
		})
	}
}

// The error of a refusal names the reason and never the token: the token is
// a secret and an error message travels to logs.
func TestRefusalNeverCarriesTheToken(t *testing.T) {
	const token = "s3cret-token-value"

	_, err := config.Parse([]string{
		"--db", dbPath(t),
		"--bind", "192.168.1.20",
		"--auth-token", token,
		"--tls-cert", filepath.Join(t.TempDir(), "cert.pem"),
	})
	if err == nil {
		t.Fatal("Parse: no error, want the missing TLS refusal")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("the error carries the token: %v", err)
	}
}

func TestLoopback(t *testing.T) {
	cases := []struct {
		bind string
		want bool
	}{
		{bind: "127.0.0.1", want: true},
		{bind: "127.0.0.53", want: true},
		{bind: "::1", want: true},
		{bind: "localhost", want: true},
		{bind: "LocalHost", want: true},
		{bind: "", want: true},
		{bind: "0.0.0.0", want: false},
		{bind: "192.168.1.20", want: false},
		{bind: "example.com", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.bind, func(t *testing.T) {
			cfg := config.Config{Bind: tc.bind}
			if got := cfg.Loopback(); got != tc.want {
				t.Errorf("Loopback(%q) = %v, want %v", tc.bind, got, tc.want)
			}
		})
	}
}

func TestTLSEnabled(t *testing.T) {
	cases := []struct {
		name string
		cert string
		key  string
		want bool
	}{
		{name: "no material", want: false},
		{name: "cert only", cert: "cert.pem", want: false},
		{name: "key only", key: "key.pem", want: false},
		{name: "both", cert: "cert.pem", key: "key.pem", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{TLSCert: tc.cert, TLSKey: tc.key}
			if got := cfg.TLSEnabled(); got != tc.want {
				t.Errorf("TLSEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}
