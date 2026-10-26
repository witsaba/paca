// Package user holds the model of the person that PACA knows.
//
// The model is the domain's own: it knows nothing about SQL, about the
// table, or about the driver. The adapter that reads the table maps the row
// onto this model, and the model carries the rules that the row cannot.
//
// The four states of the user are the same four the database accepts, and
// the same rule rejects a fifth one on both sides.
package user

import (
	"errors"
	"fmt"
	"time"
)

// ErrUnknownStatus is the error of a status that is not one of the four
// states of the user.
var ErrUnknownStatus = errors.New("the status is not one of the states of the user")

// Status is the state of a user.
type Status string

const (
	// StatusOnboarding is the state of a user that has not finished the
	// onboarding. It is the state of every user that has just been created.
	StatusOnboarding Status = "onboarding"

	// StatusActive is the state of a user that uses the product.
	StatusActive Status = "active"

	// StatusInactive is the state of a user that does not use the product
	// for now, and that may come back.
	StatusInactive Status = "inactive"

	// StatusBlocked is the state of a user that may not use the product.
	StatusBlocked Status = "blocked"
)

// statuses holds the four states of the user, in the order they are
// presented to a reader.
var statuses = []Status{
	StatusOnboarding,
	StatusActive,
	StatusInactive,
	StatusBlocked,
}

// AllStatuses returns the four states of the user. The caller receives a
// copy: the order and the content of the list never change under it.
func AllStatuses() []Status {
	all := make([]Status, len(statuses))
	copy(all, statuses)
	return all
}

// Valid reports whether the status is one of the four states of the user.
func (s Status) Valid() bool {
	for _, candidate := range statuses {
		if s == candidate {
			return true
		}
	}
	return false
}

// String returns the name of the status. It is the name the database
// stores, with no translation in between.
func (s Status) String() string {
	return string(s)
}

// ParseStatus returns the status of the given name.
//
// The database refuses a name that is not one of the four states, so a name
// that fails here would fail there too. The error wraps ErrUnknownStatus,
// so a caller can tell an unknown state from a failure of something else.
func ParseStatus(name string) (Status, error) {
	status := Status(name)
	if !status.Valid() {
		return "", fmt.Errorf("%w: %q", ErrUnknownStatus, name)
	}
	return status, nil
}

// ID is the national document ID of the person.
//
// The number is not generated: the caller supplies the number the person
// already carries, and it is the primary key of the row. The type exists so
// that an identifier never travels as a plain integer by accident.
type ID int64

// User is a person that PACA knows.
//
// A field the database stores as NULL is empty here. The core never writes
// an empty string to the database: it writes NULL. The rule is the same in
// both directions, so a value that is not set has exactly one meaning.
type User struct {
	// ID is the national document ID of the person.
	ID ID

	// FullName is the complete name of the person. Empty means the name is
	// not set.
	FullName string

	// Status is the state of the user.
	Status Status

	// StatusObservations is a free text note about the current status of
	// the user. Empty means there is no note.
	StatusObservations string

	// Phone is the phone number of the person. Empty means it is not set.
	Phone string

	// Email is the email address of the person. Empty means it is not set.
	Email string

	// CreatedAt is the moment the row was created. The database filled it.
	CreatedAt time.Time

	// UpdatedAt is the moment the row was last written. The database filled
	// it on insert; whoever updates the row is responsible for it after
	// that.
	UpdatedAt time.Time
}
