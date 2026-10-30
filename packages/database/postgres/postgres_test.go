package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/database/postgres"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*postgres.Config)
		wantErr bool
	}{
		{
			name:    "the configuration of the default is valid once it names a database",
			mutate:  func(c *postgres.Config) { c.URL = "postgres://paca@localhost:5432/paca" },
			wantErr: false,
		},
		{
			name:    "the connection string is required",
			mutate:  func(c *postgres.Config) { c.URL = "" },
			wantErr: true,
		},
		{
			name:    "the pool holds at least one connection",
			mutate:  func(c *postgres.Config) { c.URL = "postgres://paca@localhost:5432/paca"; c.MaxConns = 0 },
			wantErr: true,
		},
		{
			name:    "the pool cannot hold a negative number of connections",
			mutate:  func(c *postgres.Config) { c.URL = "postgres://paca@localhost:5432/paca"; c.MaxConns = -1 },
			wantErr: true,
		},
		{
			name: "the pool cannot keep more connections open than it holds",
			mutate: func(c *postgres.Config) {
				c.URL = "postgres://paca@localhost:5432/paca"
				c.MaxConns = 2
				c.MinConns = 3
			},
			wantErr: true,
		},
		{
			name:    "the pool cannot keep a negative number of connections open",
			mutate:  func(c *postgres.Config) { c.URL = "postgres://paca@localhost:5432/paca"; c.MinConns = -1 },
			wantErr: true,
		},
		{
			name: "the lifetime of a connection cannot be negative",
			mutate: func(c *postgres.Config) {
				c.URL = "postgres://paca@localhost:5432/paca"
				c.MaxConnLifetime = -time.Second
			},
			wantErr: true,
		},
		{
			name: "the idle time of a connection cannot be negative",
			mutate: func(c *postgres.Config) {
				c.URL = "postgres://paca@localhost:5432/paca"
				c.MaxConnIdleTime = -time.Second
			},
			wantErr: true,
		},
		{
			name: "the timeout of a connection cannot be negative",
			mutate: func(c *postgres.Config) {
				c.URL = "postgres://paca@localhost:5432/paca"
				c.ConnectTimeout = -time.Second
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := postgres.DefaultConfig()
			tc.mutate(&cfg)

			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("the configuration %+v is valid, want an error", cfg)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("the configuration %+v is invalid: %v", cfg, err)
			}
		})
	}
}

func TestConfigValidateWithoutAConnectionString(t *testing.T) {
	t.Parallel()

	err := postgres.DefaultConfig().Validate()
	if err == nil {
		t.Fatal("the default configuration is valid, want an error")
	}
	if !errors.Is(err, postgres.ErrNoURL) {
		t.Errorf("the error is %v, want it to wrap ErrNoURL", err)
	}
}

func TestOpenRejectsAnInvalidConfiguration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if _, err := postgres.Open(ctx, postgres.DefaultConfig()); !errors.Is(err, postgres.ErrNoURL) {
		t.Errorf("the error is %v, want it to wrap ErrNoURL", err)
	}
}

func TestOpenRejectsAnInvalidConnectionString(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := postgres.DefaultConfig()
	cfg.URL = "://not-a-connection-string"

	if _, err := postgres.Open(ctx, cfg); err == nil {
		t.Fatal("the pool was opened with an invalid connection string, want an error")
	}
}

func TestPoolPingWithoutAPool(t *testing.T) {
	t.Parallel()

	var pool *postgres.Pool

	if err := pool.Ping(context.Background()); err == nil {
		t.Fatal("the ping of a pool that is not open answered, want an error")
	}
	if pool.Queries() != nil {
		t.Error("the queries of a pool that is not open are not nil")
	}
	if pool.SQL() != nil {
		t.Error("the view of the schema of a pool that is not open is not nil")
	}
	pool.Close()
}

// TestSQLGivesTheSameViewEveryTime proves that the view of the schema is
// built once: the tools of the schema and the queries of the services share
// the connections of the pool, and a second view would open a second set.
func TestSQLGivesTheSameViewEveryTime(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := postgres.DefaultConfig()
	// Nothing listens on this port: the pool is lazy, so it opens anyway.
	cfg.URL = "postgres://paca:paca@127.0.0.1:1/paca?sslmode=disable"
	cfg.ConnectTimeout = 2 * time.Second

	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("the pool did not open: %v", err)
	}
	defer pool.Close()

	view := pool.SQL()
	if view == nil {
		t.Fatal("the view of the schema of an open pool is nil")
	}
	if pool.SQL() != view {
		t.Error("the view of the schema is built twice, want the same one")
	}
}

func TestOpenOpensThePoolWithoutADatabase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := postgres.DefaultConfig()
	// Nothing listens on this port: the pool is lazy, so it opens anyway,
	// and the ping is the thing that fails.
	cfg.URL = "postgres://paca:paca@127.0.0.1:1/paca?sslmode=disable"
	cfg.ConnectTimeout = 2 * time.Second

	pool, err := postgres.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("the pool did not open: %v", err)
	}
	defer pool.Close()

	if pool.Queries() == nil {
		t.Error("the queries of an open pool are nil")
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err == nil {
		t.Fatal("the ping of a pool with no database answered, want an error")
	}
}
