package database_test

import (
	"database/sql"
	"log/slog"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/witsaba/paca/apps/db-admin/internal/adapters/outbound/database"
)

func TestTheMigratorNeedsADatabase(t *testing.T) {
	t.Parallel()

	if _, err := database.NewMigrator(nil, slog.Default()); err == nil {
		t.Fatal("NewMigrator() = nil, want an error about the database")
	}
}

func TestTheMigratorIsBuiltOverTheMigratorOfTheCore(t *testing.T) {
	t.Parallel()

	// The handle is never used: the migrator of the core only needs one to
	// take the lock of the database when it applies a migration.
	db, err := sql.Open("pgx", "postgres://paca:paca@127.0.0.1:5432/paca?sslmode=disable")
	if err != nil {
		t.Fatalf("the handle of the database could not be opened: %v", err)
	}
	defer db.Close()

	migrator, err := database.NewMigrator(db, slog.Default())
	if err != nil {
		t.Fatalf("NewMigrator() = %v", err)
	}
	if migrator == nil {
		t.Fatal("NewMigrator() = nil, want the adapter of the core")
	}
}
