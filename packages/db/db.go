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
	"fmt"
	"io/fs"
	"sort"
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
func Migrations() fs.FS {
	return migrations
}

// MigrationNames returns the names of the migration files, in the order they
// have to be applied.
//
// The order is the numeric prefix of the name, not the lexical order of the
// whole name, so that the tenth migration follows the ninth. The numeric
// prefix is also the schema version each migration is expected to stamp, which
// is what the migration runner compares against PRAGMA user_version.
func MigrationNames() ([]string, error) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read the migrations directory: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		names = append(names, entry.Name())
	}

	sort.Slice(names, func(i, j int) bool { return version(names[i]) < version(names[j]) })

	return names, nil
}

// version returns the numeric prefix of a migration file name, or -1 when the
// name carries no number at all. A file without a number cannot be placed in
// the order, so it sorts first and the runner refuses it.
func version(name string) int {
	number := 0
	digits := 0

	for _, r := range name {
		if r < '0' || r > '9' {
			break
		}

		number = number*10 + int(r-'0')
		digits++
	}

	if digits == 0 {
		return -1
	}

	return number
}
