//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/witsaba/paca/packages/domain/user"
)

// emailCanonicalConstraint is the constraint that refuses an address that is
// not in its canonical form.
//
// It is the name PostgreSQL carries in its errors, and it is how the tests
// tell one refusal from another.
const emailCanonicalConstraint = "users_email_canonical_check"

// TestTheDomainStoresTheCanonicalFormOfTheAddress proves that the address
// the domain writes is the canonical one, whatever the case the person wrote
// it in, and that the row of the database holds it.
func TestTheDomainStoresTheCanonicalFormOfTheAddress(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)
	id := nextIdentifier()

	details, err := user.NewDetails("Ana María Torres", phoneOf(id), strings.ToUpper(emailOf(id)))
	if err != nil {
		t.Fatalf("the data of the person could not be cleaned: %v", err)
	}
	if details.Email != emailOf(id) {
		t.Fatalf("the cleaned address is %q, want %q", details.Email, emailOf(id))
	}

	created, err := repo.Create(ctx, user.ID(id), details)
	if err != nil {
		t.Fatalf("the person could not be registered: %v", err)
	}
	if created.Email != emailOf(id) {
		t.Errorf("the address of the person is %q, want %q", created.Email, emailOf(id))
	}

	row := readRow(ctx, t, id)
	if row.email.String != emailOf(id) {
		t.Errorf("the address of the row is %q, want %q", row.email.String, emailOf(id))
	}
}

// TestTwoAddressesOfTheSameMailboxAreTheSameAddress proves that the mailbox
// belongs to one person: the second registration of the same address, written
// in another case, is refused as the address of somebody else.
func TestTwoAddressesOfTheSameMailboxAreTheSameAddress(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := usersRepository(t)

	owner := nextIdentifier()
	details := detailsOf(t, owner, "The Owner Of The Address")
	if _, err := repo.Create(ctx, user.ID(owner), details); err != nil {
		t.Fatalf("the owner of the address could not be registered: %v", err)
	}

	other := nextIdentifier()
	same, err := user.NewDetails("The Other Person", phoneOf(other), mixedCase(details.Email))
	if err != nil {
		t.Fatalf("the data of the person could not be cleaned: %v", err)
	}
	if same.Email != details.Email {
		t.Fatalf("the address is %q, want the canonical form %q", same.Email, details.Email)
	}

	if _, err := repo.Create(ctx, user.ID(other), same); !errors.Is(err, user.ErrEmailTaken) {
		t.Fatalf("the error is %v, want %v", err, user.ErrEmailTaken)
	}

	if count := countRows(ctx, t, other); count != 0 {
		t.Errorf("the database stored %d rows with the document %d, want none", count, other)
	}
}

// TestTheTableRefusesAnAddressThatIsNotCanonical proves the defense in
// depth: the table refuses the writer that does not know the domain, and the
// refusal names the constraint that holds the rule.
func TestTheTableRefusesAnAddressThatIsNotCanonical(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	id := nextIdentifier()

	_, err := theSuite.db.ExecContext(ctx,
		`INSERT INTO users (id, email) VALUES ($1, $2)`, id, "Ana.Torres@Example.com")
	if err == nil {
		t.Fatal("the database accepted an address that is not canonical, want the constraint to refuse it")
	}

	pgErr := postgresError(t, err)
	if pgErr.ConstraintName != emailCanonicalConstraint {
		t.Errorf("the constraint is %q, want %q", pgErr.ConstraintName, emailCanonicalConstraint)
	}

	if count := countRows(ctx, t, id); count != 0 {
		t.Errorf("the database stored %d rows with the document %d, want none", count, id)
	}
}

// TestTheMigrationBringsTheAddressesToTheCanonicalForm proves the migration
// of the identity of the address against a database that holds the rows the
// migration finds.
//
// The test needs a database of its own: it applies the migrations one at a
// time, and the database of the suite is already at the last one.
func TestTheMigrationBringsTheAddressesToTheCanonicalForm(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container, err := startPostgres(ctx)
	if err != nil {
		t.Fatalf("the database of the test could not be started: %v", err)
	}
	defer func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), time.Minute)
		defer terminateCancel()
		if err := container.Terminate(terminateCtx); err != nil {
			t.Logf("the database container did not terminate: %v", err)
		}
	}()

	url, err := connectionString(ctx, container)
	if err != nil {
		t.Fatalf("the connection string of the database could not be built: %v", err)
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("the database could not be opened: %v", err)
	}
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../migrations"))
	if err != nil {
		t.Fatalf("the provider of the migrations could not be created: %v", err)
	}
	defer provider.Close()

	// The schema of the first two migrations: the row of the test has to
	// exist before the migration of the identity runs.
	if _, err := provider.UpTo(ctx, 2); err != nil {
		t.Fatalf("the first migrations could not be applied: %v", err)
	}

	id := nextIdentifier()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email) VALUES ($1, $2)`, id, "Ana.Torres@Example.com"); err != nil {
		t.Fatalf("the row of the test could not be written: %v", err)
	}

	// The migration of the identity of the address.
	if _, err := provider.UpTo(ctx, 3); err != nil {
		t.Fatalf("the migration of the identity could not be applied: %v", err)
	}

	var email sql.NullString
	if err := db.QueryRowContext(ctx,
		`SELECT email FROM users WHERE id = $1`, id).Scan(&email); err != nil {
		t.Fatalf("the row of the test could not be read: %v", err)
	}
	if email.String != "ana.torres@example.com" {
		t.Errorf("the address of the row is %q, want %q", email.String, "ana.torres@example.com")
	}
}
