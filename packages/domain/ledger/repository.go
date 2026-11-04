package ledger

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
)

// Repository is the port of the storage of the book.
//
// The port is what a use case needs, not the shape of a table: it writes a
// movement, reads the movements of a period, and reads the balance of
// every account of a business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Record writes a movement of the book and answers it with the
	// identity that the storage generated.
	//
	// The error is ErrInvalidAmount, ErrSameAccount or
	// ErrInvalidDescription when the movement breaks a rule of the
	// domain, and ErrNotFound when the business or one of the accounts is
	// not there.
	Record(ctx context.Context, entry Entry) (Entry, error)

	// List returns the movements of a business between two days, in the
	// order of the day and of the identity.
	//
	// The days are the two ends of the period: the movement of the first
	// day and the movement of the last one are both in the answer.
	//
	// The error is ErrNotFound when no business carries that identity.
	List(ctx context.Context, businessID business.ID, from, to Date) ([]Entry, error)

	// Balances returns the balance of every account of a business, as of a
	// day.
	//
	// The account that never moved is in the answer with a balance of
	// zero: the person reads the accounts of their book, and an account
	// that is missing reads like an account that does not exist.
	//
	// The error is ErrNotFound when no business carries that identity.
	Balances(ctx context.Context, businessID business.ID, asOf Date) ([]Balance, error)

	// Summary returns the money of a period of the book: what the business
	// earned and what it spent during the period.
	//
	// The two ends of the period are both included: the movement of the
	// first day and the movement of the last one are both inside the money
	// of the period.
	//
	// The error is ErrNotFound when no business carries that identity.
	Summary(ctx context.Context, businessID business.ID, from, to Date) (Summary, error)
}
