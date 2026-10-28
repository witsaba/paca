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
| T1   | Delete the `db-migrate` command and every reference to it (Makefile, README, coverage lists)   | S3      | inline  | _pending_ |
| T2   | Create the `packages/domain` module: the model of the user and the port of the storage          | S2, S5  | inline  | _pending_ |
| T3   | Create the `packages/database` module: the pool, the queries, the migrations, the migrator, the repository | S2, S5 | inline | _pending_ |
| T4   | Point the two applications at the two new modules                                               | S2, S6  | worker  | _pending_ |
| T5   | Move the assembly of the connection string into the database library                           | S7      | inline  | _pending_ |
| T6   | Wire the build: `go.work`, the root `Makefile`, the continuous integration, the documentation   | S2, S5  | inline  | _pending_ |
| T7   | The differential check and the gate of the repository                                           | S6      | verifier | _pending_ |

Each task is one work unit: one commit, with its tests and its documentation
alongside the behavior it changes.

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
