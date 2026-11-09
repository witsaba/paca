package party

import "errors"

// The errors of the domain of the party.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
var (
	// ErrNotFound is the error of a party that the storage does not hold,
	// and of the business of a party that is not there.
	ErrNotFound = errors.New("there is no party with that identity")

	// ErrUnknownKind is the error of a kind that is not one of the kinds of
	// a party.
	ErrUnknownKind = errors.New("the kind of a party is not one of the kinds of a party")

	// ErrInvalidName is the error of a party without a name.
	ErrInvalidName = errors.New("the name of a party is not empty")

	// ErrUnknownRole is the error of a role that is not one of the roles of
	// a party.
	ErrUnknownRole = errors.New("the role of a party is not one of the roles of a party")

	// ErrNoRole is the error of a party that holds no role at all: it is
	// neither a client nor a provider.
	ErrNoRole = errors.New("a party holds no role: it is neither a client nor a provider")

	// ErrMissingLegalHuman is the error of a party that requires the legal
	// human that represents it and carries none.
	ErrMissingLegalHuman = errors.New("the legal human that represents the party is not there")

	// ErrUnexpectedLegalHuman is the error of a party that refuses the
	// legal human that represents it.
	ErrUnexpectedLegalHuman = errors.New("this party of that kind is not represented by a legal human")

	// ErrWrongBusiness is the error of a party of a business that does not
	// own parties.
	ErrWrongBusiness = errors.New("the parties belong to a business of its own")

	// ErrInvalidValue is the error of a row the database refused because it
	// breaks a rule of the table.
	ErrInvalidValue = errors.New("the database refused the row")

	// ErrNoStorage is the error of a port of the storage that has no
	// storage behind it.
	ErrNoStorage = errors.New("the storage of the parties is not open")
)

// Errors returns the errors of the domain of the party.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrUnknownKind,
		ErrInvalidName,
		ErrUnknownRole,
		ErrNoRole,
		ErrMissingLegalHuman,
		ErrUnexpectedLegalHuman,
		ErrWrongBusiness,
		ErrInvalidValue,
		ErrNoStorage,
	}
}
