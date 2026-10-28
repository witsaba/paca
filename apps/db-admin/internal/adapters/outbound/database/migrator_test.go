package database_test

import (
	"context"
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

func TestTheMigratorAppliesTheMigrationsThroughTheCore(t *testing.T) {
	t.Parallel()

	// The database does not answer: nothing is listening on that port. The
	// migrator of the core takes the lock of the database before it applies
	// a migration, so it fails, and the adapter fails with it. An adapter
	// that wrote the migrations on its own would answer differently.
	db, err := sql.Open("pgx", "postgres://paca:paca@127.0.0.1:1/paca?sslmode=disable")
	if err != nil {
		t.Fatalf("the handle of the database could not be opened: %v", err)
	}
	defer db.Close()

	migrator, err := database.NewMigrator(db, slog.Default())
	if err != nil {
		t.Fatalf("NewMigrator() = %v", err)
	}

	applied, err := migrator.Up(context.Background())
	if err == nil {
		t.Fatal("Up() = nil, want the failure of the migrator of the core")
	}
	if applied != nil {
		t.Fatalf("applied = %+v, want none", applied)
	}
}
