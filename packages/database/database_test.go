package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/packages/database"
)

func TestNewMigratorWithoutADatabase(t *testing.T) {
	t.Parallel()

	migrator, err := database.NewMigrator(nil, nil)
	if err == nil {
		t.Fatalf("the migrator was created without a database: %+v", migrator)
	}
	if !errors.Is(err, database.ErrNoDatabase) {
		t.Errorf("the error is %v, want it to wrap ErrNoDatabase", err)
	}
}

func TestMigratorWithoutADatabase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	migrator := &database.Migrator{}

	t.Run("the migrations cannot be applied", func(t *testing.T) {
		t.Parallel()

		if _, err := migrator.Up(ctx); !errors.Is(err, database.ErrNoDatabase) {
			t.Errorf("the error is %v, want it to wrap ErrNoDatabase", err)
		}
	})

	t.Run("the state cannot be read", func(t *testing.T) {
		t.Parallel()

		if _, err := migrator.Status(ctx); !errors.Is(err, database.ErrNoDatabase) {
			t.Errorf("the error is %v, want it to wrap ErrNoDatabase", err)
		}
	})

	t.Run("the pending migrations cannot be read", func(t *testing.T) {
		t.Parallel()

		if _, err := migrator.Pending(ctx); !errors.Is(err, database.ErrNoDatabase) {
			t.Errorf("the error is %v, want it to wrap ErrNoDatabase", err)
		}
	})

	t.Run("the version cannot be read", func(t *testing.T) {
		t.Parallel()

		if _, err := migrator.Version(ctx); !errors.Is(err, database.ErrNoDatabase) {
			t.Errorf("the error is %v, want it to wrap ErrNoDatabase", err)
		}
	})
}
