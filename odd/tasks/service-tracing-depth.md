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
| T3  | S1, S2  | The trace of the database admin: the boot that applies the schema becomes a trace with the lock, the migrations and the state of the schema in it. | inline | — |
| T4  | S1      | The hygiene of the collector: the spans of the collector itself leave the store, and the redaction covers the keys of the payloads of the tools. | inline | — |
| T5  | S1, S2, S3 | The documentation: the model of the spans of each service, the recipe that reads the traces back, and the proof that the slice changed no behaviour. | inline | — |

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
