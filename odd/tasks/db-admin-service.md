# PACA — the database admin service

Feature document for the new backend service of the monorepository: the one
that runs the database stuff and nothing else. It applies the migrations of
the schema when it starts, it stays alive with a single health check, and it
is the container the stack waits for.

- **Project:** `apps/db-admin` (Go module `github.com/witsaba/paca/apps/db-admin`).
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-db-admin`, branch `feat/db-admin`, based on `main` at `8483384`.
- **Scope:** one new service, its stack, and the wiring that gives the database URL to it and to the MCP server. No gRPC surface yet (the user will add features later), no REST API, no read or write of the tables of the domain.

---

## Specs

Each spec is the user's own wording, quoted verbatim, except the ones marked
*(decided)*, which record a choice taken with the user.

### S1 — A new backend service for the database stuff only

> I wanna add a new feature, a new apps backend service that allows to run ONLY database stuff like the migration.

### S2 — A new git worktree

> Open a new git worktree, review the memories about it.

### S3 — The same architecture as the MCP server

> Use Golang and the same Hexagonal Architecture, Domain Driven Design and SOLID just as the current mcp one.

The service is a hexagonal application, exactly like `apps/mcp-server`:

```
apps/db-admin/
  cmd/paca-db-admin/          the binary
  internal/domain/            the models and the rules (health, schema state)
  internal/application/       the use cases (migrate, health)
  internal/ports/             the contracts: what comes in, what goes out
  internal/adapters/inbound/  the HTTP adapter (the health endpoint)
  internal/adapters/outbound/ the database adapter (the migrator, the checker)
  internal/platform/          config, logging, server, healthcheck, clock, telemetry
  internal/bootstrap/         the composition root: the only place that knows every layer
```

The dependency rule holds: the domain knows nothing about PostgreSQL, the
adapters know the ports and nothing about the services, and only the
composition root knows everything.

### S4 — It lives in the container, with a single health check, and runs the migrations

> The focus of this new backend app is to live in the container with a single health check and run the migrations.

The container runs the service, the service exposes **one** health endpoint
(`/healthz`) with **one** check — the database answers and the schema has no
pending migration — and the service applies the pending migrations when it
starts.

### S5 — The service is long running *(decided)*

> Long running, I 'll be adding some features later like some gRPC functionalities.

The service stays alive after the migrations are applied. It is not a one-shot
job: the user will add more database features to it later, and a service that
exits after the migrations cannot carry them.

The gRPC surface is **not** part of this slice. The port `50051` stays
reserved for it, and the ports layer is shaped so that a gRPC adapter can be
added later without touching the domain.

### S6 — A new stack, and the database URL for it and for the MCP server *(decided)*

> New stack + the DB URL for it and for the MCP server

Two things, in one slice:

1. `apps/db-admin/` is a stack of the monorepository, with its own compose
   file, its own image and its own Makefile, included by the root compose.
2. The MCP server — which today starts without a database, because no compose
   file sets `PACA_DATABASE_URL` — gets one, and so does the new service.

The stack does **not** declare `depends_on` between the stacks of different
files: a cross-file dependency only resolves inside the merged project, and
the stack of a service must keep running on its own.

### S7 — The name *(decided)*

> apps/db-admin

The directory is `apps/db-admin/`, the binary is `paca-db-admin`, the image is
`paca/db-admin`, and the compose service is `db-admin`.

### S8 — One writer of the schema

The core already holds the decision (Engram #4573): *the schema has exactly
one writer*. This service is that writer in the container.

The migrator of `packages/core` stays the only thing that writes the schema.
The service does not reimplement it: it uses `core.NewMigrator` through an
outbound adapter, exactly like a service uses `packages/core/postgres` for
the tables.

### S9 — Where the credentials come from *(decided)*

The credentials of the database live with the stack that owns them:
`db/postgres/.env`. They are never written twice.

The services that talk to the database load that file with `env_file`, and
their configuration is read in this order:

1. `PACA_DATABASE_URL` — the whole connection string. When it is set, it
   wins, and nothing else is read. A hosted database, or a URL built by
   hand, needs nothing else.
2. Otherwise the connection string is assembled from the credentials of the
   PostgreSQL stack: `POSTGRES_USER`, `POSTGRES_PASSWORD` and `POSTGRES_DB`
   (loaded from `db/postgres/.env` through `env_file`), with
   `PACA_DATABASE_HOST` (default `postgres`) and `PACA_DATABASE_PORT`
   (default `5432`).

The core reads no environment variable — that is its own architectural rule
("each service maps its own environment onto a config struct"), so the
mapping lives in the configuration of each service.

### S10 — The rules of the gate of the repository

The service is an application of the monorepository, so it obeys the same
rules as the MCP server:

- Its own Makefile with the same targets, and the root `Makefile` adds
  `db-admin` to `APPS`. The continuous integration never names a stack.
- A coverage gate of 80% or more, the linter, the race detector, the
  vulnerability scan, `go vet`, `gofmt`.
- The image is built from the root of the monorepository, because the service
  imports `packages/core`, which is a module of its own.
- The container runs as a non-root user, read-only, without capabilities,
  with `no-new-privileges`.
- The port is not published: the service is reachable from the containers of
  `paca_internal` only.

---

## Tasks

| ID  | Specs    | Route | Commit | What it does |
|-----|----------|-------|--------|--------------|
| T1  | S1, S3, S7 | worker | `a54bfab`, `fed2afc`, `0f9969f`, `4c5b6b8` | The module of the service: `go.mod`, the domain (`health`, `schema`), the ports, the application use cases (`migrate`, `health`) and their unit tests. |
| T2  | S3, S4, S9 | worker | `86887ec`, `4d4b74f`, `954f39d`, `7ca6122` | The platform and the composition root: `config` (with the rules of S9), `logging`, `server`, `healthcheck`, `clock`, `telemetry`, `bootstrap`, and the binary `cmd/paca-db-admin` with its `healthcheck` command. |
| T3  | S3, S4, S8 | worker | `08d7dfe`, `9923d1f` | The adapters: the inbound HTTP adapter (the health endpoint, one check) and the outbound database adapter (the migrator over `packages/core`, the checker of the database). |
| T4  | S10 | worker | `ee2ff7d`, `595ffaa` | The stack of the service: the `Dockerfile`, the `compose.yaml`, the `.dockerignore`, the `Makefile`, the `README.md`. It travels with the module it builds, so the image and the tests of the service are one unit. |
| T5  | S4, S10 | worker | `2d38595` | The container tests: the image is built, a PostgreSQL is started, the service applies the migrations, the health endpoint answers, and the schema is in place. |
| T6  | S6, S7, S10 | inline | `5c157fa` | The wiring of the monorepository: the root `Makefile` (`APPS`), the root `compose.yaml` (the include), the root `README.md` (the tables of the repository and of the ports), the workflow (the cache of the dependencies). |
| T7  | S6, S9 | inline | `43a9849` | The database URL of the MCP server: its compose file loads the credentials of the PostgreSQL stack, and its configuration assembles the connection string with the rules of S9. |
| T8  | S1–S10 | inline | — | The verification of the slice: the gate of the service, the container tests, the stack of the monorepository up and healthy, and the migrations applied by the stack. The evidence is in L8. |

The commit of every task is recorded in the Log.

### The size of the slice

The slice carries 15 commits and 6,701 changed lines. The generic guidance
of the harness is a slice of about 400 authored changed lines per commit;
the practice of this repository is one pull request per feature slice, with
the work-unit commits as the review path (PRs #25, #27 and #29), and the
deviation is stated here as it was stated in `odd/tasks/env-init-script.md`.

---

## Log

- **L1** — the original request, verbatim:

  > I wanna add a new feature, a new apps backend service that allows to run ONLY database stuff like the migration. Open a new git worktree, review the memories about it. Use Golang and the same Hexagonal Architecture, Domain Driven Design and SOLID just as the current mcp one. The focus of this new backend app is to live in the container with a single health check and run the migrations.

- **L2** — the memories reviewed before any code: Engram #4573 (the architecture of the core: one writer of the schema, sqlc + pgx + goose), #4574 (the goose provider needs `fs.Sub`), #4577 (the stack never applied the migrations: the gap this service closes), #4581 (one pull request per feature slice, the commits are the review path), #4592 (the open review of the environment variables: `PACA_DATABASE_URL` is defined nowhere).

- **L3** — the decisions taken with the user, quoted verbatim:

  > Long running, I 'll be adding some features later like some gRPC functionalities.

  > New stack + the DB URL for it and for the MCP server

  > apps/db-admin

- **L4** — the worktree: `/Users/braejan/workspace/witsaba/repositories/paca-db-admin`, branch `feat/db-admin`, based on `main` at `8483384`.

- **L5** — the mechanism of the credentials, established by experiment on the
  local Docker Compose (v5.4.0): `environment: FOO: ${BAR}` is **not**
  interpolated from the `env_file` of the same service — it is interpolated
  from the environment of the project only. The credentials of the database
  therefore travel to the services that need them through `env_file`, and
  the connection string is assembled by the service itself (S9).

- **L6** — the port of the health endpoint: `8082`. The ports `8080` and
  `50051` are already reserved by the root README for the REST backend and
  for the gRPC backend.

- **L7** — the service does not declare `depends_on` on the other stacks: a
  cross-file dependency only resolves inside the merged project, and a stack
  that must run on its own cannot depend on a stack it does not include.

- **L8** — the verification of the slice (T8), observed and not assumed:

  - `make gate` at the root of the worktree: green — the new application
    (90.5% of coverage), the MCP server (84.1%), the core (96.8%), no
    vulnerability. `make shellcheck`: green.
  - `make -C apps/db-admin test-integration`: green — 4 tests against a real
    PostgreSQL and the real image.
  - **The gap of Engram #4577 is closed, and it was proven with the schema
    dropped first.** `drop table users cascade; drop table goose_db_version;`
    left the database with *zero* tables; `docker compose up -d --build
    --wait` then brought the whole stack up healthy (postgres, mcp-server,
    db-admin, jaeger) and the log of the writer said:

    ```
    {"msg":"the migration was applied","version":1,"name":"00001_users.sql"}
    {"msg":"the migration was applied","version":2,"name":"00002_users_guards.sql"}
    {"msg":"the schema is up to date","version":2}
    ```

    `goose_db_version` carries the three rows stamped at the moment the
    stack started, and `users` is there again.
  - The health endpoint of the writer answers `{"status":"ok", ...,
    "checks":[{"name":"database","status":"ok","latency_ms":0}]}` — one
    check, as it was asked.
  - The health endpoint of the MCP server now carries a `database` check for
    the first time: the server reaches the database of the stack.
  - The chain was proven end to end: the MCP tool `onboard_user` wrote a
    person through the core, and the row was in the table
    (`1010101010 | Ada Lovelace | ada@example.com | onboarding`). The probe
    row was deleted afterwards.
  - The defect that the review of the parent found: the configuration of
    the new service trimmed the password of the role. A password that is
    trimmed is a password that is wrong. It is fixed, and the test that
    proves it (`TestLoadKeepsTheSpacesOfThePassword`) was checked against
    the old behaviour: it fails with the trim and passes without it.
  - The dead fields that the review of the parent removed: the host and the
    port of the database were carried by the configuration of the MCP
    server and read by nothing. The connection string carries them.

- **L9** — the size of the slice: 15 commits, 6,701 changed lines. The
  practice of the repository is one pull request per feature slice, with the
  commits as the review path; the deviation from the generic budget of the
  harness is stated above.
