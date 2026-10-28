//go:build integration

package integration_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/domain/user"
)

// This file holds the container tests of the status of a person, and the
// tests of the guards the table holds on its own.
func TestChangeStatusFollowsTheMapOfTheStates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	tests := []struct {
		name        string
		from        user.Status
		to          user.Status
		observation string
	}{
		{name: "from onboarding to active", from: user.StatusOnboarding, to: user.StatusActive},
		{name: "from onboarding to inactive", from: user.StatusOnboarding, to: user.StatusInactive, observation: testObservation},
		{name: "from inactive to active", from: user.StatusInactive, to: user.StatusActive},
		{name: "from blocked to inactive", from: user.StatusBlocked, to: user.StatusInactive, observation: testObservation},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id := nextIdentifier()
			before := userInStatus(t, repo, id, "The Person Of The Move", tc.from)

			moved, err := repo.ChangeStatus(ctx, user.ID(id), tc.to, tc.observation)
			if err != nil {
				t.Fatalf("the person could not be moved to %q: %v", tc.to, err)
			}

			if moved.Status != tc.to {
				t.Errorf("the state of the person is %q, want %q", moved.Status, tc.to)
			}
			if !moved.UpdatedAt.After(before.UpdatedAt) {
				t.Errorf("the update time of the person is %s, want it after %s", moved.UpdatedAt, before.UpdatedAt)
			}

			// The closed states carry the observation of the move, and the
			// open states carry none: the observation the person had before
			// the move is cleared.
			row := readRow(ctx, t, id)
			if user.RequiresObservations(tc.to) {
				if row.statusObservations.String != tc.observation {
					t.Errorf("the observation of the row is %q, want %q", row.statusObservations.String, tc.observation)
				}
				if moved.StatusObservations != tc.observation {
					t.Errorf("the observation of the person is %q, want %q", moved.StatusObservations, tc.observation)
				}
			} else {
				if row.statusObservations.Valid {
					t.Errorf("the observation of the row is %q, want NULL", row.statusObservations.String)
				}
				if moved.StatusObservations != "" {
					t.Errorf("the observation of the person is %q, want none", moved.StatusObservations)
				}
			}
		})
	}
}

// TestChangeStatusRefusesAMoveTheMapDoesNotAllow proves that the change of
// the state refuses a move the map of the domain does not hold, and that the
// row keeps the state it had.
func TestChangeStatusRefusesAMoveTheMapDoesNotAllow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	tests := []struct {
		name        string
		from        user.Status
		to          user.Status
		observation string
	}{
		{name: "from active to onboarding", from: user.StatusActive, to: user.StatusOnboarding},
		{name: "from inactive to onboarding", from: user.StatusInactive, to: user.StatusOnboarding},
		{name: "from blocked to onboarding", from: user.StatusBlocked, to: user.StatusOnboarding},
		{name: "from onboarding to onboarding", from: user.StatusOnboarding, to: user.StatusOnboarding},
		{name: "from active to active", from: user.StatusActive, to: user.StatusActive},
		{name: "from inactive to inactive", from: user.StatusInactive, to: user.StatusInactive, observation: testObservation},
		{name: "from blocked to blocked", from: user.StatusBlocked, to: user.StatusBlocked, observation: testObservation},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id := nextIdentifier()
			before := userInStatus(t, repo, id, "The Person Of The Move", tc.from)

			_, err := repo.ChangeStatus(ctx, user.ID(id), tc.to, tc.observation)
			if !errors.Is(err, user.ErrInvalidTransition) {
				t.Fatalf("the error is %v, want %v", err, user.ErrInvalidTransition)
			}

			row := readRow(ctx, t, id)
			if row.status != tc.from.String() {
				t.Errorf("the state of the row is %q, want %q", row.status, tc.from)
			}
			if row.statusObservations.String != before.StatusObservations {
				t.Errorf("the observation of the row is %q, want %q", row.statusObservations.String, before.StatusObservations)
			}
		})
	}
}

// TestChangeStatusRefusesAnObservationThatDoesNotFit proves that the change
// of the state refuses a closed state without an observation and an open
// state with one, and that the row keeps the state it had.
func TestChangeStatusRefusesAnObservationThatDoesNotFit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	tests := []struct {
		name        string
		from        user.Status
		to          user.Status
		observation string
		wantErr     error
	}{
		{
			name:    "to inactive without an observation",
			from:    user.StatusOnboarding,
			to:      user.StatusInactive,
			wantErr: user.ErrObservationRequired,
		},
		{
			name:    "to blocked without an observation",
			from:    user.StatusOnboarding,
			to:      user.StatusBlocked,
			wantErr: user.ErrObservationRequired,
		},
		{
			name:        "to active with an observation",
			from:        user.StatusOnboarding,
			to:          user.StatusActive,
			observation: testObservation,
			wantErr:     user.ErrUnexpectedObservation,
		},
		{
			name:        "from active to active with an observation",
			from:        user.StatusActive,
			to:          user.StatusActive,
			observation: testObservation,
			wantErr:     user.ErrUnexpectedObservation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id := nextIdentifier()
			before := userInStatus(t, repo, id, "The Person Of The Move", tc.from)

			_, err := repo.ChangeStatus(ctx, user.ID(id), tc.to, tc.observation)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("the error is %v, want %v", err, tc.wantErr)
			}

			row := readRow(ctx, t, id)
			if row.status != tc.from.String() {
				t.Errorf("the state of the row is %q, want %q", row.status, tc.from)
			}
			if row.statusObservations.String != before.StatusObservations {
				t.Errorf("the observation of the row is %q, want %q", row.statusObservations.String, before.StatusObservations)
			}
		})
	}
}

// TestChangeStatusOfAMissingDocument proves that the change of the state of
// a document nobody carries says that the person does not exist.
func TestChangeStatusOfAMissingDocument(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	_, err := repo.ChangeStatus(ctx, user.ID(nextIdentifier()), user.StatusActive, "")
	if !errors.Is(err, user.ErrNotFound) {
		t.Fatalf("the error is %v, want %v", err, user.ErrNotFound)
	}
}

// TestTheDatabaseRefusesWhatTheCoreRefuses proves that the guards of the
// migration 00002 hold on their own: a writer that is not the core cannot
// store a row that breaks them, the error names the guard, and nothing is
// stored.
func TestTheDatabaseRefusesWhatTheCoreRefuses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name string
		// query is the insert that breaks the guard.
		query string
		// args are the values of the insert.
		args []any
		// constraint is the guard the database must name.
		constraint string
		// column is the column the guard names, when the guard of the error
		// is not the only one that could refuse the row.
		column string
	}{
		{
			name:       "a phone that is not digits only",
			query:      `INSERT INTO users (id, phone) VALUES ($1, $2)`,
			args:       []any{"+57 321-253-9984"},
			constraint: phoneFormatConstraint,
		},
		{
			name:       "a phone that is too short",
			query:      `INSERT INTO users (id, phone) VALUES ($1, $2)`,
			args:       []any{"3212539984"},
			constraint: phoneFormatConstraint,
		},
		{
			name:       "an email that is not an address",
			query:      `INSERT INTO users (id, email) VALUES ($1, $2)`,
			args:       []any{"ana(at)example.com"},
			constraint: emailFormatConstraint,
		},
		{
			name:       "an email without a domain with a dot",
			query:      `INSERT INTO users (id, email) VALUES ($1, $2)`,
			args:       []any{"ana@example"},
			constraint: emailFormatConstraint,
		},
		{
			name:       "an empty name",
			query:      `INSERT INTO users (id, full_name) VALUES ($1, $2)`,
			args:       []any{""},
			constraint: optionalNotEmptyConstraint,
		},
		{
			name:  "an empty phone",
			query: `INSERT INTO users (id, phone) VALUES ($1, $2)`,
			args:  []any{""},
			// The empty phone breaks two guards: the one of the format and
			// the one of the empty string. The database names one of them,
			// and the one it names holds the column.
			column: "phone",
		},
		{
			name:   "an empty email",
			query:  `INSERT INTO users (id, email) VALUES ($1, $2)`,
			args:   []any{""},
			column: "email",
		},
		{
			name:       "an inactive status without an observation",
			query:      `INSERT INTO users (id, status) VALUES ($1, $2)`,
			args:       []any{"inactive"},
			constraint: statusObservationConstraint,
		},
		{
			name:       "a blocked status without an observation",
			query:      `INSERT INTO users (id, status) VALUES ($1, $2)`,
			args:       []any{"blocked"},
			constraint: statusObservationConstraint,
		},
		{
			name:       "an inactive status with an empty observation",
			query:      `INSERT INTO users (id, status, status_observations) VALUES ($1, $2, $3)`,
			args:       []any{"inactive", ""},
			constraint: statusObservationConstraint,
		},
		{
			name:       "an active status with an observation",
			query:      `INSERT INTO users (id, status, status_observations) VALUES ($1, $2, $3)`,
			args:       []any{"active", testObservation},
			constraint: statusObservationConstraint,
		},
		{
			name:       "an onboarding status with an observation",
			query:      `INSERT INTO users (id, status, status_observations) VALUES ($1, $2, $3)`,
			args:       []any{"onboarding", testObservation},
			constraint: statusObservationConstraint,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id := nextIdentifier()
			args := append([]any{id}, tc.args...)

			_, err := theSuite.db.ExecContext(ctx, tc.query, args...)
			if err == nil {
				t.Fatalf("the row was accepted, want the guard of the database to refuse it")
			}

			pgErr := postgresError(t, err)
			if pgErr.Code != checkViolation {
				t.Errorf("the code of the error is %q, want %q", pgErr.Code, checkViolation)
			}

			definition := guardDefinition(ctx, t, pgErr.ConstraintName)
			if tc.constraint != "" && pgErr.ConstraintName != tc.constraint {
				t.Errorf("the constraint of the error is %q, want %q", pgErr.ConstraintName, tc.constraint)
			}
			if tc.column != "" && !strings.Contains(definition, tc.column) {
				t.Errorf("the guard %q is %q, want it to name the column %q", pgErr.ConstraintName, definition, tc.column)
			}

			if count := countRows(ctx, t, id); count != 0 {
				t.Errorf("the database stored %d rows with the document %d, want none", count, id)
			}
		})
	}
}

// usersRepository returns the repository of the users bound to the database
// of the suite.
