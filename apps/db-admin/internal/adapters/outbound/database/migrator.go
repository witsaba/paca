// Package database provides the adapters that talk to the database: the
// migrator that applies the schema through the library, and the checker that
// gathers the health of the database into the single check of the service.
//
// It is an outbound adapter: the application asks it about the database, and
// the answer comes from the database itself. The pool and the driver belong
// to the library that owns the database, so this package names neither:
// the composition root binds the pool of the library to the port of the
// pinger.
package database

import (
	"context"

	"github.com/witsaba/paca/apps/db-admin/internal/ports"
	"github.com/witsaba/paca/packages/database"
	"github.com/witsaba/paca/packages/domain/schema"
)

// The adapters of this package are the ones the application talks to.
var (
	_ ports.Migrator = (*Migrator)(nil)
	_ ports.Checker  = (*Checker)(nil)
)

// Migrator applies the migrations of the schema, through the database
// library.
//
// The library is the single writer of the schema of the monorepository:
// this adapter never writes a migration of its own, it asks the migrator of
// the library to do it.
type Migrator struct {
	migrator *database.Migrator
}

// NewMigrator builds the adapter of the migrator of the database library.
//
// The migrator of the library takes the lock of the database while it
// applies a migration, so two replicas of the service take turns instead of
// writing the same table twice.
//
// The migrator of the library is built by the composition root, over the
// pool of the library: the adapter names the migrator, never the handle it
// applies the schema with.
func NewMigrator(migrator *database.Migrator) *Migrator {
	return &Migrator{migrator: migrator}
}

// Up applies every migration that has not been applied yet, and returns the
// ones it applied.
//
// The list is empty when the schema was already up to date: applying the
// migrations twice is not an error, it is nothing to do.
//
// The record of a migration is the one of the domain: the model has a single
// owner, and this adapter maps nothing.
func (m *Migrator) Up(ctx context.Context) ([]schema.Migration, error) {
	return m.migrator.Up(ctx)
}

// Status returns the state of every migration of the schema.
func (m *Migrator) Status(ctx context.Context) ([]schema.Migration, error) {
	return m.migrator.Status(ctx)
}

// Pending reports whether a migration is waiting to be applied.
func (m *Migrator) Pending(ctx context.Context) (bool, error) {
	return m.migrator.Pending(ctx)
}

// Version returns the version of the schema of the database.
func (m *Migrator) Version(ctx context.Context) (int64, error) {
	return m.migrator.Version(ctx)
}
