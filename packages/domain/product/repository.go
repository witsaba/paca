package product

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
)

// Repository is the port of the storage of the products.
//
// The port is what a use case needs, not the shape of a table: it opens a
// product of the business and it reads the products the business knows.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Create writes a product of the business and returns it with the
	// identity that the storage generated.
	//
	// The name of a product is taken inside the business, and one recipe
	// backs one product: the storage holds the same rules as a defense in
	// depth.
	//
	// The error is ErrNotFound when the business or the recipe of the
	// product is not there, ErrInvalidValue when the row breaks a rule of
	// the table, and ErrNoStorage when the storage is not open.
	Create(ctx context.Context, p Product) (Product, error)

	// List returns the products of the business, in the order of their
	// identity.
	//
	// The list holds the products of the business and of no other one,
	// and it is empty when the business sells nothing yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Product, error)

	// State writes the tax treatment the owner stated for the product,
	// and returns the product as it is now.
	//
	// The rule of the domain: the tax treatment is an OPTIONAL fact of
	// the product, and the name and the recipe of the product never
	// change: only the treatment the owner stated is written.
	//
	// The error is ErrNotFound when the product is not there, and
	// ErrNoStorage when the storage is not open.
	State(ctx context.Context, p Product) (Product, error)
}

// PriceRepository is the port of the storage of the price list.
//
// The port is what a use case needs: the price list is a history, and the
// storage writes a price and reads the prices of the business back.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type PriceRepository interface {
	// Set writes the price of a product for a channel and returns it with
	// the identity that the storage generated.
	//
	// The price is a fact of the business and it is never rewritten: the
	// row of a price is kept, and the price that holds is the last one
	// set on or before the day that is asked for.
	//
	// The error is ErrNotFound when the business or the product of the
	// price is not there, ErrInvalidValue when the row breaks a rule of
	// the table, and ErrNoStorage when the storage is not open.
	Set(ctx context.Context, price Price) (Price, error)

	// List returns the prices of the business, in the order of their
	// identity.
	//
	// The list holds the prices of the business and of no other one, and
	// it is empty when the business priced nothing yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Price, error)
}
