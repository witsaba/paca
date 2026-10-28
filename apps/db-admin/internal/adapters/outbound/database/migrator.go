// Package database provides the adapters that talk to the database: the
// migrator that applies the schema through the core, the pinger that asks
// the database whether it answers, and the checker that gathers the two into
// the single health check of the service.
//
// It is an outbound adapter: the application asks it about the database, and
// the answer comes from the database itself.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/witsaba/paca/apps/db-admin/internal/domain/schema"
	"github.com/witsaba/paca/apps/db-admin/internal/ports"
	"github.com/witsaba/paca/packages/core"
)

// The adapters of this package are the ones the application talks to.
var (
	_ ports.Migrator = (*Migrator)(nil)
	_ ports.Pinger   = (*Pinger)(nil)
	_ ports.Checker  = (*Checker)(nil)
)

// Migrator applies the migrations of the schema, through the core.
//
// The core is the single writer of the schema of the monorepository: this
// adapter never writes a migration of its own, it asks the migrator of the
// core to do it.
type Migrator struct {
	migrator *core.Migrator
}

// NewMigrator builds the adapter of the migrator of the core.
//
// The migrator of the core takes the lock of the database while it applies a
// migration, so two replicas of the service take turns instead of writing
// the same table twice.
func NewMigrator(db *sql.DB, logger *slog.Logger) (*Migrator, error) {
	migrator, err := core.NewMigrator(db, logger)
	if err != nil {
		return nil, fmt.Errorf("the migrator of the core could not be created: %w", err)
	}
	return &Migrator{migrator: migrator}, nil
}

// Up applies every migration that has not been applied yet, and returns the
// ones it applied.
//
// The list is empty when the schema was already up to date: applying the
// migrations twice is not an error, it is nothing to do.
func (m *Migrator) Up(ctx context.Context) ([]schema.Migration, error) {
	applied, err := m.migrator.Up(ctx)
	if err != nil {
		return nil, err
	}

	migrations := make([]schema.Migration, 0, len(applied))
	for _, migration := range applied {
		migrations = append(migrations, schema.Migration{
			Version:   migration.Version,
			Name:      migration.Name,
			Applied:   migration.Applied,
			AppliedAt: migration.AppliedAt,
		})
	}

	return migrations, nil
}

// Status returns the state of every migration of the schema.
func (m *Migrator) Status(ctx context.Context) ([]schema.Migration, error) {
	statuses, err := m.migrator.Status(ctx)
	if err != nil {
		return nil, err
	}

	migrations := make([]schema.Migration, 0, len(statuses))
	for _, status := range statuses {
		migrations = append(migrations, schema.Migration{
			Version:   status.Version,
			Name:      status.Name,
			Applied:   status.Applied,
			AppliedAt: status.AppliedAt,
		})
	}

	return migrations, nil
}

// Pending reports whether a migration is waiting to be applied.
func (m *Migrator) Pending(ctx context.Context) (bool, error) {
	return m.migrator.Pending(ctx)
}

// Version returns the version of the schema of the database.
func (m *Migrator) Version(ctx context.Context) (int64, error) {
	return m.migrator.Version(ctx)
}
