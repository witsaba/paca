// Package core is the shared persistence library of the PACA monorepository.
//
// The services of the monorepository import it: the MCP server today, the
// worker and the REST backend tomorrow. It holds what they would otherwise
// write twice — the models of the domain, the contracts of the repositories
// and the migrations of the schema.
//
// The dependency rule of the hexagonal architecture holds here too:
//
//	domain/    the models and the rules of the domain. They know nothing
//	           about the database.
//	postgres/  the adapter that talks to PostgreSQL. It knows the models
//	           and it knows nothing about the services.
//	queries/   the SQL of the queries, read by the generator.
//	migrations/ the history of the schema, applied by a single writer.
//
// The library publishes no port and runs no container. A service maps its
// own environment onto the configuration of the core, opens the pool and
// uses the models.
//
// The primary key of the tables is the national document ID of the person,
// which the caller supplies. Nothing in the core generates an identifier.
package core

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// migrationFiles holds the history of the schema: the migrations, in the
// order they were written.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// ErrNoDatabase is the error of a migrator with no database behind it.
var ErrNoDatabase = errors.New("the database of the migrator is not open")

// Migration is the state of one migration of the schema.
type Migration struct {
	// Version is the number of the migration.
	Version int64

	// Name is the name of the file of the migration.
	Name string

	// Applied tells whether the migration has been applied to the database.
	Applied bool

	// AppliedAt is the moment the migration was applied. It is the zero
	// time when the migration has not been applied.
	AppliedAt time.Time
}

// Migrator applies the migrations of the schema.
//
// The schema has one writer, and it is this migrator. A service never
// migrates on boot: a service that migrates on boot races with the other
// replicas of itself.
type Migrator struct {
	provider *goose.Provider
}

// NewMigrator returns the migrator of the schema.
//
// The migrator takes the lock of the database while it applies a migration,
// so two migrators that run at the same time take turns instead of writing
// the same table twice.
func NewMigrator(db *sql.DB, logger *slog.Logger) (*Migrator, error) {
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

	return &Migrator{provider: provider}, nil
}

// Up applies every migration that has not been applied yet, and returns the
// ones it applied.
//
// The list is empty when the schema was already up to date: applying the
// migrations twice is not an error, it is nothing to do.
func (m *Migrator) Up(ctx context.Context) ([]Migration, error) {
	if m == nil || m.provider == nil {
		return nil, ErrNoDatabase
	}

	results, err := m.provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("the migrations could not be applied: %w", err)
	}

	applied := make([]Migration, 0, len(results))
	for _, result := range results {
		if result == nil || result.Source == nil {
			continue
		}
		applied = append(applied, Migration{
			Version: result.Source.Version,
			Name:    migrationName(result.Source.Path),
			Applied: true,
		})
	}

	return applied, nil
}

// Status returns the state of every migration of the schema.
func (m *Migrator) Status(ctx context.Context) ([]Migration, error) {
	if m == nil || m.provider == nil {
		return nil, ErrNoDatabase
	}

	statuses, err := m.provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("the state of the migrations could not be read: %w", err)
	}

	migrations := make([]Migration, 0, len(statuses))
	for _, status := range statuses {
		if status == nil || status.Source == nil {
			continue
		}
		migrations = append(migrations, Migration{
			Version:   status.Source.Version,
			Name:      migrationName(status.Source.Path),
			Applied:   status.AppliedAt.After(time.Time{}),
			AppliedAt: status.AppliedAt,
		})
	}

	return migrations, nil
}

// Pending reports whether a migration is waiting to be applied.
func (m *Migrator) Pending(ctx context.Context) (bool, error) {
	if m == nil || m.provider == nil {
		return false, ErrNoDatabase
	}

	pending, err := m.provider.HasPending(ctx)
	if err != nil {
		return false, fmt.Errorf("the pending migrations could not be read: %w", err)
	}

	return pending, nil
}

// Version returns the version of the schema of the database: the number of
// the last migration that was applied.
func (m *Migrator) Version(ctx context.Context) (int64, error) {
	if m == nil || m.provider == nil {
		return 0, ErrNoDatabase
	}

	version, err := m.provider.GetDBVersion(ctx)
	if err != nil {
		return 0, fmt.Errorf("the version of the schema could not be read: %w", err)
	}

	return version, nil
}

// migrationName returns the name of the file of a migration, without the
// directory it lives in.
func migrationName(sourcePath string) string {
	return path.Base(sourcePath)
}
