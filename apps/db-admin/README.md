# The database admin service of the PACA monorepository.

The service is the **single writer of the schema**. It applies the pending
migrations of the [database library](../../packages/database) when it starts, and it stays
alive serving exactly **one** health endpoint: `/healthz`.

It does nothing else. There is no REST API and there is no gRPC surface yet:
the port `50051` stays reserved for the gRPC one, and the shape of the layers
lets it be added later without touching the domain.

The service is **private**. It publishes no port: the only things that reach
it are the other containers of the monorepository, through the `paca_internal`
network. The host and the internet do not.

The image of the service is built from the root of the monorepository, because
the service imports the shared libraries of the repository, and they are
modules of their own next to it.

---

## The architecture

The service is a hexagonal application. The rules live in the middle, the
protocol lives at the edge, and the composition root is the only place that
knows every layer at once.

```
cmd/paca-db-admin/          the binary, and the healthcheck command
internal/
  domain/health/            the health model: states, checks, reports
  application/               the use cases: migrate and health
  ports/                     the contracts between the layers
  adapters/
    inbound/http/            the HTTP adapter: the health endpoint
    outbound/database/       the migrator over the database library, the checker of the database
  platform/                  the infrastructure: config, logs, telemetry, clock, server
  bootstrap/                 the wiring of every layer
test/integration/            the container tests
```

The dependency rule holds: the domain knows nothing, the application knows
the domain and the ports, the adapters know the ports, and only the
composition root knows the adapters. Every layer is exercised through its
public interface, so a rule can be tested without a server.

## The database

**The service requires a database.** It is the writer of the schema, and a
writer without a database has nothing to write: without one the service
refuses to start, says why, and exits with a non-zero status.

This is the difference with the [MCP server](../mcp-server), which runs
without a database on purpose: its tools are still there, and every call says
that the database is not configured. The database admin service has no such
fallback.

The service imports the [database library](../../packages/database) of the
monorepository and it writes the schema through it: the migrator of the
library is the only thing that writes the schema, and the service never
writes a migration of its own. The library reads no environment variable —
the mapping of the environment onto the configuration lives here, in the
configuration of this service.

### The credentials

The credentials of the database live with the stack that owns them —
[`db/postgres/.env`](../../db/postgres) — and they are written **once**. The
compose file of the service loads that file with `env_file`, and the service
assembles the connection string out of it. The rules are, in this order:

1. `PACA_DATABASE_URL` — the whole connection string. When it is set, it
   wins, and **nothing else is read**: a hosted database, or a URL built by
   hand, needs nothing else.
2. Otherwise the connection string is assembled from the credentials of the
   PostgreSQL stack — `POSTGRES_USER`, `POSTGRES_PASSWORD` and `POSTGRES_DB`
   — with `PACA_DATABASE_HOST` (default `postgres`) and `PACA_DATABASE_PORT`
   (default `5432`).

Credentials that are half there are an error: the service says which of them
is missing and exits. An invalid value is an error, never a silent fallback.

### The migrations

The service applies the pending migrations of the schema when it starts, and
it does it through the migrator of the library, which takes the lock of the
database while it writes.

Applying the migrations is **idempotent**: a second start finds nothing to
do and says so in the log (`the schema was already up to date`). It is not an
error.

When the migrations fail, the service **does not start serving**: it logs
the failure and exits with a non-zero status. A service that did not write
the schema is not a writer.

### The wait for the database

The database of the monorepository and the writer that owns its schema start
at the same moment, and the database accepts the connections a couple of
seconds after its container starts. The writer **waits** for its database
instead of asking it once and exiting.

The difference matters: a container that exits is restarted by the runtime
of Docker, and the health check of the container declares it unhealthy
before the restart succeeds. The stack would then come up by a race, and the
race is lost more often than not.

The budget of the wait is `PACA_DB_ADMIN_DATABASE_WAIT_TIMEOUT` (60 seconds
by default). An invalid value is an error, never a silent fallback. The
start period of the health check of the container is longer than the budget,
so the container is never declared unhealthy while the writer waits.

## The endpoint

| Path       | Method | What it answers                                          |
|------------|--------|----------------------------------------------------------|
| `/healthz` | `GET`  | The health of the service: 200 while it serves, 503 when it does not. |

It is the only route of the service. The health endpoint answers a JSON body:

```json
{
  "status": "ok",
  "service": { "name": "paca-db-admin", "version": "0.1.0" },
  "uptime_seconds": 42.5,
  "checked_at": "2026-10-27T10:00:00Z",
  "checks": [ { "name": "database", "status": "ok", "latency_ms": 2 } ]
}
```

There is exactly **one** check: `database`. It reports whether the database
answers **and** whether the schema has no pending migration.

| The database…              | The check is | The service is | The endpoint answers |
|----------------------------|--------------|----------------|----------------------|
| answers, the schema is up to date | `ok`     | `ok`           | `200`                |
| answers, a migration is pending  | `degraded` | `degraded`  | `200`                |
| does not answer            | `unhealthy`  | `unhealthy`  | `503`                |

A pending migration degrades the service instead of breaking it: the service
still serves, but the schema is behind. A database that does not answer
breaks it: the service is the writer of the schema.

## The configuration

Everything comes from the environment. An invalid value is an error, never a
silent fallback.

| Variable                    | Default     | What it is                                            |
|-----------------------------|-------------|-------------------------------------------------------|
| `PACA_DB_ADMIN_HOST`        | `127.0.0.1` | The interface the server binds. The container overrides it with `0.0.0.0`. |
| `PACA_DB_ADMIN_PORT`        | `8082`      | The port reserved for the database admin service.      |
| `PACA_DB_ADMIN_CHECK_TIMEOUT` | `2s`      | The bound of a single health check.                   |
| `PACA_DB_ADMIN_SHUTDOWN_TIMEOUT` | `10s`   | The grace period of the shutdown.                     |
| `PACA_DB_ADMIN_DATABASE_WAIT_TIMEOUT` | `60s` | How long the writer waits for the database to answer when it starts. |
| `PACA_LOG_LEVEL`            | `info`      | The verbosity of the logs: `debug`, `info`, `warn`, `error`. |
| `PACA_SERVICE_VERSION`      | the version of the binary | The version the health report carries.   |
| `PACA_DATABASE_URL`         | *empty*     | The whole connection string of the database. When it is set, it wins and nothing else is read. |
| `PACA_DATABASE_HOST`        | `postgres`  | The host of the database, when the connection string is assembled. |
| `PACA_DATABASE_PORT`        | `5432`      | The port of the database, when the connection string is assembled. |
| `POSTGRES_USER`             | *empty*     | The role that owns the database.                      |
| `POSTGRES_PASSWORD`         | *empty*     | The password of the role.                              |
| `POSTGRES_DB`               | *empty*     | The application database.                              |

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
| `OTEL_SERVICE_NAME`               | `paca-db-admin` | The name the telemetry carries.                   |
| `OTEL_RESOURCE_ATTRIBUTES`        | *empty*     | The attributes of the deployment, as `key=value` pairs. |

### The logs and the traces

The logs go to the standard output, as JSON. Every record carries the
`trace_id` and the `span_id` of the trace it belongs to, when the request
that wrote it was inside a trace. That identifier is the join between the
two: paste it into the search of the trace backend, and the whole request
the line belongs to shows up.

```json
{"time":"2026-10-27T10:00:00Z","level":"INFO","msg":"request","method":"GET",
 "path":"/healthz","status":200,"duration":1200000,
 "trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7"}
```

The traces themselves live in the [observability stack](../../observability),
which keeps them for two days.

## Running it

The service needs a database. The quickest way to give it one is the stack
of PostgreSQL of the monorepository:

```sh
make -C apps/db-admin build          # build the binary into bin/
make -C apps/db-admin run            # run the service on 127.0.0.1:8082
```

The service reads the credentials of the database from the environment, the
same way the container does:

```sh
set -a; . db/postgres/.env; set +a
export PACA_DATABASE_HOST=postgres PACA_DATABASE_PORT=5432
make -C apps/db-admin run
```

Or as a container, with the stack of the monorepository:

```sh
make -C apps/db-admin compose-up     # start the stack and wait until it is healthy
make -C apps/db-admin smoke          # ask the running service whether it is alive
make -C apps/db-admin compose-down   # stop the stack
```

The stack of the service runs on its own too:

```sh
docker compose -f apps/db-admin/compose.yaml up --build -d --wait
```

It needs the database of the monorepository: the stacks share the network
`paca_internal`, so the PostgreSQL of the repository answers at
`postgres:5432` when it runs. Start it first (`make db-up`).

There is no `depends_on` on the other stacks: a cross-file dependency only
resolves inside the merged project, and a stack that must run on its own
cannot depend on a stack it does not include. The service waits for the
database by itself: it fails with a clear message when the database does not
answer, and the `restart: unless-stopped` policy of the container brings it
back.

## The quality gate

```sh
make -C apps/db-admin all
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

### The coverage gate

The gate measures the packages whose rules can be asserted **without a
database**: the domain, the application layer, the adapters of the HTTP
protocol, the configuration, and the infrastructure. It leaves two things
out, and the container tests are what prove them:

- the composition root (`internal/bootstrap`) and the binary
  (`cmd/paca-db-admin`): they open the database, apply the migrations and
  serve;
- the adapter that applies the migrations
  (`internal/adapters/outbound/database/migrator.go`): it writes the schema,
  and the schema only exists in a real database.

## The tests

The unit tests live with the code they cover. The container tests live in
`test/integration` and need Docker:

```sh
make -C apps/db-admin test-integration
```

They build the image of the service from its Dockerfile, start a PostgreSQL,
and check that the service applies the migrations of the schema, that the
table of the users is in place, that the health endpoint answers, that a
second run applies nothing, that the service names the database with the
whole connection string as well, and that a service without a database
refuses to start.

## The security of the slice

- The service is private: no published port, only the `paca_internal`
  network, which has no outbound access.
- The image runs as a non-root user, on a read-only file system, without any
  capability, with `no-new-privileges`, on a distroless base that has no
  shell and no package manager.
- The credentials of the database reach the service through the `env_file`
  of the stack that owns them, and they are never written twice.
- The headers of the responses are the ones of the browser: no sniffing, no
  framing, no referrer.
- The body of a request is never read: the service has no route that takes
  one, and the headers are bounded and their read is timed out.
- There is no authentication yet: the network is the boundary. The token and
  the TLS come with the first feature that needs them.

## The versions

| What                | Version  | Why                                                     |
|---------------------|----------|---------------------------------------------------------|
| Go                  | 1.27.3   | The latest security release of the 1.27 line (2026-10-19). The toolchain of the repository is 1.27.3. |
| golangci-lint       | v2.15.0  | Pinned in `tools/go.mod`.                               |
| govulncheck         | v1.8.0   | Pinned in `tools/go.mod`.                               |
| testcontainers-go   | v0.44.0  | The container tests.                                    |
| OpenTelemetry Go    | v1.47.0  | The tracing: the SDK, the OTLP exporter and the instrumentation of the transport. |
| otelhttp            | v0.72.0  | The instrumentation of the HTTP transport.               |
