# `packages/db` — Database Package

The PACA database schema: the migration script, the verification harness, and
the Go module that publishes the schema to the rest of the repository.

## Contents

```
packages/db/
├── db.go                           publishes the migrations to Go (go:embed)
├── migrations/
│   └── 001_initial_schema.sql    the authoritative schema (5 tables, 12 indexes)
└── tools/
    └── verify_schema.sh           automated verification (57 checks)
```

## The Go module

`packages/db` is a Go module of the workspace: `github.com/witsaba/paca/packages/db`.
It publishes the migration files as a read-only `fs.FS`, so that the code that
applies them does not depend on a path on a disk:

```go
import schema "github.com/witsaba/paca/packages/db"

migrations := schema.Migrations()   // an fs.FS holding migrations/*.sql
```

**Why a module of its own:** `go:embed` can only embed the files that live
inside the directory tree of the module that declares the embed. The runner
that applies the migrations lives in `packages/core`, which is a different
module. Publishing the files from a module that owns them keeps the SQL in the
place this package documents it, and lets the adapter depend on the schema
instead of on a copy of it.

The module publishes the files and nothing else: the order in which the
migrations are applied belongs to the runner, and not to this package.

## Apply the Migration

```sh
sqlite3 -bail paca.db < packages/db/migrations/001_initial_schema.sql
```

**Always use `-bail`.** The sqlite3 CLI continues past errors by default. The migration
is atomic — it runs in a single transaction — so a failure rolls back cleanly. With
`-bail`, the CLI stops at the first error, which is the safe behavior for a migration.

**Requires:** SQLite >= 3.37.0 (for STRICT tables).

**From Go:** the adapter applies the migrations with the same discipline — it
stops at the first error — through
[`packages/core/adapters/sqlite`](../core/adapters/sqlite).

## Verify

```sh
bash packages/db/tools/verify_schema.sh
```

The script applies the migration to a temporary database and runs 57 automated checks:
structure, every CHECK constraint, uniqueness policies, referential actions, and the
non-empty-database guard. All checks pass.

## Add a Migration

1. Create `packages/db/migrations/002_<name>.sql` (next sequential number).
2. Start with `PRAGMA foreign_keys = ON;` and `BEGIN IMMEDIATE;`.
3. Apply your changes.
4. End with `PRAGMA user_version = 2;` (the next version number), still inside the
   transaction, then `COMMIT;`.

A later migration runs against a database that already has tables, so it must not reuse
the initial migration's emptiness precheck. It should instead check that the schema it
expects is the one it finds, for example by asserting `PRAGMA user_version` is the value
it is migrating from.

For the full procedure to add a new account kind (including the 3.53.0+ path), see
[Extending the schema](../README.md#extending-the-schema) in the root README.

## Full Reference

For the complete table and column reference, the inheritance model explanation, all
`kind` vocabularies, the index guide, useful queries, and the design rationale, see
the root [README.md](../README.md).
