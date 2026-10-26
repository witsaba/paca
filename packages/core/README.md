# The core of the PACA monorepository.

The core holds what the services of the monorepository share: the models of
the domain, the contracts of the repositories and the migrations of the
schema. The MCP server imports it today; the worker and the REST backend
will import it tomorrow.

It is a **library**. It publishes no port, it runs no container, and it
starts no process. A service maps its own environment onto the configuration
of the core, opens the pool and uses the models.

---

## The layout

```
packages/core/
  core.go               the migrator: the single writer of the schema
  domain/user/          the model of the person: the identifier and the states
  postgres/             the adapter that talks to PostgreSQL
  internal/db/          the generated code: the row model and the queries
  queries/              the SQL of the queries, read by the generator
  migrations/           the history of the schema
  cmd/db-migrate/       the command that applies the migrations
```

The dependency rule of the hexagonal architecture holds here too: the domain
knows nothing about the database, the adapter knows the models and nothing
about the services, and only a service knows the core.

The generated code lives in `internal/db`: it belongs to the core alone. The
public surface of the core is curated by hand, because the visibility rule
of `internal` does not cross the boundary of a module — the services are
other modules.

## The tables

### `users`

The person that PACA knows. It is the first table of the repository, and it
arrives before its CRUD: the schema is right, and the operations come later.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, never generated                        |
| `full_name`           | `text`       | Optional                                                     |
| `status`              | `text`       | Not null, default `onboarding`, checked against four states  |
| `status_observations` | `text`       | Optional                                                     |
| `phone`               | `text`       | Optional                                                     |
| `email`               | `text`       | Optional                                                     |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The identifier is the national document ID of the person.** The caller
supplies it, and nothing in the core generates one: the number is not
sequential because it is not a counter, it is the number the person already
carries. The column is a `bigint`, because a ten-digit document number
should never meet the ceiling of a 32-bit integer.

**The status is checked by the database**, against the four states of the
domain: `onboarding`, `active`, `inactive` and `blocked`. A writer that
does not know the states cannot store one.

**The optional columns are NULL when the value is not known.** The domain
uses the empty string for the same fact, and the core never writes an empty
string to the database.

**The timestamps carry the time of the system.** The database fills them on
insert. There is no trigger: whoever updates a row is responsible for
`updated_at` after that.

## The migrations

The schema has **one writer**, and it is the command `db-migrate`. A service
never migrates on boot: a service that migrates on boot races with the other
replicas of itself.

```sh
make -C packages/core db-migrate      # apply the pending migrations
make -C packages/core db-migrate -status   # print the state, change nothing
```

The command names the database through `PACA_DATABASE_URL` and refuses to
run without it, rather than guessing a database. The migrator takes the
lock of the database while it applies a migration, so two migrators that
run at the same time take turns instead of writing the same table twice.

The migration files are embedded in the binary: the file that creates a
table is the file the generator reads, so the schema and the row model
cannot drift apart.

## The generated code

The schema is the source of truth and the SQL is written by hand. `sqlc`
reads the migrations, reads the queries and writes the row model and the
code of the queries.

```sh
make -C packages/core generate          # write the generated code
make -C packages/core generate-check    # fail when it is not what the schema describes
```

The gate of the module regenerates the code and fails when it differs from
the committed code, so a change of the schema never reaches the repository
without the code that goes with it.

The generator needs one query to write the code of the connection: the
`Ping` of the health check. It touches no table, and it is not a query of
the domain.

## The configuration of a service

The core reads no environment variable. A service maps its own environment
onto the configuration of the core, so the same core serves every service
with its own settings.

| Variable                       | Default | What it is                                          |
|--------------------------------|---------|-----------------------------------------------------|
| `PACA_DATABASE_URL`            | *empty* | The connection string of the database.               |
| `PACA_DATABASE_MAX_CONNS`      | `4`     | The number of connections the pool may open.         |
| `PACA_DATABASE_CONNECT_TIMEOUT`| `5s`    | The bound of the attempt to open a connection.       |

The database is optional: a service that does not name one starts without
it. The pool is lazy, so a database that is down does not stop a service
from starting — the health check is what asks the database whether it
answers.

## Running it

```sh
make -C packages/core all               # the full quality gate
make -C packages/core test-integration  # the container tests (Docker)
```

## The quality gate

| Target            | What it does                                                     |
|-------------------|------------------------------------------------------------------|
| `make deps`       | Downloads and verifies the dependencies.                          |
| `make tidy-check` | Fails when `go.mod` is not tidy.                                 |
| `make fmt-check`  | Fails when a source is not formatted.                             |
| `make vet`        | Runs the analyzers of the toolchain.                              |
| `make lint`       | Runs `golangci-lint`, with `gosec` among the linters.             |
| `make build`      | Builds every package of the module.                               |
| `make test`       | Runs the test suite.                                              |
| `make test-integration` | Starts a PostgreSQL with the schema and asserts the rules of the tables against it. |
| `make cover`      | Measures the coverage and **fails below 80%**.                    |
| `make race`       | Runs the test suite with the race detector.                      |
| `make bench`      | Runs the benchmarks.                                              |
| `make perf`       | Checks the allocation budgets, then runs the benchmarks.          |
| `make vuln`       | Runs `govulncheck` over the dependencies.                         |
| `make generate-check` | Fails when the generated code is not what the schema describes. |
| `make db-migrate` | Applies the migrations to the database.                           |

The tools are pinned in the repository (`tools/go.mod`), so every machine
and every container uses the same versions.

### The coverage, in two numbers

The gate of the module measures the packages whose rules can be asserted
without a database: the domain and the adapter. The packages that talk to
the database — the migrator and its command — are covered by the container
tests, which report their own coverage. The generated code is not measured
at all: it is written by the generator, never by hand.

## The tests

The unit tests live with the code they cover. The container tests live in
`test/integration` and need Docker:

```sh
make -C packages/core test-integration
```

They start a PostgreSQL, apply the migrations through the single writer, and
assert every rule of the tables against the real database: the primary key
that refuses a duplicate, the identifier that the database never invents,
the status that the database checks, the optional columns that hold NULL,
and the timestamps that the system fills.

They run in the workflow of the continuous integration, next to the gate of
the applications.

## The versions

| What             | Version  | Why                                                     |
|------------------|----------|---------------------------------------------------------|
| Go               | 1.27.3   | The latest security release of the 1.27 line.            |
| pgx              | v5.11.0  | The driver of PostgreSQL, and the pool of the core.      |
| sqlc             | v1.31.1  | The generator of the row model and of the queries.       |
| goose            | v3.29.0  | The runner of the migrations.                            |
| testcontainers-go| v0.44.0  | The container tests.                                     |
| golangci-lint    | v2.15.0  | Pinned in `tools/go.mod`.                                |
| govulncheck      | v1.8.0   | Pinned in `tools/go.mod`.                                |
