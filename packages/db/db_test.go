package db_test

import (
	"io/fs"
	"testing"

	"github.com/witsaba/paca/packages/db"
)

// The published migrations are the SQL files of the repository, and the
// initial schema is one of them.
func TestMigrationsHoldsTheSchemaFiles(t *testing.T) {
	got, err := fs.ReadDir(db.Migrations(), "migrations")
	if err != nil {
		t.Fatalf("read the published migrations: %v", err)
	}

	if len(got) == 0 {
		t.Fatal("the published migrations are empty, want at least the initial schema")
	}

	var found bool
	for _, entry := range got {
		if entry.Name() == "001_initial_schema.sql" {
			found = true
		}
	}

	if !found {
		t.Errorf("the published migrations do not hold 001_initial_schema.sql, got %d files", len(got))
	}
}

// The names come back in the order the migrations have to be applied, which is
// the numeric prefix of the name and not the lexical order of the file name.
func TestMigrationNamesAreOrderedByNumber(t *testing.T) {
	names, err := db.MigrationNames()
	if err != nil {
		t.Fatalf("list the migration names: %v", err)
	}

	if len(names) == 0 {
		t.Fatal("no migration names, want at least the initial schema")
	}

	if names[0] != "001_initial_schema.sql" {
		t.Errorf("the first migration is %q, want 001_initial_schema.sql", names[0])
	}
}
