//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/user"
	"github.com/witsaba/paca/packages/core/postgres"
)

// The names of the guards of the migration 00002.
//
// They are the names PostgreSQL carries in its errors, and they are how the
// tests tell one refusal from another.
const (
	// phoneFormatConstraint is the constraint that checks the shape of the
	// phone number.
	phoneFormatConstraint = "users_phone_format_check"

	// emailFormatConstraint is the constraint that checks the shape of the
	// email address.
	emailFormatConstraint = "users_email_format_check"

	// optionalNotEmptyConstraint is the constraint that refuses an empty
	// string in an optional column.
	optionalNotEmptyConstraint = "users_optional_not_empty_check"

	// statusObservationConstraint is the constraint that binds the
	// observation to the status.
	statusObservationConstraint = "users_status_observation_check"
)

// The observation the tests use to close a user.
const testObservation = "the person asked to leave"

// This file holds the container tests of the registration of a person, and
// the helpers every test of the repository uses.
func TestCreateRegistersThePersonInOnboarding(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)
	id := nextIdentifier()

	details, err := user.NewDetails("  Ana María Torres  ", "+57 321-253-9984", "  Ana.Torres@Example.com  ")
	if err != nil {
		t.Fatalf("the data of the person could not be cleaned: %v", err)
	}
	if details.Phone != "573212539984" {
		t.Fatalf("the cleaned phone is %q, want %q", details.Phone, "573212539984")
	}
	if details.Email != "Ana.Torres@Example.com" {
		t.Fatalf("the cleaned address is %q, want %q", details.Email, "Ana.Torres@Example.com")
	}

	before := time.Now().UTC()
	created, err := repo.Create(ctx, user.ID(id), details)
	if err != nil {
		t.Fatalf("the person could not be registered: %v", err)
	}
	after := time.Now().UTC()

	if created.ID != user.ID(id) {
		t.Errorf("the document of the person is %d, want %d", created.ID, id)
	}
	if created.Status != user.StatusOnboarding {
		t.Errorf("the state of the person is %q, want %q", created.Status, user.StatusOnboarding)
	}
	if created.StatusObservations != "" {
		t.Errorf("the observation of the person is %q, want none", created.StatusObservations)
	}
	if created.FullName != details.FullName {
		t.Errorf("the name of the person is %q, want %q", created.FullName, details.FullName)
	}
	if created.Phone != details.Phone {
		t.Errorf("the phone of the person is %q, want %q", created.Phone, details.Phone)
	}
	if created.Email != details.Email {
		t.Errorf("the address of the person is %q, want %q", created.Email, details.Email)
	}
	if created.CreatedAt.IsZero() {
		t.Error("the creation time of the person is empty, want the database to fill it")
	}
	if created.UpdatedAt.IsZero() {
		t.Error("the update time of the person is empty, want the database to fill it")
	}

	// The clock of the machine and the clock of the container are the same
	// clock, so a moment of tolerance is enough for the two readings.
	tolerance := time.Second
	if created.CreatedAt.Before(before.Add(-tolerance)) || created.CreatedAt.After(after.Add(tolerance)) {
		t.Errorf("the creation time of the person is %s, want it between %s and %s", created.CreatedAt, before, after)
	}
	if !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Errorf("the creation time of the person is %s and the update time is %s, want them equal", created.CreatedAt, created.UpdatedAt)
	}

	// The row of the database holds the same facts.
	row := readRow(ctx, t, id)
	if row.status != user.StatusOnboarding.String() {
		t.Errorf("the state of the row is %q, want %q", row.status, user.StatusOnboarding)
	}
	if row.statusObservations.Valid {
		t.Errorf("the observation of the row is %q, want NULL", row.statusObservations.String)
	}
	if row.phone.String != details.Phone {
		t.Errorf("the phone of the row is %q, want %q", row.phone.String, details.Phone)
	}
	if row.email.String != details.Email {
		t.Errorf("the address of the row is %q, want %q", row.email.String, details.Email)
	}
	if !row.createdAt.Equal(created.CreatedAt) {
		t.Errorf("the creation time of the row is %s, want %s", row.createdAt, created.CreatedAt)
	}
	if !row.updatedAt.Equal(created.UpdatedAt) {
		t.Errorf("the update time of the row is %s, want %s", row.updatedAt, created.UpdatedAt)
	}
}

// TestCreateWithADocumentIDThatIsTaken proves that the database refuses the
// second registration of a document, that the refusal says so, and that
// nothing extra is stored.
func TestCreateWithADocumentIDThatIsTaken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)
	id := nextIdentifier()

	registerUser(t, repo, id, "The First Person")

	_, err := repo.Create(ctx, user.ID(id), detailsOf(t, nextIdentifier(), "The Second Person"))
	if !errors.Is(err, user.ErrAlreadyExists) {
		t.Fatalf("the error is %v, want %v", err, user.ErrAlreadyExists)
	}

	if count := countRows(ctx, t, id); count != 1 {
		t.Errorf("the database holds %d rows with the document %d, want one", count, id)
	}
}

// TestCreateWithAnEmailThatIsTaken proves that the database refuses an
// address that another user already carries, and that the address is
// compared without the case of the name.
func TestCreateWithAnEmailThatIsTaken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	owner := nextIdentifier()
	registerUser(t, repo, owner, "The Owner Of The Address")

	tests := []struct {
		name  string
		email string
	}{
		{name: "the same address", email: emailOf(owner)},
		{name: "the same address in another case", email: strings.ToUpper(emailOf(owner))},
		{name: "the same address in a mixed case", email: mixedCase(emailOf(owner))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			id := nextIdentifier()
			details, err := user.NewDetails("The Other Person", phoneOf(id), tc.email)
			if err != nil {
				t.Fatalf("the data of the person could not be cleaned: %v", err)
			}

			_, err = repo.Create(ctx, user.ID(id), details)
			if !errors.Is(err, user.ErrEmailTaken) {
				t.Fatalf("the error is %v, want %v", err, user.ErrEmailTaken)
			}

			if count := countRows(ctx, t, id); count != 0 {
				t.Errorf("the database stored %d rows with the document %d, want none", count, id)
			}
		})
	}
}

// TestCreateWithAPhoneThatIsTaken proves that the database refuses a phone
// number that another user already carries, and that nothing is stored.
func TestCreateWithAPhoneThatIsTaken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	owner := nextIdentifier()
	registerUser(t, repo, owner, "The Owner Of The Phone")

	id := nextIdentifier()
	details, err := user.NewDetails("The Other Person", phoneOf(owner), emailOf(id))
	if err != nil {
		t.Fatalf("the data of the person could not be cleaned: %v", err)
	}

	_, err = repo.Create(ctx, user.ID(id), details)
	if !errors.Is(err, user.ErrPhoneTaken) {
		t.Fatalf("the error is %v, want %v", err, user.ErrPhoneTaken)
	}

	if count := countRows(ctx, t, id); count != 0 {
		t.Errorf("the database stored %d rows with the document %d, want none", count, id)
	}
}

// TestGetByIDReturnsThePersonOfTheDocument proves that the read returns the
// person of the document, with the facts the row holds.
func usersRepository(t *testing.T) *postgres.UserRepository {
	t.Helper()

	repo := postgres.NewUserRepository(theSuite.pool)
	if repo == nil {
		t.Fatal("the repository of the users could not be created, want one bound to the database of the suite")
	}

	return repo
}

// phoneOf returns a phone number no other test uses: the country code and
// the identifier, as digits.
func phoneOf(id int64) string {
	return "57" + strconv.FormatInt(id, 10)
}

// emailOf returns an address no other test uses.
func emailOf(id int64) string {
	return fmt.Sprintf("person%d@example.com", id)
}

// mixedCase returns the address with the first letter in another case.
//
// The address is the same mailbox written differently, and the unique index
// of the email compares the addresses without the case.
func mixedCase(email string) string {
	if email == "" {
		return email
	}

	return strings.ToUpper(email[:1]) + email[1:]
}

// detailsOf returns the data of a person, cleaned the way the domain cleans
// it, with a phone and an address no other test uses.
func detailsOf(t *testing.T, id int64, fullName string) user.Details {
	t.Helper()

	details, err := user.NewDetails(fullName, phoneOf(id), emailOf(id))
	if err != nil {
		t.Fatalf("the data of the person could not be cleaned: %v", err)
	}

	return details
}

// registerUser registers a person through the repository and returns it.
func registerUser(t *testing.T, repo *postgres.UserRepository, id int64, fullName string) user.User {
	t.Helper()

	created, err := repo.Create(context.Background(), user.ID(id), detailsOf(t, id, fullName))
	if err != nil {
		t.Fatalf("the person could not be registered: %v", err)
	}

	return created
}

// userInStatus registers a person and moves it to the wanted state.
//
// The registration writes the onboarding state, and the map of the domain
// holds every move the test needs to reach the other three.
func userInStatus(t *testing.T, repo *postgres.UserRepository, id int64, fullName string, status user.Status) user.User {
	t.Helper()

	created := registerUser(t, repo, id, fullName)
	if status == user.StatusOnboarding {
		return created
	}

	observation := ""
	if user.RequiresObservations(status) {
		observation = testObservation
	}

	moved, err := repo.ChangeStatus(context.Background(), user.ID(id), status, observation)
	if err != nil {
		t.Fatalf("the person could not be moved to %q: %v", status, err)
	}

	return moved
}

// storedRow holds the values of a row of the table of the users.
//
// The values are read with the raw connection of the suite, so the
// assertions see what the database holds and not what the adapter makes of
// it.
type storedRow struct {
	fullName           sql.NullString
	status             string
	statusObservations sql.NullString
	phone              sql.NullString
	email              sql.NullString
	createdAt          time.Time
	updatedAt          time.Time
}

// readRow reads the row of a document from the database.
func readRow(ctx context.Context, t *testing.T, id int64) storedRow {
	t.Helper()

	var row storedRow
	err := theSuite.db.QueryRowContext(ctx, `
		SELECT full_name, status, status_observations, phone, email, created_at, updated_at
		FROM users
		WHERE id = $1`, id,
	).Scan(
		&row.fullName,
		&row.status,
		&row.statusObservations,
		&row.phone,
		&row.email,
		&row.createdAt,
		&row.updatedAt,
	)
	if err != nil {
		t.Fatalf("the row of the document %d could not be read: %v", id, err)
	}

	return row
}

// countRows returns the number of rows the database holds for a document.
func countRows(ctx context.Context, t *testing.T, id int64) int {
	t.Helper()

	var count int
	if err := theSuite.db.QueryRowContext(ctx,
		`SELECT count(*) FROM users WHERE id = $1`, id,
	).Scan(&count); err != nil {
		t.Fatalf("the rows of the document %d could not be counted: %v", id, err)
	}

	return count
}

// guardDefinition returns the definition of a guard of the table of the
// users.
//
// The test fails when the name is not one of the guards of the migration
// 00002: the database must refuse the row with a rule of that migration and
// not with a rule of anywhere else.
func guardDefinition(ctx context.Context, t *testing.T, name string) string {
	t.Helper()

	switch name {
	case phoneFormatConstraint, emailFormatConstraint, optionalNotEmptyConstraint, statusObservationConstraint:
	default:
		t.Fatalf("the constraint of the error is %q, want one of the guards of the migration 00002", name)
	}

	var definition string
	if err := theSuite.db.QueryRowContext(ctx, `
		SELECT pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE conrelid = 'public.users'::regclass AND conname = $1`, name,
	).Scan(&definition); err != nil {
		t.Fatalf("the guard %q could not be read: %v", name, err)
	}

	return definition
}
