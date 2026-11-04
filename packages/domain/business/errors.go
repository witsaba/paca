package business

import "errors"

// The errors of the domain of the business.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
var (
	// ErrNotFound is the error of a business that the storage does not hold.
	ErrNotFound = errors.New("there is no business with that identity")

	// ErrAlreadyExists is the error of a business that the storage already
	// holds: the owner already has a personal business, or the name of the
	// business is taken inside the business of its owner.
	ErrAlreadyExists = errors.New("that business already exists")

	// ErrInvalidName is the error of a business without a name.
	ErrInvalidName = errors.New("the name of a business is not empty")

	// ErrInvalidCurrency is the error of a business without a currency.
	ErrInvalidCurrency = errors.New("the currency of a business is not empty")

	// ErrUnknownKind is the error of a kind that is not one of the kinds of
	// a business.
	ErrUnknownKind = errors.New("the kind is not one of the kinds of a business")
)

// Errors returns the errors of the domain of the business.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrAlreadyExists,
		ErrInvalidName,
		ErrInvalidCurrency,
		ErrUnknownKind,
	}
}
