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
| T7  | The gate pins the rules it claims: one writer, no dead API, no copy of a shared layer, no rule over an empty set | S1, S2, S3, S5 | inline | — |
| T8  | The environment of the stack declares the variables it reads          | S10     | inline  | — |
| T9  | The MCP server refuses an unsafe bind and checks the Host            | S9      | worker  | — |
| T10 | The concurrent change has a container test                           | S11     | inline  | — |
| T11 | The read of a person reaches the agent through the port              | S6, S12 | inline  | — |
| T12 | The list of the people reaches the agent through the port            | S6, S12 | inline  | — |
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
