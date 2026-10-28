package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/witsaba/paca/packages/domain/user"
	"github.com/witsaba/paca/packages/database/internal/db"
)

func TestUserFromRow(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, time.October, 27, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, time.October, 28, 11, 30, 0, 0, time.UTC)

	row := db.User{
		ID:                 80000001,
		FullName:           pgtype.Text{String: "Ada Lovelace", Valid: true},
		Status:             "active",
		StatusObservations: pgtype.Text{String: "the onboarding is finished", Valid: true},
		Phone:              pgtype.Text{String: "+573001234567", Valid: true},
		Email:              pgtype.Text{String: "ada@example.org", Valid: true},
		CreatedAt:          pgtype.Timestamptz{Time: created, Valid: true},
		UpdatedAt:          pgtype.Timestamptz{Time: updated, Valid: true},
	}

	got, err := userFromRow(row)
	if err != nil {
		t.Fatalf("the row did not map: %v", err)
	}

	if got.ID != user.ID(80000001) {
		t.Errorf("the identifier is %d, want 80000001", got.ID)
	}
	if got.FullName != "Ada Lovelace" {
		t.Errorf("the name is %q, want %q", got.FullName, "Ada Lovelace")
	}
	if got.Status != user.StatusActive {
		t.Errorf("the status is %q, want %q", got.Status, user.StatusActive)
	}
	if got.StatusObservations != "the onboarding is finished" {
		t.Errorf("the observations are %q, want %q", got.StatusObservations, "the onboarding is finished")
	}
	if got.Phone != "+573001234567" {
		t.Errorf("the phone is %q, want %q", got.Phone, "+573001234567")
	}
	if got.Email != "ada@example.org" {
		t.Errorf("the email is %q, want %q", got.Email, "ada@example.org")
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("the creation is %s, want %s", got.CreatedAt, created)
	}
	if !got.UpdatedAt.Equal(updated) {
		t.Errorf("the update is %s, want %s", got.UpdatedAt, updated)
	}
}

func TestUserFromRowWithNullColumns(t *testing.T) {
	t.Parallel()

	row := db.User{
		ID:     80000002,
		Status: "onboarding",
	}

	got, err := userFromRow(row)
	if err != nil {
		t.Fatalf("the row did not map: %v", err)
	}

	if got.FullName != "" {
		t.Errorf("the name is %q, want the empty string", got.FullName)
	}
	if got.StatusObservations != "" {
		t.Errorf("the observations are %q, want the empty string", got.StatusObservations)
	}
	if got.Phone != "" {
		t.Errorf("the phone is %q, want the empty string", got.Phone)
	}
	if got.Email != "" {
		t.Errorf("the email is %q, want the empty string", got.Email)
	}
	if !got.CreatedAt.IsZero() {
		t.Errorf("the creation is %s, want the zero time", got.CreatedAt)
	}
	if !got.UpdatedAt.IsZero() {
		t.Errorf("the update is %s, want the zero time", got.UpdatedAt)
	}
}

func TestUserFromRowRefusesAnUnknownStatus(t *testing.T) {
	t.Parallel()

	row := db.User{
		ID:     80000003,
		Status: "unknown",
	}

	got, err := userFromRow(row)
	if err == nil {
		t.Fatalf("the row mapped into %+v, want an error", got)
	}
	if !errors.Is(err, user.ErrUnknownStatus) {
		t.Errorf("the error is %v, want it to wrap ErrUnknownStatus", err)
	}
}
