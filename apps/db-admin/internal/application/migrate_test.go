package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/witsaba/paca/apps/db-admin/internal/application"
	"github.com/witsaba/paca/apps/db-admin/internal/domain/schema"
)

// fakeMigrator is the migrator of the tests: it answers what the test
// decided, and it remembers what it was asked.
type fakeMigrator struct {
	applied    []schema.Migration
	upErr      error
	upCalls    int
	statuses   []schema.Migration
	statusErr  error
	pending    bool
	pendingErr error
	version    int64
	versionErr error
}

// Up applies the migrations of the test.
func (m *fakeMigrator) Up(context.Context) ([]schema.Migration, error) {
	m.upCalls++
	if m.upErr != nil {
		return nil, m.upErr
	}
	return m.applied, nil
}

// Status returns the state of the migrations of the test.
func (m *fakeMigrator) Status(context.Context) ([]schema.Migration, error) {
	if m.statusErr != nil {
		return nil, m.statusErr
	}
	return m.statuses, nil
}

// Pending tells whether a migration is waiting.
func (m *fakeMigrator) Pending(context.Context) (bool, error) {
	if m.pendingErr != nil {
		return false, m.pendingErr
	}
	return m.pending, nil
}

// Version returns the version of the schema of the test.
func (m *fakeMigrator) Version(context.Context) (int64, error) {
	if m.versionErr != nil {
		return 0, m.versionErr
	}
	return m.version, nil
}

func TestTheMigrationServiceNeedsAMigrator(t *testing.T) {
	t.Parallel()

	if _, err := application.NewMigrateService(nil); err == nil {
		t.Fatal("NewMigrateService() = nil, want an error about the migrator")
	}
}

func TestTheMigrationServiceAppliesThePendingMigrations(t *testing.T) {
	t.Parallel()

	migrator := &fakeMigrator{
		applied: []schema.Migration{
			{Version: 1, Name: "00001_users.sql", Applied: true},
			{Version: 2, Name: "00002_users_guards.sql", Applied: true},
		},
		version: 2,
	}

	service, err := application.NewMigrateService(migrator)
	if err != nil {
		t.Fatalf("NewMigrateService() = %v", err)
	}

	state, err := service.Migrate(context.Background())
	if err != nil {
		t.Fatalf("Migrate() = %v", err)
	}

	if len(state.Applied) != 2 {
		t.Fatalf("applied = %+v, want the two migrations of the test", state.Applied)
	}
	if state.Version != 2 {
		t.Fatalf("version = %d, want %d", state.Version, 2)
	}
	if state.Pending {
		t.Fatal("the state is pending, want the schema to be up to date")
	}
	if !state.IsUpToDate() {
		t.Fatal("the schema is not up to date, want it to be up to date")
	}
}

func TestTheMigrationServiceIsIdempotent(t *testing.T) {
	t.Parallel()

	// The schema is already up to date: the migrator applies nothing.
	migrator := &fakeMigrator{version: 2}

	service, err := application.NewMigrateService(migrator)
	if err != nil {
		t.Fatalf("NewMigrateService() = %v", err)
	}

	state, err := service.Migrate(context.Background())
	if err != nil {
		t.Fatalf("Migrate() = %v, want no error", err)
	}

	if len(state.Applied) != 0 {
		t.Fatalf("applied = %+v, want none", state.Applied)
	}
	if state.Version != 2 {
		t.Fatalf("version = %d, want %d", state.Version, 2)
	}
}

func TestTheMigrationServiceAppliesTheMigrationsOnce(t *testing.T) {
	t.Parallel()

	migrator := &fakeMigrator{version: 2}

	service, err := application.NewMigrateService(migrator)
	if err != nil {
		t.Fatalf("NewMigrateService() = %v", err)
	}

	if _, err := service.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() = %v", err)
	}

	if migrator.upCalls != 1 {
		t.Fatalf("the migrations were applied %d times, want once", migrator.upCalls)
	}
}

func TestTheMigrationServiceFailsWhenTheMigrationsCannotBeApplied(t *testing.T) {
	t.Parallel()

	service, err := application.NewMigrateService(&fakeMigrator{
		upErr: errors.New("the migration 00001_users.sql failed"),
	})
	if err != nil {
		t.Fatalf("NewMigrateService() = %v", err)
	}

	if _, err := service.Migrate(context.Background()); err == nil {
		t.Fatal("Migrate() = nil, want the failure of the migrations")
	}
}

func TestTheMigrationServiceFailsWhenTheVersionCannotBeRead(t *testing.T) {
	t.Parallel()

	service, err := application.NewMigrateService(&fakeMigrator{
		versionErr: errors.New("the version could not be read"),
	})
	if err != nil {
		t.Fatalf("NewMigrateService() = %v", err)
	}

	if _, err := service.Migrate(context.Background()); err == nil {
		t.Fatal("Migrate() = nil, want the failure of the version")
	}
}

func TestTheMigrationServiceFailsWhenThePendingMigrationsCannotBeRead(t *testing.T) {
	t.Parallel()

	service, err := application.NewMigrateService(&fakeMigrator{
		pendingErr: errors.New("the pending migrations could not be read"),
	})
	if err != nil {
		t.Fatalf("NewMigrateService() = %v", err)
	}

	if _, err := service.Migrate(context.Background()); err == nil {
		t.Fatal("Migrate() = nil, want the failure of the pending migrations")
	}
}

func TestTheMigrationServiceReadsTheStateOfTheMigrations(t *testing.T) {
	t.Parallel()

	migrator := &fakeMigrator{
		statuses: []schema.Migration{{Version: 1, Name: "00001_users.sql", Applied: true}},
	}

	service, err := application.NewMigrateService(migrator)
	if err != nil {
		t.Fatalf("NewMigrateService() = %v", err)
	}

	migrations, err := service.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() = %v", err)
	}

	if len(migrations) != 1 || migrations[0].Name != "00001_users.sql" {
		t.Fatalf("migrations = %+v, want the one of the test", migrations)
	}
}

func TestTheMigrationServiceFailsWhenTheStateCannotBeRead(t *testing.T) {
	t.Parallel()

	service, err := application.NewMigrateService(&fakeMigrator{
		statusErr: errors.New("the state could not be read"),
	})
	if err != nil {
		t.Fatalf("NewMigrateService() = %v", err)
	}

	if _, err := service.Status(context.Background()); err == nil {
		t.Fatal("Status() = nil, want the failure of the state")
	}
}
