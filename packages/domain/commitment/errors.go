package commitment

import "errors"

// The errors of the domain of the commitment.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
var (
	// ErrNotFound is the error of a commitment that the storage does not
	// hold.
	ErrNotFound = errors.New("there is no commitment with that identity")

	// ErrMissingBusiness is the error of a commitment of no business.
	ErrMissingBusiness = errors.New("the commitment belongs to no business")

	// ErrMissingWhat is the error of a commitment that says nothing about
	// what is due: a promise that says nothing is not a promise.
	ErrMissingWhat = errors.New("the commitment says nothing about what is due")

	// ErrMissingDay is the error of a commitment that carries no day it is
	// due on.
	ErrMissingDay = errors.New("the commitment carries no day")

	// ErrMissingMetDay is the error of a commitment that was met with no
	// day of meeting: the promise that was kept stays in the book with the
	// day it was kept.
	ErrMissingMetDay = errors.New("the commitment carries no day of meeting")

	// ErrAlreadyMet is the error of a commitment that was met already: a
	// promise is kept once, and the book never hides the day it was kept.
	ErrAlreadyMet = errors.New("the commitment was met already")

	// ErrWrongBusiness is the error of a commitment of a business that
	// does not hold promises.
	ErrWrongBusiness = errors.New("the commitments belong to a business of its own")

	// ErrBusinessNotFound is the error of the business of the commitments
	// that is not there.
	ErrBusinessNotFound = errors.New("the business of the commitments is not there")

	// ErrInvalidValue is the error of a row the database refused because
	// it breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no
	// storage behind it.
	ErrNoStorage = errors.New("the storage of the commitments is not open")
)

// Errors returns the errors of the domain of the commitment.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrMissingBusiness,
		ErrMissingWhat,
		ErrMissingDay,
		ErrMissingMetDay,
		ErrAlreadyMet,
		ErrWrongBusiness,
		ErrBusinessNotFound,
		ErrInvalidValue,
		ErrNoStorage,
	}
}
