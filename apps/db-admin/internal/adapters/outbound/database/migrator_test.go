package database_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/witsaba/paca/apps/db-admin/internal/adapters/outbound/database"
	core "github.com/witsaba/paca/packages/database"
	"github.com/witsaba/paca/packages/database/postgres"
)

// quietLogger is the logger of the tests of the migrator: it says nothing.
func quietLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// openPool opens the pool of the library, without waiting for the database.
//
// The pool is lazy: it opens no connection until one is needed, so a pool
// over a database that is down is a pool like any other.
func openPool(t *testing.T, url string) *postgres.Pool {
	t.Helper()

	poolCfg := postgres.DefaultConfig()
	poolCfg.URL = url

	pool, err := postgres.Open(context.Background(), poolCfg)
	if err != nil {
		t.Fatalf("the pool of the library could not be opened: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func TestTheMigratorNeedsADatabase(t *testing.T) {
	t.Parallel()

	// The migrator of the library is the one that refuses a database that
	// is not there: the composition root builds it, and the error is the
	// one of the library.
	if _, err := core.NewMigrator(nil, quietLogger()); err == nil {
		t.Fatal("NewMigrator() = nil, want an error about the database")
	}
}

func TestTheAdapterRefusesAMigratorThatIsNotThere(t *testing.T) {
	t.Parallel()

	// The adapter over no migrator of the library fails instead of
	// panicking: the migrator of the library answers the failure itself.
	migrator := database.NewMigrator(nil)

	if _, err := migrator.Up(context.Background()); err == nil {
		t.Fatal("Up() = nil, want an error about the database")
	}
}

func TestTheMigratorIsBuiltOverTheMigratorOfTheCore(t *testing.T) {
	t.Parallel()

	// The handle is never used: the migrator of the core only needs one to
	// take the lock of the database when it applies a migration. The pool
	// is the only door to PostgreSQL, and the migrator of the core speaks
	// through its view.
	pool := openPool(t, "postgres://paca:paca@127.0.0.1:5432/paca?sslmode=disable")

	libraryMigrator, err := core.NewMigrator(pool.SQL(), quietLogger())
	if err != nil {
		t.Fatalf("NewMigrator() = %v", err)
	}

	migrator := database.NewMigrator(libraryMigrator)
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
	pool := openPool(t, "postgres://paca:paca@127.0.0.1:1/paca?sslmode=disable")

	libraryMigrator, err := core.NewMigrator(pool.SQL(), quietLogger())
	if err != nil {
		t.Fatalf("NewMigrator() = %v", err)
	}

	migrator := database.NewMigrator(libraryMigrator)

	applied, err := migrator.Up(context.Background())
	if err == nil {
		t.Fatal("Up() = nil, want the failure of the migrator of the core")
	}
	if applied != nil {
		t.Fatalf("applied = %+v, want none", applied)
	}
}
