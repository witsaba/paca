// Package schema holds the state of the schema of the database: the version
// it is at, the migrations that have been applied, and whether one of them
// is still waiting.
//
// The model is pure. It knows nothing about SQL, about the driver, or about
// the migrator that writes the schema, so the rules of the migration use
// case can be tested without a database.
//
// The package lives in the module of the domain because it has two sides: the
// service that writes the schema and the library that applies the migrations.
// One model, one owner: the two sides speak the same record, and the adapter
// between them maps nothing.
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
	// time when the moment is not known: a migration that was applied a
	// moment ago carries it until the state of the schema is read.
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
