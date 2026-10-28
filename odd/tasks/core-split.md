# PACA — the split of the core into a domain module and a database module

Feature document for the refactoring of the shared library of the monorepository:
`packages/core` becomes two modules — `packages/domain` (the models, the rules and
the port of the storage) and `packages/database` (the implementation: the pool, the
queries, the migrations, the migrator and the repository) — and the command
`db-migrate` is deleted, so the database admin is the only writer of the schema.

- **Project:** `packages/domain` and `packages/database` (modules
  `github.com/witsaba/paca/packages/domain` and `github.com/witsaba/paca/packages/database`).
  `packages/core` is removed.
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-core-split`, branch
  `feat/core-split`, based on `main` at `5f87399`.
- **Scope:** the shared library and the wiring of the two applications that import it.
  No change of behavior, no new table, no new endpoint, no gRPC surface, no new
  environment variable.

---

## Specs

Each spec is the user's own wording, quoted verbatim, except the ones marked
*(decided)*, which record a choice taken with the user.

### S1 — The database stuff in one place

> I wanna separate the database stuff only in a specific app.

*(decided)* The separation happens in a **library**, not in an application. Go's
`internal` rule forbids `apps/mcp-server` from importing `apps/db-admin/internal/...`,
so putting the implementation in the application would force a network boundary
between the two. The database admin keeps the ownership of the schema at runtime; the
implementation of the database lives in the library.

### S2 — SOLID and the standards of Go decide the shape

> Following a proper SOLID principles and a good standars for golang, where shall I have the database stuff in a library like the core or directly in the db-admin service?

*(decided)* Two modules, because a `go.mod` boundary is the only boundary the
compiler enforces:

- `packages/domain` — the models, the rules and the port of the storage. It imports
  nothing but the standard library.
- `packages/database` — the pool, the queries, the migrations, the migrator and the
  repository. It imports `packages/domain` and never an application.

A split into two packages of a single module would organize the code without
enforcing anything: `packages/core/internal/db` is already reachable from
`packages/core/domain/user`, because the visibility rule of `internal` stops at the
root of the module.

### S3 — The second writer of the schema is deleted

> And yeap, delete the packages/core/cmd/db-migrate.

The command `db-migrate`, its Makefile target, its documentation and its tests are
deleted. The database admin is the only writer of the schema, as
`odd/tasks/core-data-service.md` (S5) already promised.

### S4 — A new git worktree

> Open a new git worktree to working on in this feature.

### S5 — The recommendation is approved

> your recommendation sounds pretty good to me.

The four units of the recommendation are the scope of the slice: the deletion of the
command, the shared configuration of the database, the split of the module and the
differential check.

### S6 — Nothing changes for the applications *(decided)*

The observable behavior of `paca-mcp-server` and `paca-db-admin` does not change:
same stdout, same stderr, same exit code, same HTTP endpoints, same MCP tools. The
slice is a refactoring, and the proof is a differential check against the previous
code.

### S7 — The duplicated configuration of the database leaves the applications *(decided)*

The two applications assemble their connection string in their own way today
(`apps/mcp-server/internal/platform/config/config.go` and
`apps/db-admin/internal/platform/config/config.go`). The assembly moves into the
database library, so the two applications build the same string from the same
environment variables.

The library reads no environment variable: a service maps its own environment onto
it, as the core has always promised. The shared piece is the pure assembly of the
connection string; the resolution of *which* variable wins stays in each application,
because each one states it in its own words and its own tests assert those words.

---

## Tasks

| ID   | Task                                                                                          | Specs   | Route   | Commit    |
|------|-----------------------------------------------------------------------------------------------|---------|---------|-----------|
| T1   | Delete the `db-migrate` command and every reference to it (Makefile, README, coverage lists)   | S3      | inline  | `f3f29a5` |
| T2   | Create the `packages/domain` module: the model of the user and the port of the storage          | S2, S5  | inline  | `d6ea254` |
| T3   | Create the `packages/database` module: the pool, the queries, the migrations, the migrator, the repository | S2, S5 | inline | `266e48b` |
| T4   | Point the two applications at the two new modules                                               | S2, S6  | inline  | `266e48b` |
| T5   | Move the assembly of the connection string into the database library                           | S7      | worker  | `8f6efd2` |
| T6   | Wire the build: `go.work`, the root `Makefile`, the continuous integration, the documentation   | S2, S5  | inline  | `90b572d` |
| T7   | The differential check and the gate of the repository                                           | S6      | verifier | _pending_ |

Each task is one work unit: one commit, with its tests and its documentation
alongside the behavior it changes.

T4 landed with T3: the rewrite of the import paths of the two applications is
the same mechanical operation as the move of the library, and splitting it in
two commits would have produced a commit that does not build.

The feature document of the slice is `a5a2db7`.

---

## Log

### L1 — the original request, verbatim

> I recently merged the db admin and that is a very good news to me due to I wanna separate the database stuff only in a specific app. Review the @apps/mcp-server/ and check what database implementation are in there and most to be in @apps/db-admin/ show me your analyze.

### L2 — the question that decided the shape, verbatim

> Following a proper SOLID principles and a good standars for golang, where shall I have the database stuff in a library like the core or directly in the db-admin service?

### L3 — the approval and the two instructions, verbatim

> your recommendation sounds pretty good to me. And yeap, delete the packages/core/cmd/db-migrate. Open a new git worktree to working on in this feature.

### L4 — the evidence of the analysis

Read from the repository at `main` @ `5f87399`:

- `grep -rn "jackc\|goose\|sqlc" apps/mcp-server/` returns nothing outside the tests.
  The MCP server writes no row of its own: it calls
  `postgres.NewUserRepository(pool)` from the shared library.
- The implementation of the database lives in `packages/core`: `postgres/` (the pool,
  the configuration, the repository, the mapping of the rows), `internal/db/` (the code
  generated by sqlc), `queries/` (the SQL), `migrations/` (the history of the schema)
  and `core.go` (the migrator, with goose).
- The database admin only applies the schema and serves one health endpoint. Its own
  feature document (`odd/tasks/db-admin-service.md`, S4) says: *"no read or write of
  the tables of the domain."*
- Four leaks: two writers of the schema (`packages/core/cmd/db-migrate` and the
  database admin), the duplicated configuration of the database in the two
  applications, two connection stacks (`pgxpool` in the library and `database/sql` in
  the database admin, because goose needs a `*sql.DB`), and one module that mixes the
  domain with the persistence.

### L5 — the decision on the shape

The answer to L2, agreed with the user in L3: **a library, not an application**; and
**two modules, not one**, because the visibility rule of `internal` does not cross
the boundary of a module, so only a `go.mod` boundary makes *"the domain knows
nothing about the database"* a compile error instead of a comment. The database admin
keeps the ownership of the schema at runtime and imports the migrator of the
library; it does not gain a network surface, and nothing of the shared library is
thrown away if a gRPC surface is added later.

### L6 — the worktree and the branch, created before the first source write

`/Users/braejan/workspace/witsaba/repositories/paca-core-split`, branch
`feat/core-split`, based on `main` at `5f87399`. The main worktree is clean and in
sync with `origin/main`.

### L7 — the shape of the two modules, in detail

```
packages/domain/            module github.com/witsaba/paca/packages/domain
  domain.go                 the package comment: the models and the rules, nothing else
  user/                     User, ID, Details, Status, the rules, and the Repository port

packages/database/          module github.com/witsaba/paca/packages/database
  database.go               the Migrator: the writer of the schema (was packages/core/core.go)
  migrations/               the history of the schema
  queries/                  the SQL, read by the generator
  internal/db/              the code generated by sqlc, protected by the internal rule
  postgres/                 the Pool, the Config and the UserRepository
  config/                   the assembly of the connection string, shared by the services
```

`packages/domain` imports nothing but the standard library. `packages/database`
imports `packages/domain` and never an application. The applications import both.

### L8 — the writer of the slice, and the work units

The writer of T5 was a worker, with a bounded surface: the new package
`packages/database/config`, the configuration of the two services, and the
README of the library. The writer proved the RED (the package did not exist) and
the GREEN, and it proved with a scratch script that the connection string of the
library is byte-for-byte the one the two services built before.

### L9 — the defect the writer found in the work of T3

The writer reported that the lint of `packages/database` failed on three files of
the adapter: the rewrite of the import paths moved `packages/domain/user` before
`packages/database/internal/db`, and the group of the local imports was not sorted
anymore. `gofmt` refused the files, and the gate of the module failed on them.

The defect was mine, from T3, and it was fixed in `9c72c6d`: the import group is
sorted again. A fourth file, the one of the container tests, was not in the report
of the writer because the linter does not see it — it carries the build tag of the
integration tests — and `make fmt-check` runs over every file. It is sorted too.

### L10 — the documentation of the past slices is left alone

The feature documents of the past slices (`odd/tasks/core-data-service.md`,
`odd/tasks/db-admin-service.md`, `odd/tasks/user-crud-mcp.md`) name the module
`packages/core`. They are the record of what happened then, and they keep the
paths they were written with. The documentation that describes the repository
today — the root README, the README of the two services, the README of the two
libraries — says where the library and the domain live now.

### L11 — the differential check of the slice

Sixteen scenarios were run against the binaries of the previous code (the commit
`5f87399`, built in a scratch directory outside the worktree) and against the
binaries of the change, with the same environment: the stdout, the stderr and the
exit code of every one of them are identical. The scenarios cover the refusals of
the configuration (an empty connection string, a connection string of spaces, a
number of connections that is not a number, a timeout that is not a duration, a
port that is not a number, a port out of range, an empty host), the start of the
MCP server with and without a database, the refusals of the database admin (no
database, an empty connection string, a wait timeout that is not a duration, a
port that is not a number, half of the credentials), the give-up of the writer
when the database does not answer, and the health check without a service.

The script is a scratch tool and it is not committed. The timestamp of the log
line is normalized, because it is the only thing two runs of the same code never
share.

### L12 — the size of the slice

The slice is larger than the four hundred lines a work unit aims at: the two
modules, the move of the code, the wiring of the build and the documentation. The
repository's practice is one pull request per feature slice, and the commits are
the review path. The deviation is recorded here, and the maintainer decides
whether to split it.

### L13 — the defect that the container tests of the packages found

The root of the monorepository calls the container tests of every package, and
the new module of the domain brought no target to call: `make
packages-integration` stopped on it before it ran a single test.

The module talks to no database, and the rules of the domain are proven by the
unit tests. The module now declares its own answer with a `test-integration`
target that says so and succeeds, and the root keeps dispatching without naming
anything. It is fixed in `9851df4`.

### L14 — the evidence of the gate

- `make gate`: green. The format check, the check of the generated code, the
  analyzers, the linter (0 issues in every module), the build, the tests, the
  race detector, the coverage gate (96.3% for the database library, 97.2% for
  the domain) and the vulnerability scan (no vulnerability) of the two
  applications and the two libraries.
- `make packages-integration`: green. The container tests of the database library
  assert every rule of the tables against a real PostgreSQL.
- `make apps-integration`: green. The images of the two services are built from
  the root of the repository and run against a PostgreSQL.

### L15 — what the independent verification found, and what it changed

An independent verifier audited the slice against the base commit `5f87399` and
reproduced the differential check with its own harness, of thirty-two
scenarios. It found one regression and two defects that the parent had missed:

1. **A regression of medium severity.** `apps/db-admin/go.mod` carried the
   `replace` of the database library but not the one of the domain. A replace of
   a dependency is ignored, so the module could not resolve its own graph
   outside the workspace of the repository: `GOWORK=off go list -m all` tried to
   fetch the sibling from the network and failed. The build, the tests and the
   image still worked, because the workspace hides the hole. The `replace` is
   there now.

2. **Three broken links.** The README of the MCP server pointed at
   `../packages/database` and `../packages/domain`, which resolve to
   `apps/packages/...`: the file is at `apps/mcp-server/README.md`, so the path
   needs two levels up. One of the three was already broken before the slice;
   the other two came with it.

3. **Two advisories.** The comments of `packages/database/sqlc.yaml` still spoke
   of the core, and the cache of the continuous integration named a `go.sum` that
   the domain does not have — it has no dependency.

The two classes of defect are now pinned by `scripts/tests/structure-test.sh`:
every module of the repository resolves its own graph without the workspace, and
every relative link of the documentation points at a file that exists. The
script has thirty-two assertions, and it was proved to discriminate: the two
defects were reintroduced and the script failed with four failures, and they
were fixed again and the script passed.

The target is `make structure-test`, it runs in the continuous integration next
to the tests of the installer, and it is a step of `make all`.



