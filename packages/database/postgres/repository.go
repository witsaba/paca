package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/database/internal/db"
	"github.com/witsaba/paca/packages/domain/user"
)

// The codes of the errors PostgreSQL raises when it refuses a row.
const (
	// uniqueViolation is the code of a row that breaks a unique index.
	uniqueViolation = "23505"

	// checkViolation is the code of a row that breaks a check constraint.
	checkViolation = "23514"
)

// The names the database gives to the constraints of the table of the users.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// adapter tells one refusal from another.
const (
	primaryKeyConstraint = "users_pkey"
	emailUniqueIndex     = "users_email_key"
	phoneUniqueIndex     = "users_phone_key"
)

// UserRepository is the adapter of the storage of the users.
//
// It writes one statement at a time. It does not read a row to find out
// whether it may write it: the primary key and the unique indexes tell the
// truth by themselves, and the adapter maps that truth to the errors of the
// port.
//
// The only place the adapter reads before it writes is the change of the
// status: the map of the states lives in the domain, and the map needs to
// know the state the user is in.
type UserRepository struct {
	queries *db.Queries
}

// NewUserRepository returns the repository of the users, bound to the pool.
//
// The repository needs an open pool, and the constructor refuses anything
// else: a repository built on nothing would be a nil pointer inside an
// interface, and the interface would not be nil. A caller that has no
// database holds no repository at all, and says so.
//
// The error is ErrNoPool when the pool is not there.
func NewUserRepository(pool *Pool) (*UserRepository, error) {
	if pool == nil {
		return nil, ErrNoPool
	}

	return &UserRepository{queries: pool.Queries()}, nil
}

// Create registers a person.
//
// The statement writes the onboarding state and no observation: the new user
// starts at the entry of the machine, and the caller does not choose it.
//
// The data is cleaned before it is written, so the row never carries a phone
// with its punctuation or an address with its spaces, whatever the caller
// sends.
//
// The error is user.ErrAlreadyExists when the document ID is taken.
func (r *UserRepository) Create(ctx context.Context, id user.ID, details user.Details) (user.User, error) {
	clean, err := user.NewDetails(details.FullName, details.Phone, details.Email)
	if err != nil {
		return user.User{}, err
	}

	if r == nil || r.queries == nil {
		return user.User{}, user.ErrNoStorage
	}

	row, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		ID:       int64(id),
		FullName: optionalText(clean.FullName),
		Phone:    optionalText(clean.Phone),
		Email:    optionalText(clean.Email),
	})
	if err != nil {
		return user.User{}, writeError(err)
	}

	return userFromRow(row)
}

// Find returns the user of the document ID.
//
// The read is a helper of the adapter: the change of the status reads the row
// it is about to write, and no use case asks for a row by itself.
//
// The error is user.ErrNotFound when no user carries that document ID.
func (r *UserRepository) Find(ctx context.Context, id user.ID) (user.User, error) {
	if r == nil || r.queries == nil {
		return user.User{}, user.ErrNoStorage
	}

	row, err := r.queries.GetUserByID(ctx, int64(id))
	if err != nil {
		return user.User{}, writeError(err)
	}

	return userFromRow(row)
}

// List returns the people of the storage, in the order of their document ID.
//
// The limit is the number of people the caller wants to read: the storage
// returns at most that many, and fewer when there are not that many.
func (r *UserRepository) List(ctx context.Context, limit int) ([]user.User, error) {
	if r == nil || r.queries == nil {
		return nil, user.ErrNoStorage
	}

	rows, err := r.queries.ListUsers(ctx, int64(limit))
	if err != nil {
		return nil, writeError(err)
	}

	users := make([]user.User, 0, len(rows))
	for _, row := range rows {
		found, err := userFromRow(row)
		if err != nil {
			return nil, err
		}
		users = append(users, found)
	}

	return users, nil
}

// UpdateDetails writes the data of a user.
//
// The statement writes the fields that carry a value and leaves the others
// as they are: an empty field is a field that does not change.
//
// The data is cleaned before it is written, exactly as it is on the way in.
//
// The error is user.ErrNotFound when no user carries that document ID, and
// user.ErrEmailTaken or user.ErrPhoneTaken when the value belongs to
// somebody else.
func (r *UserRepository) UpdateDetails(ctx context.Context, id user.ID, details user.Details) (user.User, error) {
	clean, err := user.NewDetails(details.FullName, details.Phone, details.Email)
	if err != nil {
		return user.User{}, err
	}

	if r == nil || r.queries == nil {
		return user.User{}, user.ErrNoStorage
	}

	row, err := r.queries.UpdateUserDetails(ctx, db.UpdateUserDetailsParams{
		FullName: optionalText(clean.FullName),
		Phone:    optionalText(clean.Phone),
		Email:    optionalText(clean.Email),
		ID:       int64(id),
	})
	if err != nil {
		return user.User{}, writeError(err)
	}

	return userFromRow(row)
}

// ChangeStatus moves a user to another status and writes the observation of
// the move.
//
// The move is refused with user.ErrInvalidTransition when the map of the
// states does not allow it, and with user.ErrObservationRequired or
// user.ErrUnexpectedObservation when the observation does not fit the status.
//
// The statement also requires the state to be the one the adapter read, so a
// user that moved at the same time is reported with
// user.ErrConcurrentChange instead of being written over.
func (r *UserRepository) ChangeStatus(ctx context.Context, id user.ID, status user.Status, observation string) (user.User, error) {
	// The rules of the domain are checked before the repository is touched:
	// they do not depend on the storage, and a caller that breaks them
	// deserves to hear about it before anything else.
	if !status.Valid() {
		return user.User{}, fmt.Errorf("%w: %q", user.ErrUnknownStatus, status)
	}

	if err := user.ValidateObservation(status, observation); err != nil {
		return user.User{}, err
	}

	if r == nil || r.queries == nil {
		return user.User{}, user.ErrNoStorage
	}

	current, err := r.Find(ctx, id)
	if err != nil {
		return user.User{}, err
	}

	if !current.Status.CanMoveTo(status) {
		return user.User{}, fmt.Errorf("%w: from %q to %q", user.ErrInvalidTransition, current.Status, status)
	}

	row, err := r.queries.ChangeUserStatus(ctx, db.ChangeUserStatusParams{
		Status:             status.String(),
		StatusObservations: observationText(status, observation),
		ID:                 int64(id),
		CurrentStatus:      current.Status.String(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user.User{}, fmt.Errorf("%w: the status of the user moved while it was being changed", user.ErrConcurrentChange)
		}
		return user.User{}, writeError(err)
	}

	return userFromRow(row)
}

// optionalText returns the value of an optional column, or NULL when the
// value is not set.
//
// The domain never writes an empty string: it writes NULL, and the column
// holds the same fact.
func optionalText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}

	return pgtype.Text{String: value, Valid: true}
}

// observationText returns the observation of a status, or NULL when the
// status carries none.
//
// The two open states carry no observation, and the database refuses one: a
// note on an active user would explain nothing.
func observationText(status user.Status, observation string) pgtype.Text {
	if !user.RequiresObservations(status) {
		return pgtype.Text{}
	}

	return pgtype.Text{String: strings.TrimSpace(observation), Valid: true}
}

// writeError maps the error of the database to the error of the port.
//
// A row that does not exist is user.ErrNotFound. A row that breaks the
// primary key is a user that is already registered, and a row that breaks a
// unique index is an email or a phone that belongs to somebody else.
func writeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return user.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	switch pgErr.Code {
	case uniqueViolation:
		switch pgErr.ConstraintName {
		case primaryKeyConstraint:
			return fmt.Errorf("%w: the document ID is already registered", user.ErrAlreadyExists)
		case emailUniqueIndex:
			return user.ErrEmailTaken
		case phoneUniqueIndex:
			return user.ErrPhoneTaken
		}
	case checkViolation:
		return fmt.Errorf("%w: the constraint %q refused the row", user.ErrInvalidValue, pgErr.ConstraintName)
	}

	return err
}
