package account

import "errors"

// The errors of the domain of the account.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
var (
	// ErrNotFound is the error of an account that the storage does not hold.
	ErrNotFound = errors.New("there is no account with that identity")

	// ErrAlreadyExists is the error of an account whose name is taken
	// inside the business that owns it.
	ErrAlreadyExists = errors.New("that account already exists in this business")

	// ErrInvalidName is the error of an account without a name.
	ErrInvalidName = errors.New("the name of an account is not empty")

	// ErrUnknownKind is the error of a kind that is not one of the kinds of
	// an account.
	ErrUnknownKind = errors.New("the kind is not one of the kinds of an account")

	// ErrDebtNotAllowed is the error of a counterparty or of a due date
	// offered to an account that cannot owe and cannot be owed.
	ErrDebtNotAllowed = errors.New("an account of this kind carries no counterparty and no due date")

	// ErrInvalidValue is the error of a row the database refused because it
	// breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no
	// storage behind it.
	ErrNoStorage = errors.New("the storage of the accounts is not open")
)

// Errors returns the errors of the domain of the account.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrAlreadyExists,
		ErrInvalidName,
		ErrUnknownKind,
		ErrDebtNotAllowed,
		ErrInvalidValue,
		ErrNoStorage,
	}
}
