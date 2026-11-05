# The domain of the PACA monorepository.

The domain holds what the services of the monorepository share: the model
of a *person*, the rules a person carries, the port of the storage that keeps
them, the state of the *schema* of the database — the version it is at, the
migrations it carries and what is pending — and the model of the *health* of
a service: the states it is in, the checks it runs and the report it answers.

A model has one owner. The person lives here because more than one service
needs it, the state of the schema because two sides need it — the service
that writes the schema and the library that applies the migrations — and the
health because every service with a health endpoint speaks it. Both sides
speak the same record, and the adapter between them maps nothing.

It is a **library**. It publishes no port on the network, it runs no
container, and it starts no process.

---

## The layout

```
packages/domain/
  domain.go         the package comment: the models and the rules, nothing else
  user/             the model of the person: the identifier, the states, the
                    details, the registration and the port of the storage
  business/         the model of the business that owns a book: the identity
                    the database generates, the kinds and the port of the
                    storage
  account/          the model of the account of a book: the five kinds of the
                    accounting, the debt and the seed of the personal book
  ledger/           the model of the movements of the book: the entry, the
                    amount, the day, the balance of an account, the money of
                    the business, the period of a reading and the debt of an
                    account
  purchase/         the model of the draft of a purchase and of the purchase
                    itself: the candidate the agent proposes, the lines, the
                    measure, the money, what the draft still needs, the
                    confirmation and the port of the storage
  store/            the port of a transaction of the storage: the unit of work
  schema/           the state of the schema of the database: the version, the
                    migrations and what is pending
  health/           the model of the health of a service: the states, the
                    checks, the report and the rule that one failing
                    component makes the whole service unhealthy
```

The dependency rule of the hexagonal architecture holds here, and the
toolchain enforces it: **the module imports nothing but the standard
library.** A model that reached for a driver, for a table or for a
connection string would break the build of every service of the
monorepository — the compiler says so before a reviewer has to.

That is the reason this module is a module of its own, and not a package of
the database library: the boundary between the two is a `go.mod`, and the
visibility rule of `internal` does not cross it. A package of the same
module would organize the code, but nothing would forbid the domain from
importing the generated code of the queries.

## The model of the person

`user.User` is the person that PACA knows. It carries the identifier, the
details, the state and the timestamps of the person.

**The identifier is the national document ID of the person.** The caller
supplies it, and nothing in the domain generates one: the number is not
sequential because it is not a counter, it is the number the person already
carries. A person writes `1.234.567` and `1234567`, and both are the same
document.

## The rules of the person

The domain package holds the rules that a table cannot.

**The phone is cleaned the way a person writes it.** The plus of the
country code, the spaces, the dashes, the dots and the parentheses go away,
and what is left are the digits: `+57 321-253-9984` is stored as
`573212539984`. A number that carries no country code is refused: ten
digits are exactly the length of a Colombian number, and that is the number
that must be written with its code.

**The email is checked** and stored in its canonical form: an address that
does not look like an address is refused, and the canonical form of an
address is its lower case. Two addresses that differ only in the case of the
name are the same mailbox, and the mailbox belongs to one person, so the
domain stores one value for them and never two.

The table holds the same rules as a defense in depth: its unique index
refuses two rows that carry the same mailbox, and its constraint refuses an
address that is not in its canonical form. The table guards against the
writer that does not know the domain; the domain is still the one that
refuses.

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

**The map is published.** `user.AllStatuses()` returns the four states and
`user.AllTargets()` the states a person may reach, and `user.AllowedTargets()`
the states one state may move to. The surfaces that show the states of the
person — the schema of a tool, the instruction an agent reads — read them from
here, so they never drift from the rule.

**The observation belongs to the status.** The two closed states —
`inactive` and `blocked` — carry the reason why the person is not using the
product, and the two open states carry none.

## The repository of the people

The domain publishes the port of the storage of the people:

```go
Create(ctx, id, details) (user.User, error)
Find(ctx, id) (user.User, error)
UpdateDetails(ctx, id, details) (user.User, error)
ChangeStatus(ctx, id, status, observation) (user.User, error)
```

There is no delete and no list: a person that must go is `blocked`, and
nothing consumes a list yet.

The read is there because a use case asks for a person by its document: the
tool that reads the data of a person needs it, and a port holds the methods
the use cases call. The adapter reads the row it needs by itself, when the
move of a state needs it, and that read is the same one.

The port is what a use case needs, not the shape of a table. It lives here,
in the module of the models, because both sides of it need it: the service
that writes the people, and the adapter that reads the table. An adapter of
another storage — a gRPC client of a data service, tomorrow — implements
the same port, and no service changes.

## The model of the book

The domain of PACA is the person, and the book of that person is the second
thing it carries: the **business** that owns the money and the **accounts**
that hold it.

### The business

A business is the thing that owns a book: the Personal Business of a person,
and the businesses of their own — the bakery of that same person.

**The identity of a business is generated by the database, and the document
ID of the owner is its owner and never its key.** The owner is found through
`(owner_id, kind = 'personal')`, and a person may own more than one
business: the key space is the businesses, never the people. The port of the
storage offers `ListByOwner`, which reads every business of a person in the
order of their identity.

**A person has exactly one personal business.** The domain states the rule
in `FindPersonalByOwner`, and the storage holds it as a defense in depth: the
partial unique index of the table refuses a second personal business of the
same owner, and it leaves the other kinds free.

The kinds are `personal` and `business`. The second one is in the vocabulary
of the domain, and the use case of the registration opens one: the name of a
business is taken inside the businesses of its owner, and the domain holds
that rule.

### The account

An account is a bucket of money inside the book of a business. The five kinds
are the five of the accounting, and the kind says what the account means: an
asset is what the person has, a liability is what the person owes, an income
is what the person earns, an expense is what the person spends, and an
equity is what the person put into the book of their own business.

**The name of an account is unique inside its business**, because the person
reads the accounts of their book by name.

**A debt is an account, and never a table of its own.** A loan with a friend
is a liability account with a counterparty and a due date, and the domain
refuses the pair of values that breaks that rule: an account of any other
kind carries neither.

**The domain owns the seeds of the books.** `DefaultPersonalAccounts`
returns the accounts that the Personal Business of a person starts with, and
`DefaultBusinessAccounts` returns the accounts that a business of the kind
`business` starts with: the cash, the bank, the inventory, the sales, the
cost of the sales, the labor, the overhead, the waste and the capital. The
contract the agent reads is derived from them — the list of the accounts
and the kinds of an account are read from the domain and never written a
second time.

## The movements of the book

The `ledger` package holds the money of a person: the movements that fill the
book and the balance that comes out of it.

**Every movement is money that goes from the credit account to the debit
account.** The money lands where it is debited and it comes from where it is
credited. The amount is positive, the two sides are not the same account, and
the description is not empty: a movement the person cannot read is a
movement they cannot recognise.

**The amount is an integer and never a float.** The arithmetic of money does
not forgive a rounding error, and the person reads the numbers that the book
carries. It counts the units of the currency of the book, and the currency of
PACA is the Colombian peso.

**The day of a movement is a day and not a moment.** The time of the day is
not a fact of the money, and the column of the table holds no time.

**The balance of an account is the difference between its debits and its
credits, with the sign that the kind of the account asks for.** An asset and
an expense count as they are: what the person has and what they spend are
positive when they grow. A liability, an income and an equity count the
other way round: what the person owes, what they earn and what they put into
the book are positive when they grow. The result reads "what it means to
the person" in every kind.

**The money of the business is read from those balances:** what it has, what
it owes, what it earned, what it spent, and what it is worth.

### The period of the book

A reading of the book covers a **period**: the stretch of days that the
answer takes. `ledger.Period` holds the two ends of it, and it includes
both of them — the movement of the first day and the movement of the last
one are inside the period. `NewPeriod` refuses the ends that are in the
wrong order, and `Month` returns the period of a month of the calendar, with
the first day and the last day of the month that the calendar really has.

**The money of a period is what the person earned and what they spent.** The
two numbers of `ledger.Summary` are positive magnitudes — what came in and
what went out — and the net is what is left of the income once the expenses
are taken out of it. The net is negative when the person spent more than
they earned, and that is a fact of the month and not an error of the book.
The port of the book carries the reading of that money: `Summary` asks for
the money of a period and answers it.

### The debt

A debt is an account and never a table of its own. `account.IsDebt` says
when an account carries one: its kind owes or is owed, and it names the
other side of the money.

**The debt of the book is the account and what it still holds.**
`ledger.Debt` is the pair, and `ledger.Debts` reads the debts of a book out
of the accounts and their balances: the amount is the balance of the account
read the way the person reads it — what the person still owes for a
liability, what the person is still owed for an asset — and only the open
debts are in the answer.

**The debts come first by the day they fall due, and the debt without a day
comes last.** Two debts of the same day come by name, so the answer is the
same every time it is read.

**A payment may not pass what the debt holds.** `Debt.AcceptsPayment` asks
whether a payment may be paid against the debt: the payment must be
positive, and it may not be bigger than the amount the debt still holds.

## The model of the draft

`purchase.Draft` is the candidate that the agent proposes from what the
owner said. It is a proposal and not a fact: it is kept by the business
that owns it, with its state, its revision and the sentence it came from,
and it waits for the yes of the owner before anything is written in the
book. The transport of the MCP server is stateless, and a candidate that
lives only in the context of the model is a candidate that can be
paraphrased, lost or silently changed between the proposal and the yes.

**The draft carries the three parts of the answer of Paca.** What it
understood are its lines, what it still needs is `Missing`, what it was not
sure about is `Ambiguous` — and an ambiguity always carries the options
Paca offers, because Paca never asks an open question when it can name the
options — and what Paca filled by herself is `Inferred`.

**The money is never inferred.** `purchase.NewLine` refuses a price the
agent filled (`ErrInferredPrice`), and a line the owner did not price is
what the draft still needs. The quantity and its unit travel together:
`purchase.NewMeasure` refuses a quantity without a unit. When the owner
named both the money that was paid and the price of the unit, the two must
agree, and the domain refuses the pair that does not.

**The day is one of the three things Paca may fill by herself.** A draft
that names no day is the day of today, and the answer names it in
`Inferred`. The day, the currency and the business are the three, and the
money is never one of them.

**A draft expires seven days after the day it was proposed.** The price of
a market does not wait. An amendment grows the revision of the draft and
keeps the sentence the owner said the first time.

## The model of the purchase

`purchase.Purchase` is the fact of what the business bought: the business,
the day, the market, the invoice, its lines, the sentence of the owner and
the word she used to confirm it. A purchase never changes.

**The confirmation is a rule of the domain.** `purchase.Confirm` turns a
draft into the purchase it describes, and it refuses what the domain
forbids: a confirmation without the word of the owner
(`ErrMissingConfirmation`), a draft that does not wait any more
(`ErrNotProposed`), a draft that changed under the eyes of the owner
(`ErrStaleRevision`), a draft that expired (`ErrExpiredDraft`) and a draft
that still needs the money of one of its lines (`ErrStillNeeds`).

**The money of the book counts whole pesos.** `purchase.Price.Amount`
answers the money of a line: the money that was paid, and the price of the
unit times the quantity rounded to the nearest peso when the owner named
only the price of the unit. The sentence of the owner keeps the exact
wording, and the money of the book is whole pesos.

**The confirmation answers with the three parts of Paca.** What it
recorded are the lines of the purchase, what it did not understand are the
ambiguities the draft still carried, and what it still needs is empty: a
purchase without the money of its lines is refused.

**The money of the payment moves from the account that pays to the
inventory.** `purchase.ThePayingAccount` answers the account the money
comes from — the cash of the business, and the account the owner names
when she names another one — and `purchase.TheInventoryAccount` answers the
account of the inventory of the business. The role of an account belongs to
the seed of the book of a business: `account.OfRole` answers the account of
a role without writing its name down twice.

**A draft is confirmed once.** The state of a draft moves from `proposed`
to `confirmed`, and the storage refuses the confirmation of a draft that is
not the one the owner saw: the revision she saw is the revision the row
carries.

## The transaction of the storage

The unit of work is the port of a transaction of the storage, and it lives
in its own package because it is a capability of the storage and not of a
person or of a book:

```go
Do(ctx, work) error
```

The work runs inside one transaction: the rows are written whole or not at
all. The caller never sees the transaction, and never commits by itself. The
error of the work is the error of the call.

## The model of the health

`health.Status` is the state a service is in: `ok`, `degraded` or
`unhealthy`. `health.Check` is one component the service watches, with the
time it took to answer, and `health.Report` is the answer of the service:
its identity, its state, the checks it ran and the time it has been up.

**The state of the service is the worst state of its components.** The rule
lives here, in `Status.Worst`, and every service answers the same way: one
failing component makes the whole service unhealthy, one degraded component
makes it degraded, and a service with no checks at all is healthy.

**An unknown state is never trusted.** A status that the domain does not know
is `unhealthy`, never `ok`: the report of a service that nobody understands
must not be read as a promise.

## Running it

```sh
make -C packages/domain all    # the full quality gate
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
| `make cover`      | Measures the coverage and **fails below 80%**.                    |
| `make race`       | Runs the test suite with the race detector.                      |
| `make bench`      | Runs the benchmarks.                                              |
| `make perf`       | Checks the allocation budgets, then runs the benchmarks.          |
| `make vuln`       | Runs `govulncheck` over the dependencies.                         |

The tools are pinned in the repository (`tools/go.mod`), so every machine
and every container uses the same versions.

### The coverage, in one number

The module talks to no database, so the whole suite runs without one and
the gate measures it as a whole: the rules of the person are asserted
without a PostgreSQL, and the assertions are the tests.

## The tests

The unit tests live with the code they cover. There is no container test
here: the rules of the domain are proven without a database, and the rules
of the table are proven against a real one by the container tests of the
database library (`packages/database`).

## The versions

| What             | Version  | Why                                                     |
|------------------|----------|---------------------------------------------------------|
| Go               | 1.27.3   | The latest security release of the 1.27 line.            |
