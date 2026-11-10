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

	// Deliver writes the delivery of the orders: the state of every OPEN
	// order of the list moves to the delivered one, and of no other one.
	//
	// The rule of the domain: the delivery of an order is written with
	// the ROUTE of the round, and the orders that the route delivers are
	// the ones of its stops. The delivery is written once: an order that
	// is not open any more is never reopened.
	//
	// The error is ErrAlreadyDelivered when the order is not open any
	// more, ErrNotFound when the order is not there, ErrInvalidValue when
	// the row breaks a rule of the table, and ErrNoStorage when the
	// storage is not open.
	Deliver(ctx context.Context, orders []Order) error
}

// RouteRepository is the port of the storage of the routes.
//
// The route is a fact of the business: the stops of the round of a day, in
// the order the agent will visit the clients in, and what each stop costs.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type RouteRepository interface {
	// Create writes the route with its stops and returns it with the
	// identity that the storage generated.
	//
	// The route and its stops are written in one transaction: a route
	// without its stops visits nobody, and a half-written route is a
	// route nobody can read.
	//
	// The route visits every order once, and the rule travels with the
	// port: a route that names the same order in two stops is refused
	// (ErrDuplicateStop), because the total of the route is derived from
	// the stops.
	//
	// The error is ErrRouteAlreadyWritten when the business already wrote
	// the route of that day, ErrDuplicateStop when the route visits the
	// same order twice, ErrInvalidValue when the row breaks a rule of the
	// table, and ErrNoStorage when the storage is not open.
	Create(ctx context.Context, r Route) (Route, error)

	// List returns the routes of the business, in the order of their
	// identity, with their stops.
	//
	// The list holds the routes of the business and of no other one, and
	// it is empty when the business planned no round yet.
	//
	// The error is ErrNoStorage when the storage is not open.
	List(ctx context.Context, own business.ID) ([]Route, error)
}
