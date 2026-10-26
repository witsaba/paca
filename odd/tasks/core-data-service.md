# PACA — the core data service

Feature document for the shared persistence component of the monorepository: the
module that holds the domain models, the repositories and the migrations, so
that the MCP server and the future services never write them twice.

- **Project:** `packages/core` (Go module `github.com/witsaba/paca/packages/core`).
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-core-data`, branch `feat/core-data-service`.
- **Scope:** the shared persistence library, its single schema writer, and the first real table: `users`. No CRUD, no MCP tool, no REST endpoint, no gRPC service, no authentication.

---

## Specs

Each spec is the user's own wording, quoted verbatim, except the ones marked
*(decided)*, which record a choice taken during the spike and the discussion.

### S1 — One core service for the database stuff

> I need you create a core service that can be used for the current MCP server, and for other future services projects like a worker services and, i dont know, a rest service.

### S2 — No repeated models, repositories and migrations

> I wanna avoid to repeat the models, repositories, migrations code and this services must to be only for the database stuff.

### S3 — Architectural research first

> Web research about given this mono repo what is the best architectural way to add the service to this monorepo.

### S4 — Spike and discuss, no code

> I dont wanna you code yet or start making changes, i wanna you spike and discuss it with me. present to me three diferent options with pro/cons and tradeofds

### S5 — The hybrid shape *(decided)*

The three options were: a shared library module, a standalone gRPC data
service, and the hybrid. **The hybrid was chosen.**

The component is a Go library module, imported by every service of the
monorepository. It owns the schema, the models, the repositories and the
migrations, and it exposes no network surface of its own.

Two commitments come with it:

1. **The schema has exactly one writer.** The migrations are applied by a
   dedicated command, never by a service at boot. A service that migrates on
   boot is a service that races with its own replicas.
2. **The repositories are shaped as ports.** They are defined by what a use
   case needs, never by the shape of a table. That way a gRPC client adapter
   can be added later against the same contracts, without a redesign.

### S6 — sqlc over pgx *(decided)*

**sqlc with pgx was chosen** over raw pgx and over GORM or Ent.

The SQL stays the source of truth and the generated Go is the only thing that
touches a row. The domain never learns an ORM, which is the rule the hexagonal
architecture of the MCP server already enforces. Postgres-specific features
(pgvector, JSONB, partial indexes) stay reachable, which a generic ORM
narrows.

### S7 — The module lives in `packages/` *(decided)*

The layout of the repository reserves `packages/` for the libraries the
services share, and `packages/core` is the first one. It is its own Go module,
listed in `go.work`, so it keeps working on its own.

### S8 — A consumer declares the dependency explicitly *(decided)*

A module of the workspace that another module imports must carry the
dependency in its own `go.mod`:

```
require github.com/witsaba/paca/packages/core v0.0.0
replace github.com/witsaba/paca/packages/core => ../../packages/core
```

A workspace-only import builds, but `go mod tidy` resolves it from the network
and fails, and the `tidy-check` gate of the repository would stay red. The
`replace` is what makes the module both tidy and self-contained.

A `replace` inside `go.work` is **not** an option: the toolchain rejects it
with `workspace module ... is replaced at all versions in the go.work file`.

### S9 — No `internal/` in the shared module *(decided)*

`packages/core/internal/...` is unreachable from `apps/mcp-server`: the
visibility rule of `internal` is scoped to the module, and the two are
different modules. The public surface of the core is curated by hand instead,
and the packages that are not part of the contract stay unexported inside their
own package.

### S10 — The configuration belongs to the service *(decided)*

The core reads no environment variable and opens no socket on its own. It
takes a configuration struct — the address, the pool size, the timeouts — and
returns a pool. Each service maps its own environment onto that struct, so
the same core serves the MCP server, the worker and the REST backend with
their own settings.

### S11 — The migrations travel with the code *(decided)*

The migration files live in `packages/core/migrations`, embedded with
`embed.FS`, and the runner lives in the same module. The schema then travels
with the code that owns it, and the version of the code and the version of the
schema are always the same commit.

`db/postgres/migrations` was the alternative, next to the stack that runs the
database. It keeps the DDL beside the image and the smoke test, but it splits
the owner of the schema across two directories, and the runner needs an extra
path outside its own module.

### S12 — goose runs the schema *(decided)*

`goose` runs the versioned SQL files. It embeds them and runs them from a Go
program, which is what the single writer needs, and it keeps the failure mode
readable: a migration that fails leaves the version recorded, and the next run
tells the truth about it.

`golang-migrate` was the alternative, with the same embedding and the same
programmatic runner. Atlas stays the declarative option for the day the schema
grows enough to want a diff. The tool is pinned in `tools/go.mod`, like every
other tool of the repository.

### S13 — The quality gate of the module *(decided)*

`packages/core` brings the same Makefile as every other Go module of the
repository: the format check, the analyzers, the linter, the build, the test
suite, the race detector, the coverage gate at 80%, the benchmarks and the
vulnerability scan. The tools are pinned in `tools/go.mod`, and the CI job
runs the gate of the core next to the gate of the applications.

### S14 — The tests run against a real database *(decided)*

The repositories are tested against a real PostgreSQL with pgvector, started
with testcontainers, exactly like the container tests of the MCP server. The
tests apply the migrations first, so the migration path is exercised too.

### S15 — The gRPC facade is out of scope *(decided)*

No gRPC service is built in this slice. What the slice delivers is the *seam*:
the repository contracts that a facade would implement. The reserved port
50051 stays free for the future gRPC backend, as the repository registry says.

### S16 — The first real table: `users`

> let add a initial table called users, must to have a unique pk int id not sequential, a way to store the complete name but optional, a status check in database by default `onboarding` but with the `active`, `inactive`, `blocked` not null. a status observations optional,  A phone number optional, a email optional, a created_at and updated_at with system time default. For now DO NOT ADD ANY CRUD, is just the very first table for adding a real one and ensure a proper implementation

The table carries:

| Column                | Type         | Rules                                              |
|-----------------------|--------------|----------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, not sequential, never generated |
| `full_name`           | `text`       | Optional                                           |
| `status`              | `text`       | Not null, default `onboarding`, checked against the four states |
| `status_observations` | `text`       | Optional                                           |
| `phone`               | `text`       | Optional                                           |
| `email`               | `text`       | Optional                                           |
| `created_at`          | `timestamptz`| Not null, default `now()`                          |
| `updated_at`          | `timestamptz`| Not null, default `now()`                          |

The status check lives in the database, so no writer can store a state that
does not exist:

```sql
CONSTRAINT users_status_check CHECK (status IN ('onboarding', 'active', 'inactive', 'blocked'))
```

### S17 — The primary key is the Colombian national document ID

> Must to be asked to the user, this ID number is in the national document ID in colombia.

The `id` is **not generated**. It is the national document ID of the person,
supplied by the caller, and it is the primary key of the row.

That answers why it is not sequential: it is not a counter, it is a real-world
number that the person already carries. Three consequences follow, and they
are all good ones:

- The identifier is stable for the life of the person, so the primary key is
  never updated.
- The values are spread over the whole space, so the index of the primary key
  does not suffer the page splits of a serial key.
- The core needs no ID generator at all, which is one less piece of machinery
  to get wrong.

The column is a `bigint`, because the document number comes from outside the
system and a ten-digit cédula should never meet the ceiling of a 32-bit
integer. It still holds the exact number, with no encoding and no prefix.

### S18 — The status observations are a text field *(decided)*

> a status observations optional

A single optional `status_observations text` column: a free-text note about
the user's current status. It was chosen over a JSONB column and over a
separate observations table, both of which were offered.

### S19 — `updated_at` is a default, not a trigger *(decided)*

> a created_at and updated_at with system time default

Both timestamps are `NOT NULL DEFAULT now()`. The database fills them on
insert. There is no trigger: `updated_at` is not kept current by a write
that the database performs on its own, so whoever updates a row is
responsible for setting it.

### S20 — No CRUD in this slice *(decided)*

> For now DO NOT ADD ANY CRUD, is just the very first table for adding a real one and ensure a proper implementation

No create, no read, no update, no delete. Not as an MCP tool, not as a
repository method, not as a SQL query.

What the slice delivers is the table itself, done properly: the migration
that creates it, the generated row model, the domain model that carries the
states, and the tests that prove every rule of the table holds against a real
database.

The repository port comes with the slice that adds the first CRUD operation,
because a port with no operation is not a contract.

### S21 — A new worktree for the work *(decided)*

> i forgot to mention that you must to open a new git worktree to working on it

The work happens in its own worktree, on its own branch, and never on `main`:

- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-core-data`
- **Branch:** `feat/core-data-service`

---

## Tasks

| ID  | Specs                  | Route | Commit | Summary |
|-----|------------------------|-------|--------|---------|
| T1  | S7, S8, S13            | inline | `22d8f7a` | The module `packages/core`, the `go.work` entry, the Makefile and the CI entry of the gate |
| T2  | S16, S17, S18, S19     | inline | `22d8f7a` | The domain model of the user: the identifier, the four states and the invariants |
| T3  | S6, S16                | inline | `482a9ab` | The sqlc pipeline: the schema of the table and the generated row model |
| T4  | S2, S10                | inline | `3dc1cb6` | The postgres adapter: the pool, the configuration and the ping of the database |
| T5  | S1, S11, S12, S16, S19 | inline | `03cd28a`, `74a5c2b` | The migration of the `users` table and the single writer that applies it |
| T6  | S1, S2, S8             | inline | `781c7a3` | The wiring into `apps/mcp-server`: the dependency, the configuration and the health check of the database |
| T7  | S14, S16, S17, S18, S19, S20 | worker | `10b977c` | The testcontainers suite: the migration, and every rule of the table proven against a real database |
| T8  | S3, S13                | inline | `8ac8154` | The documentation: the README of the core and the registry of the monorepository |

### The slice boundaries

The delivery is split into work units of roughly 400 authored changed lines or
less, so every commit can be reviewed on its own. Every commit builds, and
every commit keeps the test suite green.

1. The module of the core, its Makefile and its entry in the CI.
2. The domain model of the user.
3. The sqlc pipeline and the generated model.
4. The postgres adapter.
5. The migration of the table and the single writer.
6. The wiring into the MCP server.
7. The container tests that prove the table.
8. The documentation, which closes the feature.

The commit that carries the migration carries it alone, so a reviewer reads
the schema change without the code around it.

### The review slices

The feature is bigger than one review, so it is delivered as a stack of
pull requests, one per slice, each branch cut from the branch of the slice
below it.

| Slice | Branch             | What it carries                                          | Commits |
|-------|--------------------|----------------------------------------------------------|---------|
| 1     | `feat/core-data-service` | The specification, the module of the core, the gate and the model of the user | `15fa3ac`, `22d8f7a` |
| 2     | `feat/core-data-table`   | The sqlc pipeline, the postgres adapter, the migration of the `users` table and the single writer | `482a9ab`, `3dc1cb6`, `03cd28a` |
| 3     | `feat/core-data-consumers` | The wiring into the MCP server and the container tests that prove the table | `781c7a3`, `74a5c2b`, `10b977c` |
| 4     | `feat/core-data-docs`    | The documentation of the core and the registry of the monorepository | `8ac8154`, `3770c7b` |

Slice 1 is the branch of this slice of work. Slice 2 is cut from it, slice 3
from slice 2, and slice 4 from slice 3: the stack is merged from the bottom,
so a reviewer reads the module, then the table, then the consumer, then the
words.

The last commit of the feature carries the whole slice. The commit that
touches the migrations carries them alone, so a reviewer can read the schema
change without the code around it.

---

## Log

### L1 — Original request

> I need you create a core service that can be used for the current MCP server, and for other future services projects like a worker services and, i dont know, a rest service. I wanna avoid to repeat the models, repositories, migrations code and this services must to be only for the database stuff. Web research about given this mono repo what is the best architectural way to add the service to this monorepo. I dont wanna you code yet or start making changes, i wanna you spike and discuss it with me. present to me three diferent options with pro/cons and tradeofds

### L2 — The spike and its findings (2026-10-27)

The repository as it stands: a Go workspace with `apps/mcp-server` (one
module, hexagonal, strict), `db/postgres` (PostgreSQL 18.6 with pgvector
0.8.7, internal only), `observability` (Jaeger 2), and `packages/` reserved
for the shared libraries. No persistence code exists yet: no driver, no
schema, no migration.

The three options that were presented, with their tradeoffs:

| | Shared library module | Standalone gRPC data service | Hybrid |
|---|---|---|---|
| Latency | none | one hop per query | none for now |
| Atomic write across repositories | yes | no, sagas needed | yes for now |
| Compile-time types | yes | no, generated from proto | yes for now |
| Single schema owner | by convention | yes | yes |
| New operational surface | none | high | low |
| Non-Go consumers | no | yes | yes, later |
| Cost of the wrong choice | low | high | low |

### L3 — Decisions taken in the questionnaire

- Architecture: **the hybrid** (S5).
- Persistence: **sqlc with pgx** (S6).

### L4 — The mechanics of the workspace, verified in a scratch directory

- An `internal` package of one module cannot be imported by another module.
  The visibility rule is module-scoped, so the shared core cannot hide its
  internals behind `packages/core/internal` (S9).
- A workspace-only import builds and tests, but `go mod tidy` looks the
  module up on the network and fails. `go work sync` fails the same way.
- The shape that works is `require` plus `replace` in the `go.mod` of the
  consumer. With it, `go mod tidy` succeeds and the module still builds on
  its own, as the repository requires (S8).
- A `replace` in `go.work` alone is rejected by the toolchain.

### L5 — The last two open specs, answered

- **S11:** the migration files live in `packages/core/migrations`, embedded.
- **S12:** `goose` runs them.

### L6 — The first table, requested

> let add a initial table called users, must to have a unique pk int id not sequential, a way to store the complete name but optional, a status check in database by default `onboarding` but with the `active`, `inactive`, `blocked` not null. a status observations optional,  A phone number optional, a email optional, a created_at and updated_at with system time default. For now DO NOT ADD ANY CRUD, is just the very first table for adding a real one and ensure a proper implementation

### L7 — The identifier, clarified

> Must to be asked to the user, this ID number is in the national document ID in colombia.

The `id` is the Colombian national document ID, supplied by the caller, never
generated. The table therefore needs no sequence, no identity column and no ID
generator in the core. The column is a `bigint` (S17).

The two other questions of the same round were answered:

- **S18:** the status observations are a `text` column.
- **S19:** `updated_at` is a default, with no trigger.

### L8 — The worktree

> i forgot to mention that you must to open a new git worktree to working on it

The work happens in `/Users/braejan/workspace/witsaba/repositories/paca-core-data`,
on the branch `feat/core-data-service`. `main` stays untouched (S21).

### L9 — The slice boundary and the delivery strategy

The feature is bigger than one review, so the branch is divided into review
slices, and the boundary of each one is recorded in the table above. The
commits of each slice are:

| Slice | Commits | What it carries |
|-------|---------|-----------------|
| 1 | `15fa3ac`, `22d8f7a` | The specification, the module of the core, the gate and the model of the user |
| 2 | `482a9ab`, `3dc1cb6`, `03cd28a` | The table, the generated row model, the postgres adapter and the single writer of the schema |
| 3 | `781c7a3`, `74a5c2b`, `10b977c` | The wiring into the MCP server, the defect the container tests found, and the container tests that prove the rules of the table |
| 4 | `8ac8154`, `3770c7b`, `805afde`, `71e58f1` | The documentation of the core, the boundary of the slices, the evidence and the issue and the pull request |

The slices are cut as stacked branches only when a stack of pull requests is
wanted; the whole feature is one pull request otherwise, as the slices of
the MCP server were.

The history of the branch was rewritten once, after the container tests were
written: a commit of the documentation had captured a copy of the test files
that was still being written, and that commit did not compile. The commits
above are the ones on the branch now, and every one of them builds.

### L10 — What the implementation found

- **sqlc and goose read the same files.** The migration carries the goose
  annotations and the two directions of the migration, and sqlc reads the
  very same directory as its schema. The `DROP TABLE` of the down section
  does not remove the table from the catalog of the generator, so the schema
  and the row model cannot drift apart. The tool was proved on a scratch
  directory before the module was written.
- **sqlc v1.31.1 writes nothing without a query.** The generator refuses a
  configuration with no queries at all. The suite carries one query that is
  not CRUD: the `Ping` of the health check, which touches no table.
- **`internal` does not cross the boundary of a module.** The generated code
  lives in `internal/db` of the core, where it belongs, and the public
  surface of the core is curated by hand (S9).
- **The coverage of the core is in two numbers.** The gate measures the
  packages whose rules can be asserted without a database; the migrator and
  its command are covered by the container tests, which report their own
  coverage. The generated code is not measured at all.
- **The image of the MCP server is built from the root of the repository.**
  The server imports the core, the core is a module of its own, and the
  local dependency of the server on it resolves outside the context of the
  old build. The sources inside the image mirror the layout of the
  repository, so the same paths hold in both places. The image was built
  and the binary it holds came out of the new context.

### L11 — The decisions the implementation took

- The identifier column is a `bigint`: the document number comes from
  outside the system, and a ten-digit cédula should never meet the ceiling
  of a 32-bit integer (S17).
- The database of the MCP server is **optional** for now. The server starts
  without one, exactly as it did before, and the first tool that reads the
  database makes it required. The health report carries the `database` check
  when the service names one.
- The credentials of the database stay with the stack that owns them. The
  stack of the MCP server does not name a database yet, so it does not need
  the credentials of the stack of the database.
- The composition root opens the pool, and the health use case receives the
  checkers it watches. The domain of the service never learns that a
  database exists.

### L12 — What is out of scope of this slice

- The gRPC facade and the reserved port 50051 (S15).
- **The CRUD of the `users` table** (S20). No create, no read, no update, no
  delete, and no repository method: the repository port arrives with the
  slice that adds the first operation.
- Any MCP tool, REST endpoint or worker that reads the database. The slice
  delivers the core and its proof; the first consumer arrives after it.
- Authentication, TLS, row-level security and multi-tenancy.

### L13 — Three guards that are deliberately not in the table

The request describes the table, and the table is implemented as it is
described. These three guards are **not** in it, and each one is a decision
the user may want to take in a later slice:

- **A positive-value check on the identifier.** A document ID of zero or less
  is meaningless, and the column would accept it today.
- **A guard against the empty string.** Every optional text column accepts
  `''` today, so an empty email and a missing email are two different facts.
- **A uniqueness rule on the email.** The request makes only the primary key
  unique. Two people may share an email today, and the database will not stop
  them.

### L14 — The defect the container tests found

The suite could not reach the database: the runner of the migrations looks
for them at the root of the files it receives, and the migrator handed it the
whole embedded filesystem, whose files live in a directory. The migrator
found no migration at all, and the schema was never written. The defect was
in the code of the core, not in the tests: the directory becomes the root
now (`74a5c2b`).

### L15 — What the slice proves

The gate of the monorepository is green: the format check, the analyzers of
the toolchain, `golangci-lint` with `gosec`, the build, the test suite, the
race detector, the coverage gate and `govulncheck` (**no vulnerabilities**)
for the MCP server (**84.7%**) and for the core (**96.9%**).

The container tests of the core start a real PostgreSQL, apply the
migrations through the single writer, and prove the thirteen rules of the
`users` table against it: **14 tests and 20 subtests, all passing**.

The image of the MCP server was built from the root of the monorepository,
the new context, and the binary it holds came out of that build.

### L16 — The issue and the pull request of the slice

- **Issue:** #24 — "feat(core): the core of the monorepository — the shared
  models, the schema and the first table".
- **Pull request:** #25 — the branch `feat/core-data-service`, the label
  `type:feature`, and the body closes #24.
