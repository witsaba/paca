package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/db-admin/internal/adapters/outbound/database"
	"github.com/witsaba/paca/packages/domain/health"
	"github.com/witsaba/paca/packages/domain/schema"
)

// fakePinger is the database of the test: it answers what the test decided.
type fakePinger struct {
	err error
}

// Ping answers the question of the test.
func (p *fakePinger) Ping(context.Context) error {
	return p.err
}

// fakeMigrator is the migrator of the test: it tells what the test decided
// about the state of the schema.
type fakeMigrator struct {
	pending    bool
	pendingErr error
}

// Up applies nothing.
func (m *fakeMigrator) Up(context.Context) ([]schema.Migration, error) {
	return nil, nil
}

// Status returns nothing.
func (m *fakeMigrator) Status(context.Context) ([]schema.Migration, error) {
	return nil, nil
}

// Pending tells whether a migration is waiting.
func (m *fakeMigrator) Pending(context.Context) (bool, error) {
	if m.pendingErr != nil {
		return false, m.pendingErr
	}
	return m.pending, nil
}

// Version returns the version of the schema.
func (m *fakeMigrator) Version(context.Context) (int64, error) {
	return 1, nil
}

func TestTheCheckerNamesTheComponentItWatches(t *testing.T) {
	t.Parallel()

	if got := database.NewChecker(&fakePinger{}, &fakeMigrator{}).Name(); got != "database" {
		t.Fatalf("Name() = %q, want %q", got, "database")
	}
}

func TestTheCheckerIsHappyWhenTheDatabaseAnswersAndTheSchemaIsUpToDate(t *testing.T) {
	t.Parallel()

	checker := database.NewChecker(&fakePinger{}, &fakeMigrator{})

	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("Check() = %v, want nil", err)
	}
}

func TestTheCheckerFailsWhenTheDatabaseDoesNotAnswer(t *testing.T) {
	t.Parallel()

	failure := errors.New("the database did not answer")
	checker := database.NewChecker(&fakePinger{err: failure}, &fakeMigrator{})

	err := checker.Check(context.Background())
	if err == nil {
		t.Fatal("Check() = nil, want the failure of the database")
	}
	if !errors.Is(err, failure) {
		t.Fatalf("Check() = %v, want %v", err, failure)
	}
	if errors.Is(err, health.ErrDegraded) {
		t.Fatal("the check is degraded, want it to be a failure")
	}
}

func TestTheCheckerIsDegradedWhenTheSchemaIsBehind(t *testing.T) {
	t.Parallel()

	checker := database.NewChecker(&fakePinger{}, &fakeMigrator{pending: true})

	err := checker.Check(context.Background())
	if err == nil {
		t.Fatal("Check() = nil, want the pending migrations")
	}
	if !errors.Is(err, health.ErrDegraded) {
		t.Fatalf("Check() = %v, want a degraded error", err)
	}
	if !errors.Is(err, database.ErrPendingMigrations) {
		t.Fatalf("Check() = %v, want %v", err, database.ErrPendingMigrations)
	}
}

func TestTheCheckerFailsWhenTheStateOfTheSchemaCannotBeRead(t *testing.T) {
	t.Parallel()

	checker := database.NewChecker(&fakePinger{}, &fakeMigrator{
		pendingErr: errors.New("the state of the schema could not be read"),
	})

	if err := checker.Check(context.Background()); err == nil {
		t.Fatal("Check() = nil, want the failure of the state of the schema")
	}
}

func TestThePingerWithoutADatabaseFails(t *testing.T) {
	t.Parallel()

	if err := database.NewPinger(nil).Ping(context.Background()); err == nil {
		t.Fatal("Ping() = nil, want an error about the database")
	}
}
