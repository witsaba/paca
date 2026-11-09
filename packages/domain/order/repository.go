package order

import (
	"context"

	"github.com/witsaba/paca/packages/domain/business"
)

// Repository is the port of the storage of the orders.
//
// The port is what a use case needs, not the shape of a table: it writes an
// order with its lines and it reads the orders of a business.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Create writes an order with its lines and returns it with the
	// identity that the storage generated.
	//
	// The order and its lines are written in one transaction: an order
	// without its lines orders nothing, and a half-written order is an
	// order nobody can read.
	//
	// The error is ErrNotFound when the business, the party or the
	// product of the order is not there, ErrInvalidValue when the row
	// breaks a rule of the table, and ErrNoStorage when the storage is
	// not open.
	Create(ctx context.Context, o Order) (Order, error)

	// List returns the orders of the business, in the order of their
	// identity, with their lines.
	//
	// The list holds the orders of the business and of no other one, and
	// it is empty when the business owes nothing yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Order, error)
}
