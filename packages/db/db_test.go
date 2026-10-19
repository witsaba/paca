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
