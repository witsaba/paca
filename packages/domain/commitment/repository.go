package commitment

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
)

// Repository is the port of the storage of the commitments.
//
// The port is what a use case needs, not the shape of a table: it writes a
// promise, it writes the day a promise was met, and it reads the promises of
// a business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Create writes the promise and returns it with the identity that the
	// storage generated.
	//
	// The promise is written OPEN: it carries no day of meeting, and the
	// book never writes one at the creation.
	//
	// The error is ErrBusinessNotFound when the business of the promise is
	// not there, ErrInvalidValue when the row breaks a rule of the table,
	// and ErrNoStorage when the storage is not open.
	Create(ctx context.Context, c Commitment) (Commitment, error)

	// Meet writes the day the promise was met and returns the promise that
	// was kept.
	//
	// The rule of the domain: a promise is kept once. The day of meeting is
	// written over an OPEN promise and of no other one, so a caller that
	// bypasses the domain never hides the day the book already holds.
	//
	// The error is ErrAlreadyMet when the promise carries a day of meeting
	// already, ErrNotFound when the promise is not there, ErrInvalidValue
	// when the row breaks a rule of the table, and ErrNoStorage when the
	// storage is not open.
	Meet(ctx context.Context, met Commitment) (Commitment, error)

	// List returns the promises of the business, in the order of their
	// identity, with the day each one was met when it was met.
	//
	// The list holds the promises of the business and of no other one, and
	// it is empty when the business promised nothing yet. A promise is
	// never deleted: the list holds the ones that were kept too.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Commitment, error)
}
