package schema_test

import (
	"testing"
	"time"

	"github.com/witsaba/paca/packages/domain/schema"
)

func TestTheSchemaIsUpToDateWhenNothingIsPending(t *testing.T) {
	t.Parallel()

	state := schema.State{Version: 2}

	if !state.IsUpToDate() {
		t.Fatal("IsUpToDate() = false, want true")
	}
}

func TestTheSchemaIsNotUpToDateWhenAMigrationIsPending(t *testing.T) {
	t.Parallel()

	state := schema.State{Version: 2, Pending: true}

	if state.IsUpToDate() {
		t.Fatal("IsUpToDate() = true, want false")
	}
}

func TestTheAppliedVersionsAreTheNumbersOfTheAppliedMigrations(t *testing.T) {
	t.Parallel()

	state := schema.State{
		Version: 2,
		Applied: []schema.Migration{
			{Version: 1, Name: "00001_users.sql", Applied: true},
			{Version: 2, Name: "00002_users_guards.sql", Applied: true},
		},
	}

	versions := state.AppliedVersions()
	if len(versions) != 2 {
		t.Fatalf("versions = %v, want two of them", versions)
	}
	if versions[0] != 1 || versions[1] != 2 {
		t.Fatalf("versions = %v, want them in the order they were applied", versions)
	}
}

func TestTheAppliedVersionsOfAnUpToDateSchemaAreEmpty(t *testing.T) {
	t.Parallel()

	versions := schema.State{Version: 2}.AppliedVersions()
	if len(versions) != 0 {
		t.Fatalf("versions = %v, want none", versions)
	}
}

func TestTheMigrationCarriesTheMomentItWasApplied(t *testing.T) {
	t.Parallel()

	appliedAt := time.Date(2026, time.October, 27, 10, 0, 0, 0, time.UTC)
	migration := schema.Migration{Version: 1, Name: "00001_users.sql", Applied: true, AppliedAt: appliedAt}

	if !migration.Applied {
		t.Fatal("the migration is not applied, want it to be applied")
	}
	if !migration.AppliedAt.Equal(appliedAt) {
		t.Fatalf("applied at = %v, want %v", migration.AppliedAt, appliedAt)
	}
}
