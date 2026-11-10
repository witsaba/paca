package party

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
)

// Repository is the port of the storage of the parties.
//
// The port is what a use case needs, not the shape of a table: it writes a
// party and it reads the parties of a business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Create writes a party of the business and returns it with the
	// identity that the storage generated.
	//
	// The name of a party is not unique inside the business: the book
	// never refuses two parties that share a name, and the storage holds
	// the same rule as a defense in depth.
	//
	// The error is ErrNotFound when the business of the party is not
	// there, ErrInvalidValue when the row breaks a rule of the table, and
	// ErrNoStorage when the storage is not open.
	Create(ctx context.Context, p Party) (Party, error)

	// List returns the parties of a business, in the order of their
	// identity.
	//
	// The list holds every party of the business, the ones that share a
	// name among them. It is empty when the business knows no party yet.
	//
	// The use case reads the business before it reads the parties: a
	// business that is not there is refused with ErrNotFound, and the list
	// is empty only when the business knows no party.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, businessID business.ID) ([]Party, error)
}
