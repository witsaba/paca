package business

import (
	"context"

	"github.com/witsaba/paca/packages/domain/user"
)

// Repository is the port of the storage of the businesses.
//
// The port is what a use case needs, not the shape of a table: it creates a
// business, reads it by its identity and reads the personal business of a
// person.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Create writes a business and returns it with the identity that the
	// storage generated.
	//
	// The error is ErrAlreadyExists when the owner already holds a
	// personal business, or when the name of the business is taken inside
	// the business of its owner, and ErrNoStorage when the storage is not
	// open.
	Create(ctx context.Context, owner user.ID, business Business) (Business, error)

	// Find returns the business that carries that identity.
	//
	// The error is ErrNotFound when no business carries that identity, and
	// ErrNoStorage when the storage is not open.
	Find(ctx context.Context, id ID) (Business, error)

	// FindPersonalByOwner returns the personal business of a person.
	//
	// The rule of the domain: a person has exactly one personal business.
	// The storage holds the same rule as a defense in depth.
	//
	// The error is ErrNotFound when the person holds no personal business,
	// and ErrNoStorage when the storage is not open.
	FindPersonalByOwner(ctx context.Context, owner user.ID) (Business, error)
}
