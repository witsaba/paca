// Package postgres is the adapter of the database library that talks to
// PostgreSQL.
//
// The package owns the pool of connections and the configuration of it. It
// hides the driver: a service that imports the library never names pgx, never
// names a connection string, and never opens a connection of its own.
//
// The dependency rule holds: the package knows the models of the domain and
// the generated code, and the domain knows nothing about it.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/packages/database/internal/db"
)

// ErrNoURL is the error of a configuration without a connection string.
var ErrNoURL = errors.New("the connection string of the database is empty")

// ErrNoPool is the error of a pool of connections that is not open.
var ErrNoPool = errors.New("the pool of connections is not open")

// Config is the configuration of the pool of connections.
//
// The library reads no environment variable: a service maps its own environment
// onto this configuration, so the same library serves every service of the
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

// DefaultConfig returns the configuration the library uses when the service
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
// It is the only door of the library to PostgreSQL. The type hides the driver
// on purpose: the services depend on the library, not on pgx.
type Pool struct {
	pool *pgxpool.Pool

	// tracer traces the statements the pool runs. It is nil when the
	// caller asked for no tracing, and the statements leave no span then.
	tracer trace.Tracer

	// mu guards the view of the handle that the tools of the schema need.
	mu sync.Mutex

	// sql is the view of the pool that database/sql speaks to.
	sql *sql.DB
}

// Option configures the pool of connections.
type Option func(*Pool)

// WithTracer makes the pool trace every statement it runs.
//
// The tracer is the API of the tracing: the library builds no provider,
// exports no span and reads no environment variable. The composition root
// of the service owns that, and it hands the tracer over here.
//
// A pool without a tracer traces nothing: the statements leave no span
// behind, and the behaviour of the library is the same.
func WithTracer(tracer trace.Tracer) Option {
	return func(p *Pool) {
		p.tracer = tracer
	}
}

// Open opens the pool of connections.
//
// The pool is lazy: it opens no connection until one is needed, and a
// database that is down does not stop a service from starting. The health
// check of the service is what asks the database whether it answers.
func Open(ctx context.Context, cfg Config, opts ...Option) (*Pool, error) {
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

	opened := &Pool{pool: pool}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(opened)
	}

	return opened, nil
}

// Ping asks the database whether it answers.
//
// The question goes through the generated code, on the same connection and
// with the same driver as everything else the library does.
//
// The question is the health of the service and not work: the container
// runtime asks it every few seconds, and a span for each question would
// fill the traces with the noise of the runtime. The statement of the
// health check is therefore not traced.
func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return ErrNoPool
	}

	queries := db.New(p.healthDoor())
	if _, err := queries.Ping(ctx); err != nil {
		return fmt.Errorf("the database did not answer: %w", err)
	}
	return nil
}

// Queries returns the generated queries, bound to the pool.
//
// The repositories of the library use them. A service that needs a query of
// its own goes through a repository, never through this door.
func (p *Pool) Queries() *db.Queries {
	if p == nil || p.pool == nil {
		return nil
	}
	return db.New(p.door(nil))
}

// healthDoor returns the door of the health check: the pool itself, never
// the decorated one.
//
// The health check is the runtime asking whether the service is alive. It
// is not the work of the service, and a span for every question would fill
// the traces with the noise of the runtime.
func (p *Pool) healthDoor() db.DBTX {
	return p.pool
}

// door returns the door of the generated queries: the transaction of the
// context, or the pool when it carries none, decorated with the spans of
// the tracing.
//
// The seam is the one the generator owns, so every query of every
// repository of the library goes through it.
func (p *Pool) door(inner db.DBTX) db.DBTX {
	if inner == nil {
		inner = p.pool
	}

	if p.tracer == nil {
		return inner
	}

	return &tracedDBTX{inner: inner, tracer: p.tracer}
}

// SQL returns the handle of the database that the tools of the schema need.
//
// The migrator of the library speaks through database/sql: goose is the tool
// that applies the migrations, and it asks for a *sql.DB. The view is over
// the same pool, so the migrations and the queries of the services share the
// connections, the driver and the limits of the pool.
//
// The view is built once and kept: the pool closes it when the pool closes.
// A pool that is not there has no view, and the caller says so.
func (p *Pool) SQL() *sql.DB {
	if p == nil || p.pool == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.sql == nil {
		p.sql = stdlib.OpenDBFromPool(p.pool)
	}

	return p.sql
}

// Close closes the pool and waits for the connections to end.
func (p *Pool) Close() {
	if p == nil || p.pool == nil {
		return
	}

	p.mu.Lock()
	if p.sql != nil {
		// The view shares the connections of the pool: closing it never
		// closes a connection, and the pool closes them right after.
		_ = p.sql.Close()
		p.sql = nil
	}
	p.mu.Unlock()

	p.pool.Close()
}
