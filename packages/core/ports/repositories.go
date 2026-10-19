package ports

import (
	"context"

	"github.com/witsaba/paca/packages/core/domain"
)

// The repository ports of PACA.
//
// A repository is a driven port: the application needs to read and to write
// the state of the world, and the interface is what it needs, not what the
// storage happens to offer. The SQLite adapter implements these interfaces;
// a future adapter over another storage implements them too, and the
// application never notices.
//
// The rules every implementation shares:
//
//   - A failure is a typed error of the domain. A record that does not exist
//     is a not-found error, never an empty result and never a nil.
//   - A stored entity comes back with its identity filled in.
//   - The order of a list is defined by the port, so that two implementations
//     return the same list in the same order.

// AccountRepository stores the accounts, the places where money is held.
type AccountRepository interface {
	// Create stores a new account and returns it with its identity filled
	// in.
	Create(ctx context.Context, account domain.Account) (domain.Account, error)

	// GetByID returns the account with the given identity. An account that
	// does not exist is a not-found error.
	GetByID(ctx context.Context, id int64) (domain.Account, error)

	// List returns every account, ordered by identity.
	List(ctx context.Context) ([]domain.Account, error)

	// ListActive returns the accounts that are still open, ordered by
	// identity.
	ListActive(ctx context.Context) ([]domain.Account, error)
}

// ContactRepository stores the contacts, the people and the entities that
// take part in a transaction.
type ContactRepository interface {
	// Create stores a new contact and returns it with its identity filled
	// in.
	Create(ctx context.Context, contact domain.Contact) (domain.Contact, error)

	// GetByID returns the contact with the given identity. A contact that
	// does not exist is a not-found error.
	GetByID(ctx context.Context, id int64) (domain.Contact, error)

	// List returns every contact, ordered by identity.
	List(ctx context.Context) ([]domain.Contact, error)

	// ListByKind returns the contacts of the given kind, ordered by
	// identity.
	ListByKind(ctx context.Context, kind domain.ContactKind) ([]domain.Contact, error)
}

// LoanRepository stores the loans, the debt contracts.
type LoanRepository interface {
	// Create stores a new loan and returns it with its identity filled in.
	Create(ctx context.Context, loan domain.Loan) (domain.Loan, error)

	// GetByID returns the loan with the given identity. A loan that does
	// not exist is a not-found error.
	GetByID(ctx context.Context, id int64) (domain.Loan, error)

	// List returns every loan, ordered by identity.
	List(ctx context.Context) ([]domain.Loan, error)

	// ListByStatus returns the loans with the given status, ordered by
	// identity.
	ListByStatus(ctx context.Context, status domain.LoanStatus) ([]domain.Loan, error)

	// ListByContact returns the loans with the given contact, ordered by
	// identity.
	ListByContact(ctx context.Context, contactID int64) ([]domain.Loan, error)

	// Update stores the changes of an existing loan. A loan that does not
	// exist is a not-found error.
	Update(ctx context.Context, loan domain.Loan) error
}

// MovementRepository stores the movements, the money events.
type MovementRepository interface {
	// Create stores a new movement and returns it with its identity filled
	// in.
	Create(ctx context.Context, movement domain.Movement) (domain.Movement, error)

	// GetByID returns the movement with the given identity. A movement that
	// does not exist is a not-found error.
	GetByID(ctx context.Context, id int64) (domain.Movement, error)

	// List returns the movements that match the filter, most recent first.
	// An empty filter matches every movement.
	List(ctx context.Context, filter MovementFilter) ([]domain.Movement, error)

	// Update stores the changes of an existing movement. A movement that
	// does not exist is a not-found error.
	Update(ctx context.Context, movement domain.Movement) error

	// Delete removes a movement. A movement that does not exist is a
	// not-found error.
	Delete(ctx context.Context, id int64) error

	// BalanceByAccount returns the balance of every account that holds
	// movements, in Colombian pesos. The balance is the money in minus the
	// money out.
	BalanceByAccount(ctx context.Context) (map[int64]int64, error)

	// TotalByKind returns the total of every kind of movement over a
	// period, in Colombian pesos. An empty period covers every movement.
	TotalByKind(ctx context.Context, period Period) (map[domain.MovementKind]int64, error)

	// TotalByCategory returns the total of the movements of every category
	// over a period, in Colombian pesos. An empty period covers every
	// movement.
	TotalByCategory(ctx context.Context, period Period) (map[string]int64, error)
}

// LoanPaymentRepository stores the obligations of the loans.
type LoanPaymentRepository interface {
	// Create stores a new obligation and returns it with its identity
	// filled in.
	Create(ctx context.Context, payment domain.LoanPayment) (domain.LoanPayment, error)

	// GetByID returns the obligation with the given identity. An obligation
	// that does not exist is a not-found error.
	GetByID(ctx context.Context, id int64) (domain.LoanPayment, error)

	// ListByLoan returns the obligations of a loan, ordered by their
	// position in the schedule.
	ListByLoan(ctx context.Context, loanID int64) ([]domain.LoanPayment, error)

	// ListUnpaid returns the obligations that are still outstanding, ordered
	// by the day they fall due.
	ListUnpaid(ctx context.Context) ([]domain.LoanPayment, error)

	// MarkPaid settles an obligation: it records the day it was paid and
	// the movement that settled it. An obligation that does not exist is a
	// not-found error.
	MarkPaid(ctx context.Context, id int64, paidOn string, movementID *int64) error
}

// MovementFilter narrows a query over the movements.
//
// An empty filter matches every movement. The fields that are set narrow the
// result: a filter over a kind matches only that kind, and a period that
// holds only a start matches everything from that day on.
type MovementFilter struct {
	// AccountID narrows the result to the movements of one account.
	AccountID int64

	// Kinds narrows the result to the given kinds. An empty list matches
	// every kind.
	Kinds []domain.MovementKind

	// LoanID narrows the result to the movements of one loan.
	LoanID int64

	// Period narrows the result to the movements of a period.
	Period

	// Limit caps the number of movements returned. Zero means no cap.
	Limit int
}

// Period is a range of days, as YYYY-MM-DD.
//
// An empty start or an empty end means the range is open on that side.
type Period struct {
	// From is the first day of the period, inclusive.
	From string

	// To is the last day of the period, inclusive.
	To string
}
