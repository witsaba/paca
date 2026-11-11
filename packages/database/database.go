// Package database is the persistence library of the PACA monorepository.
//
// The services of the monorepository import it: the MCP server today, the
// worker and the REST backend tomorrow. It holds what they would otherwise
// write twice — the pool of connections, the queries, the migrations of the
// schema and the repositories that write the rows.
//
// The dependency rule of the hexagonal architecture holds here too:
//
//	postgres/  the adapter that talks to PostgreSQL. It knows the models
//	           of the domain and it knows nothing about the services.
//	queries/   the SQL of the queries, read by the generator.
//	migrations/ the history of the schema, applied by a single writer.
//
// The models of the domain live in another module, packages/domain, and
// this one depends on it. The rule of the direction is the toolchain:
// packages/domain imports nothing but the standard library, so the domain
// cannot reach the queries of this module.
//
// The library publishes no port and runs no container. A service maps its
// own environment onto the configuration of the library, opens the pool
// and uses the repositories.
//
// The primary key of the tables is the national document ID of the person,
// which the caller supplies. Nothing in the library generates an identifier.
package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/packages/domain/schema"
)

// migrationFiles holds the history of the schema: the migrations, in the
// order they were written.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// ErrNoDatabase is the error of a migrator with no database behind it.
var ErrNoDatabase = errors.New("the database of the migrator is not open")

// Migrator applies the migrations of the schema.
//
// The schema has one writer, and it is this migrator. A service never
// migrates on boot: a service that migrates on boot races with the other
// replicas of itself.
type Migrator struct {
	provider *goose.Provider

	// tracer traces the work of the migrator. It is nil when the caller
	// asked for no tracing, and the work leaves no span then.
	tracer trace.Tracer
}

// NewMigrator returns the migrator of the schema.
//
// The migrator takes the lock of the database while it applies a migration,
// so two migrators that run at the same time take turns instead of writing
// the same table twice.
func NewMigrator(db *sql.DB, logger *slog.Logger, opts ...MigratorOption) (*Migrator, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	if logger == nil {
		logger = slog.Default()
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("the lock of the database could not be taken: %w", err)
	}

	migrator := &Migrator{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(migrator)
	}

	if migrator.tracer != nil {
		locker = &tracedSessionLocker{inner: locker, tracer: migrator.tracer}
	}

	// The runner looks for the migrations at the root of the files it
	// receives, and the embedded files live in a directory. The directory
	// becomes the root here.
	migrationFS, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("the migrations could not be read: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		migrationFS,
		goose.WithSessionLocker(locker),
		goose.WithSlog(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("the migrator could not be created: %w", err)
	}

	migrator.provider = provider

	return migrator, nil
}

// Up applies every migration that has not been applied yet, and returns the
// ones it applied.
//
// The list is empty when the schema was already up to date: applying the
// migrations twice is not an error, it is nothing to do.
func (m *Migrator) Up(ctx context.Context) ([]schema.Migration, error) {
	if m == nil || m.provider == nil {
		return nil, ErrNoDatabase
	}

	ctx, span := m.start(ctx, spanMigrate)
	defer span.End()

	results, err := m.provider.Up(ctx)
	if err != nil {
		recordFailure(span, err)
		return nil, fmt.Errorf("the migrations could not be applied: %w", err)
	}

	applied := make([]schema.Migration, 0, len(results))
	for _, result := range results {
		if result == nil || result.Source == nil {
			continue
		}
		applied = append(applied, schema.Migration{
			Version: result.Source.Version,
			Name:    migrationName(result.Source.Path),
			Applied: true,
		})
	}

	span.SetAttributes(attribute.String(attributeSchemaApplied, theNamesOf(applied)))

	return applied, nil
}

// Pending reports whether a migration is waiting to be applied.
func (m *Migrator) Pending(ctx context.Context) (bool, error) {
	if m == nil || m.provider == nil {
		return false, ErrNoDatabase
	}

	ctx, span := m.start(ctx, spanPending)
	defer span.End()

	pending, err := m.provider.HasPending(ctx)
	if err != nil {
		recordFailure(span, err)
		return false, fmt.Errorf("the pending migrations could not be read: %w", err)
	}

	span.SetAttributes(attribute.Bool(attributeSchemaPending, pending))

	return pending, nil
}

// Version returns the version of the schema of the database: the number of
// the last migration that was applied.
func (m *Migrator) Version(ctx context.Context) (int64, error) {
	if m == nil || m.provider == nil {
		return 0, ErrNoDatabase
	}

	ctx, span := m.start(ctx, spanVersion)
	defer span.End()

	version, err := m.provider.GetDBVersion(ctx)
	if err != nil {
		recordFailure(span, err)
		return 0, fmt.Errorf("the version of the schema could not be read: %w", err)
	}

	span.SetAttributes(attribute.Int64(attributeSchemaVersion, version))

	return version, nil
}

// start opens the span of a piece of work of the migrator.
//
// The migrator never opens a **root** span: it enriches a trace that is
// already open. The boot of the writer of the schema opens one, and the
// health check of the service opens none — the runtime asking whether the
// service is alive is not work, and a span of its own would be an orphan
// in the store.
//
// A migrator without a tracer opens no span at all: the work of the schema
// leaves no trace behind, and the behaviour of the library is the same.
func (m *Migrator) start(ctx context.Context, name string) (context.Context, trace.Span) {
	if m.tracer == nil || !trace.SpanContextFromContext(ctx).IsValid() {
		return ctx, trace.SpanFromContext(ctx)
	}

	return m.tracer.Start(ctx, name)
}

// theNamesOf answers the names of the migrations, as one text.
//
// The name of a migration is the name of its file, and the file carries a
// number: the name says which migration it was, and the number says in
// which order.
func theNamesOf(applied []schema.Migration) string {
	names := make([]string, 0, len(applied))
	for _, one := range applied {
		names = append(names, fmt.Sprintf("%d_%s", one.Version, one.Name))
	}

	return strings.Join(names, ",")
}

// migrationName returns the name of the file of a migration, without the
// directory it lives in.
func migrationName(sourcePath string) string {
	return path.Base(sourcePath)
}
