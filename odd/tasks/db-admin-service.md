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

### S11 — `make all` is the gate of the repository, and it works from nothing

> before merging, ensure the make all in the repository.

`make all` is the one command that proves the repository. It runs every
check the continuous integration runs, and it leaves no container behind.

*(decided)* The rules:

- It works on a machine with nothing: no image, no container, no volume and
  no `.env`. It writes the environment it needs, it starts what it must
  smoke, and it stops what it started.
- It runs the quality gate of every application and every package, the
  container tests of the packages, the container tests of the applications,
  the shellcheck of the scripts, the tests of the installer of the
  environment, the smoke test of every database and the smoke test of the
  collector of the traces.
- A step that fails stops the gate, and the exit code says so.

### S12 — The migration runs when the stack starts, from nothing

> ensure that when starting the docker compose the migration run properly

`docker compose up --wait` on a machine with no image, no container and no
volume brings the whole stack up healthy, and the schema of the database is
written by the service that owns it.

*(decided)* The rules:

- The writer **waits** for the database instead of exiting: the service
  stays up while the database is still starting, and it writes the schema as
  soon as the database answers. The container is not restarted in the
  meantime, and the stack comes up healthy the first time.
- The budget of the wait is a setting of the service:
  `PACA_DB_ADMIN_DATABASE_WAIT_TIMEOUT`, 60 seconds by default. An invalid
  value is an error, never a silent fallback.
- The health check of the container tolerates the wait: the container is not
  declared unhealthy while the service waits for its database.
- The rule is proven continuously: the continuous integration starts the
  whole stack from nothing and asserts that the schema was written.

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
| T9  | S11 | inline | `37ac2ce` | The gate of the repository: `make all` runs every check of the continuous integration, from nothing, and leaves no container behind. The evidence is in L14. |
| T10 | S12 | worker | `5e29199` | The writer waits for the database: the service retries it until the database answers, the container tolerates the wait, and the stack comes up healthy the first time. The evidence is in L14. |

The commit of every task is recorded in the Log.

### The size of the slice

The slice carries 17 commits and 7,688 changed lines. The generic guidance
of the harness is a slice of about 400 authored changed lines per commit;
the practice of this repository is one pull request per feature slice, with
the work-unit commits as the review path (PRs #25, #27 and #29), and the
deviation is stated here as it was stated in `odd/tasks/env-init-script.md`.

The two commits of the last round are the boundary of the slice: `5e29199`
(the writer waits for the database, S12) and `37ac2ce` (the gate of the
repository, S11). They are the last work units of the feature: the slice
ends with them, and the pull request #31 carries them as its last two
commits.

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

- **L10** — the independent verification round (T8), and what it changed.

  The verifier was an adversary with the job of finding what the author
  missed. It mutated the code in a scratch copy and checked that a test
  fails when the behaviour is broken. It found:

  - **A regression the author had not seen.** Half of the credentials of the
    database in the environment of the MCP server made the server stop with
    an error. Before the slice, the same environment started the server
    without a database: the guarantee "a server with no database configured
    starts without one" was broken by a change nobody asked for. Fixed: half
    credentials are ignored, the server runs without a database and says so
    in its log. The test now pins the old behaviour.
  - **A rule without a test.** The MCP server did not guard the rule that
    the password of the database keeps its spaces. The rule exists in the
    db-admin and it is now in the MCP server too.
  - **Rules that only the container tests covered**: the service applies the
    migrations when it starts, it carries exactly one check, and it keeps
    serving. The container tests were never in the continuous integration.
    They are now: the root Makefile gained the target `apps-integration`,
    and the workflow runs it.
  - **A test that proved nothing**: the test of the migrator asserted that
    the adapter was built, and not that it writes through the core. It now
    fails when the adapter stops calling the core, and the migration use
    case counts the calls, so "the migrations are applied once" is proven.
  - **Two broken container tests of the MCP server**, older than this slice
    and never run by the workflow: the image was built with the wrong
    context (the Dockerfile of the server expects the root of the
    monorepository), and the protocol test still expected a server without
    tools. Both are fixed, so the target of the continuous integration is
    green.

  The gates after the fixes: `make gate` green (the MCP server at 83.7% of
  coverage, the db-admin at 90.5%, the core at 96.8%, no vulnerability),
  `make shellcheck` green, `make apps-integration` green.

- **L11** — the request of the user, verbatim, before the merge of the slice:

  > before merging, ensure the make all in the repository. Clean the all
  > docker images and volumes and containers and build from zero and ensure
  > that when starting the docker compose the migration run properly

- **L12** — the machine was wiped before the verification: every container,
  every image, every volume and the build cache of Docker were removed
  (`docker rm -f`, `docker rmi -f`, `docker volume rm -f`, `docker network
  prune -f`, `docker builder prune -af`). The machine came back with 0
  containers, 0 images, 0 volumes and 0 bytes of build cache. The volume
  `witsaba-local-home-mac_witsaba-postgres-data`, which belongs to another
  project of the machine, was removed as well: the user chose the full
  wipe over the scoped one.

- **L13** — the verification from zero, and the two defects it found.

  **Defect 1 — `make all` did not work from a clean state.** The target
  `all` of the root `Makefile` was `gate shellcheck db-smoke`: it ran the
  smoke test of the database against a database it never started. On a
  machine where the database was not running, `make all` ended with:

  ```
  == container state ==
    FAIL the postgres container is not running (run 'make -C db/postgres up' first)
  make[2]: *** [smoke] Error 1
  make[1]: *** [smoke] Error 1
  make: *** [db-smoke] Error 2
  ```

  The continuous integration never saw it: the workflow runs `make db-up`
  before `make db-smoke`, in a job of its own. The target `all` — the one
  command a person runs to prove the repository — was broken, and the
  merge was about to ship it broken. Fixed by S11 and T9.

  **Defect 2 — the stack did not come up from nothing, and the migration
  did not run.** With the whole machine wiped, `docker compose up -d
  --build --wait` **failed with the exit code 1**:

  ```
  container paca-db-admin is unhealthy
  ```

  The cause was a race, and it was in the service and not in the compose
  file. The database accepts connections about two seconds after its
  container starts, and the writer started at the same moment as the
  database: it asked the database once, and the answer was `connection
  refused`:

  ```
  paca-db-admin: the database did not answer: failed to connect to
  `user=paca database=paca`: 172.19.0.3:5432 (postgres): dial error:
  dial tcp 172.19.0.3:5432: connect: connection refused
  ```

  The service exited, the `restart: unless-stopped` policy of the container
  brought it back, and it restarted **three** times before it won the race.
  The health check of the container declared it unhealthy in the meantime,
  so `--wait` gave up and the whole stack never came up. The migration was
  applied on the fourth attempt, and the schema was correct — but the
  command a person runs to start the stack failed, and the way it failed
  was a race, not a rule.

  The rule the writer obeys is now S12: it waits for the database instead
  of asking it once, the container tolerates the wait, and the stack comes
  up healthy the first time.

  The evidence of the fix is in L14.

- **L14** — the fix of the two defects, and the proof from nothing.

  **The writer waits for the database (S12, T10, `5e29199`).** The wait
  is an adapter of the database — `apps/db-admin/internal/adapters/outbound/database/waiter.go`
  — and the package is under the coverage gate, so the rule is proven by
  unit tests that never sleep: the writer asks again until the database
  answers, it gives up when the budget of the wait is spent, and it stops
  when the service is cancelled. The budget of the wait is a setting of
  the service, `PACA_DB_ADMIN_DATABASE_WAIT_TIMEOUT` (60 seconds by
  default), and an invalid value is an error, never a silent fallback. The
  start period of the health check of the container is 90 seconds, so the
  container is not declared unhealthy while the writer waits.

  **The gate of the repository (S11, T9, `37ac2ce`).** `make all` runs
  every check of the continuous integration from nothing, and it stops the
  containers it started when a step fails: the whole recipe is one shell,
  so a trap does it. The new targets `stack-up`, `stack-schema` and
  `stack-down` bring the whole project up, assert that the schema of the
  database was written by the service that owns it, and stop it. The
  assertion also proves that the writer was **not** restarted.

  **The proof, from a machine with nothing.** The Docker state was wiped
  again (0 containers, 0 images, 0 volumes, 0 bytes of build cache), and
  the stack was brought up from there:

  ```
  $ docker compose up -d --build --wait
  ...
  Container paca-db-admin Healthy
  Container paca-mcp-server Healthy
  Container paca-postgres Healthy
  Container paca-jaeger Healthy
  === EXIT: 0 ===
  ```

  The **first** attempt, with no retry of the command and no manual
  intervention. The log of the writer says what happened, and it is the
  wait and not the race:

  ```
  {"msg":"the service starts"}
  {"msg":"the database did not answer, the writer waits for it",
   "error":"the database did not answer: ... connect: connection refused",
   "pause":1000000000}
  {"msg":"the migration was applied","version":1,"name":"00001_users.sql"}
  {"msg":"the migration was applied","version":2,"name":"00002_users_guards.sql"}
  {"msg":"the schema is up to date","version":2}
  {"msg":"the database admin is serving","addr":"[::]:8082"}
  ```

  One refusal, one pause, one success. The container was **not** restarted:
  `RestartCount` is 0, where it was 3 before the fix. The schema is in
  place: `goose_db_version` carries the three rows of the versions and the
  table `users` exists.

  The gate of the repository was then run against the same zero machine,
  and it was green from the first step to the last one:

  ```
  $ make all
  ==> the environment of the stacks
  ==> the quality gate
  ==> the container tests of the packages
  ==> the container tests of the applications
  ==> the shellcheck of the scripts
  ==> the tests of the installer of the environment
  ==> the database            (up, smoke, down)
  ==> the collector of the traces
  ==> the whole stack         (up, schema, down)
  All the steps of the gate passed.
  === EXIT: 0 ===
  ```

  The gate left no container behind, and the assertion of the schema was
  green inside it:

  ```
  == the writer of the schema ==
    ok   the restart count of paca-db-admin = 0
  == the schema ==
    ok   the applied migrations = 3
    ok   the migrations that are not applied = 0
    ok   the maximum applied version = 2
    ok   the table users = 1
  ```

  **A third defect, found by the gate.** The smoke test of the database
  asserted the isolation of the stack with the role and the database of
  the manifest hardcoded (`-U paca -d paca`), while every other assertion
  of the same script reads the credentials from inside the container. The
  `.env` of the machine is not committed and names the database
  `paca-pg`, so the assertion failed against a stack that was configured
  correctly. The script reads the role and the database from the container
  now, the way it reads the password. It was a latent defect of the
  repository: it never showed itself because the default of the manifest
  and the name of the database of the machine were the same.

