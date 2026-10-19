package sqlite_test

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/witsaba/paca/packages/core/adapters/sqlite"
	"github.com/witsaba/paca/packages/core/domain"
	schema "github.com/witsaba/paca/packages/db"
)

// The initial schema of the repository, read from the published migrations.
func readInitialSchema(t *testing.T) string {
	t.Helper()

	content, err := fs.ReadFile(schema.Migrations(), "migrations/001_initial_schema.sql")
	if err != nil {
		t.Fatalf("read the initial schema: %v", err)
	}

	return string(content)
}

// The tables the initial schema declares.
var schemaTables = []string{"accounts", "contacts", "loans", "movements", "loan_payments"}

// userVersion returns the schema version the database holds.
func userVersion(t *testing.T, db *sqlite.DB) int {
	t.Helper()

	var version int
	if err := db.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read the user_version pragma: %v", err)
	}

	return version
}

// tableNames returns the names of the tables the database holds.
func tableNames(t *testing.T, db *sqlite.DB) []string {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), "SELECT name FROM sqlite_master WHERE type = 'table'")
	if err != nil {
		t.Fatalf("read the tables of the database: %v", err)
	}

	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan the name of a table: %v", err)
		}

		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate the tables of the database: %v", err)
	}

	return names
}

// The runner applies the schema of the repository to an empty database.
func TestMigrateAppliesTheSchema(t *testing.T) {
	db := openTestDB(t)

	if err := sqlite.MigrateSchema(context.Background(), db); err != nil {
		t.Fatalf("migrate the database: %v", err)
	}

	if got := userVersion(t, db); got != 1 {
		t.Errorf("the schema version is %d, want 1", got)
	}

	for _, table := range schemaTables {
		var found bool
		for _, name := range tableNames(t, db) {
			if name == table {
				found = true
			}
		}

		if !found {
			t.Errorf("the migration did not create the table %q", table)
		}
	}
}

// Running the runner again changes nothing: the migration is not applied a
// second time, and the database is left as it was.
func TestMigrateDoesNotApplyTheInitialSchemaTwice(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := sqlite.MigrateSchema(ctx, db); err != nil {
		t.Fatalf("migrate the database: %v", err)
	}

	if err := sqlite.MigrateSchema(ctx, db); err != nil {
		t.Fatalf("migrate the database a second time: %v", err)
	}

	if got := userVersion(t, db); got != 1 {
		t.Errorf("the schema version is %d, want 1", got)
	}

	if got := len(tableNames(t, db)); got != len(schemaTables) {
		t.Errorf("the database holds %d tables, want %d", got, len(schemaTables))
	}
}

// The initial schema refuses to run against a database that already holds
// tables, and the refusal leaves the database exactly as it was.
func TestTheInitialSchemaRefusesToRunTwice(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := sqlite.MigrateSchema(ctx, db); err != nil {
		t.Fatalf("migrate the database: %v", err)
	}

	before := tableNames(t, db)

	_, err := db.ExecContext(ctx, readInitialSchema(t))
	if err == nil {
		t.Fatal("the initial schema ran a second time, want a refusal")
	}

	if got := userVersion(t, db); got != 1 {
		t.Errorf("the schema version is %d, want 1", got)
	}

	after := tableNames(t, db)
	if len(after) != len(before) {
		t.Errorf("the refused migration left %d tables, want %d", len(after), len(before))
	}
}

// A migration that is added later is applied by the same runner, without any
// change to it.
func TestMigrateAppliesANewMigration(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	migrations := fstest.MapFS{
		"migrations/001_initial_schema.sql": &fstest.MapFile{Data: []byte(readInitialSchema(t))},
		"migrations/002_notes.sql": &fstest.MapFile{Data: []byte(`PRAGMA foreign_keys = ON;
BEGIN IMMEDIATE;
CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT NOT NULL) STRICT;
PRAGMA user_version = 2;
COMMIT;
`)},
	}

	if err := sqlite.Migrate(ctx, db, migrations); err != nil {
		t.Fatalf("migrate the database: %v", err)
	}

	if got := userVersion(t, db); got != 2 {
		t.Errorf("the schema version is %d, want 2", got)
	}

	var found bool
	for _, name := range tableNames(t, db) {
		if name == "notes" {
			found = true
		}
	}

	if !found {
		t.Error("the new migration did not create the table it declares")
	}
}

// The runner stops at the first error: the migrations that follow a failing
// one are not applied.
func TestMigrateStopsAtTheFirstError(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	migrations := fstest.MapFS{
		"migrations/001_initial_schema.sql": &fstest.MapFile{Data: []byte(readInitialSchema(t))},
		"migrations/002_broken.sql": &fstest.MapFile{Data: []byte(`INSERT INTO a_table_that_does_not_exist (id) VALUES (1);
`)},
		"migrations/003_later.sql": &fstest.MapFile{Data: []byte(`PRAGMA foreign_keys = ON;
BEGIN IMMEDIATE;
CREATE TABLE later (id INTEGER PRIMARY KEY) STRICT;
PRAGMA user_version = 3;
COMMIT;
`)},
	}

	err := sqlite.Migrate(ctx, db, migrations)
	if err == nil {
		t.Fatal("the migration of a broken file succeeded, want an error")
	}

	if got := userVersion(t, db); got != 1 {
		t.Errorf("the schema version is %d, want 1: the failing migration must not stamp a version", got)
	}

	for _, name := range tableNames(t, db) {
		if name == "later" {
			t.Error("a migration after the failing one was applied")
		}
	}
}

// A filesystem that holds no migrations is a failure, not a silent no-op.
func TestMigrateRefusesAFilesystemWithoutMigrations(t *testing.T) {
	db := openTestDB(t)

	err := sqlite.Migrate(context.Background(), db, fstest.MapFS{})
	if err == nil {
		t.Fatal("the runner accepted a filesystem without migrations, want an error")
	}

	if got := domain.CodeOf(err); got != domain.CodeInternal {
		t.Errorf("the code of the failure is %q, want %q", got, domain.CodeInternal)
	}
}
