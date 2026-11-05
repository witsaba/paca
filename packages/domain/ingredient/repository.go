package ingredient

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
)

// Repository is the port of the storage of the catalog.
//
// The port is what a use case needs, not the shape of a table: it reads the
// ingredients of a business and it writes one of them.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// List returns the ingredients of a business, in the order of their
	// identity.
	//
	// The list is empty when the business bought nothing yet: the catalog
	// of a business begins empty.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, businessID business.ID) ([]Ingredient, error)

	// Create writes an ingredient of the catalog of a business, and answers
	// it with the identity that the storage generated.
	//
	// The rule of the domain: two ingredients of the same business never
	// carry the same name. The storage holds the same rule as a defense in
	// depth.
	//
	// The error is ErrDuplicateName when the name is taken inside the
	// business, and ErrNoStorage when the storage is not open.
	Create(ctx context.Context, item Ingredient) (Ingredient, error)
}
