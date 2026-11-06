package ingredient

import (
	"errors"
	"fmt"
)

// The errors of the domain of the ingredient.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
var (
	// ErrNotFound is the error of an ingredient that the storage does not
	// hold, and of the business of an ingredient that is not there.
	ErrNotFound = errors.New("there is no ingredient with that identity")

	// ErrInvalidName is the error of an ingredient without a name, and of a
	// name the owner did not say.
	ErrInvalidName = errors.New("the name of an ingredient is not empty")

	// ErrDuplicateName is the error of an ingredient whose name is taken
	// inside the business that owns it.
	ErrDuplicateName = errors.New("two ingredients of the same business carry the same name")

	// ErrWrongBusiness is the error of an ingredient of another business.
	ErrWrongBusiness = errors.New("the ingredient belongs to another business")

	// ErrInvalidValue is the error of a row the database refused because it
	// breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no
	// storage behind it.
	ErrNoStorage = errors.New("the storage of the ingredients is not open")

	// ErrUnknownUnit is the error of a unit the book does not know: the
	// gram, the kilogram, the package and the piece are the units of the
	// book, and the message names them.
	ErrUnknownUnit = fmt.Errorf("the unit is not one of the units the book knows: %s", namesOfTheUnits())

	// ErrInvalidQuantity is the error of a quantity that is not positive.
	ErrInvalidQuantity = errors.New("the quantity of an ingredient is not positive")

	// ErrMissingPackageWeight is the error of a package whose weight the
	// owner has not declared: the weight of a package is never assumed.
	ErrMissingPackageWeight = errors.New("the weight of the package is not declared")

	// ErrInvalidPackageWeight is the error of a weight of a package that
	// is not positive.
	ErrInvalidPackageWeight = errors.New("the weight of the package is not positive")

	// ErrInvalidCertification is the error of a certification without a
	// claim: there is nothing to prove.
	ErrInvalidCertification = errors.New("the certification carries no name")

	// ErrMissingEvidence is the error of a certification without the
	// document that proves it: a claim nobody can check is a wish.
	ErrMissingEvidence = errors.New("the certification carries the evidence of its claim")

	// ErrInvalidConversion is the error of a conversion that is not
	// positive: the conversion is a number of grams.
	ErrInvalidConversion = errors.New("the conversion of the ingredient is not positive")

	// ErrUnknownConversion is the error of a unit that is never converted
	// into grams by assumption: the piece is one unit of the ingredient,
	// and its weight is not declared.
	ErrUnknownConversion = errors.New("the unit is never converted into grams by assumption")
)

// Errors returns the errors of the domain of the ingredient.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrInvalidName,
		ErrDuplicateName,
		ErrWrongBusiness,
		ErrInvalidValue,
		ErrNoStorage,
		ErrUnknownUnit,
		ErrInvalidQuantity,
		ErrMissingPackageWeight,
		ErrInvalidPackageWeight,
		ErrInvalidCertification,
		ErrMissingEvidence,
		ErrInvalidConversion,
		ErrUnknownConversion,
	}
}
