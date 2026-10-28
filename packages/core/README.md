# The core of the PACA monorepository.

The core holds what the services of the monorepository share: the models of
the domain, the contracts of the repositories and the migrations of the
schema. The MCP server and the database admin import it today; the worker
and the REST backend will import it tomorrow.

It is a **library**. It publishes no port, it runs no container, and it
starts no process. A service maps its own environment onto the configuration
of the core, opens the pool and uses the models.

---

## The layout

```
packages/core/
  core.go               the migrator: the single writer of the schema
  domain/user/          the model of the person: the identifier, the states
                        and the port of the storage
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
carries the schema and the rules of the people.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, never generated                        |
| `full_name`           | `text`       | Optional                                                     |
| `status`              | `text`       | Not null, default `onboarding`, checked against four states  |
| `status_observations` | `text`       | The reason of the closed states; the open states carry none   |
| `phone`               | `text`       | Optional, unique, digits only with the country code          |
| `email`               | `text`       | Optional, unique without case, the shape of an address       |
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

**The observation belongs to the status.** The two closed states —
`inactive` and `blocked` — carry the reason why the person is not using the
product, and the two open states carry none. A CHECK on the table holds the
rule, so no writer breaks it.

**The optional columns are NULL when the value is not known.** The domain
uses the empty string for the same fact, and the core never writes an empty
string to the database.

**The email and the phone are unique when they are set.** The indexes are
partial: a person may have no email and no phone, and two people without an
email do not collide. The email is compared without case, because
`Foo@x.com` and `foo@x.com` are the same mailbox.

**The formats are checked by the database too.** The phone holds digits only
and carries its country code — eleven to fifteen digits — and the email
holds the shape of an address.

**The timestamps carry the time of the system.** The database fills them on
insert. There is no trigger: whoever updates a row is responsible for
`updated_at` after that — and the core writes `updated_at = now()` inside
the statement of the update.

## The rules of the person

The domain package holds the rules that the table cannot.

**The phone is cleaned the way a person writes it.** The plus of the
country code, the spaces, the dashes, the dots and the parentheses go away,
and what is left are the digits: `+57 321-253-9984` is stored as
`573212539984`. A number that carries no country code is refused: ten
digits are exactly the length of a Colombian number, and that is the number
that must be written with its code.

**The email is checked** and trimmed: an address that does not look like an
address is refused, and the same rule is a CHECK of the table.

**The document ID keeps its digits.** A person writes `1.234.567` and
`1234567`, and both are the same document.

**The states move through a map.** The map of the transitions lives in the
domain, and it is the only place that knows it:

```
onboarding ──▶ active      active     ──▶ inactive   inactive   ──▶ active
onboarding ──▶ inactive    active     ──▶ blocked    inactive   ──▶ blocked
onboarding ──▶ blocked     blocked    ──▶ active     blocked    ──▶ inactive
```

No move leads back to the onboarding: that state is the entry of the
machine, the state of a person that has just been created, and it is never
a destination.

## The repository of the people

The core publishes the port of the storage of the people, and the adapter
that writes it:

```go
Create(ctx, id, details) (user.User, error)
GetByID(ctx, id) (user.User, error)
UpdateDetails(ctx, id, details) (user.User, error)
ChangeStatus(ctx, id, status, observation) (user.User, error)
```

There is no delete and no list: a person that must go is `blocked`, and
nothing consumes a list yet.

**The writes are direct.** The adapter does not read a row to find out
whether it may write it: the insert tells the truth with its primary key,
and the update tells the truth with the row it touched. A document that is
already registered is `ErrAlreadyExists`, an email that belongs to somebody
else is `ErrEmailTaken`, and a document that does not exist is
`ErrNotFound`.

**A field that is empty is a field that does not change.** The details of a
person carry the values that are set; the statement writes them with
`COALESCE`, and the ones that are absent keep the value they had.

**The change of the status is the only place the adapter reads before it
writes.** The map of the states lives in the domain, and the map needs to
know the state the user is in. The statement also requires that state, so a
user that moved at the same time is reported instead of written over.

## The migrations

The schema has **one writer**. In the container it is the `db-admin` service
(`apps/db-admin/`): it applies the pending migrations when it starts, and
nothing else writes the schema. From a machine, the writer is the command
`db-migrate`.

The other services never migrate on boot: a service that migrates on boot
races with the other replicas of itself. The writer is the exception, and
the lock of the database is what keeps it safe.

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
the domain. The queries of the people live in `queries/users.sql`, and they
are one statement per operation.

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
without a database: the domain and the adapter. The parts that talk to the
database — the migrator, its command and the file of the repository — are
covered by the container tests, which report their own coverage. The
generated code is not measured at all: it is written by the generator, never
by hand.

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
the timestamps that the system fills, the email and the phone that belong
to one person only, and the observation that belongs to the closed states.

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
