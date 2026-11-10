package order

import "errors"

// The errors of the domain of the order.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
var (
	// ErrNotFound is the error of an order that the storage does not
	// hold.
	ErrNotFound = errors.New("there is no order with that identity")

	// ErrMissingBusiness is the error of an order of no business.
	ErrMissingBusiness = errors.New("the order belongs to no business")

	// ErrMissingParty is the error of an order that names no party.
	ErrMissingParty = errors.New("the order names no party")

	// ErrPartyOfAnotherBusiness is the error of an order whose party is
	// a party of another business.
	ErrPartyOfAnotherBusiness = errors.New("the party of the order belongs to another business")

	// ErrNotAClient is the error of an order whose party does not hold
	// the role of the client.
	ErrNotAClient = errors.New("the party of the order does not hold the role of the client")

	// ErrMissingProduct is the error of a line of an order that names no
	// product.
	ErrMissingProduct = errors.New("the line of the order names no product")

	// ErrProductOfAnotherBusiness is the error of an order whose product
	// is a product of another business.
	ErrProductOfAnotherBusiness = errors.New("the product of the order belongs to another business")

	// ErrNoLines is the error of an order that carries no line.
	ErrNoLines = errors.New("the order carries no line")

	// ErrInvalidQuantity is the error of a line whose quantity is not
	// positive.
	ErrInvalidQuantity = errors.New("the quantity of a line of the order is not positive")

	// ErrSplitPiece is the error of a line by the piece whose quantity
	// splits a piece: the book never splits a piece.
	ErrSplitPiece = errors.New("the quantity of a line of the order splits a piece")

	// ErrMissingUnit is the error of a line of an order that carries no
	// unit.
	ErrMissingUnit = errors.New("the line of the order carries no unit")

	// ErrUnknownUnit is the error of a line whose unit is not one of the
	// three ways the owner charges.
	ErrUnknownUnit = errors.New("the unit of the line of the order is not one of the units of the book")

	// ErrBusinessNotFound is the error of the business of the orders that
	// is not there.
	ErrBusinessNotFound = errors.New("the business of the orders is not there")

	// ErrMissingDay is the error of an order that carries no day.
	ErrMissingDay = errors.New("the order carries no day")

	// ErrWrongBusiness is the error of an order of a business that does
	// not hold orders.
	ErrWrongBusiness = errors.New("the orders belong to a business of its own")

	// ErrUnknownState is the error of a state that is not one of the
	// states of an order.
	ErrUnknownState = errors.New("the state of an order is not one of the states of the book")

	// ErrInvalidValue is the error of a row the database refused because
	// it breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no
	// storage behind it.
	ErrNoStorage = errors.New("the storage of the orders is not open")

	// ErrMissingRouteDay is the error of a route that carries no day.
	ErrMissingRouteDay = errors.New("the route carries no day")

	// ErrNoStops is the error of a route that carries no stop.
	ErrNoStops = errors.New("the route carries no stop")

	// ErrRouteAlreadyWritten is the error of a second route for a day that
	// was already written.
	ErrRouteAlreadyWritten = errors.New("the route of that day was already written")

	// ErrStopNotAnOrder is the error of a stop that is not an order of
	// the business.
	ErrStopNotAnOrder = errors.New("the stop is not an order of that business")

	// ErrDuplicateStop is the error of a route that visits the same order
	// in two stops.
	ErrDuplicateStop = errors.New("the route visits the same order in two stops")

	// ErrInvalidCost is the error of a stop whose cost is less than
	// nothing: the money of the book counts whole pesos, and no stop costs
	// less than nothing.
	ErrInvalidCost = errors.New("the cost of the stop is less than nothing")

	// ErrAlreadyDelivered is the error of an order that was delivered
	// already.
	ErrAlreadyDelivered = errors.New("the order was delivered already")
)

// Errors returns the errors of the domain of the order.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrMissingBusiness,
		ErrMissingParty,
		ErrPartyOfAnotherBusiness,
		ErrNotAClient,
		ErrMissingProduct,
		ErrProductOfAnotherBusiness,
		ErrNoLines,
		ErrInvalidQuantity,
		ErrSplitPiece,
		ErrMissingUnit,
		ErrUnknownUnit,
		ErrBusinessNotFound,
		ErrMissingDay,
		ErrWrongBusiness,
		ErrUnknownState,
		ErrInvalidValue,
		ErrNoStorage,
		ErrMissingRouteDay,
		ErrNoStops,
		ErrRouteAlreadyWritten,
		ErrStopNotAnOrder,
		ErrDuplicateStop,
		ErrInvalidCost,
		ErrAlreadyDelivered,
	}
}
