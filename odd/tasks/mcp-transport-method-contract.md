# The method contract of the MCP transport

This slice answers the two defects that the monitor of the traces of the stack
opened against the transport of the MCP server (`GET /mcp` and `HEAD /mcp`
answered with `405`). The transport is stateless and the `405` is the answer
the protocol asks for; the defect was the documentation of the service, which
promised methods the transport does not serve. The contract is now pinned by a
test and stated by the README.

---

## Specs

The specs quote the request of the maintainer and the contract of the
repository (`AGENTS.md`), verbatim. Nothing is added that was not asked for.

**S1** — From the request of the maintainer:

> check in the repository issues, where paca, with another model is currently
> running in this PC as a production one, in docker compose. So I need you be
> carefull due to is needed to do not delete anything inside the current
> running. Paca is creating issues with a new github user so read them and
> explore the errors mentioned. IF the issue is real, proceed to fix it, if do
> not find any reason to do it, just document it and close it.

The stack that runs on the machine is not touched: no container is stopped,
removed or rebuilt, and no volume is removed. The change is a branch and a
worktree of its own, and the production image keeps running the code it
already runs.

**S2** — From the request of the maintainer:

> IF the issue is real, proceed to fix it, if do not find any reason to do it,
> just document it and close it.

Each issue is read, the error it names is reproduced against the running
stack, and the answer is written in the issue itself. An issue whose finding
is real is fixed; an issue whose finding is not real is documented and closed.

**S3** — From `AGENTS.md`, "The tests are the proof of the contract":

> A test asserts **every observable effect** of the rule it covers: the exit
> code, the exact output, the error, and the state that was left behind.

The method contract of the transport is proved against the transport itself: a
`GET`, a `HEAD` and a `DELETE` on `/mcp` are answered with `405` and with the
header that names the method that is allowed.

**S4** — From `apps/mcp-server/README.md`, the table of the endpoints said:

> | `/mcp`     | `POST` (`GET`, `DELETE`) | The MCP transport, over the streamable HTTP transport. |

The README states the contract the transport enforces: `POST`, and the `405`
that every other method receives, with the reason the protocol gives for it.

**S5** — From `apps/mcp-server/AGENTS.md`, "The shape of a change here":

> 5. The gate of the service is green: `make -C apps/mcp-server all`, and the container
>    tests: `make apps-integration`.

The gate of the service is green before the work unit is committed.

---

## Tasks

| ID   | Task                                                          | Specs   | Route   | Commit |
|------|---------------------------------------------------------------|---------|---------|--------|
| T1  | The feature document of the slice                              | S1      | inline  | `1d05630` (the document) |
| T2  | The method contract of the transport is pinned by a test       | S3      | inline  | `5d175d2` (the test) |
| T3  | The README states the method contract the transport enforces   | S4      | inline  | `5d175d2` (the documentation) |
| T4  | The evidence: the gate of the service is green                 | S5      | inline  | — |
| T5  | The issues 78 and 79 are answered and closed                   | S2      | inline  | — |

---

## Log

**L1** (2026-11-16): The request of the maintainer, verbatim:

> check in the repository issues, where paca, with another model is currently
> running in this PC as a production one, in docker compose. So I need you be
> carefull due to is needed to do not delete anything inside the current
> running. Paca is creating issues with a new github user so read them and
> explore the errors mentioned. IF the issue is real, proceed to fix it, if do
> not find any reason to do it, just document it and close it.

The branch is `fix/mcp-transport-methods`, the worktree is
`/Users/braejan/workspace/witsaba/repositories/paca-worktrees/mcp-transport-methods`,
created from `main` at `2c7642c`. The stack that runs on the machine is the
production one of `witsaba/paca`: `paca-mcp-server`, `paca-db-admin`,
`paca-postgres`, `paca-jaeger`, `paca-engram` and `paca-hermes`. Nothing of it
is stopped, removed, rebuilt or restarted by this slice.

**L2** (2026-11-16): What the two issues found.

The issues are #78 (`GET /mcp` answered with `405`) and #79 (`HEAD /mcp`
answered with `405`), both opened by the account of the harness of the agent
(`witsaba-harness-2`) from the traces of the stack.

The answer of the transport is the right one. The transport is the official Go
SDK of the protocol (`github.com/modelcontextprotocol/go-sdk v1.8.0`) built with
`Stateless: true` (`apps/mcp-server/internal/adapters/inbound/mcpserver/server.go`),
and a stateless server of the SDK serves the messages of the protocol with
`POST` alone: every other method is answered with `405 Method Not Allowed` and
the header `Allow: POST`, as RFC 9110 §15.5.6 asks (`mcp/streamable.go`,
`serveStateless`). The protocol of MCP lets a server offer a stream of
server-initiated messages with `GET` and asks a server that does not offer it
to answer `405`; a stateless server has no server-initiated message to send
and no session to end.

It was proved against the running stack, without touching it: a `GET`, a
`HEAD` and an `OPTIONS` on `http://paca-mcp-server:8081/mcp` from the
container of the agent are answered with `405` and the header `allow: POST`.

**The defect was the documentation.** The README of the service promised
`GET` and `DELETE` on `/mcp` (S4): a contract the code never implemented and
that no test pinned. The mismatch is what the monitor read as a defect.

**L3** (2026-11-16): The work units.

The test (T2) pins the method contract against the transport itself: a `GET`,
a `HEAD` and a `DELETE` on `/mcp` are answered with `405` and with
`Allow: POST`. The test was green when it was written: the behavior was
already right, and what was missing was the proof of it and the truth of the
README. A test that proves a rule the code already holds is worth writing
here: without it, the next change of the transport could have turned the
`405` into a defect with nothing to fail.

The README (T3) states the contract the transport enforces, the reason the
protocol gives for the `405`, and the two ways a probe can check the transport
alive: a `POST` with an `initialize` message, or a read of `/healthz`.

**Risk:** none. The change touches no code of the running stack: it adds a
test and corrects a line of documentation that named a contract the service
never implemented. The observable behavior of the service is the one it had
before the change.

**L4** (2026-11-16): The evidence, and the answer of the two issues.

The gate of the service is **green**: the format check, the analyzers, the
linter, the build, the tests, the race detector, the coverage floor and the
vulnerability scan. The coverage of the module is `88.7%`, above the floor of
`80%`, and the scan found no vulnerability in the code of the service.

The container tests (`make apps-integration`) were not run locally: they build the
image of the service and start it, and the stack of the machine is the
production one of the maintainer, which this slice does not touch. The gate
of the service that covers the change is the one that was run.

**The continuous integration of the pull request #81 is green**, and it runs
what the machine could not: the gate of every module of the monorepository,
the container tests of the shared libraries and the container tests of the
applications, the shellcheck of the scripts, the tests of the structure of the
repository, and the whole stack with the schema written by the service that
owns it. The jobs of the database, of the memory and of the collector of the
traces were skipped: the change cannot affect what they prove.

**Paca reviewed the pull request** as a second outside view, with the account
of the harness of the agent, and published its verdict on the pull request
itself: the resolution is right, the `405` is the answer the protocol asks
for a stateless transport, the correction of the README is exact, the test
pins the contract against the real SDK and not against a mock, and the
narrowing of the change carries no risk for the stack that runs. It found no
blocking finding. The one thing it asks is outside this repository: the probe
of the monitor that opened the two issues must be pointed at `/healthz`.

Both issues are answered with the proof and closed. The answer is the same for
the two: the `405` is the answer a stateless transport of the protocol gives,
the header `Allow: POST` is there, and the defect was the documentation of
the service, which is now corrected and pinned by a test.

The test that the issue #79 proposes was read before it was written. It asks
the router for a `HEAD` with a **fake** transport, and the fake answers `202`
whatever the method is: it would have been green before the change and after
it, and it would have proved nothing about the method contract of the real
transport. The test of this slice asks the transport itself.
