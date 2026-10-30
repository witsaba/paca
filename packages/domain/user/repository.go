package user

import (
	"context"
	"errors"
)

// ErrNotFound is the error of a user that the storage does not hold.
var ErrNotFound = errors.New("there is no user with that document ID")

// ErrAlreadyExists is the error of a user whose document ID is already
// registered.
var ErrAlreadyExists = errors.New("there is already a user with that document ID")

// ErrEmailTaken is the error of an email address that another user already
// carries.
var ErrEmailTaken = errors.New("that email address is already registered")

// ErrPhoneTaken is the error of a phone number that another user already
// carries.
var ErrPhoneTaken = errors.New("that phone number is already registered")

// ErrConcurrentChange is the error of a row that changed between the moment
// the storage read it and the moment it wrote it.
var ErrConcurrentChange = errors.New("the user changed at the same time")

// ErrInvalidValue is the error of a row the database refused because it
// breaks a rule of the table.
var ErrInvalidValue = errors.New("the database refused the row")

// ErrNoStorage is the error of a port of the storage that has no storage
// behind it.
//
// The service that has no database still exposes the port, and the port
// refuses to answer: it is the error of a storage that is not open.
var ErrNoStorage = errors.New("the storage of the users is not open")

// The bounds of a list of people.
//
// A list is bounded: the caller asks for what it wants, and the port never
// returns more than the bound.
const (
	// DefaultListedUsers is the number of people a list returns when the
	// caller does not ask for another one.
	DefaultListedUsers = 50

	// MaxListedUsers is the largest number of people a list returns.
	MaxListedUsers = 100
)

// ListedUsers returns the number of people a list returns for the limit the
// caller asks for.
//
// A limit that is not positive is the default: the caller that does not know
// how many people it wants reads a page, not the whole table. A limit that
// is over the bound is the bound: a list that returns everything is a list
// that does not answer a question.
func ListedUsers(limit int) int {
	switch {
	case limit <= 0:
		return DefaultListedUsers
	case limit > MaxListedUsers:
		return MaxListedUsers
	default:
		return limit
	}
}

// Repository is the port of the storage of the users.
//
// The port is what a use case needs, not the shape of the table: it creates,
// reads, updates and moves a user, and it never asks for a row it does not
// need.
//
// The rules of the domain travel with the port. An implementation refuses
// what the domain refuses, and says so with the errors of this package.
type Repository interface {
	// Create registers a person. The new user is in the onboarding status
	// and carries no observation: the storage decides that, not the caller.
	//
	// The error is ErrAlreadyExists when the document ID is taken.
	Create(ctx context.Context, id ID, details Details) (User, error)

	// UpdateDetails writes the data of a user. A field that is empty is left
	// as it is: the storage writes the fields that carry a value.
	//
	// The error is ErrNotFound when no user carries that document ID, and
	// ErrEmailTaken or ErrPhoneTaken when the value is already registered.
	UpdateDetails(ctx context.Context, id ID, details Details) (User, error)

	// Find returns the person who carries that document ID.
	//
	// The error is ErrNotFound when no user carries that document ID.
	Find(ctx context.Context, id ID) (User, error)

	// List returns the people of the storage, in the order of their
	// document ID.
	//
	// The limit is the number of people the caller wants to read: the
	// storage returns at most that many, and fewer when there are not that
	// many. The bound of the list belongs to the domain, and ListedUsers
	// returns the number the port answers.
	List(ctx context.Context, limit int) ([]User, error)

	// ChangeStatus moves a user to another status and writes the observation
	// of the move.
	//
	// The move is refused with ErrInvalidTransition when the map of the
	// states does not allow it, and with ErrObservationRequired or
	// ErrUnexpectedObservation when the observation does not fit the status.
	// The error is ErrNotFound when no user carries that document ID.
	ChangeStatus(ctx context.Context, id ID, status Status, observation string) (User, error)
}
