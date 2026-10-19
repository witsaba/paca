package sqlite_test

import (
	"context"
	"testing"

	"github.com/witsaba/paca/packages/core/adapters/sqlite"
)

// openMigratedDB opens a database in a temporary directory and applies the
// schema of the repository to it.
//
// Every repository test needs a database that holds the schema, so the
// helper is shared: the tests of a table never define their own.
func openMigratedDB(t *testing.T) *sqlite.DB {
	t.Helper()

	db := openTestDB(t)

	if err := sqlite.MigrateSchema(context.Background(), db); err != nil {
		t.Fatalf("migrate the test database: %v", err)
	}

	return db
}
