# The database library of the PACA monorepository.

The library holds the implementation of the persistence: the pool of
connections, the queries, the migrations of the schema and the repositories
that write the rows. The MCP server imports it today; the worker and the
REST backend will import it tomorrow.

It is a **library**. It publishes no port on the network, it runs no
container, and it starts no process. A service maps its own environment
onto the configuration of the library, opens the pool and uses the
repositories.

The models of the domain — what a person is, and the rules a person
carries — live in another module, `packages/domain`. This one depends on
it, never the other way around.

---

## The layout

```
packages/database/
  database.go         the migrator: the single writer of the schema. It
                      speaks in the model of the domain, so the service and
                      the library share one record of a migration
  postgres/           the adapter that talks to PostgreSQL: the pool, the
                      configuration and the repositories
  config/             the assembly of the connection string, shared by the
                      services
  internal/db/        the generated code: the row model and the queries
  queries/            the SQL of the queries, read by the generator
  migrations/         the history of the schema
  test/integration/   the container tests, against a real PostgreSQL
```

The dependency rule of the hexagonal architecture holds here, and the
toolchain enforces it: the module depends on `packages/domain`, and
`packages/domain` imports nothing but the standard library. The domain
cannot reach the generated code of the queries, because the visibility rule
of `internal` does not cross the boundary of a module.

The generated code lives in `internal/db`: it belongs to this module alone.
The public surface of the library is curated by hand, because the services
are other modules.

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
| `email`               | `text`       | Optional, unique without case, the shape of an address, canonical |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The identifier is the national document ID of the person.** The caller
supplies it, and nothing in the library generates one: the number is not
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
uses the empty string for the same fact, and the library never writes an
empty string to the database.

**The email and the phone are unique when they are set.** The indexes are
partial: a person may have no email and no phone, and two people without an
email do not collide. The email is compared without case, because
`Foo@x.com` and `foo@x.com` are the same mailbox.

**The email is stored in its canonical form.** The domain owns that rule: it
writes the lower case of the address, and the constraint of the table refuses
an address that is not in it. The migration of the identity brought the rows
that existed to that form.

**The formats are checked by the database too.** The phone holds digits only
and carries its country code — eleven to fifteen digits — and the email
holds the shape of an address.

**The timestamps carry the time of the system.** The database fills them on
insert. There is no trigger: whoever updates a row is responsible for
`updated_at` after that — and the library writes `updated_at = now()`
inside the statement of the update.

The rules that a table cannot hold — the cleaning of the phone, the shape
of the email, the map of the states — live in the module of the domain, and
the domain is the one that refuses them.

## The repository of the people

The adapter writes the people through the port of the domain
(`user.Repository`):

```go
Create(ctx, id, details) (user.User, error)
UpdateDetails(ctx, id, details) (user.User, error)
ChangeStatus(ctx, id, status, observation) (user.User, error)
```

There is no delete and no list: a person that must go is `blocked`, and
nothing consumes a list yet. There is no read either: the port holds the
methods the use cases call, and none of them asks for a person by its
document. The adapter reads the row it needs by itself, when the move of a
state needs it.

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

The schema has **one writer**: the `db-admin` service (`apps/db-admin/`). It
applies the pending migrations when it starts, and nothing else writes the
schema — not a service, not a command.

The other services never migrate on boot: a service that migrates on boot
races with the other replicas of itself. The writer is the exception, and
the lock of the database is what keeps it safe.

```sh
make stack-up          # the writer applies the schema when it starts
make stack-schema      # assert that the schema was written by its owner
```

The migrator takes the lock of the database while it applies a migration,
so two migrators that run at the same time take turns instead of writing
the same table twice.

The migration files are embedded in the binary: the file that creates a
table is the file the generator reads, so the schema and the row model
cannot drift apart.

## The generated code

The schema is the source of truth and the SQL is written by hand. `sqlc`
reads the migrations, reads the queries and writes the row model and the
queries themselves.

```sh
make -C packages/database generate          # write the generated code
make -C packages/database generate-check    # fail when it is not what the schema describes
```

The gate of the module regenerates the code and fails when it differs from
the committed code, so a change of the schema never reaches the repository
without the code that goes with it.

The generator needs one query to write the code of the connection: the
`Ping` of the health check. It touches no table, and it is not a query of
the domain. The queries of the people live in `queries/users.sql`, and they
are one statement per operation.

## The configuration of a service

The library reads no environment variable. A service maps its own environment
onto the configuration of the library, so the same library serves every
service with its own settings.

The services that name their database with the credentials of the
PostgreSQL stack build the connection string in one place, in
`packages/database/config`. The package is pure: it reads no variable, no
file and no clock, it takes the credentials and returns the string. Which
variable wins, which default applies and which value is invalid stay in
the service, because every service states them in its own words and in its
own tests.

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
make -C packages/database all               # the full quality gate
make -C packages/database test-integration  # the container tests (Docker)
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

The tools are pinned in the repository (`tools/go.mod`), so every machine
and every container uses the same versions.

### The coverage, in two numbers

The gate of the module measures the packages whose rules can be asserted
without a database: the adapter. The parts that talk to the database — the
migrator and the file of the repository — are covered by the container
tests, which report their own coverage. The generated code is not measured
at all: it is written by the generator, never by hand.

## The tests

The unit tests live with the code they cover. The container tests live in
`test/integration` and need Docker:

```sh
make -C packages/database test-integration
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
