# The MCP server of the PACA monorepository.

The server speaks the [Model Context Protocol](https://modelcontextprotocol.io)
over the official Go SDK, answers a health endpoint, and exposes **three
tools**: the ones that take care of the people who use PACA.

The service is **private**. It publishes no port: the only things that reach it
are the other containers of the monorepository, through the `paca_internal`
network. The host and the internet do not.

The image of the server is built from the root of the monorepository, because
the server imports the shared libraries of the repository, and they are
modules of their own next to it.

---

## The architecture

The server is a hexagonal application. The rules live in the middle, the
protocol lives at the edge, and the composition root is the only place that
knows every layer at once.

```
cmd/paca-mcp-server/          the composition root: it wires the layers
internal/
  application/                 the use cases: the rules of the service
  ports/                       the contracts between the layers
  adapters/
    inbound/http/              the HTTP adapter: the health endpoint
    inbound/mcpserver/         the MCP adapter: the official Go SDK
    outbound/serving/          the checker of the MCP transport
    outbound/database/         the checker of the database
  platform/config/             the configuration of this service: the only
                               infrastructure the service owns
  bootstrap/                   the wiring of every layer
test/integration/              the container tests
```

The models and the infrastructure live in the shared modules: the person and the health
in [packages/domain](../../packages/domain), the clock, the logs, the telemetry and the
server in [packages/platform](../../packages/platform). The service owns only the models
that belong to it alone, and the configuration that is its own.

The dependency rule holds: the domain knows nothing, the application knows
the domain and the ports, the adapters know the ports, and only the
composition root knows the adapters. Every layer is exercised through its
public interface, so a rule can be tested without a server.

## The database

The server imports the [database library](../../packages/database) and the
[domain](../../packages/domain) of the monorepository, and it reads and writes
the people through them: the pool, the driver and the queries belong to the
database library, the rules of the person belong to the domain, and the
server never names them.

The database is **still optional**. The server starts without one, the three
tools are still there, and every call answers that the database is not
configured. When the configuration names one, the health report carries a
`database` check, and a database that does not answer makes the service
unhealthy.

The pool is lazy: a database that is down does not stop the server from
starting. The health check is what asks the database whether it answers.

**The stack points the server at the database.** The credentials live with
the stack that owns them — `db/postgres/.env` — and the compose of the
server loads that file with `env_file`, so they are written once and only
once. The server assembles the connection string itself, because Docker
Compose does not interpolate `${VAR}` from the `env_file` of a service into
its `environment` block.

The connection string is resolved in this order:

1. `PACA_DATABASE_URL` — the whole connection string. When it is set, it
   wins, and nothing else is read.
2. Otherwise the connection string is assembled from the credentials of the
   PostgreSQL stack — `POSTGRES_USER`, `POSTGRES_PASSWORD` and
   `POSTGRES_DB` — with `PACA_DATABASE_HOST` (default `postgres`) and
   `PACA_DATABASE_PORT` (default `5432`):
   `postgres://<user>:<password>@<host>:<port>/<database>?sslmode=disable`.
   The user and the password are escaped the way a URL carries them, so a
   password with an `@` or a `/` in it does not break the connection string.
3. When neither of them is there, the server runs **without a database**, and
   its health report says nothing about it. Credentials that are half there
   are ignored the same way: the server runs without a database and says so
   in its log. A variable that is there and holds something invalid is
   still an error, never a silent fallback.

The URL names the container, so it is the same URL the host does not reach:
the database publishes no port, and only the containers of the internal
network do. To point the server at a database by hand, name it whole:

```sh
export PACA_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"
docker compose up -d --wait
```

## The tools

The server exposes five tools: the three moments of the journey of a person
inside PACA, and the two reads that tell the agent what PACA knows.

| Tool                      | What it does                                                    |
|---------------------------|-----------------------------------------------------------------|
| `onboard_user`            | Registers a person and starts their onboarding.                  |
| `get_user`                | Reads the data of a person who is already registered.            |
| `list_users`              | Reads the people PACA knows, as many of them as the agent asks. |
| `change_user_status`      | Moves a user to another status, and says why when it needs it.  |
| `update_user_contact_data`| Writes the name, the phone and the email of a user.              |

**The description of a tool is not a comment.** It is the instruction the
agent reads before it calls the tool, so it says what the tool needs, what it
does with what it receives, and what it must not do. The description of
`onboard_user` tells the agent that the document ID is required and that
PACA never invents one, that the name is optional but worth asking for
kindly, that the phone must carry its country code, that the email must be a
valid address, and that it should gather as much as the person is willing to
give. The description of `change_user_status` tells the agent the moves the
map of the states allows and the rule of the observation. The description of
`update_user_contact_data` tells the agent that an empty field is a field
that stays as it is. The description of `get_user` tells the agent that the
tool answers with what PACA knows about the person, and the description of
`list_users` tells the agent to ask for the people it needs and to ask again
when the answer carries more people than it asked for.

**The failures are answered in words the agent can act on.** A document that
is not a number, a phone without a country code, an address that is not an
address, a document that is already registered, an email that belongs to
somebody else, a move the map does not allow, and an observation that does
not fit the status: each one says what happened and what to do about it.

The rules themselves live in the [domain](../../packages/domain): the cleaning of
the phone, the check of the email, the map of the states and the rule of the
observation. The tools only carry them to the agent.

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
| `PACA_MCP_ALLOW_NON_LOOPBACK` | `false`  | The server may bind an address that is not the loopback interface. The container sets it: it is the deployment where the other containers of the stack reach the server. |
| `PACA_MCP_ALLOWED_HOSTS`    | `localhost,127.0.0.1,[::1]` | The hosts the server answers for. The `Host` header of a request that names another one is refused. The container names itself. |
| `PACA_MCP_TRUSTED_ORIGINS`  | *empty*     | The origins the server trusts. The rule of the standard library — the same origin as the `Host` header — is the rule when the list is empty. |
| `PACA_LOG_LEVEL`            | `info`      | The verbosity of the logs: `debug`, `info`, `warn`, `error`. |
| `PACA_SERVICE_VERSION`      | the version of the binary | The version the clients see.             |
| `PACA_DATABASE_URL`         | *empty*     | The connection string of the database. When it is set, it wins and nothing else is read. Empty means the server runs without one. |
| `PACA_DATABASE_MAX_CONNS`   | `4`         | The number of connections the pool of the database may open. |
| `PACA_DATABASE_CONNECT_TIMEOUT` | `5s`    | The bound of the attempt to open a connection. |
| `PACA_DATABASE_HOST`        | `postgres`  | The host the database answers on. It is only read when the connection string is assembled from the credentials. |
| `PACA_DATABASE_PORT`        | `5432`      | The port the database answers on. It is only read when the connection string is assembled from the credentials. |
| `POSTGRES_USER`             | *empty*     | The role that owns the application database. It travels from `db/postgres/.env` with the `env_file` of the compose. |
| `POSTGRES_PASSWORD`         | *empty*     | The password of the role. It travels from `db/postgres/.env` with the `env_file` of the compose. |
| `POSTGRES_DB`               | *empty*     | The application database. It travels from `db/postgres/.env` with the `env_file` of the compose. |

Without `PACA_DATABASE_URL` and without the credentials of the PostgreSQL
stack, the server runs without a database. Credentials that are half there
are ignored: the server runs without a database and says so in its log.

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

The stack of the server loads the credentials of the PostgreSQL stack, so it
expects a database to answer at `postgres:5432`: the database of the
monorepository, on the network the stacks share. Start the database first
(`make db-up`) or start the whole monorepository (`docker compose up
--wait`). Without a database the server still serves, and its health report
says that the database does not answer.

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
- The server **refuses a bind that is not the loopback interface**, unless
  the operator allows it with `PACA_MCP_ALLOW_NON_LOOPBACK`. The tools
  carry the data of the people, and a bind that the network of the machine
  reaches is a bind that anybody on the network reaches. The container is
  the deployment where the other containers of the stack reach the server,
  over a network that has no route to the internet, and the compose file of
  the service says so.
- The server **answers only for the hosts of its allowlist**
  (`PACA_MCP_ALLOWED_HOSTS`): the `Host` header of a request that names
  another host is refused with a `403`, and the refusal never reaches the
  tools. The default is the loopback, and the container names itself.
- The cross-origin protection of the standard library stays on, and its rule
  is **stated and not left to the default**: the service trusts the origins
  of `PACA_MCP_TRUSTED_ORIGINS`, and the rule of the standard library — the
  same origin as the `Host` header — is the rule when the list is empty. A
  cross-origin request from a browser is rejected, and the requests of a
  client that is not a browser are let through.
- The body of a request is bounded, the headers are bounded, and the header
  read is timed out.
- There is no authentication yet: the network is the boundary. The token and
  the TLS come with the first tool that needs them.
- The five tools carry **personal data**: the document, the name, the phone
  and the email of a person. The boundary has not moved — it is still the
  network — and the hardening that was proposed for it (refusing a bind that
  is not loopback, a Host allowlist and the explicit Origin rule) is closed.
  There is still no authentication: the token and the TLS come with the
  first tool that needs them.

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
