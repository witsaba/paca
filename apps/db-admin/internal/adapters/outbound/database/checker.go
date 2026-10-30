package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/witsaba/paca/apps/db-admin/internal/ports"
	"github.com/witsaba/paca/packages/domain/health"
)

// ErrPendingMigrations is what the checker reports when the schema of the
// database is behind the migrations of the repository.
//
// It is a degraded error: the service still works, and it still serves, but
// the schema it writes is not the one the code expects.
var ErrPendingMigrations = errors.New("the schema has pending migrations")

// pendingError is the error of a schema that is behind.
//
// It is a degraded error: the service still works, and it still serves, but
// the schema it writes is not the one the code expects.
type pendingError struct{}

// Error says what is wrong with the schema.
func (pendingError) Error() string {
	return ErrPendingMigrations.Error()
}

// Is makes the pending error a degraded one, so the health use case reports
// the service as degraded instead of unhealthy.
func (pendingError) Is(target error) bool {
	return target == health.ErrDegraded || target == ErrPendingMigrations
}

// Checker reports whether the database answers and whether the schema of it
// is up to date.
//
// It is the single check of the service: the service does nothing but the
// database, so the database is the only thing that can be wrong with it.
type Checker struct {
	pinger   ports.Pinger
	migrator ports.Migrator
}

// NewChecker builds the checker of the database.
//
// The pinger is the port of the capability: the pool of the library answers
// the question, and the adapter never names the handle that answers it.
func NewChecker(pinger ports.Pinger, migrator ports.Migrator) *Checker {
	return &Checker{pinger: pinger, migrator: migrator}
}

// Name identifies the component the checker watches.
func (c *Checker) Name() string {
	return "database"
}

// Check asks the database whether it answers, and whether the schema of it
// has nothing left to apply.
//
// A database that does not answer fails the check: the service is the writer
// of the schema, and a writer that cannot reach the database is not a
// service. A schema with a pending migration degrades the check: the
// service still serves, but the schema is behind.
func (c *Checker) Check(ctx context.Context) error {
	if err := c.pinger.Ping(ctx); err != nil {
		return err
	}

	pending, err := c.migrator.Pending(ctx)
	if err != nil {
		return fmt.Errorf("the state of the schema could not be read: %w", err)
	}

	if pending {
		return pendingError{}
	}

	return nil
}
