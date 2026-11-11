# PACA — Service tracing depth: spans for the services of the monorepository

Feature document for the second tracing slice of the monorepository: the spans that make the
work of PACA's own services visible — the MCP server, the database library and the database
admin — after the diagnostic proved that every trace held a single HTTP span.

- **Project:** `apps/mcp-server`, `apps/db-admin`, `packages/database` (Go modules of the monorepository).
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-tracing-depth`, branch `feat/service-tracing-depth`.
- **Stack:** no new container. The collector of `observability/` is reconfigured, and the two
  services and the database library are instrumented.
- **Scope:** the operations of PACA's own services. The agent is **out of scope**: it is not
  instrumented, and no cross-process trace is aimed for.

---

## Specs

Each spec is the user's own wording, quoted verbatim. The task entries expand them into
implementation rules and verification.

### S1 — The diagnosis that opened the slice

> investigate why the traces are mostly empty, there are calls to the mcp server but just show son POST/GET calls. I need a deep diagnostic.

### S2 — The scope of the slice: the agent stays out, the own services come in

> ok the hermes tracing can be ommitted for now, due to even, i can use another agent in the future like gentle-shell or something like that. I need to track effectively my own creations, like my own mcp server, the db-admin that are something new. Re plan your recommenation with this constraint

### S3 — The work happens on its own worktree, in ODD, and the `.env` files are never touched

> ok you're authorized to start. Use ODD for organic implementation and work as usual in a apart git worktree. Due to there is a new gitworktree, is needed to add the `.env` files that are in the current folder, when needed, just tell me the commands to copy them in the gitworktree due to you're not allowed to manipulate this files.

---

## Tasks

Each task lists the specs it covers, the route it takes, and the commit that holds it. The
`Commit` column is filled in as the work units are committed.

| ID  | Specs   | Task                                                                                                                                                              | Route  | Commit |
|-----|---------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------|--------|
| T1  | S1, S2  | The spans of the tool calls: one span per call, named after the tool, child of the HTTP span, carrying the name of the tool and the type of the error, and a line of the log that carries the tool and the trace. | inline | `6524b2b` |
| T2  | S1, S2  | The spans of the queries: one span per statement of the database library, over the seam the generator owns, with the tracer given to the library and not taken from it. | inline | `22354f3` |
| T3  | S1, S2  | The trace of the database admin: the boot that applies the schema becomes a trace with the lock, the migrations and the state of the schema in it. | inline | `2e6d292` |
| T4  | S1      | The hygiene of the collector: the spans of the collector itself leave the store, and the redaction covers the keys of the payloads of the tools. | inline | `015c393` |
| T5  | S1, S2, S3 | The documentation: the model of the spans of each service, the recipe that reads the traces back, and the proof that the slice changed no behaviour. | inline | `772a524` |
| T6  | S1      | The two noises the cutover proved: the statement of the health check and the spans of the collector itself leave the store. | inline | `b217272`, `71c6c41` |
| T7  | S1      | The gate of the pull request: the toolchain of the repository moves to the security release that closes the advisory of the Go standard library that made the gate red. | inline | `76e0c3c` |

---

## Log

### L1 — The original request

> Check using docker (i already up the service and i'm testing it rn) and review the conversation that I got with Paca (hermes) and search for the tools or calls the agent made so far, and review too the database directly with docker cli in order to do not use passwords and review the records. Then review jaeger, and investigate why the traces are mostly empty, there are calls to the mcp server but just show son POST/GET calls. I need a deep diagnostic.

### L2 — What the diagnostic measured

- 26 spans, 26 traces, **one span per trace**, and **no span with a parent**: every span is the
  `otelhttp` server span of a `POST /mcp`, named after the route, with no MCP method and no tool
  name in it. Durations: 0.02 ms to 7.48 ms.
- The only instrumented layer of the services is the HTTP one
  (`packages/platform/server/middleware.go`, `otelhttp.NewHandler`). The MCP transport registers
  its tools without a middleware, and `packages/database` holds no OpenTelemetry at all.
- The agent sends no `traceparent` and exports no spans, so every call is a root trace. The
  conversation is therefore N disconnected traces, and the tool that ran is in none of them.
- `apps/db-admin` emits **no span at all**: its only route is the health endpoint, which is
  excluded from the tracing, and its real work — applying the schema — happens at boot, before
  it serves.
- The collector traces its own query API and stores those spans in the same backend: 200 spans
  of the service `jaeger` in three hours, most of them the polling of the interface.

### L3 — The scope the user fixed

> ok the hermes tracing can be ommitted for now, due to even, i can use another agent in the future like gentle-shell or something like that. I need to track effectively my own creations, like my own mcp server, the db-admin that are something new. Re plan your recommenation with this constraint

The consequence accepted: without an instrumented agent there is no parent, so the unit of
tracking is **the operation** — one tool call is one named trace — and not the turn of a
conversation. The GenAI conventions are dropped with the agent: no span of the services carries
them, so the redaction that named them protected nothing.

### L4 — The authorization

> ok you're authorized to start. Use ODD for organic implementation and work as usual in a apart git worktree. Due to there is a new gitworktree, is needed to add the `.env` files that are in the current folder, when needed, just tell me the commands to copy them in the gitworktree due to you're not allowed to manipulate this files.

The `.env` files of the stacks are read by nobody and written by nobody: the session states the
commands that copy them into the worktree, and the person who runs them owns them.

### L5 — The decisions of the slice

- The span of a tool call is created in the **adapter** of the transport, with the middleware of
  the SDK (`Server.AddReceivingMiddleware`), and it is the child of the HTTP span: the context of
  the middleware descends from the context of the request.
- The span of a tool call carries **no argument and no result**: the product holds personal and
  financial data, and the conventions of the protocol mark the payloads of the tools as
  sensitive and opt-in. The span says what ran and how long it took, never with what data.
- The methods that carry no work — `ping`, `initialize`, the notifications, the list of the
  tools — leave **no span**. The ping of the agent every three minutes was a third of the
  traces of the server.
- The identifier of the JSON-RPC request is **not reachable** through the API of the SDK: the
  request of the middleware carries the parameters, the session and the extra, and nothing
  else. The join between a trace and the conversation is therefore the line of the log of the
  tool call, which carries the name of the tool and the identifiers of the trace, and the
  second in which it was written.
- The spans of the queries are created in the **adapter** of the database, over the seam the
  generator owns (`DBTX`), so that every query of every repository is covered, present and
  future, by one decorator.
- The database library takes the **API** of the tracer (`go.opentelemetry.io/otel/trace`) and
  receives the tracer at its construction; it does not depend on the platform, and the SDK stays
  in the platform. The option is additive: no caller of the library breaks.
- The trace of the database admin is a **root** trace: the boot of the service is not a request,
  and a root span is the honest shape of it.

### L6 — The verification plan

- The unit tests of every unit that changes: the span of a tool call (its name, its parent, its
  attributes, the type of the error), the span of a query (its name, its statement, its parent),
  and the trace of the boot of the database admin (its children).
- The quality gate of every module that changes: `make -C apps/mcp-server all`,
  `make -C packages/database all`, `make -C apps/db-admin all`.
- The stack, after the person who owns the session restarts it: a tool call, and the trace read
  back from the query API of the collector with the recipe of the documentation.
- The differential check: the tools of the MCP server and the commands of the database admin
  behave exactly as before — the slice adds spans and changes nothing else.

### L7 — What was verified, and how

- **The tests of the slice.** The call of a tool leaves a span named after
  the tool, of the kind server, child of the span of the request, with the
  method and the tool in its attributes and with no argument and no result
  in it; a failing tool leaves the type of the failure and the status Error;
  a method that carries no work leaves no span; a transport without a
  tracer leaves no span and does not fail. A statement leaves a span named
  `pg.query`, of the kind client, child of the span of the work that ran
  it, with the system, the namespace and the statement; a failing statement
  leaves the type of the failure; the row of a statement ends its span when
  it is read. The boot of the database admin leaves a root trace, and the
  migrator receives a context that carries it.
- **The gates of the three modules.** The format check, the analyzers, the
  linter, the build, the tests, the race detector and the coverage floor
  pass for `apps/mcp-server`, `packages/database` and `apps/db-admin`. The
  vulnerability scan fails with **GO-2026-6859**, an advisory of the Go
  standard library: the same scan fails the same way on `main`, so it is
  not a defect of this slice.
- **The configuration of the collector.** It was proved with a collector of
  its own, built from the same image and the same file, while the stack of
  the monorepository kept running: the collector starts with the filter, a
  span of the service `jaeger` is dropped and a span of another service is
  stored.
- **The stack of the monorepository was not restarted.** The session was
  testing against it, and the cutover is the decision of the person who owns
  it. The commands are in L9.

### L8 — The limits of the slice, written down

- The type of the failure of a tool call is `tool.error`: the SDK wraps the
  error of the tool in the result of the call, so the type of the error of
  the domain is lost at the boundary of the protocol. The reason of the
  refusal stays with the client, and the errors the service cannot map are
  written in the log.
- A span of a tool call carries no argument and no result, and a span of a
  statement carries no value: the trace says what ran and how long it
  took, never with what data. The people of the product are the reason.
- The agent is not instrumented: the join between a trace and the
  conversation is the line of the log of the call — the tool, the outcome
  and the identifiers of the trace — and the second in which it was written.

### L9 — The environment of the worktree, and the cutover

The `.env` files of the stacks belong to the person who runs the project:
the session reads none and writes none. The worktree has no `.env` of its
own, and the stacks that need one get it with these commands, run by the
person who owns them:

```sh
cd /Users/braejan/workspace/witsaba/repositories/paca-tracing-depth
cp ../paca/db/postgres/.env db/postgres/.env
cp ../paca/hermes/.env hermes/.env
cp ../paca/engram/.env engram/.env
```

The cutover rebuilds the two services and restarts the collector. It
interrupts the session of the agent against the MCP server, and the person
who owns the stack decides when it happens:

```sh
cd /Users/braejan/workspace/witsaba/repositories/paca-tracing-depth
docker compose build mcp-server db-admin jaeger
docker compose up -d --wait mcp-server db-admin jaeger
```

After the cutover, a call of a tool is read back with the recipe of the
documentation of the observability stack.

### L10 — The native review of the slice: started, and blocked on the host

The switch of the review is on, so the assessment of the slice ran, and it
asked for the review: the tier of the candidate is high, and the reason it
gives is a *process boundary* in `packages/database/postgres/tracing.go` —
the word "process" of the comments of the file, read as code that starts
processes. It is not a defect of the slice.

The lineage of the review is `review-63e43867b7af30f5`, with the lenses
`review-risk` and `review-reliability`, and it waits for the two artifacts
of the reviewers. **The capture of those artifacts is blocked on the host:**
the facade rejects the binding of the collector, both as the complete group
and as a single slot, with *"collectBindings must be the complete distinct
current reviewer group in exact provider order"* and *"collectBinding is
missing or stale for current STATUS"*. The binding the status issues is
copied exactly and nothing was captured; the lineage stays in the state
`reviewing` and no verdict was admitted.

Nothing of the review was submitted, and the delivery of the slice does not
depend on it. The person who owns the repository can finish it with the
collector itself:

```sh
gentle-ai review status \
  --cwd=/Users/braejan/workspace/witsaba/repositories/paca-tracing-depth \
  --contract=gentle-ai.review-integration/v2 --next-transition=true \
  --lineage=review-63e43867b7af30f5 \
  --repository-context=rctx2_8eaddf0ec74fa9d10adb776bc075e461b24b66d2121629000fcd8d3a37f47c1b \
  --agent=pi --base-ref=01e834ad5025fd069ab19f0fa2bcd0034e59f8a6 --committed-only=true
```

### L11 — The independent verification of the slice could not run

The risk line of the slice names item 3 (the public surface of the shared
library changes), so the slice needs an **independent** verifier and not
only the self-verification of the writer.

The verifier was launched and it **never ran a probe**: the provider of the
subagents answered with a quota error. It is the same error that killed the
explorer and the two writers of the slice, so it is the quota of the
provider of the session and not a defect of the brief.

**The proof of the independent verification is therefore missing.** What
exists is the self-verification of the writer, which is not the same thing:

- The three modules build, vet and test green, and the gates are green
  except for the advisory of the Go standard library of L7.
- The layering holds: no OpenTelemetry in the domain, the application, the
  ports or the generated code of the library; `packages/database` does not
  depend on `packages/platform`; no production file of the library imports
  the SDK of the tracing, only its API.
- The contract did not break: the call sites of `main` — `postgres.Open(ctx,
  cfg)`, `core.NewMigrator(nil, logger)`, `database.NewMigrator(nil)` — are
  unchanged and still compile, because the two options are variadic. The
  only call site that changed is the composition root of the database admin,
  which hands the tracer over.
- The privacy promise is mechanical: no span writes an argument, a result
  or a message; the SQL of the library binds every value and never inlines
  one.

The independent verifier is the missing proof, and the person who owns the
repository can run it when the quota of the provider is back.

### L12 — What the cutover of the stack proved, and the two noises it found

The cutover was made by the person who owns the stack, and the trace of a
call of a tool was read back from the collector:

```
POST /mcp          1.14 ms   the request
└── list_users     0.94 ms   mcp.method.name=tools/call, mcp.tool.name=list_users
    └── pg.query   0.80 ms   db.statement=SELECT ... FROM users ORDER BY id LIMIT $1::bigint
```

The call that failed answered `error.type=tool.error` on the span of the
tool, and the boot of the database admin left its root trace with the
migrations, the version and the lock under it.

Two noises were found in the store of the fresh stack:

- **The statement of the health check.** The container runtime asks the
  health endpoint every few seconds, the endpoint is excluded from the
  tracing, and the statement of that question became a **root** span of its
  own: six orphan spans in three minutes, one per question of each
  service. The health check is the runtime asking whether the service is
  alive, not work: the statement of it is not traced any more.
- **The spans of the collector itself.** The filter of the pipeline drops
  the spans that arrive over OTLP with the name of the collector — proved
  on the live stack with a probe: the probe of the service `jaeger` was
  dropped and the probe of another service was stored. The spans of the
  *query API* are written straight to the storage and never travel through
  the pipeline, so no processor can filter them: the switch of the SDK of
  the collector is what stops them, and it costs the logs of the collector
  nothing (measured in a collector of its own: zero self-spans, the logs
  unchanged).

### L13 — The state of the store after the last cutover

The last cutover left the store with exactly the work of the services and
nothing else:

```
paca-mcp-server   3 spans   POST /mcp → list_users → pg.query
paca-db-admin     5 spans   db-admin.migrate → schema.migrate, schema.version → schema.lock, schema.pending
jaeger            0 spans
```

The rules that came out of the cutover:

- The pool does not trace the statement of the **health check**: the
  runtime asking whether the service is alive is not work.
- The migrator never opens a **root** span: it enriches a trace that is
  already open. The boot of the writer of the schema opens the trace, and
  the health check leaves nothing behind.
- The collector does not trace itself: its query API writes its spans
  straight to the storage, so the switch of its SDK is what stops them.

### L14 — The acceptance of the slice

The slice was accepted by the person who owns the repository, after the
cutover of the stack and the reading of the traces back: *"lgtm"*.

The acceptance covers the implementation, the tests and the evidence of the
slice. Two things stay open on the host and neither of them blocks the
slice: the native review (lineage `review-63e43867b7af30f5`, waiting for
the artifacts of the reviewers) and the independent verification (the
provider of the subagents answered with a quota error, so the agent of the
verification never ran a probe).

The branch is `feat/service-tracing-depth`, with the worktree at
`/Users/braejan/workspace/witsaba/repositories/paca-tracing-depth`. The
push, the pull request and the merge remain the decisions of the person
who owns the repository.

### L15 — The gate of the pull request was red, and why it was fixed here

The gate of the pull request failed at the `vuln` step with **GO-2026-6859**:
an advisory of the Go standard library against `net/http@go1.27.3`, fixed in
`net/http@go1.27.4`. The repository pins `go 1.27.3` in `go.work` and in
every `go.mod` of the tree, and the images of the build pin the image of
the same version by digest.

The failure was not of this slice — the same scan fails the same way on
`main` — but a red gate is a red gate, and the person who owns the
repository asked for it to be fixed in the same pull request.

The toolchain moved to **1.27.4**, and every pin of the tree with it:

- the `go` directive of `go.work` and of the seven modules of the tree;
- the image of the build of the two services, with its digest
  (`golang:1.27.4-alpine3.23@sha256:146e3b9c…`);
- the tables of the versions of the four READMEs that name it.

The evidence: the scan of the MCP server answers *"Your code is affected by
0 vulnerabilities"* with the new toolchain, and the gates of the three
modules that change pass whole. The run of the continuous integration of
the pull request is **green**: every job succeeded, the quality gate
included, with the container tests of the shared libraries and of the
applications running for the first time in this branch — the toolchain
moved, so every module of the tree is in the diff of the gate.
