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

	// ErrMissingConfirmation is the error of a confirmation without the
	// word the owner used: the word is kept with the record.
	ErrMissingConfirmation = errors.New("the confirmation carries the word of the owner")

	// ErrNotProposed is the error of the confirmation of a draft that is
	// not waiting for the yes of the owner.
	ErrNotProposed = errors.New("the draft is not waiting for the yes of the owner")

	// ErrNotCancellable is the error of the cancellation of a draft that
	// does not wait any more: it is confirmed already, it is cancelled
	// already or it expired.
	ErrNotCancellable = errors.New("the draft does not wait any more")

	// ErrStaleRevision is the error of the confirmation of a draft that
	// changed under the eyes of the owner.
	ErrStaleRevision = errors.New("the draft changed under the eyes of the owner")

	// ErrExpiredDraft is the error of the confirmation of a draft that
	// expired: the price of a market does not wait.
	ErrExpiredDraft = errors.New("the draft expired")

	// ErrStillNeeds is the error of the confirmation of a draft that still
	// needs the money of one of its lines.
	ErrStillNeeds = errors.New("the draft still needs the money of a line")

	// ErrInvalidValue is the error of a row the database refused because it
	// breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no
	// storage behind it.
	ErrNoStorage = errors.New("the storage of the drafts is not open")

	// ErrMissingIngredient is the error of a reference that names nothing
	// of the catalog of the business.
	ErrMissingIngredient = errors.New("the reference names the ingredient of the catalog")

	// ErrInvalidReferencePrice is the error of a reference whose price is
	// not a price.
	ErrInvalidReferencePrice = errors.New("the price of the reference is positive")

	// ErrMissingReferenceUnit is the error of a reference whose price
	// carries no unit: a price without a unit is a number nobody can
	// compare with anything.
	ErrMissingReferenceUnit = errors.New("the price of the reference carries its unit")

	// ErrMissingSource is the error of a reference without the place the
	// price was read: a price without a source is a price nobody can
	// check.
	ErrMissingSource = errors.New("the reference carries the source where the price was read")

	// ErrMissingDay is the error of a reference without the day the price
	// was seen.
	ErrMissingDay = errors.New("the reference carries the day it was seen")

	// ErrLayerWithoutIngredient is the error of a layer that names
	// nothing of the catalog: the layer of a line is derived from the
	// purchase, and the line names the ingredient it is of.
	ErrLayerWithoutIngredient = errors.New("the layer names the ingredient of the catalog")

	// ErrInvalidLayer is the error of a layer that is not whole: what is
	// left of it is never more than what it bought, and the unit of what
	// is left of it is the unit it bought.
	ErrInvalidLayer = errors.New("the layer is not whole: what it holds is not what it bought")

	// ErrInvalidPosition is the error of a layer that comes from no line
	// of its purchase: the position of the line is positive, and it is
	// the order the owner said the line in.
	ErrInvalidPosition = errors.New("the position of the line of the layer is positive")

	// ErrInvalidUnit is the error of a layer whose unit is not one of the
	// four units of the book: the layer carries the unit the book knows,
	// and the way the owner said it is kept in the purchase.
	ErrInvalidUnit = errors.New("the unit of the layer is one of the units of the book")

	// ErrLayerOfAnotherBusiness is the error of a layer that claims
	// another business than the purchase it comes from: the layer is
	// derived from the purchase, and it belongs to its business.
	ErrLayerOfAnotherBusiness = errors.New("the layer belongs to the business of its purchase")

	// ErrMissingBusiness is the error of a stock that is read for a
	// business that is not there: the stock belongs to a business, and
	// there is no stock of nobody.
	ErrMissingBusiness = errors.New("the stock is read for a business")
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
		ErrMissingConfirmation,
		ErrNotProposed,
		ErrNotCancellable,
		ErrStaleRevision,
		ErrExpiredDraft,
		ErrStillNeeds,
		ErrInvalidValue,
		ErrNoStorage,
		ErrMissingIngredient,
		ErrInvalidReferencePrice,
		ErrMissingReferenceUnit,
		ErrMissingSource,
		ErrMissingDay,
		ErrLayerWithoutIngredient,
		ErrInvalidLayer,
		ErrInvalidPosition,
		ErrInvalidUnit,
		ErrLayerOfAnotherBusiness,
		ErrMissingBusiness,
	}
}
