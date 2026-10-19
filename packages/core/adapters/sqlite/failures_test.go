package sqlite_test

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/witsaba/paca/packages/core/adapters/sqlite"
	"github.com/witsaba/paca/packages/core/domain"
	"github.com/witsaba/paca/packages/core/ports"
)

// The details the adapter gives to the constraint violations, as the
// contract states them.
const (
	detailRequired   = "a required field is missing"
	detailConstraint = "the request breaks a rule of the schema"
)

// dropTables removes the given tables, in the order they are given.
//
// The order matters: a table that another table references cannot be removed
// while the foreign key is on, so the tables that depend on a table are
// removed first.
func dropTables(t *testing.T, db *sqlite.DB, tables ...string) {
	t.Helper()

	for _, table := range tables {
		if _, err := db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+table); err != nil {
			t.Fatalf("drop the table %s: %v", table, err)
		}
	}
}

// A database that cannot be created is a failure, not an empty database.
func TestOpenFailsWhenTheDatabaseCannotBeCreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a-directory-that-does-not-exist", "paca.db")

	db, err := sqlite.Open(context.Background(), path)
	if err == nil {
		db.Close()
		t.Fatal("the database was opened, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeInternal {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeInternal)
	}
}

// A query against a table that does not exist is a failure.
func TestQueryFailsWhenTheTableIsMissing(t *testing.T) {
	db := openTestDB(t)

	_, err := db.QueryContext(context.Background(), "SELECT * FROM a_table_that_does_not_exist")
	if err == nil {
		t.Fatal("the query succeeded, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeInternal {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeInternal)
	}
}

// A transaction reads and writes, and its work is visible once it commits.
func TestATransactionReadsAndCommits(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin the transaction: %v", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO accounts (kind, name, currency, opened_on, is_active) VALUES ('cash', 'Cash', 'COP', '2024-01-15', 1)`); err != nil {
		t.Fatalf("write inside the transaction: %v", err)
	}

	rows, err := tx.QueryContext(ctx, "SELECT name FROM accounts")
	if err != nil {
		t.Fatalf("read inside the transaction: %v", err)
	}

	var count int
	for rows.Next() {
		count++
	}

	if err := rows.Close(); err != nil {
		t.Fatalf("close the rows of the transaction: %v", err)
	}

	if count != 1 {
		t.Errorf("the transaction read %d rows, want 1", count)
	}

	var name string
	if err := tx.QueryRowContext(ctx, "SELECT name FROM accounts").Scan(&name); err != nil {
		t.Fatalf("read one row inside the transaction: %v", err)
	}

	if name != "Cash" {
		t.Errorf("the row of the transaction is %q, want %q", name, "Cash")
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit the transaction: %v", err)
	}

	var total int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM accounts").Scan(&total); err != nil {
		t.Fatalf("count the rows outside the transaction: %v", err)
	}

	if total != 1 {
		t.Errorf("the committed transaction left %d rows, want 1", total)
	}
}

// A statement that fails inside a transaction is a failure, and the
// transaction can still be rolled back.
func TestATransactionRefusesAFailingStatement(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin the transaction: %v", err)
	}

	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "INSERT INTO a_table_that_does_not_exist (id) VALUES (1)"); err == nil {
		t.Fatal("the failing statement succeeded, want a failure")
	}

	if _, err := tx.QueryContext(ctx, "SELECT * FROM a_table_that_does_not_exist"); err == nil {
		t.Fatal("the failing query succeeded, want a failure")
	}
}

// The pool holds one connection, so a second transaction waits for the first
// one and gives up when its context ends.
func TestASecondTransactionWaitsForTheConnection(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin the first transaction: %v", err)
	}

	defer tx.Rollback()

	waiting, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	if _, err := db.BeginTx(waiting, nil); err == nil {
		t.Fatal("the second transaction started, want a failure")
	}
}

// Every repository reports a failure when the table it reads is gone, and it
// does not turn that failure into a missing record: the failure is not about
// a record that does not exist.
func TestTheRepositoriesReportAFailureWhenTheirTableIsMissing(t *testing.T) {
	cases := []struct {
		name string
		drop []string
		list func(ctx context.Context, db *sqlite.DB) error
		read func(ctx context.Context, db *sqlite.DB) error
	}{
		{
			name: "accounts",
			drop: []string{"loan_payments", "movements", "accounts"},
			list: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewAccountRepository(db).List(ctx)
				return err
			},
			read: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewAccountRepository(db).GetByID(ctx, 1)
				return err
			},
		},
		{
			name: "contacts",
			drop: []string{"loan_payments", "loans", "contacts"},
			list: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewContactRepository(db).List(ctx)
				return err
			},
			read: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewContactRepository(db).GetByID(ctx, 1)
				return err
			},
		},
		{
			name: "loans",
			drop: []string{"loan_payments", "loans"},
			list: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewLoanRepository(db).List(ctx)
				return err
			},
			read: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewLoanRepository(db).GetByID(ctx, 1)
				return err
			},
		},
		{
			name: "movements",
			drop: []string{"loan_payments", "movements"},
			list: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewMovementRepository(db).List(ctx, ports.MovementFilter{})
				return err
			},
			read: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewMovementRepository(db).GetByID(ctx, 1)
				return err
			},
		},
		{
			name: "loan payments",
			drop: []string{"loan_payments"},
			list: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewLoanPaymentRepository(db).ListUnpaid(ctx)
				return err
			},
			read: func(ctx context.Context, db *sqlite.DB) error {
				_, err := sqlite.NewLoanPaymentRepository(db).GetByID(ctx, 1)
				return err
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			db := openMigratedDB(t)
			ctx := context.Background()

			dropTables(t, db, testCase.drop...)

			if err := testCase.list(ctx, db); err == nil {
				t.Error("the list succeeded, want a failure")
			} else if got := domain.CodeOf(err); got != domain.CodeInternal {
				t.Errorf("the code of the failure of the list is %q, want %q", got, domain.CodeInternal)
			}

			err := testCase.read(ctx, db)
			if err == nil {
				t.Fatal("the read succeeded, want a failure")
			}

			if got := domain.CodeOf(err); got != domain.CodeInternal {
				t.Errorf("the code of the failure of the read is %q, want %q", got, domain.CodeInternal)
			}
		})
	}
}

// The reports of the movements fail when the movements are gone.
func TestTheReportsOfTheMovementsFailWhenTheTableIsMissing(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	dropTables(t, db, "loan_payments", "movements")

	repository := sqlite.NewMovementRepository(db)

	if _, err := repository.BalanceByAccount(ctx); err == nil {
		t.Error("the balance succeeded, want a failure")
	}

	if _, err := repository.TotalByKind(ctx, ports.Period{}); err == nil {
		t.Error("the total by kind succeeded, want a failure")
	}

	if _, err := repository.TotalByCategory(ctx, ports.Period{}); err == nil {
		t.Error("the total by category succeeded, want a failure")
	}
}

// The runner ignores a directory that sits among the migration files: a
// directory is not a migration.
func TestTheRunnerIgnoresADirectoryInTheMigrations(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	migrations := fstest.MapFS{
		"migrations/001_initial_schema.sql": &fstest.MapFile{Data: []byte(readInitialSchema(t))},
		"migrations/002_a_directory/":       &fstest.MapFile{Mode: fs.ModeDir},
	}

	if err := sqlite.Migrate(ctx, db, migrations); err != nil {
		t.Fatalf("migrate the database: %v", err)
	}

	if got := userVersion(t, db); got != 1 {
		t.Errorf("the schema version is %d, want 1", got)
	}
}

// The runner refuses two migrations that carry the same number: the order of
// the migrations cannot be decided.
func TestTheRunnerRefusesTwoMigrationsWithTheSameNumber(t *testing.T) {
	db := openTestDB(t)

	migrations := fstest.MapFS{
		"migrations/001_first.sql":  &fstest.MapFile{Data: []byte("")},
		"migrations/001_second.sql": &fstest.MapFile{Data: []byte("")},
	}

	err := sqlite.Migrate(context.Background(), db, migrations)
	if err == nil {
		t.Fatal("the runner accepted two migrations with the same number, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeInternal {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeInternal)
	}
}

// The runner refuses a migration file that carries no number at all.
func TestTheRunnerRefusesAMigrationWithNoNumber(t *testing.T) {
	db := openTestDB(t)

	migrations := fstest.MapFS{
		"migrations/initial.sql": &fstest.MapFile{Data: []byte("")},
	}

	err := sqlite.Migrate(context.Background(), db, migrations)
	if err == nil {
		t.Fatal("the runner accepted a migration with no number, want a failure")
	}
}

// The runner refuses a migration file whose name carries no separator.
func TestTheRunnerRefusesAMigrationWithNoSeparator(t *testing.T) {
	db := openTestDB(t)

	migrations := fstest.MapFS{
		"migrations/initial_schema.sql": &fstest.MapFile{Data: []byte("")},
	}

	err := sqlite.Migrate(context.Background(), db, migrations)
	if err == nil {
		t.Fatal("the runner accepted a migration with no separator, want a failure")
	}
}

// The runner refuses a migration file whose number is not a number.
func TestTheRunnerRefusesAMigrationWhoseNumberIsNotANumber(t *testing.T) {
	db := openTestDB(t)

	migrations := fstest.MapFS{
		"migrations/first_schema.sql": &fstest.MapFile{Data: []byte("")},
	}

	err := sqlite.Migrate(context.Background(), db, migrations)
	if err == nil {
		t.Fatal("the runner accepted a migration with a number that is not a number, want a failure")
	}
}

// The runner refuses a migration that does not stamp the version it
// promises: the next run would not know what has been applied.
func TestTheRunnerRefusesAMigrationThatDoesNotStampItsVersion(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	migrations := fstest.MapFS{
		"migrations/001_initial_schema.sql": &fstest.MapFile{Data: []byte(readInitialSchema(t))},
		"migrations/002_unstamped.sql": &fstest.MapFile{Data: []byte(`PRAGMA foreign_keys = ON;
BEGIN IMMEDIATE;
CREATE TABLE unstamped (id INTEGER PRIMARY KEY) STRICT;
COMMIT;
`)},
	}

	err := sqlite.Migrate(ctx, db, migrations)
	if err == nil {
		t.Fatal("the runner accepted a migration that stamps no version, want a failure")
	}

	if got := userVersion(t, db); got != 1 {
		t.Errorf("the schema version is %d, want 1", got)
	}
}

// firstContactID returns the identity of the first contact of the database.
func firstContactID(t *testing.T, db *sqlite.DB) int64 {
	t.Helper()

	var id int64
	if err := db.QueryRowContext(context.Background(), "SELECT id FROM contacts ORDER BY id LIMIT 1").Scan(&id); err != nil {
		t.Fatalf("read the first contact: %v", err)
	}

	return id
}

// A movement keeps the contact and the loan it points at.
func TestAMovementKeepsItsCounterpartyAndItsLoan(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	loanID := storedLoan(t, db)
	contactID := firstContactID(t, db)

	repository := sqlite.NewMovementRepository(db)

	stored, err := repository.Create(ctx, domain.Movement{
		AccountID:             movementAccount(t, db, "Cash in the drawer"),
		Kind:                  domain.MovementKindLoanLent,
		Amount:                500000,
		OccurredOn:            "2024-02-01",
		CounterpartyContactID: &contactID,
		LoanID:                &loanID,
	})
	if err != nil {
		t.Fatalf("store the movement: %v", err)
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the movement: %v", err)
	}

	if read.CounterpartyContactID == nil || *read.CounterpartyContactID != contactID {
		t.Errorf("the counterparty of the movement is %v, want %d", read.CounterpartyContactID, contactID)
	}

	if read.LoanID == nil || *read.LoanID != loanID {
		t.Errorf("the loan of the movement is %v, want %d", read.LoanID, loanID)
	}
}

// The filter of the movements narrows the list to the movements of one loan.
func TestTheFilterNarrowsByLoan(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	loanID := storedLoan(t, db)
	accountID := movementAccount(t, db, "Cash in the drawer")

	repository := sqlite.NewMovementRepository(db)

	if _, err := repository.Create(ctx, incomeMovement(accountID, 10000, "2024-01-05")); err != nil {
		t.Fatalf("store the income: %v", err)
	}

	if _, err := repository.Create(ctx, domain.Movement{
		AccountID:  accountID,
		Kind:       domain.MovementKindLoanLent,
		Amount:     500000,
		OccurredOn: "2024-02-01",
		LoanID:     &loanID,
	}); err != nil {
		t.Fatalf("store the loan movement: %v", err)
	}

	movements, err := repository.List(ctx, ports.MovementFilter{LoanID: loanID})
	if err != nil {
		t.Fatalf("list the movements of the loan: %v", err)
	}

	if len(movements) != 1 {
		t.Fatalf("the list holds %d movements, want 1", len(movements))
	}

	if movements[0].LoanID == nil || *movements[0].LoanID != loanID {
		t.Errorf("the movement of the list belongs to the loan %v, want %d", movements[0].LoanID, loanID)
	}
}

// A loan keeps the interest rate it was granted with.
func TestALoanKeepsItsInterestRate(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewLoanRepository(db)

	rate := int64(1500)
	loan := borrowedLoan(storedContact(t, db, "Luis"))
	loan.InterestRateBP = &rate

	stored, err := repository.Create(ctx, loan)
	if err != nil {
		t.Fatalf("store the loan: %v", err)
	}

	if stored.InterestRateBP == nil {
		t.Fatal("the stored loan carries no rate, want the rate it was granted with")
	}

	read, err := repository.GetByID(ctx, stored.ID)
	if err != nil {
		t.Fatalf("read the loan: %v", err)
	}

	if read.InterestRateBP == nil || *read.InterestRateBP != 1500 {
		t.Errorf("the rate of the loan is %v, want 1500", read.InterestRateBP)
	}
}

// A movement that does not exist cannot be updated.
func TestUpdatingAMovementThatDoesNotExistIsNotFound(t *testing.T) {
	db := openMigratedDB(t)
	ctx := context.Background()

	repository := sqlite.NewMovementRepository(db)

	movement := incomeMovement(movementAccount(t, db, "Cash in the drawer"), 10000, "2024-01-05")
	movement.ID = 404

	err := repository.Update(ctx, movement)
	if err == nil {
		t.Fatal("the missing movement was updated, want a failure")
	}

	if got := domain.CodeOf(err); got != domain.CodeNotFound {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeNotFound)
	}
}

// A value that a column refuses because it is required is reported as a
// missing field.
func TestANotNullViolationIsARequiredField(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "CREATE TABLE people (id INTEGER PRIMARY KEY, name TEXT NOT NULL)"); err != nil {
		t.Fatalf("create the table: %v", err)
	}

	_, err := db.ExecContext(ctx, "INSERT INTO people (id, name) VALUES (1, NULL)")
	if err == nil {
		t.Fatal("the insert succeeded, want a failure")
	}

	if got := domain.DetailOf(err); got != detailRequired {
		t.Errorf("the detail of the failure is %q, want %q", got, detailRequired)
	}
}

// A trigger that refuses a row is a rule of the schema, and it is reported
// as one.
func TestATriggerViolationBreaksARuleOfTheSchema(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `CREATE TABLE guarded (id INTEGER PRIMARY KEY, value TEXT);
CREATE TRIGGER guard BEFORE INSERT ON guarded BEGIN SELECT RAISE(ABORT, 'not allowed'); END;`); err != nil {
		t.Fatalf("create the table and its trigger: %v", err)
	}

	_, err := db.ExecContext(ctx, "INSERT INTO guarded (id, value) VALUES (1, 'anything')")
	if err == nil {
		t.Fatal("the insert succeeded, want a failure")
	}

	if got := domain.DetailOf(err); got != detailConstraint {
		t.Errorf("the detail of the failure is %q, want %q", got, detailConstraint)
	}
}
