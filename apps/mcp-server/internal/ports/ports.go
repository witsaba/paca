// Package ports holds the contracts of the hexagonal application: what comes
// in, and what goes out.
//
// The application layer depends on these interfaces only. The adapters
// implement them, and nothing in the middle knows which adapter it is talking
// to.
package ports

import (
	"context"
	"time"

	"github.com/witsaba/paca/apps/mcp-server/internal/domain/health"
)

// HealthQuerier is the inbound port of the health use case. Every adapter
// that exposes the health of the service depends on this contract, never on
// the implementation of the use case.
type HealthQuerier interface {
	// Health returns the health of the service at this moment.
	Health(ctx context.Context) (health.Report, error)
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

// Clock is the outbound port that tells the application what time it is.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}
