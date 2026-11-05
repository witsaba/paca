package purchase

import "errors"

// The errors of the domain of the draft.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
var (
	// ErrNotFound is the error of a draft that the storage does not hold.
	ErrNotFound = errors.New("there is no draft with that identity")

	// ErrInvalidName is the error of a line or of an ambiguity without a
	// name.
	ErrInvalidName = errors.New("the name of a line is not empty")

	// ErrInvalidQuantity is the error of a measure whose quantity is not
	// positive.
	ErrInvalidQuantity = errors.New("the quantity of a line is not positive")

	// ErrMissingUnit is the error of a measure without a unit.
	ErrMissingUnit = errors.New("the measure of a line carries its unit")

	// ErrInvalidPrice is the error of a price that is not a money.
	ErrInvalidPrice = errors.New("the money of a line is not negative")

	// ErrPriceMismatch is the error of a line whose money that was paid and
	// price of the unit disagree.
	ErrPriceMismatch = errors.New("the money paid and the price of the unit do not agree")

	// ErrInferredPrice is the error of a line whose money was filled by
	// the agent: the money is never inferred.
	ErrInferredPrice = errors.New("the money of a line is never inferred")

	// ErrMissingUtterance is the error of a draft without the sentence of
	// the owner.
	ErrMissingUtterance = errors.New("the draft carries the sentence of the owner")

	// ErrNoLines is the error of a draft that proposes nothing.
	ErrNoLines = errors.New("the draft carries at least one line")

	// ErrNoOptions is the error of an ambiguity without options: Paca
	// never asks an open question when it can name the options.
	ErrNoOptions = errors.New("the ambiguity carries the options Paca offers")

	// ErrWrongBusiness is the error of a draft of another business.
	ErrWrongBusiness = errors.New("the draft belongs to another business")

	// ErrInvalidValue is the error of a row the database refused because it
	// breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no
	// storage behind it.
	ErrNoStorage = errors.New("the storage of the drafts is not open")
)

// Errors returns the errors of the domain of the draft.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrInvalidName,
		ErrInvalidQuantity,
		ErrMissingUnit,
		ErrInvalidPrice,
		ErrPriceMismatch,
		ErrInferredPrice,
		ErrMissingUtterance,
		ErrNoLines,
		ErrNoOptions,
		ErrWrongBusiness,
		ErrInvalidValue,
		ErrNoStorage,
	}
}
