package sqlite

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/witsaba/paca/packages/core/domain"
	schema "github.com/witsaba/paca/packages/db"
)

// The directory of the filesystem that holds the migration files, and the
// prefix of the name of a migration file.
const (
	migrationsDir      = "migrations"
	migrationSeparator = "_"
	migrationSuffix    = ".sql"
)

// migration is one migration file of the schema.
type migration struct {
	name    string
	version int
}

// Migrate applies the migrations of the schema to the database.
//
// The migrations are applied in the order of their number, and only the ones
// the database has not applied yet: the version the database holds, read from
// PRAGMA user_version, is the version of the last migration that ran. Running
// the runner twice is therefore safe and changes nothing.
//
// The runner stops at the first error. A migration that fails leaves the
// database as it was, and the migrations that follow it are not applied.
//
// Each migration file manages its own transaction and stamps the version it
// produces, exactly like the initial schema does. The runner checks that
// stamp: a migration that does not stamp the version it promises is a
// failure, because the next run would not know what has been applied.
func Migrate(ctx context.Context, database *DB, migrations fs.FS) error {
	current, err := userVersion(ctx, database)
	if err != nil {
		return err
	}

	files, err := listMigrations(migrations)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.version <= current {
			continue
		}

		if err := applyMigration(ctx, database, migrations, file); err != nil {
			return err
		}
	}

	return nil
}

// MigrateSchema applies the migrations that the repository publishes to the
// database.
//
// It is the runner of the schema of PACA: the migrations of packages/db, in
// the order of their number.
func MigrateSchema(ctx context.Context, database *DB) error {
	return Migrate(ctx, database, schema.Migrations())
}

// userVersion returns the version of the schema the database holds.
func userVersion(ctx context.Context, database *DB) (int, error) {
	var version int
	if err := database.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return 0, err
	}

	return version, nil
}

// listMigrations returns the migration files of the filesystem, in the order
// they have to be applied.
//
// The order is the number of the file, and the number is also the version of
// the schema the migration produces. A file whose name carries no number, and
// two files that carry the same number, are failures: the order of the
// migrations cannot be decided.
func listMigrations(migrations fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(migrations, migrationsDir)
	if err != nil {
		return nil, domain.Wrap(domain.CodeInternal, "the migrations are missing", err)
	}

	files := make([]migration, 0, len(entries))
	versions := make(map[int]bool, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version, err := migrationVersion(entry.Name())
		if err != nil {
			return nil, err
		}

		if versions[version] {
			return nil, domain.New(domain.CodeInternal, "two migrations carry the same number")
		}

		versions[version] = true
		files = append(files, migration{name: entry.Name(), version: version})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })

	return files, nil
}

// migrationVersion returns the version a migration file produces, read from
// the number at the front of its name.
func migrationVersion(name string) (int, error) {
	prefix, _, found := strings.Cut(name, migrationSeparator)
	if !found {
		return 0, domain.New(domain.CodeInternal, "a migration file carries no number")
	}

	version, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, domain.Wrap(domain.CodeInternal, "a migration file carries no number", err)
	}

	return version, nil
}

// applyMigration runs one migration file and checks the version it stamps.
func applyMigration(ctx context.Context, database *DB, migrations fs.FS, file migration) error {
	script, err := fs.ReadFile(migrations, migrationsDir+"/"+file.name)
	if err != nil {
		return domain.Wrap(domain.CodeInternal, "a migration file cannot be read", err)
	}

	if _, err := database.ExecContext(ctx, string(script)); err != nil {
		// The script may have left a transaction open when it failed. The
		// rollback is a formality: it fails on a connection that holds no
		// transaction, and that failure is not the one worth reporting.
		_, _ = database.ExecContext(ctx, "ROLLBACK")

		return domain.Wrap(domain.CodeInternal, fmt.Sprintf("the migration %s failed", file.name), err)
	}

	version, err := userVersion(ctx, database)
	if err != nil {
		return err
	}

	if version != file.version {
		return domain.New(domain.CodeInternal, fmt.Sprintf("the migration %s did not stamp the version it promises", file.name))
	}

	return nil
}
