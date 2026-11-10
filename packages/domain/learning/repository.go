package learning

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
)

// Repository is the port of the storage of the learning log.
//
// The port is what a use case needs, not the shape of a table: it writes an
// entry and it reads the entries of a business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
//
// There is NO update and NO delete: the log is never rewritten, and the
// table has no query that changes the words of an entry.
type Repository interface {
	// Create writes the entry and answers it with the identity that the
	// storage generated.
	//
	// The entry is about a batch of the business, and it may name a
	// production of that batch and the entry it answers: the table carries
	// the business with the answer and with the batch, so the row of the
	// entry of another business is refused.
	//
	// The error is ErrBusinessNotFound when the business of the entry is not
	// there, ErrRecipeOfAnotherBusiness when the batch is not a batch of the
	// business, ErrProductionOfAnotherBusiness when the production is not a
	// production of the business, ErrEntryOfAnotherBusiness when the entry
	// it answers is not an entry of the business, ErrAlreadyAnswered when
	// the entry it answers was answered already, ErrInvalidValue when the
	// row breaks a rule of the table, and ErrNoStorage when the storage is
	// not open.
	Create(ctx context.Context, entry Entry) (Entry, error)

	// List returns the entries of the business, in the order they were
	// written, each one with the batch it names and the entry it answers.
	//
	// The list holds the entries of the business and of no other one, and it
	// is empty when the business wrote nothing yet. An entry is never
	// deleted: the list holds the ones that were answered too.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Entry, error)
}
