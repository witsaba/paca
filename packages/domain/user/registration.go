package user

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNameRequired is the error of a registration that carries no name.
//
// A person is registered with their name: the onboarding asks for it, and the
// team does not recognise a person they cannot name.
var ErrNameRequired = errors.New("the name of a person is not empty")

// Registration is the data that registers a person.
//
// The name is required, and the phone and the email are optional: the
// person may give them or not, and the storage writes NULL for the ones
// that are absent.
//
// The value cannot be built broken: the constructor refuses the registration
// without a name, and the port of the storage takes this type and not a set
// of loose strings. A registration that breaks the rule does not reach the
// table.
type Registration struct {
	// FullName is the complete name of the person. It is not empty.
	FullName string

	// Phone is the phone number of the person, as digits with the country
	// code. Empty means the person did not give it.
	Phone string

	// Email is the email address of the person. Empty means the person did
	// not give it.
	Email string
}

// NewRegistration cleans and validates the data that registers a person.
//
// The name loses the spaces around it, and it may not be empty. The phone
// and the email are cleaned and validated exactly as they are in a partial
// update, and the ones that are not given stay empty.
//
// The error is ErrNameRequired when the name is absent, and the error of the
// contact when the phone or the email is not what it must be.
func NewRegistration(fullName, phone, email string) (Registration, error) {
	trimmed := strings.TrimSpace(fullName)
	if trimmed == "" {
		return Registration{}, fmt.Errorf("%w: the registration carries no name", ErrNameRequired)
	}

	cleanPhone, err := CleanPhone(phone)
	if err != nil {
		return Registration{}, err
	}

	cleanEmail, err := CleanEmail(email)
	if err != nil {
		return Registration{}, err
	}

	return Registration{
		FullName: trimmed,
		Phone:    cleanPhone,
		Email:    cleanEmail,
	}, nil
}
