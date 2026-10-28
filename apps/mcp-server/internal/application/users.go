package application

import (
	"context"

	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/domain/user"
)

// UsersService is the use case of the users.
//
// It owns the journey of a person inside PACA: the onboarding that
// registers them, the change of their status, and the correction of their
// data. It speaks in the model of the core and it writes through the port of
// the storage, and it knows nothing about the protocol that reaches it.
type UsersService struct {
	repository user.Repository
}

// NewUsersService builds the use case of the users.
//
// The repository is the port of the storage: the use case does not know
// whether the rows live in PostgreSQL or somewhere else.
func NewUsersService(repository user.Repository) *UsersService {
	return &UsersService{repository: repository}
}

// Onboard registers a person and starts their onboarding.
//
// The document ID is the national document of the person, and it is
// required: the use case refuses a call that does not bring it. The name, the
// phone and the email are optional, and the core of the monorepository
// cleans them before they are written.
//
// The new user starts in the onboarding status, with no observation: the
// core decides that, not the caller.
func (s *UsersService) Onboard(ctx context.Context, documentID, fullName, phone, email string) (user.User, error) {
	if s == nil || s.repository == nil {
		return user.User{}, ports.ErrNoDatabase
	}

	id, err := user.ParseID(documentID)
	if err != nil {
		return user.User{}, err
	}

	details, err := user.NewDetails(fullName, phone, email)
	if err != nil {
		return user.User{}, err
	}

	return s.repository.Create(ctx, id, details)
}

// ChangeStatus moves a user to another status and writes the observation of
// the move.
//
// The status is the name of one of the four states of the user, and the
// observation is the reason of the move: it is mandatory for the two closed
// states and refused for the two open ones. The core of the monorepository
// holds those rules, and it refuses a move the map of the states does not
// allow.
func (s *UsersService) ChangeStatus(ctx context.Context, documentID, status, observation string) (user.User, error) {
	if s == nil || s.repository == nil {
		return user.User{}, ports.ErrNoDatabase
	}

	id, err := user.ParseID(documentID)
	if err != nil {
		return user.User{}, err
	}

	target, err := user.ParseStatus(status)
	if err != nil {
		return user.User{}, err
	}

	return s.repository.ChangeStatus(ctx, id, target, observation)
}

// UpdateContactData writes the data of a user.
//
// A field that is empty is a field that does not change: the caller sends
// the fields it wants to write, and the rest stays as it is.
func (s *UsersService) UpdateContactData(ctx context.Context, documentID, fullName, phone, email string) (user.User, error) {
	if s == nil || s.repository == nil {
		return user.User{}, ports.ErrNoDatabase
	}

	id, err := user.ParseID(documentID)
	if err != nil {
		return user.User{}, err
	}

	details, err := user.NewDetails(fullName, phone, email)
	if err != nil {
		return user.User{}, err
	}

	return s.repository.UpdateDetails(ctx, id, details)
}
