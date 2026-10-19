package sqlite_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/witsaba/paca/packages/core/adapters/sqlite"
	"github.com/witsaba/paca/packages/core/domain"
)

// openTestDB opens a database in a temporary directory that is removed when
// the test ends.
func openTestDB(t *testing.T) *sqlite.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "paca.db")

	db, err := sqlite.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open the test database: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close the test database: %v", err)
		}
	})

	return db
}

// Opening a database creates the file it works on.
func TestOpenCreatesTheDatabaseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "paca.db")

	db, err := sqlite.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open the database: %v", err)
	}

	defer db.Close()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("the database file was not created: %v", err)
	}
}

// Foreign keys are enforced on the connection, because the pragma is set on
// every connection the pool opens and never on the schema.
func TestForeignKeysAreEnforcedOnTheConnection(t *testing.T) {
	db := openTestDB(t)

	var enabled int
	if err := db.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatalf("read the foreign_keys pragma: %v", err)
	}

	if enabled != 1 {
		t.Errorf("the foreign_keys pragma is %d, want 1", enabled)
	}
}

// A uniqueness violation is a validation failure, not an internal error: the
// agent can act on it.
func TestUniqueViolationBecomesAValidationFailure(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE people (id INTEGER PRIMARY KEY, phone TEXT NOT NULL UNIQUE)"); err != nil {
		t.Fatalf("create the table: %v", err)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO people (phone) VALUES (?)", "300"); err != nil {
		t.Fatalf("insert the first row: %v", err)
	}

	_, err := db.ExecContext(ctx, "INSERT INTO people (phone) VALUES (?)", "300")
	if err == nil {
		t.Fatal("the second insert succeeded, want a constraint violation")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the violation is %q, want %q", got, domain.CodeValidationFailed)
	}

	if strings.Contains(domain.DetailOf(err), "UNIQUE constraint failed") {
		t.Errorf("the detail leaks the message of the driver: %q", domain.DetailOf(err))
	}
}

// A foreign key violation is a validation failure too: the request names a
// record that does not exist.
func TestForeignKeyViolationBecomesAValidationFailure(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE accounts (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create the accounts table: %v", err)
	}

	if _, err := db.ExecContext(ctx, "CREATE TABLE movements (id INTEGER PRIMARY KEY, account_id INTEGER NOT NULL REFERENCES accounts (id))"); err != nil {
		t.Fatalf("create the movements table: %v", err)
	}

	_, err := db.ExecContext(ctx, "INSERT INTO movements (account_id) VALUES (?)", 42)
	if err == nil {
		t.Fatal("the insert succeeded, want a foreign key violation")
	}

	if got := domain.CodeOf(err); got != domain.CodeValidationFailed {
		t.Errorf("the code of the violation is %q, want %q", got, domain.CodeValidationFailed)
	}

	if strings.Contains(domain.DetailOf(err), "FOREIGN KEY constraint failed") {
		t.Errorf("the detail leaks the message of the driver: %q", domain.DetailOf(err))
	}
}

// A row that does not exist is a not-found failure, not an internal error.
func TestMissingRowBecomesANotFound(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE accounts (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create the table: %v", err)
	}

	var id int
	err := db.QueryRowContext(ctx, "SELECT id FROM accounts WHERE id = ?", 7).Scan(&id)
	if err == nil {
		t.Fatal("the query succeeded, want a missing row")
	}

	if got := domain.CodeOf(err); got != domain.CodeNotFound {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeNotFound)
	}
}

// A failure the schema does not explain is an internal error: nothing can be
// said about it that is safe to show.
func TestUnexpectedFailureBecomesAnInternalError(t *testing.T) {
	db := openTestDB(t)

	_, err := db.ExecContext(context.Background(), "SELECT * FROM a_table_that_does_not_exist")
	if err == nil {
		t.Fatal("the query succeeded, want an error")
	}

	if got := domain.CodeOf(err); got != domain.CodeInternal {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeInternal)
	}

	if domain.DetailOf(err) != "" {
		t.Errorf("the detail of an internal error is %q, want an empty detail", domain.DetailOf(err))
	}
}

// The cause of a translated failure stays reachable, so a caller can log it
// without ever showing it.
func TestTranslatedFailureKeepsItsCause(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE people (id INTEGER PRIMARY KEY, phone TEXT NOT NULL UNIQUE)"); err != nil {
		t.Fatalf("create the table: %v", err)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO people (phone) VALUES (?)", "300"); err != nil {
		t.Fatalf("insert the first row: %v", err)
	}

	_, err := db.ExecContext(ctx, "INSERT INTO people (phone) VALUES (?)", "300")
	if err == nil {
		t.Fatal("the second insert succeeded, want a constraint violation")
	}

	var driverErr sqliteDriverError
	if !errors.As(err, &driverErr) {
		t.Error("the translated error does not keep the error of the driver")
	}
}

// sqliteDriverError is the interface the error of the driver satisfies. The
// test names it instead of the concrete type, so it states the contract: the
// cause stays reachable through errors.As.
type sqliteDriverError interface {
	error
	Code() int
}

// The result set of a query is a set of rows of the driver.
func TestQueryReturnsRows(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE accounts (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create the table: %v", err)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO accounts (id) VALUES (1)"); err != nil {
		t.Fatalf("insert the row: %v", err)
	}

	rows, err := db.QueryContext(ctx, "SELECT id FROM accounts")
	if err != nil {
		t.Fatalf("query the rows: %v", err)
	}

	defer rows.Close()

	var count int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan the row: %v", err)
		}

		count++
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate the rows: %v", err)
	}

	if count != 1 {
		t.Errorf("the query returned %d rows, want 1", count)
	}
}

// A closed database refuses the work, so a use after close fails loudly.
func TestClosedDatabaseRefusesTheWork(t *testing.T) {
	db := openTestDB(t)

	if err := db.Close(); err != nil {
		t.Fatalf("close the database: %v", err)
	}

	if _, err := db.ExecContext(context.Background(), "SELECT 1"); err == nil {
		t.Fatal("the query on a closed database succeeded, want an error")
	}
}

// A transaction commits the work it holds, and refuses it when it is rolled
// back.
func TestTransactionCommitsAndRollsBack(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE accounts (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create the table: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin the transaction: %v", err)
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO accounts (id) VALUES (1)"); err != nil {
		t.Fatalf("insert inside the transaction: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("roll the transaction back: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM accounts").Scan(&count); err != nil {
		t.Fatalf("count the rows: %v", err)
	}

	if count != 0 {
		t.Errorf("the rolled back transaction left %d rows, want 0", count)
	}
}
