package user

// Errors are the errors the port of the storage of the users can return.
//
// The catalogue exists so that a caller can prove that it maps every one of
// them. An error that no caller maps reaches the user unmapped, and a
// catalogue that no test pins is a wish.
//
// The errors of the package live where their rule lives: the errors of the
// identity and of the contact are in `contact.go`, the errors of the status
// are in `status.go`, the errors of the storage are in `repository.go`, and
// the catalogue names them all.
func Errors() []error {
	return []error{
		ErrNotFound,
		ErrAlreadyExists,
		ErrEmailTaken,
		ErrPhoneTaken,
		ErrConcurrentChange,
		ErrInvalidValue,
		ErrNoStorage,
		ErrInvalidTransition,
		ErrObservationRequired,
		ErrUnexpectedObservation,
		ErrInvalidPhone,
		ErrInvalidEmail,
		ErrInvalidID,
		ErrUnknownStatus,
	}
}
