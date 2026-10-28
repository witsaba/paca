// Package schema holds the state of the schema of the database: the version
// it is at, the migrations that have been applied, and whether one of them
// is still waiting.
//
// The model is pure. It knows nothing about SQL, about the driver, or about
// the migrator that writes the schema, so the rules of the migration use
// case can be tested without a database.
package schema

import "time"

// Migration is the state of one migration of the schema.
type Migration struct {
	// Version is the number of the migration.
	Version int64

	// Name is the name of the file of the migration.
	Name string

	// Applied tells whether the migration has been applied to the database.
	Applied bool

	// AppliedAt is the moment the migration was applied. It is the zero
	// time when the migration has not been applied.
	AppliedAt time.Time
}

// State is the state of the schema of the database at a point in time.
type State struct {
	// Version is the version of the schema: the number of the last
	// migration that has been applied.
	Version int64

	// Pending tells whether a migration is waiting to be applied.
	Pending bool

	// Applied holds the migrations that the last run applied. It is empty
	// when the schema was already up to date: applying the migrations
	// twice is not an error, it is nothing to do.
	Applied []Migration
}

// IsUpToDate tells whether the schema has nothing left to apply.
func (s State) IsUpToDate() bool {
	return !s.Pending
}

// AppliedVersions returns the numbers of the migrations that the last run
// applied, in the order they were applied.
func (s State) AppliedVersions() []int64 {
	versions := make([]int64, 0, len(s.Applied))
	for _, migration := range s.Applied {
		versions = append(versions, migration.Version)
	}
	return versions
}
