package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/domain/user"
)

func TestNewUserRepositoryWithoutAPool(t *testing.T) {
	t.Parallel()

	repository, err := NewUserRepository(nil)

	if !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
	if repository != nil {
		t.Errorf("the repository is %v, want none", repository)
	}
}

func TestOptionalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  pgtype.Text
	}{
		{name: "a value is a value", value: "Ana", want: pgtype.Text{String: "Ana", Valid: true}},
		{name: "an empty value is NULL", value: "", want: pgtype.Text{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := optionalText(tc.value)

			if got != tc.want {
				t.Errorf("the value is %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestObservationText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		status      user.Status
		observation string
		want        pgtype.Text
	}{
		{
			name:        "a closed status carries the observation",
			status:      user.StatusInactive,
			observation: "  the person asked to leave  ",
			want:        pgtype.Text{String: "the person asked to leave", Valid: true},
		},
		{
			name:        "an open status carries none",
			status:      user.StatusActive,
			observation: "the payment was confirmed",
			want:        pgtype.Text{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := observationText(tc.status, tc.observation)

			if got != tc.want {
				t.Errorf("the observation is %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name:    "a row that does not exist is not found",
			err:     pgx.ErrNoRows,
			wantErr: user.ErrNotFound,
		},
		{
			name:    "a primary key that is taken is a user that exists",
			err:     &pgconn.PgError{Code: uniqueViolation, ConstraintName: primaryKeyConstraint},
			wantErr: user.ErrAlreadyExists,
		},
		{
			name:    "an email that is taken is an email that is taken",
			err:     &pgconn.PgError{Code: uniqueViolation, ConstraintName: emailUniqueIndex},
			wantErr: user.ErrEmailTaken,
		},
		{
			name:    "a phone that is taken is a phone that is taken",
			err:     &pgconn.PgError{Code: uniqueViolation, ConstraintName: phoneUniqueIndex},
			wantErr: user.ErrPhoneTaken,
		},
		{
			name:    "a check that is broken is a value the database refused",
			err:     &pgconn.PgError{Code: checkViolation, ConstraintName: "users_phone_format_check"},
			wantErr: user.ErrInvalidValue,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := writeError(tc.err)

			if !errors.Is(got, tc.wantErr) {
				t.Fatalf("the error is %v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestWriteErrorKeepsAnUnknownError(t *testing.T) {
	t.Parallel()

	original := errors.New("the connection is gone")

	got := writeError(original)

	if !errors.Is(got, original) {
		t.Fatalf("the error is %v, want %v", got, original)
	}
}

func TestWriteErrorKeepsAnUnknownUniqueIndex(t *testing.T) {
	t.Parallel()

	original := &pgconn.PgError{Code: uniqueViolation, ConstraintName: "other_index"}

	got := writeError(original)

	if !errors.Is(got, original) {
		t.Fatalf("the error is %v, want %v", got, original)
	}
}

func TestTheConstructorRefusesAPoolThatIsNotOpen(t *testing.T) {
	t.Parallel()

	repository, err := NewUserRepository(nil)

	if !errors.Is(err, ErrNoPool) {
		t.Fatalf("the error is %v, want %v", err, ErrNoPool)
	}
	if repository != nil {
		t.Fatalf("the repository is %v, want none", repository)
	}
}

func TestTheRepositoryWithoutADatabaseRefusesToWork(t *testing.T) {
	t.Parallel()

	repository := &UserRepository{}

	if _, err := repository.Create(t.Context(), 1, user.Details{}); !errors.Is(err, user.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, user.ErrNoStorage)
	}
	if _, err := repository.getByID(t.Context(), 1); !errors.Is(err, user.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, user.ErrNoStorage)
	}
	if _, err := repository.UpdateDetails(t.Context(), 1, user.Details{}); !errors.Is(err, user.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, user.ErrNoStorage)
	}
	if _, err := repository.ChangeStatus(t.Context(), 1, user.StatusActive, ""); !errors.Is(err, user.ErrNoStorage) {
		t.Errorf("the error is %v, want %v", err, user.ErrNoStorage)
	}
}

func TestTheRepositoryRefusesAnUnknownStatus(t *testing.T) {
	t.Parallel()

	repository := &UserRepository{}

	_, err := repository.ChangeStatus(t.Context(), 1, user.Status("unknown"), "")

	if !errors.Is(err, user.ErrUnknownStatus) {
		t.Fatalf("the error is %v, want %v", err, user.ErrUnknownStatus)
	}
}

func TestTheRepositoryRefusesAnObservationThatDoesNotFit(t *testing.T) {
	t.Parallel()

	repository := &UserRepository{}

	_, err := repository.ChangeStatus(t.Context(), 1, user.StatusActive, "the payment was confirmed")

	if !errors.Is(err, user.ErrUnexpectedObservation) {
		t.Fatalf("the error is %v, want %v", err, user.ErrUnexpectedObservation)
	}
}

func TestTheRepositoryRefusesAMissingObservation(t *testing.T) {
	t.Parallel()

	repository := &UserRepository{}

	_, err := repository.ChangeStatus(t.Context(), 1, user.StatusBlocked, "")

	if !errors.Is(err, user.ErrObservationRequired) {
		t.Fatalf("the error is %v, want %v", err, user.ErrObservationRequired)
	}
}

func TestTheRepositoryCleansTheDataBeforeItWritesIt(t *testing.T) {
	t.Parallel()

	repository := &UserRepository{}

	_, err := repository.Create(t.Context(), 1, user.Details{Phone: "3212539984"})
	if !errors.Is(err, user.ErrInvalidPhone) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidPhone)
	}

	_, err = repository.UpdateDetails(t.Context(), 1, user.Details{Phone: "3212539984"})
	if !errors.Is(err, user.ErrInvalidPhone) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidPhone)
	}

	_, err = repository.Create(t.Context(), 1, user.Details{Email: "not an address"})
	if !errors.Is(err, user.ErrInvalidEmail) {
		t.Fatalf("the error is %v, want %v", err, user.ErrInvalidEmail)
	}
}

func TestTheRepositoryNamesTheStatusItRefuses(t *testing.T) {
	t.Parallel()

	repository := &UserRepository{}

	_, err := repository.ChangeStatus(t.Context(), 1, user.Status("unknown"), "")

	if !errors.Is(err, user.ErrUnknownStatus) {
		t.Fatalf("the error is %v, want %v", err, user.ErrUnknownStatus)
	}
	if got := fmt.Sprint(err); got == "" {
		t.Error("the error says nothing about the status")
	}
}
