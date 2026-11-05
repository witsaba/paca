package ingredient

import "errors"

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
	}
}
