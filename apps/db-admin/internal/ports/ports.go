// Package ports holds the contracts of the hexagonal application: what comes
// in, and what goes out.
//
// The application layer depends on these interfaces only. The adapters
// implement them, and nothing in the middle knows which adapter it is talking
// to.
package ports

import (
	"context"

	"github.com/witsaba/paca/packages/domain/schema"
	healthports "github.com/witsaba/paca/packages/health/ports"
)

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

// Checker is the outbound port of the health of the service: it reports
// whether one component of the service is healthy.
//
// The health has one owner: the shared health library. The service publishes
// the contract under its own name, and the contract it publishes is the one
// of the library.
type Checker = healthports.Checker

// HealthQuerier is the inbound port of the health of the service.
//
// The health has one owner: the shared health library. The service publishes
// the contract under its own name, and the contract it publishes is the one
// of the library.
type HealthQuerier = healthports.HealthQuerier
