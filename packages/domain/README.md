# The domain of the PACA monorepository.

The domain holds what the services of the monorepository share about the
*people*: the model of a person, the rules a person carries, and the port of
the storage that keeps them. The MCP server imports it today; the worker and
the REST backend will import it tomorrow.

It is a **library**. It publishes no port on the network, it runs no
container, and it starts no process.

---

## The layout

```
packages/domain/
  domain.go         the package comment: the models and the rules, nothing else
  user/             the model of the person: the identifier, the states, the
                    details and the port of the storage
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

**The email is checked** and trimmed: an address that does not look like an
address is refused, and the same rule is a CHECK of the table.

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
GetByID(ctx, id) (user.User, error)
UpdateDetails(ctx, id, details) (user.User, error)
ChangeStatus(ctx, id, status, observation) (user.User, error)
```

There is no delete and no list: a person that must go is `blocked`, and
nothing consumes a list yet.

The port is what a use case needs, not the shape of a table. It lives here,
in the module of the models, because both sides of it need it: the service
that writes the people, and the adapter that reads the table. An adapter of
another storage — a gRPC client of a data service, tomorrow — implements
the same port, and no service changes.

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
