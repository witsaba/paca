// Package db publishes the PACA schema files to the rest of the repository.
//
// The SQL migration files are the source of truth of the schema. They live
// under migrations/ in this package, and this package is the only way to read
// them from Go.
//
// Why a module of its own: go:embed can only embed files that live inside the
// directory tree of the module that declares the embed. The adapter that runs
// the migrations lives in packages/core, which is a different module, so it
// cannot embed packages/db/migrations on its own. Publishing the files from a
// module that owns them keeps the SQL in the place the ADR and the roadmap
// require, and lets the adapter depend on the schema instead of on a copy of
// it.
//
// Dependency rule: this package imports nothing from the project, and nothing
// but the standard library.
package db

import (
	"embed"
	"io/fs"
)

// migrations holds the SQL files of the schema, embedded at build time.
//
//go:embed migrations/*.sql
var migrations embed.FS

// Migrations returns the filesystem that holds the SQL migration files.
//
// The files are named <number>_<name>.sql, and the number is the schema
// version the migration produces: 001_initial_schema.sql produces version 1.
// The returned filesystem is read-only.
//
// The order in which the files have to be applied belongs to the runner that
// applies them, and not to this package: a caller that needs the order asks
// the runner, which decides it for any filesystem, and not only for this one.
func Migrations() fs.FS {
	return migrations
}
