package account

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
)

// Repository is the port of the storage of the accounts.
//
// The port is what a use case needs, not the shape of a table: it writes the
// accounts of a business and reads them back.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Create writes one account and returns it with the identity that the
	// storage generated.
	//
	// The error is ErrAlreadyExists when the name of the account is taken
	// inside the business that owns it.
	Create(ctx context.Context, account Account) (Account, error)

	// CreateMany writes the accounts of a business in one transaction, and
	// returns them with the identities that the storage generated.
	//
	// The set of accounts of a business is written whole or not at all: a
	// book that starts with half of its accounts is a book that lies about
	// what the person has.
	//
	// The error is ErrAlreadyExists when the name of one of the accounts is
	// taken inside the business that owns them.
	CreateMany(ctx context.Context, accounts []Account) ([]Account, error)

	// Find returns the account that carries that identity.
	//
	// The error is ErrNotFound when no account carries that identity.
	Find(ctx context.Context, id ID) (Account, error)

	// List returns the accounts of a business, in the order of their
	// identity.
	//
	// The error is ErrNotFound when no business carries that identity.
	List(ctx context.Context, businessID business.ID) ([]Account, error)
}
