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

	"github.com/witsaba/paca/packages/domain/health"
	"github.com/witsaba/paca/packages/domain/user"
)

// ErrNoDatabase is the error of a use case with no database behind it.
//
// The service may run without a database: the tools are still there, and
// every call says that the database is not.
var ErrNoDatabase = errors.New("the service has no database configured")

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

// Pinger is the outbound port that asks the database whether it answers.
//
// The capability has one owner: the library that owns the database publishes
// the pool, and the pool answers the question. The service depends on the
// question and never on the handle that answers it.
type Pinger interface {
	// Ping returns nil when the database answers.
	Ping(ctx context.Context) error
}

// Clock is the outbound port that tells the application what time it is.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// UsersQuerier is the inbound port of the three tools of the users.
//
// The tools of the Model Context Protocol speak in text: a document ID, a
// name, a phone, an email. The use case speaks in the model of the domain,
// and the port is where the two meet.
type UsersQuerier interface {
	// Onboard registers a person and starts their onboarding.
	//
	// The document ID is required: it is the number the person carries, and
	// nothing generates it. The name, the phone and the email are optional.
	Onboard(ctx context.Context, documentID, fullName, phone, email string) (user.User, error)

	// ChangeStatus moves a user to another status and writes the
	// observation of the move.
	ChangeStatus(ctx context.Context, documentID, status, observation string) (user.User, error)

	// UpdateContactData writes the data of a user. A field that is empty is
	// a field that does not change.
	UpdateContactData(ctx context.Context, documentID, fullName, phone, email string) (user.User, error)
}
