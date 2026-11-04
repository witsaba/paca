// Package ports holds the contracts of the health of a service.
//
// The contracts are narrow: one interface, one capability, and the methods
// the use case actually calls. A port is owned by the consumer of the
// contract, and it is written in the language of the consumer: the health is
// the model of the domain, never the model of the protocol.
//
// The package imports nothing but the standard library and the model of the
// health, so a service that binds these contracts never drags a protocol
// along.
package ports

import (
	"context"
	"time"

	"github.com/witsaba/paca/packages/domain/health"
)

// HealthQuerier is the inbound port of the health: the endpoint asks for the
// health of the service and receives a report.
//
// The endpoint depends on this contract and never on the implementation of
// the reporter, so the report can come from somewhere else without the
// endpoint changing.
type HealthQuerier interface {
	// Health returns the health of the service at this moment.
	Health(ctx context.Context) (health.Report, error)
}

// Checker is the outbound port of the reporter: it reports whether one
// component of the service is healthy.
//
// The implementation of the checker belongs to the service: the transport of
// the MCP server checks itself, and the database of the services answers
// when it is asked. The reporter knows the question and never the answer.
type Checker interface {
	// Name identifies the component the checker watches.
	Name() string
	// Check returns nil when the component is healthy, health.ErrDegraded
	// when the component still works but not perfectly, and any other error
	// when it failed.
	Check(ctx context.Context) error
}

// Clock is the outbound port that tells the reporter what time it is.
//
// The reporter measures the uptime of the service and the latency of every
// check, and both are facts of the clock: the implementation of the clock
// belongs to the platform, and the reporter only knows the question.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}
