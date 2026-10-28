// Package ports holds the contracts of the hexagonal application: what comes
// in, and what goes out.
//
// The application layer depends on these interfaces only. The adapters
// implement them, and nothing in the middle knows which adapter it is talking
// to.
package ports

import (
	"context"
	"errors"
	"time"

	"github.com/witsaba/paca/apps/db-admin/internal/domain/health"
	"github.com/witsaba/paca/apps/db-admin/internal/domain/schema"
)

// ErrNoDatabase is the error of a use case with no database behind it.
//
// The database admin service is the writer of the schema: it cannot run
// without one. The error exists so a caller can tell the difference between
// a database that is down and a database that was never configured.
var ErrNoDatabase = errors.New("the service has no database configured")

// Migrator is the outbound port of the migration use case: it applies the
// migrations of the schema and reads its state.
//
// The migrator of the core implements it. The service never writes a
// migration of its own.
type Migrator interface {
	// Up applies every migration that has not been applied yet, and
	// returns the ones it applied. The list is empty when the schema was
	// already up to date.
	Up(ctx context.Context) ([]schema.Migration, error)

	// Status returns the state of every migration of the schema.
	Status(ctx context.Context) ([]schema.Migration, error)

	// Pending reports whether a migration is waiting to be applied.
	Pending(ctx context.Context) (bool, error)

	// Version returns the version of the schema of the database.
	Version(ctx context.Context) (int64, error)
}

// Pinger is the outbound port that asks the database whether it answers.
type Pinger interface {
	// Ping returns nil when the database answers.
	Ping(ctx context.Context) error
}

// Checker is the outbound port of the health use case: it reports whether one
// component of the service is healthy.
type Checker interface {
	// Name identifies the component the checker watches.
	Name() string
	// Check returns nil when the component is healthy, health.ErrDegraded
	// when the component still works but not perfectly, and any other error
	// when it failed.
	Check(ctx context.Context) error
}

// HealthQuerier is the inbound port of the health use case. Every adapter
// that exposes the health of the service depends on this contract, never on
// the implementation of the use case.
type HealthQuerier interface {
	// Health returns the health of the service at this moment.
	Health(ctx context.Context) (health.Report, error)
}

// Clock is the outbound port that tells the application what time it is.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}
