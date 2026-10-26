# PACA — Observability: tracing and logging with OpenTelemetry

Feature document for the observability foundation of the monorepository: OpenTelemetry tracing
exported over OTLP, and structured logging correlated with the traces, for the MCP server of
`apps/mcp-server`.

- **Project:** `apps/mcp-server` (Go module `github.com/witsaba/paca/apps/mcp-server`).
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-observability`, branch `feature/observability-tracing`.
- **Stack:** new top-level `observability/` directory, one stack, included from the root compose.
- **Scope:** the HTTP surface of the MCP server. No MCP method spans, no GenAI spans, no metrics.

---

## Specs

Each spec is the user's own wording, quoted verbatim. The task entries expand them into
implementation rules and verification.

### S1 — Tracing and logging with OpenTelemetry

> I wanna add tracing and proper logging to this project using OTel.

### S2 — The difference between logging and tracing must be understood, not assumed

> I use to use Jaeger but I didn't no have clear the differences in logging and tracing

### S3 — A storage for the telemetry, with a clean time strategy

> is possible to have a kind of database for the logging with a clean time strategy? Only in memory
> loses when the server restart or something like that rigth?

### S4 — The collector is Jaeger, configured by this repository

> is Jaeger v2 with its own collector right? why are you mention `our own collector`?

### S5 — Jaeger is the only backend

> 1, Backend Jaeger only

### S6 — Tracing the HTTP surface is enough for now

> 3, HTTP will work good so far. Paca is the entire system the MCP is just a server part of the
> entire Paca monorepo.

### S7 — The work happens on its own worktree

> Open a new git worktree to working on it.

### S8 — The first slice is the one with no extra container

> Ok, let start working in the recommended one. ... Let start with A.

Option A, as recommended and accepted: **Jaeger v2 with the Badger backend** (persistent, with a
time-based retention), **no extra container**, and **no separate log database**: the logs stay on
the standard output, correlated with the traces.

---

## Tasks

Each task lists the specs it covers, the route it takes, and the commit that holds it. The
`Commit` column is filled in as the work units are committed.

| ID  | Specs   | Task                                                                                                                                                              | Route  | Commit |
|-----|---------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------|--------|
| T1  | S3, S4, S5, S8 | The `observability/` stack: Jaeger v2 with the Badger backend, the collector configuration owned by the repository, the Makefile and the documentation. | worker | `3945ac1` |
| T2  | S1, S4 | The `internal/platform/telemetry` package: the resource, the tracer provider, the sampler, the propagator and the shutdown.                                          | inline | `9ea6265` |
| T3  | S1     | The configuration: the strict validation of the OpenTelemetry environment, and the decision to trace or not.                                                       | inline | `6dc6d01` |
| T4  | S2, S6 | The HTTP instrumentation and the logs that carry the trace: the server spans, the exclusion of the health endpoint and the identifiers of the trace in the records. | inline | `9923802` |
| T5  | S1, S6 | The deployment of the server: the OTLP endpoint in the compose of the application, and the documentation of the environment.                                       | inline | `9923802` |
| T6  | S1, S3, S5 | The wiring of the monorepository: the root compose, the root Makefile, the reserved ports and the continuous integration.                                           | inline | `a9b490b` |

---

## Log

### L1 — The original request

> I wanna add tracing and proper logging to this project using OTel. I need you explore and explain me
> too how to proper handle it. I use to use Jaeger but I didn't no have clear the differences in
> logging and tracing and I wanna you deep web research how is the industry and the opensource
> community are handling the softweare tracing for AI Agents. I wanna first that you investigate and
> explore and give me your insigths

### L2 — The answers accepted by the user

> Ok 1, Backend Jaeger only
> 2: is better to have a our own collector rigth? Can be another apps project inside?
> 3, HTTP will work good so far. Paca is the entire system the MCP is just a server part of the entire Paca monorepo.
> 4: To be honest I'm not sure, help to me to clarify this point.

### L3 — The question that settled the storage

> is possible to have a kind of database for the logging with a clean time strategy? Only in memory
> loses when the server restart or something like that rigth? Limit to only answer

### L4 — The final instruction

> Ok, let start working in the recommended one. Open a new git worktree to working on it. Do you have
> now enough context to start. Let start with A.

### L5 — The decisions taken (research findings, 2026)

- **Jaeger v2 is a customized distribution of the OpenTelemetry Collector.** It reads the same YAML
  configuration format and ships the OTLP receiver, `batch`, `memory_limiter`, the `attributes` and
  `filter` processors, tail sampling, the `jaeger_storage_exporter` and the `jaeger_query`
  extension. The repository owns that configuration; there is no separate collector container.
- **Jaeger v2 has no logs receiver.** It stores traces and metrics. A Jaeger-only deployment has no
  destination for OTLP log records, so the logs of this slice stay on the standard output and the
  correlation is made by the `trace_id` inside the record.
- **The storage with a time strategy is Badger.** It is embedded in the Jaeger process, it survives
  the restarts when its directories are on a volume, and it has a native TTL
  (`jaeger_storage.backends.<name>.badger.ttl.spans`). It is single-node, which is what a
  development stack is.
- **The semantic conventions of the Model Context Protocol are not used in this slice.** They are
  `Development`, and the server registers no tool yet, so the HTTP surface is the whole signal.
- **The application keeps its network posture.** It stays on `paca_internal`, with no outbound
  access; the OTLP receiver is reachable on that network only.

### L6 — The verification plan
- The unit tests of every package of `internal/platform` and of the HTTP adapter, with an in-memory
  span exporter: the span name, the kind, the resource attributes, the exclusion of the health
  endpoint and the correlation of the logs.
- The quality gate of the module: `make -C apps/mcp-server all` (format, analyzers, linter, build,
  tests, race, coverage gate at 80%, benchmarks, vulnerabilities).
- The stack: `docker compose up --wait` with the observability stack, then a request against the MCP
  endpoint and a trace read back from the Jaeger query API.
- The differential check: the existing commands and the existing behaviour of the server, before and
  after the change.

### L7 — The boundaries of the work units

The work is committed in the units below. Two of them are bigger than the usual slice, and the
reason is in the gate itself:

| Unit | What it holds | Why it cannot be smaller |
|------|---------------|--------------------------|
| `3945ac1` | The `observability/` stack. | The stack is one unit: the compose file, the collector configuration, the Makefile and the documentation of the same thing. |
| `9ea6265` | The `internal/platform/telemetry` package and the dependencies. | The package and its tests belong together: a package without its tests fails the coverage gate of the module, which is one of the targets of `make all`. |
| the next ones | The configuration, the instrumentation, the wiring. | They are the ordinary slices. |

### L8 — The facts of the runtime that shaped the stack

Each one was measured, not assumed:

- **The v1 query API is gone in Jaeger v2.22.** The endpoint `/api/services` answers 404; the
  health check and the smoke test of the stack ask `/api/v3/services` instead.
- **Docker does not program the published ports of a container whose only networks are internal
  ones.** The `NetworkSettings.Ports` of such a container comes back empty, and the port of the
  interface is dead. The collector therefore joins a second network, private to the stack, that
  carries the published port of the interface. The price of it: the collector reaches the internet
  through that network. The alternative — a sealed collector and no interface on the host — is one
  line of the compose file away, and the trade-off is written down in the README of the stack.
- **The daemon creates the named volume as root, and the image runs as uid 10001.** The collector
  cannot write the traces unless it runs as root. It keeps no capability, and its file system is
  read-only: the traces are the only thing it writes.

### L9 — What was measured, end to end

The whole stack was started, a request was sent to the MCP transport, and the trace was read back
from the collector:

```
span: POST /mcp | kind: SERVER | trace: 6418e6fc6a8fd03a62a953215db510f3
resource: service.name=paca-mcp-server, deployment.environment.name=development
```

And the log of the same request, on the standard output of the container:

```json
{"level":"INFO","msg":"request","method":"POST","path":"/mcp","status":200,
 "trace_id":"6418e6fc6a8fd03a62a953215db510f3","span_id":"7be20952b526bd69"}
```

The identifier of the trace in the log is the identifier of the span in the collector: that is the
join between the two signals. The requests to the health endpoint carry no identifier, because they
are not traced.

### L10 — The pull request

The work was pushed on the branch `feat/observability-tracing` and opened as the pull request
[#22](https://github.com/witsaba/paca/pull/22), which closes the issue
[#21](https://github.com/witsaba/paca/issues/21). The branch was renamed from
`feature/observability-tracing` before the push, to the `type/description` form the repository uses
for every other branch.

The verification of the security of the stack was made by an independent verifier, on the live
stack: the verdict was that the claim holds, with three advisories that are now written down in the
README of the stack and in the configuration of the collector.
