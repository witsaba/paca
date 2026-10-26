// Package postgres is the adapter of the core that talks to PostgreSQL.
//
// The package owns the pool of connections and the configuration of it. It
// hides the driver: a service that imports the core never names pgx, never
// names a connection string, and never opens a connection of its own.
//
// The dependency rule holds: the package knows the models of the domain and
// the generated code, and the domain knows nothing about it.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/witsaba/paca/packages/core/internal/db"
)

// ErrNoURL is the error of a configuration without a connection string.
var ErrNoURL = errors.New("the connection string of the database is empty")

// Config is the configuration of the pool of connections.
//
// The core reads no environment variable: a service maps its own environment
// onto this configuration, so the same core serves every service of the
// monorepository with its own settings.
type Config struct {
	// URL is the connection string of the database. It is required.
	URL string

	// MaxConns is the number of connections the pool may open at the same
	// time. It is the number of the requests the database may serve in
	// parallel.
	MaxConns int32

	// MinConns is the number of connections the pool keeps open even when
	// it is idle.
	MinConns int32

	// MaxConnLifetime is the longest a connection may live. A connection
	// that reaches it is closed and replaced.
	MaxConnLifetime time.Duration

	// MaxConnIdleTime is the longest a connection may sit idle before the
	// pool closes it.
	MaxConnIdleTime time.Duration

	// ConnectTimeout is the bound of a single attempt to open a
	// connection.
	ConnectTimeout time.Duration
}

// DefaultConfig returns the configuration the core uses when the service
// says nothing: a small pool, with the connections replaced before a
// database closes them.
func DefaultConfig() Config {
	return Config{
		URL:             "",
		MaxConns:        4,
		MinConns:        0,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		ConnectTimeout:  5 * time.Second,
	}
}

// Validate returns the error of a configuration the pool cannot open.
//
// An invalid configuration is an error, never a silent fallback: a service
// that starts with a pool it never asked for is a service nobody asked for.
func (c Config) Validate() error {
	if c.URL == "" {
		return ErrNoURL
	}
	if c.MaxConns < 1 {
		return fmt.Errorf("the pool of connections must hold at least one connection, got %d", c.MaxConns)
	}
	if c.MinConns < 0 {
		return fmt.Errorf("the pool of connections cannot hold a negative number of connections, got %d", c.MinConns)
	}
	if c.MinConns > c.MaxConns {
		return fmt.Errorf("the pool of connections cannot keep %d connections open when it holds %d", c.MinConns, c.MaxConns)
	}
	if c.MaxConnLifetime < 0 {
		return fmt.Errorf("the lifetime of a connection cannot be negative, got %s", c.MaxConnLifetime)
	}
	if c.MaxConnIdleTime < 0 {
		return fmt.Errorf("the idle time of a connection cannot be negative, got %s", c.MaxConnIdleTime)
	}
	if c.ConnectTimeout < 0 {
		return fmt.Errorf("the timeout of a connection cannot be negative, got %s", c.ConnectTimeout)
	}
	return nil
}

// Pool is the pool of connections to the database.
//
// It is the only door of the core to PostgreSQL. The type hides the driver
// on purpose: the services depend on the core, not on pgx.
type Pool struct {
	pool *pgxpool.Pool
}

// Open opens the pool of connections.
//
// The pool is lazy: it opens no connection until one is needed, and a
// database that is down does not stop a service from starting. The health
// check of the service is what asks the database whether it answers.
func Open(ctx context.Context, cfg Config) (*Pool, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("the connection string of the database is invalid: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("the pool of connections could not be opened: %w", err)
	}

	return &Pool{pool: pool}, nil
}

// Ping asks the database whether it answers.
//
// The question goes through the generated code, on the same connection and
// with the same driver as everything else the core does.
func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return errors.New("the pool of connections is not open")
	}

	queries := db.New(p.pool)
	if _, err := queries.Ping(ctx); err != nil {
		return fmt.Errorf("the database did not answer: %w", err)
	}
	return nil
}

// Queries returns the generated queries, bound to the pool.
//
// The repositories of the core use them. A service that needs a query of
// its own goes through a repository, never through this door.
func (p *Pool) Queries() *db.Queries {
	if p == nil || p.pool == nil {
		return nil
	}
	return db.New(p.pool)
}

// Close closes the pool and waits for the connections to end.
func (p *Pool) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
}
