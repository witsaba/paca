# The open defects of PACA

This slice closes every defect the exploration of the repository found open:
the seven findings the review of the architecture slice left open, the rules of
the contract the gate does not pin, and the four older items that were still
open. Each finding becomes one work unit, with its own tests and its own
commit.

---

## Specs

The specs quote the contract of the repository (`AGENTS.md`) and the request of
the maintainer, verbatim. Nothing is added that was not asked for.

**S1** — From `AGENTS.md`, "The rules that are never broken":

> **No second writer of the schema.** One service owns the migrations of the
> database, and it applies them at the start. No service migrates on boot.

The gate pins it: no application but the database admin may name the migrator
of the library of the database, and the assertion proves it discriminates.

**S2** — From `AGENTS.md`, "The rules that are never broken":

> **No dead API.** A function that no production code calls is not a promise,
> it is debris: it is removed, or the caller that needs it is written.

The debris of the repository is removed, and the gate pins the rule: the
exported functions of the shared libraries that no production code calls are
reported as failures.

**S3** — From `AGENTS.md`, "The rules that are never broken":

> **No copy of a shared layer.** The platform and the shared domain are shared:
> they are extended, never forked. The infrastructure of a service lives in
> `packages/platform`, and the models that more than one service needs live in
> `packages/domain`.

The middleware of the HTTP router, the names of the environment of the
database and the wiring of the telemetry live once, in the platform, and the
gate pins that the two services do not carry a second copy of them.

**S4** — From `AGENTS.md`, "The contract of the outside world derives from the
domain":

> Every error of the domain that a caller can see has a message **where the
> caller is**, written in the words of that caller. No error of the domain
> reaches a user unmapped.

The mapping of the errors of the MCP server maps every error it can receive,
the default branch no longer returns the raw error, and every branch of the
mapping is proved by a test.

**S5** — From `AGENTS.md`, "The rules that are never broken":

> **No rule that only a comment states.** A rule that the code cannot break is
> a wish. The gate pins the rules that matter.

Every rule of the gate proves that it runs and proves that it discriminates: a
rule that runs over an empty set is reported, and every rule has its probe.

**S6** — From `AGENTS.md`, "The ports are contracts, and they are narrow":

> The port is **narrow**: one interface, one capability, the methods the use
> case actually calls. A method that no use case calls is not a method of the
> port yet.

The port of the users carries the read and the list that the read and the list
tools call, and nothing else.

**S7** — From `AGENTS.md`, "The workflow of a change":

> The gate is green before a work unit is committed: the format check, the
> analyzers, the linter, the build, the tests, the race detector, the coverage
> floor and the vulnerability scan of every module, and then the container
> tests.

The gate of the repository is green from nothing, and the container tests run
in the containers of the repository.

**S8** — From the request of the maintainer:

> ok the golang stuff is if you really needed locally, if you can run the stuff
> in the containers, much better. Open a new git worktree and work on it
> fixinng all the defects listed to fix.

The work happens in the worktree of the slice. The toolchain of the machine is
not changed: the containers of the repository build and run the code with the
version the repository pins.

**S9** — From `apps/mcp-server/README.md`:

> the hardening that was proposed for it (refusing a bind that is not loopback,
> a Host allowlist and the explicit Origin rule) is still open.

The server refuses a bind that is not the loopback interface outside a
container, checks the `Host` header against an allowlist, and states the
`Origin` rule explicitly.

**S10** — The environment of the stack names its own variables. The connection
string of the database is named by `PACA_DATABASE_URL`, which no manifest and
no compose file declares; the stack reaches the database through the
credentials of the PostgreSQL stack, and the file that carries them is
optional, so the server starts with no database and says so.

**S11** — The branch of the adapter that reports a user that moved at the same
time (`ErrConcurrentChange`) is proved by a container test: a writer that
changes the row between the read and the write.

**S12** — The tools of the MCP server carry the read of a person and the list
of the people, through the port of the domain and through the use cases. The
description of a tool promises what the tool does, and nothing it does not do.

---

## Tasks

| ID   | Task                                                                 | Specs   | Route   | Commit |
|------|----------------------------------------------------------------------|---------|---------|--------|
| T1  | The feature document of the slice                                     | S8      | inline  | `94a002b` (the document) |
| T2  | The middleware of the HTTP router belongs to the platform             | S3      | worker  | `32606be` (the fix) |
| T3  | The names of the environment of the database belong to the platform   | S3      | inline  | `f53c695` (the fix) |
| T4  | The wiring of the telemetry belongs to the platform                  | S3      | inline  | `d523bcf` (the fix) |
| T5  | The mapping of the errors of the MCP server maps every error         | S4      | inline  | `642c947` (the fix) |
| T6  | The dead API of the repository is removed                            | S2      | inline  | `1db2660` (the fix) |
| T7  | The gate pins the rules it claims: one writer, no dead API, no copy of a shared layer, no rule over an empty set | S1, S2, S3, S5 | inline | `77e2b80` (the gate) |
| T8  | The environment of the stack declares the variables it reads          | S10     | inline  | `8b8d730` (the database admin) |
| T9  | The MCP server refuses an unsafe bind and checks the Host            | S9      | inline  | `9d51e6c` (the fix) |
| T10 | The concurrent change has a container test                           | S11     | inline  | `ee2f4d3` (the test) |
| T11 | The read of a person reaches the agent through the port              | S6, S12 | inline  | `50e7031` (the fix) |
| T12 | The list of the people reaches the agent through the port            | S6, S12 | inline  | `0cee5c1` (the fix) |
| T13 | The evidence: the gate green from nothing, and the review            | S7      | inline  | — |

---

## Log

**L1** (2026-10-30): The request of the maintainer, verbatim:

> ok the golang stuff is if you really needed locally, if you can run the stuff
> in the containers, much better. Open a new git worktree and work on it
> fixinng all the defects listed to fix.

The branch is `fix/open-defects`, the worktree is
`/Users/braejan/workspace/witsaba/repositories/paca-worktrees/open-defects`,
created from `main` at `1f12ede`.

**L2** (2026-10-30): The exploration that fixed the scope of the slice. The
defects are the seven findings of the review of the architecture slice (the
log entry L3 of `odd/tasks/architecture-defects.md`), the rules the gate does
not pin, and the four older items: the review of every environment variable,
the hardening of the MCP server, the read and the list tools, and the
container test of the concurrent change. The exploration is in the observation
4646 of the memory.

The state of the repository when the slice starts: `main` at `1f12ede`, clean,
in sync with the origin, no worktree, no branch, no stash, no container. The
gate of the structure proves 337 assertions and no failure.

**L3** (2026-10-30): The first three work units, and the boundary of the
slice.

The middleware of the HTTP router moved to the platform (`32606be`). The
middleware is the same code, the same chain and the same order; it lives
once now, in `packages/platform/server`, and the two routers call it. The
tests of the routers of the two services passed untouched: they are the
proof that the behavior did not change. The platform carries the promise and
the test of it: the security headers, the line of the log, the recovery of a
panic, the no-op provider and the paths that leave no span.

The names of the environment of the database moved to the platform too, with
the convention that goes with them: **an empty value is no value**. The
installer of the environment of the monorepository never writes an empty
value, and a variable that is set to nothing is a variable that is not set.
Two services read the environment in two ways before this work unit — the
MCP server failed to start when the connection string was empty, and the
database admin did the same — and the README of the MCP server said that an
empty connection string means a server without a database, which was not what
the code did. The platform holds the names, the defaults of the host and the
port, and the two readers (`Value` for a name, `Secret` for a password that
may hold a space). The password is never trimmed: a password that is trimmed
is a password that is wrong.

The wiring of the telemetry moved to the platform as well: the options of the
telemetry are built from the configuration of the tracing, the name of the
service is resolved by the configuration itself, and the grace period of the
shutdown belongs to the platform. Neither service writes them twice now.

The count of the assertions of the gate of the structure moved from 337 to
325: the count is a function of the number of imports of the tree, and the
middleware that left the adapters of the two services took nine imports with
it. The rules still hold and the gate still proves no failure.

**The boundary of the slice.** The commit of the middleware carries 414
authored lines and 191 removed ones: it is over the ~400 lines of the
repository, and the exception is recorded here. The work units that follow
stay under it.

**L4** (2026-10-30): The errors, the debris and the gate.

**The mapping of the errors** (`642c947`). Every error of the domain reaches
the agent with a message of its own, and the text of a Go error never does.
The catalogue of the errors of the domain (`user.Errors`) names them all, and
a test proves that every one of them has a message: an error that no message
explains would reach the agent as the text of a Go error, or as the message
of a failure the service does not know. The library of the database speaks
the vocabulary of the domain now: the error of a repository without a
database behind it is `user.ErrNoStorage`, and the error that the mapping
did not know is logged with its cause and answered with a message that does
not carry it.

**The dead API** (`1db2660`). The debris is gone: the state of the
migrations of the use case, the state of the migrations of the port, of the
adapter and of the library, the versions of the applied migrations, the
health of the status, the propagator of the providers and the SDK server of
the transport. The rule of the schema that was up to date moved where it
belongs: the composition root of the database admin asks the domain and says
what it says. The analysis of the dead code found two more functions that
nothing called — the address of the application of the two services — and
they found their caller: the log of the service now says the address of the
server through them.

**The gate** (`77e2b80`). The gate of the structure proves three more rules
of the contract, and every rule says what it proved:

- **No dead API.** The analysis starts at the entry point of every service,
  so a function that only a test calls is dead. The tool is pinned in
  `tools/go.mod` and it builds offline: the module is already in the cache
  of the machine.
- **One writer of the schema.** Only the composition root of the database
  admin names the migrator of the library.
- **No copy of a shared layer.** The bodies of the functions of the two
  services are compared: a behavior that two services write is a behavior
  that belongs to the shared layer. The copies that are known today are
  listed in the gate and reported on every run: the use case of the health
  and the endpoint of the health are written twice, and they wait for a
  shared application layer and a shared adapter. They are the only two
  copies left, and the gate says so every time it runs.

The gate also says what each rule proved, and a rule that proves nothing
without saying why is a failure: the rule of the domain of the applications
runs over an empty set today — neither service has a domain layer — and it
says so in the report.

Every one of the three rules was proved with a probe: a function that
nothing calls, a second writer of the schema, a copy of the code of the
other service and a rule that proves nothing all make the gate fail.

**The environment of the stack** (`8b8d730`). The variables of the database
say where they come from: the manifest of the stack that owns the
credentials writes them, the compose file of the service names the host and
the port, and `PACA_DATABASE_URL` is the override of an operator — the stack
does not set it, so a value that comes from outside is not shadowed by it.
The rule of an empty value is now the same everywhere: an empty value is no
value, and the credentials are read instead.

**The user that moved at the same time** (`ee2f4d3`). The branch of the
adapter that reports a user that changed between the read and the write is
proved by a container test: the other writer takes the row and changes its
state without saying it yet, the adapter reads the state the row had before
and its write waits for the lock, and the guard of the table refuses the
write when the other writer says its change. The test waits for the lock
instead of sleeping, and a mutation that maps the error to another one makes
it fail.

**L5** (2026-10-30): The read and the list of the people.

The read is a new promise of the port of the storage, and it is a promise
that a tool keeps: the tool `get_user` reads the person the document ID
carries, and it changes nothing. The list is the same promise with a bound:
the tool `list_users` answers with the people PACA knows, in the order of
their document ID, and the number of people it answers is bounded by the
domain — the caller asks for what it wants, the domain decides how many the
list carries, and a limit that is over the bound is the bound.

Both methods are narrow: `Find` and `List` are the two methods the use cases
call, and the gate of the dead code proves that they are called.

The list of the people was not in the findings of the audit: it was in the
older list of the open items of the repository, and it came with the read,
because a tool that reads one person and a tool that reads many people are
the same promise with two shapes.

**L6** (2026-10-30): The hardening of the MCP server.

The hardening that the README of the service named as open is closed:

- **The bind.** The server refuses a bind that is not the loopback
  interface, and the operator allows it with one variable
  (`PACA_MCP_ALLOW_NON_LOOPBACK`) that is false by default. The tools carry
  the data of the people, and a bind that the network of the machine reaches
  is a bind that anybody on the network reaches. The container is the
  deployment where the other containers of the stack reach the server, over
  a network that has no route to the internet, and the compose file and the
  image of the service say so.
- **The hosts.** The server answers only for the hosts of its allowlist
  (`PACA_MCP_ALLOWED_HOSTS`): the `Host` header of a request that names
  another host is refused with a `403`, and the refusal never reaches the
  tools. The filter lives in the platform, next to the middleware of the
  HTTP server, and the platform does not read the environment: the service
  hands it the hosts it answers for.
- **The origins.** The rule of the `Origin` is stated and not left to the
  default: the service trusts the origins of `PACA_MCP_TRUSTED_ORIGINS`,
  and the rule of the standard library — the same origin as the `Host`
  header — is the rule when the list is empty. An origin that the
  protection of the standard library cannot read is an error of the
  configuration and not a panic of the process.

The worker that was to write this work unit died with the rate limit of the
provider before it wrote a line, and the work unit was written here.

**L7** (2026-10-30): The evidence, and what the review found.

The gate of the repository is **green from nothing**: the format check, the
analyzers, the linter, the build, the tests, the race detector, the coverage
floor and the vulnerability scan of every module, the container tests of the
packages and of the applications, the shellcheck of the scripts, the tests of
the structure, the tests of the filters of the continuous integration, the
database, the collector of the traces, and the whole stack with the schema
written by the service that owns it. It took two minutes and it left no
container behind.

The gate of the structure proves 332 assertions and no failure, and it says
what every rule proved. The rules it added are proved with a probe each: a
function that nothing calls, a second writer of the schema, a copy of the
code of the other service, and a rule that proves nothing without saying
why — the four make the gate fail.

**What the slice changed, in one list:**

| The defect | What it is now |
|------------|----------------|
| The one writer of the schema is not pinned | The gate refuses a second writer. |
| The dead API is not pinned, and there is dead API | The debris is gone and the gate refuses the next one. |
| The middleware of the router is written twice | It lives in the platform, and the gate compares the bodies of the two services. |
| The names of the environment of the database are written twice | They live in the platform, with the rule of an empty value. |
| The wiring of the telemetry is written twice | It lives in the platform. |
| The default branch of the mapping of the errors returns the raw error | Every error has a message, and the catalogue of the domain proves it. |
| The rule of the domain layer of the applications proves nothing today | Every rule says what it proved, and a rule that proves nothing says why. |
| The environment of the stack does not say where its variables come from | Every variable says who writes it. |
| The hardening of the MCP server is open | The bind, the hosts and the origins are closed. |
| The read and the list of the people are not there | They are, through the port of the domain. |
| The branch of the concurrent change has no test | It has a container test that waits for the lock. |

**What is still open.** The gate reports two copies on every run: the use
case of the health and the endpoint of the health are written twice, one in
each service. They wait for a shared application layer and a shared adapter,
which the repository does not have yet. The gate names them, so they cannot
be forgotten, and any other copy is a failure.

**Risk:** item 2 — the hardening of the MCP server is a change of the
boundary of the service. The bind of the container is the same as it was,
the allowlist of the hosts names the container itself, and the whole stack
comes up healthy with the new variables. The rule of the ports is untouched:
the service publishes no port and it sits on the network that has no route to
the internet.
