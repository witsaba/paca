# The MCP server of the PACA monorepository.

The server speaks the [Model Context Protocol](https://modelcontextprotocol.io)
over the official Go SDK, and answers a health endpoint. **It registers no tool
yet**: this is the initialization of the server, not its first feature.

The service is **private**. It publishes no port: the only things that reach it
are the other containers of the monorepository, through the `paca_internal`
network. The host and the internet do not.

---

## The architecture

The server is a hexagonal application. The rules live in the middle, the
protocol lives at the edge, and the composition root is the only place that
knows every layer at once.

```
cmd/paca-mcp-server/          the composition root: it wires the layers
internal/
  domain/health/              the health model: states, checks, reports
  application/                 the use cases: the rules of the service
  ports/                       the contracts between the layers
  adapters/
    inbound/http/              the HTTP adapter: the health endpoint
    inbound/mcpserver/         the MCP adapter: the official Go SDK
    outbound/serving/          the checker of the MCP transport
  platform/                    the infrastructure: config, logs, telemetry, clock, server
  bootstrap/                   the wiring of every layer
test/integration/              the container tests
```

The dependency rule holds: the domain knows nothing, the application knows
the domain and the ports, the adapters know the ports, and only the
composition root knows the adapters. Every layer is exercised through its
public interface, so a rule can be tested without a server.

## The endpoints

| Path       | Method            | What it answers                                          |
|------------|-------------------|----------------------------------------------------------|
| `/healthz` | `GET`             | The health of the service: 200 while it serves, 503 when it does not. |
| `/mcp`     | `POST` (`GET`, `DELETE`) | The MCP transport, over the streamable HTTP transport. |

The health endpoint answers a JSON body:

```json
{
  "status": "ok",
  "service": { "name": "paca-mcp-server", "version": "0.1.0" },
  "uptime_seconds": 42.5,
  "checked_at": "2026-10-27T10:00:00Z",
  "checks": [ { "name": "mcp-transport", "status": "ok", "latency_ms": 0 } ]
}
```

## The configuration

Everything comes from the environment. An invalid value is an error, never a
silent fallback.

| Variable                    | Default     | What it is                                            |
|-----------------------------|-------------|-------------------------------------------------------|
| `PACA_MCP_HOST`             | `127.0.0.1` | The interface the server binds. The container overrides it with `0.0.0.0`. |
| `PACA_MCP_PORT`             | `8081`      | The port reserved for the MCP server.                 |
| `PACA_MCP_SHUTDOWN_TIMEOUT` | `10s`       | The grace period of the shutdown.                     |
| `PACA_MCP_CHECK_TIMEOUT`    | `2s`        | The bound of a single health check.                   |
| `PACA_MCP_MAX_REQUEST_BYTES`| `4MiB`      | The largest body the MCP transport accepts.           |
| `PACA_LOG_LEVEL`            | `info`      | The verbosity of the logs: `debug`, `info`, `warn`, `error`. |
| `PACA_SERVICE_VERSION`      | the version of the binary | The version the clients see.             |

### The tracing

The service traces its requests with [OpenTelemetry](https://opentelemetry.io)
and sends the traces to a collector over OTLP. The settings are the ones of
the OpenTelemetry specification, so an operator who knows the specification
knows how to turn the tracing on and off.

| Variable                          | Default     | What it is                                            |
|-----------------------------------|-------------|-------------------------------------------------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT`     | *empty*     | The address of the collector: `host:port`. The tracing is **off** without it: there is nowhere to send the traces. |
| `OTEL_EXPORTER_OTLP_PROTOCOL`     | `grpc`      | The protocol of the exporter: `grpc` or `http/protobuf`. |
| `OTEL_EXPORTER_OTLP_INSECURE`     | `false`     | Whether the exporter speaks to the collector without TLS. |
| `OTEL_SDK_DISABLED`               | `false`     | Turns the tracing off.                                 |
| `OTEL_TRACES_SAMPLER`             | `parentbased_always_on` | The sampler: `always_on`, `always_off`, `traceidratio`, `parentbased_always_on`, `parentbased_traceidratio`. |
| `OTEL_TRACES_SAMPLER_ARG`         | *empty*     | The ratio of the traces that are kept, for the samplers that take one. |
| `OTEL_SERVICE_NAME`               | `paca-mcp-server` | The name the telemetry carries.                  |
| `OTEL_RESOURCE_ATTRIBUTES`        | *empty*     | The attributes of the deployment, as `key=value` pairs. |

An invalid value is an error, never a silent fallback: a server that starts
with a collector it cannot reach, or with a sampler it cannot read, is a
server that nobody asked for.

### The logs and the traces

The logs go to the standard output, as JSON. Every record carries the
`trace_id` and the `span_id` of the trace it belongs to, when the request
that wrote it was inside a trace. That identifier is the join between the
two: paste it into the search of the trace backend, and the whole request
the line belongs to shows up.

A record written outside of a request carries no identifier: there is no
trace to point at.

```json
{"time":"2026-10-27T10:00:00Z","level":"INFO","msg":"request","method":"POST",
 "path":"/mcp","status":200,"duration":1200000,
 "trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7"}
```

The traces themselves live in the [observability stack](../../observability),
which keeps them for two days.

## Running it

```sh
make -C apps/mcp-server build          # build the binary into bin/
make -C apps/mcp-server run            # run the server on 127.0.0.1:8081
```

Or as a container, with the stack of the monorepository:

```sh
make -C apps/mcp-server compose-up     # start the stack and wait until it is healthy
make -C apps/mcp-server smoke          # ask the running server whether it is alive
make -C apps/mcp-server compose-down   # stop the stack
```

## The quality gate

```sh
make -C apps/mcp-server all
```

The gate runs, in this order: the format check, the analyzers of the
toolchain, the linter, the build, the test suite, the race detector, the
coverage gate and the vulnerability scan.

| Target            | What it does                                                     |
|-------------------|------------------------------------------------------------------|
| `make deps`       | Downloads and verifies the dependencies.                          |
| `make tidy-check` | Fails when `go.mod` is not tidy.                                 |
| `make fmt-check`  | Fails when a source is not formatted.                             |
| `make vet`        | Runs the analyzers of the toolchain.                              |
| `make lint`       | Runs `golangci-lint`, with `gosec` among the linters.             |
| `make build`      | Builds the binary into `bin/`.                                    |
| `make test`       | Runs the test suite.                                              |
| `make test-integration` | Builds the image, starts the container and talks to it.     |
| `make cover`      | Measures the coverage and **fails below 80%**.                    |
| `make race`       | Runs the test suite with the race detector.                      |
| `make bench`      | Runs the benchmarks.                                              |
| `make perf`       | Checks the allocation budgets, then runs the benchmarks.          |
| `make vuln`       | Runs `govulncheck` over the dependencies.                         |

The tools are pinned in the repository (`tools/go.mod`), so `go tool` runs the
same version everywhere. A contributor needs nothing but the Go toolchain.

## The tests

The unit tests live with the code they cover. The container tests live in
`test/integration` and need Docker:

```sh
make -C apps/mcp-server test-integration
```

They build the image of the server from its Dockerfile, start it, and check
that it comes up, that it answers its health, that it speaks the MCP
protocol, and that it keeps its transport to itself.

## The security of the slice

- The server is private: no published port, only the `paca_internal`
  network, which has no outbound access.
- The image runs as a non-root user, on a read-only file system, without any
  capability, with `no-new-privileges`, on a distroless base that has no
  shell and no package manager.
- The DNS-rebinding protection of the MCP SDK stays on: a request that
  arrives through the loopback interface with a foreign `Host` header is
  rejected.
- The cross-origin protection of the standard library stays on: a
  cross-origin request from a browser is rejected, and the requests of a
  client that is not a browser are let through.
- The body of a request is bounded, the headers are bounded, and the header
  read is timed out.
- There is no authentication yet: the network is the boundary. The token and
  the TLS come with the first tool that needs them.

## The versions

| What                | Version  | Why                                                     |
|---------------------|----------|---------------------------------------------------------|
| Go                  | 1.27.3   | The latest security release of the 1.27 line (2026-10-19). The local toolchain of the repository is 1.26.3, which is below the security release 1.26.10. |
| MCP Go SDK          | v1.8.0   | The official SDK, with the protocol revision `2026-07-28`. |
| golangci-lint       | v2.15.0  | Pinned in `tools/go.mod`.                               |
| govulncheck         | v1.8.0   | Pinned in `tools/go.mod`.                               |
| testcontainers-go   | v0.44.0  | The container tests.                                    |
| OpenTelemetry Go    | v1.47.0  | The tracing: the SDK, the OTLP exporter and the instrumentation of the transport. |
| otelhttp            | v0.72.0  | The instrumentation of the HTTP transport.               |
