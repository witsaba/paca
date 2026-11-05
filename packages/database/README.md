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

### `businesses`

The business that owns a book. It is the table of the Personal Business of
a person, and of the businesses that will come.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `owner_id`            | `bigint`     | Not null, the person who owns the business                    |
| `kind`                | `text`       | Not null, checked against `personal` and `business`          |
| `name`                | `text`       | Not null, never empty                                         |
| `currency`            | `text`       | Not null, default `COP`, never empty                          |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The identity of a business is generated by the database.** It is a counter
of the businesses and not the document of a person: the owner is the owner,
and it is not the key.

**A person has exactly one personal business.** The rule is in the domain,
and the table holds it as a defense in depth: the index
`businesses_one_personal_per_owner` is partial, and it refuses a second
personal business of the same owner while leaving the other kinds free. The
bakery of the future is a second row of this table, with an identity of its
own: the key space is the businesses, never the people.

### `accounts`

The accounts of a book: the cash, the bank, the salary, the food, the money
the person owes.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the account                  |
| `kind`                | `text`       | Not null, checked against the five kinds of the accounting    |
| `name`                | `text`       | Not null, never empty, unique inside the business             |
| `counterparty`        | `text`       | Optional: the other side of a debt                            |
| `due_date`            | `date`       | Optional: the day a debt falls due                            |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The name of an account is unique inside its business.** The person reads
the accounts of their book by name, and two accounts of the same business
that share a name are one account.

**The counterparty and the due date belong to a debt.** A debt is an account
that owes or is owed — a liability or an asset — and an account of any other
kind carries neither. The CHECK of the table holds the rule of the domain as
a defense in depth.

**There is no table of the loans.** A loan with a friend is a liability
account with a counterparty, and the balance of that account is what the
person still owes.

### `purchase_drafts`

The draft of a purchase: the candidate the agent proposes from what the
owner said, and the state it waits in.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the draft                    |
| `state`               | `text`       | Not null, default `proposed`, checked against the states      |
| `revision`            | `integer`    | Not null, default 1, positive                                 |
| `utterance`           | `text`       | Not null, never empty                                         |
| `lines`               | `jsonb`      | Not null, default `[]`, an array                              |
| `missing`             | `jsonb`      | Not null, default `[]`, an array                              |
| `ambiguous`           | `jsonb`      | Not null, default `[]`, an array                              |
| `inferred`            | `jsonb`      | Not null, default `[]`, an array                              |
| `day`                 | `date`       | Optional                                                      |
| `market`              | `text`       | Optional                                                      |
| `invoice`             | `text`       | Optional                                                      |
| `proposed_on`         | `date`       | Not null                                                      |
| `expires_on`          | `date`       | Not null, never before the day it was proposed                |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**A draft is a proposal, not a fact.** Its lines, what it still needs, what
it was not sure about and what Paca filled by herself travel as JSONB: they
are the reading of a sentence, and the money enters the book with the
purchase and not with the draft.

**The state is checked by the database**, against the states of the domain:
`proposed`, `confirmed`, `cancelled` and `expired`. A draft is confirmed
once, and the confirmation of a draft that changed under the eyes of the
owner is refused by the row that carries the revision she saw.

**A draft that waits can be dropped.** The row moves to the state
`cancelled` only when it carries the state of a draft that waits: a draft
that is confirmed or that is cancelled already does not wait any more, and
the row is not touched. The expiry is computed by the domain when the draft
is read — a draft that was neither confirmed nor cancelled seven days after
the day it was proposed reads as `expired` — and the table needs no job to
expire anything.

### `purchases`

The purchase: the fact of what the business bought, and the lines it
carries.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that bought                           |
| `draft_id`            | `bigint`     | Not null, the draft the purchase came from, unique            |
| `day`                 | `date`       | Not null                                                      |
| `market`              | `text`       | Optional                                                      |
| `invoice`             | `text`       | Optional                                                      |
| `utterance`           | `text`       | Not null, never empty                                         |
| `confirmed_with`      | `text`       | Not null, never empty: the word the owner used               |
| `total`               | `bigint`     | Not null, positive                                            |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

### `purchase_lines`

The lines of a purchase: what was bought, how much of it and what it cost.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `purchase_id`         | `bigint`     | Not null, the purchase the line belongs to                   |
| `position`            | `integer`    | Not null, positive, unique inside the purchase                |
| `name`                | `text`       | Not null, never empty: the name the owner said                |
| `quantity`            | `double precision` | Not null, positive                                      |
| `unit`                | `text`       | Not null, never empty                                        |
| `total`               | `bigint`     | Not null, never negative                                     |
| `unit_price`          | `bigint`     | Not null, never negative                                     |
| `ingredient_id`       | `bigint`     | Optional: the ingredient of the catalog the name belongs to   |

**A purchase never changes.** The lines are rows of their own table, and
the position of a line is the order the owner said it in.

**The name of the line is the name the owner said.** The identity of the
ingredient is optional: it is there when the catalog knows the name, and it
is empty when it does not.

### `ingredients`

The catalog of the names the business buys.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that buys it                          |
| `name`                | `text`       | Not null, never empty, unique inside the business             |
| `aliases`             | `jsonb`      | Not null, default `[]`, an array of the names she uses        |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |
| `updated_at`          | `timestamptz`| Not null, default `now()`                                     |

**The name of an ingredient is unique inside its business.** The index is
`ingredients_one_name_per_business`, and it compares the name the way a
person reads it: trimmed, and without case. The rule lives in the domain,
and the index holds it as a defense in depth.

**The aliases are the names the owner uses for the thing she buys.** They
travel as JSONB: they are a list of names, and the list is what the domain
reads back.

**The catalog of a business begins empty.** There is no seed: the first
purchase of the business creates the first name.

## The repository of the people

The adapter writes the people through the port of the domain
(`user.Repository`):

```go
Create(ctx, id, registration) (user.User, error)
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

### `ledger_entries`

The movements of the book of a business.

| Column                | Type         | Rules                                                        |
|-----------------------|--------------|--------------------------------------------------------------|
| `id`                  | `bigint`     | Primary key, not null, generated by the database              |
| `business_id`         | `bigint`     | Not null, the business that owns the book                     |
| `debit_account_id`    | `bigint`     | Not null, the account where the money lands                   |
| `credit_account_id`   | `bigint`     | Not null, the account the money comes from                    |
| `amount`              | `bigint`     | Not null, positive                                            |
| `occurred_on`         | `date`       | Not null, the day of the movement                             |
| `description`         | `text`       | Not null, never empty                                         |
| `created_at`          | `timestamptz`| Not null, default `now()`                                     |

**The amount is an integer and never a float.** The arithmetic of money does
not forgive a rounding error, and the column counts the pesos of the book.

**The two sides of a movement are not the same account,** and the
description is not empty: the CHECKs of the table hold the rules of the
domain as a defense in depth.

**The index of the book is the one the person reads it by:** the business,
the day and the identity of the movement.

## The repositories of the book

The adapter writes the book of a person through the ports of the domain
(`business.Repository` and `account.Repository`):

```go
Create(ctx, owner, business) (business.Business, error)
Find(ctx, id) (business.Business, error)
FindPersonalByOwner(ctx, owner) (business.Business, error)

Create(ctx, account) (account.Account, error)
CreateMany(ctx, accounts) ([]account.Account, error)
Find(ctx, id) (account.Account, error)
List(ctx, businessID) ([]account.Account, error)
```

The writes are direct, like the writes of the people: the identity of the
business tells the truth with its primary key, the partial unique index
tells the truth about the second personal business, and the unique index of
the name tells the truth about the account that is taken.

**The set of accounts is written in one transaction.** A book that starts
with half of its accounts is a book that lies about what the person has.

```go
Record(ctx, entry) (ledger.Entry, error)
List(ctx, businessID, from, to) ([]ledger.Entry, error)
Balances(ctx, businessID, asOf) ([]ledger.Balance, error)
Summary(ctx, businessID, from, to) (ledger.Summary, error)
```

**The balance of the accounts is a sum, and the database does it.** The money
of a person is read from the rows of the book, and the rows are the ones the
library wrote. The account that never moved is in the answer with a balance
of zero, because an account that is missing reads like an account that does
not exist.

**The money of a period is a sum of two sides, and the database does it too.**
The income is the money that landed in the accounts of an income, and the
expenses are the money that landed in the accounts of an expense. The two
sides of a movement are read apart and never joined into one row, so a
movement between two accounts of the same kind is counted once and not
twice. A period without movements reads as an empty one: zero income, zero
expenses.

## The repositories of the utterance

The adapter writes the utterance of the owner through the ports of the domain
(`purchase.Repository`, `purchase.Recorder` and `ingredient.Repository`):

```go
Save(ctx, draft) (purchase.Draft, error)
Find(ctx, id) (purchase.Draft, error)
Confirm(ctx, id, revision) (purchase.Draft, error)

Record(ctx, purchase) (purchase.Purchase, error)

List(ctx, businessID) ([]ingredient.Ingredient, error)
Create(ctx, ingredient) (ingredient.Ingredient, error)
```

**The lines of a draft travel as JSONB.** The adapter encodes them and
decodes them, and the domain reads them back through its own constructors:
the row never decides what a line is.

**The confirmation of a draft is guarded by the revision the owner saw.**
The row is only confirmed when it still carries that revision, and the
storage refuses the confirmation of a draft that changed under her eyes.

**The confirmation writes the purchase, its lines and the catalog in one
transaction.** The unit of work opens it, and every repository called
inside it joins the transaction it finds: the purchase, the lines and the
new names of the catalog are written whole or not at all.

**The name of an ingredient is unique inside its business.** The index of
the table tells the truth about the name that is taken, and the adapter
says it with `ingredient.ErrDuplicateName`.

## The transaction of the storage

The unit of work is the port of a transaction of the storage
(`store.UnitOfWork`):

```go
Do(ctx, work) error
```

The work that the caller hands over runs inside one transaction: the rows
are written whole or not at all. The caller never sees the transaction and
never commits by itself.

**The transaction travels in the context.** The repositories of the library
are bound to the pool and not to a transaction: the unit of work opens the
transaction, and every repository called inside it joins the transaction it
finds. A repository called outside one writes by itself.

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
