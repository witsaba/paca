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
    inbound/http/              the HTTP adapter: the router of the service
    inbound/mcpserver/         the MCP adapter: the official Go SDK
    outbound/serving/          the checker of the MCP transport
    outbound/database/         the checker of the database
  platform/config/             the configuration of this service: the only
                               infrastructure the service owns
  bootstrap/                   the wiring of every layer
test/integration/              the container tests
```

The models and the infrastructure live in the shared modules: the person, the book of
their money and the health in [packages/domain](../../packages/domain), the health itself
— the use case that reports it and the endpoint that answers it — in
[packages/health](../../packages/health), and the clock, the logs, the telemetry and the
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

The server exposes seventeen tools: the way PACA onboards a person, the four
moments of the journey of a person inside PACA, the two tools of the
businesses of a person, the three tools of the book of their money, the
four tools of their debts and of the closing of their month, the two tools
of the utterance — the purchase the owner just said, and the purchase she
confirmed — and the tool of the catalog: the name she said, against the
names the business buys.

| Tool                      | What it does                                                    |
|---------------------------|-----------------------------------------------------------------|
| `complete_onboarding`     | Registers a person, opens their Personal Business with its accounts, and moves them to the active status. One call, one transaction. |
| `get_user`                | Reads the data of a person who is already registered.            |
| `list_users`              | Reads the people PACA knows, as many of them as the agent asks. |
| `change_user_status`      | Moves a user to another status, and says why when it needs it.  |
| `update_user_contact_data`| Writes the name, the phone and the email of a user.              |
| `open_business`           | Opens a business of a person, with the book of accounts it starts with. One call, one transaction. |
| `read_businesses`         | Reads the businesses of a person: the Personal Business and the others they own. |
| `record_movement`         | Writes a movement of money in the book: it goes from one account to another one. |
| `read_movements`          | Reads the movements of a period: the libro diario.              |
| `read_balances`           | Reads the balance of every account and the money of the business: the libro mayor. |
| `record_debt`             | Opens a debt with its counterparty and its due date, and writes the money that opened it. One call, one transaction. |
| `list_debts`              | Reads the debts the book still holds: what the person owes, to whom, and the day it falls due. |
| `record_debt_payment`     | Writes the payment of a debt, and refuses a payment that passes what the debt still holds. |
| `read_month_summary`      | Closes a month: the income, the expenses, the net, and the debts that were open at the end of it. |
| `draft_purchase`           | Proposes the purchase the owner just said and keeps it as a draft of the business, with what it understood, what it still needs, what it was not sure about and what it filled. |
| `confirm_purchase`         | Writes the purchase of the draft the owner confirmed and moves the money of the payment from the account that pays to the inventory. One call, one transaction. |
| `cancel_draft`             | Drops the draft the owner does not want: it moves to the state `cancelled` and it never waits again. It writes nothing in the book. |
| `read_drafts`              | Reads the pending drafts of a business, each one with what it still needs, what it was not sure about and what PACA filled, so that a lost conversation can be recovered. It writes nothing. |
| `match_items`             | Matches the name the owner just said against the catalog of the names the business buys: the ingredient it names, the options it offers, or the news that the catalog does not know it. It writes nothing. |

**There is one tool for the registration and the onboarding.** The tool of
the onboarding registers the person and opens their book in one call, and a
tool that registers a person and stops there is a tool the agent can call
halfway: it leaves a person in the onboarding with no business, and the
agent has to remember to come back.

**The description of a tool is not a comment.** It is the instruction the
agent reads before it calls the tool, so it says what the tool needs, what it
does with what it receives, and what it must not do. The description of
`complete_onboarding` tells the agent that the onboarding is ONE call and
ONE transaction, that the document ID and the name are required and that
PACA never invents the first, that the phone and the email are a favour to
ask for kindly, that the phone must carry its country code, that the email
must be a valid address, that the book of the person starts empty, that the
call is safe to repeat, and it names the accounts the book starts with — a
list read from the domain, never written a second time. The description of `change_user_status` tells the agent the moves the
map of the states allows and the rule of the observation. The description of
`update_user_contact_data` tells the agent that an empty field is a field
that stays as it is. The description of `get_user` tells the agent that the
tool answers with what PACA knows about the person, and the description of
`list_users` tells the agent to ask for the people it needs and to ask again
when the answer carries more people than it asked for.

**The business of a person has a book of its own.** The tool that opens a
business writes the business and the accounts of its book in one
transaction: a book that starts with half of its accounts lies about what
the owner has. The name of the business is required, the currency is
optional and it means the peso when the agent names none, and the kind of
the business is always `business` — the Personal Business of a person is
opened by the onboarding. The name of a business is taken inside the
businesses of its owner, and PACA refuses the duplicate. The description of
`open_business` names the accounts the book starts with and the kinds of a
business, and the description of `read_businesses` says that the tool
changes nothing: both lists are read from the domain and never written a
second time.

**The money goes from one account to another one.** The tool of the movement
takes the account the money comes from and the account it lands in, the
way the person says it: a salary lands in the cash and comes from the
salary. The amount is positive and it is never a fraction, the day is the
day the money moved — the day of today when the agent names none — and the
description is what the person reads in their book.

**The utterance is a draft, and a draft waits for the yes.** The tool
`draft_purchase` takes the business and the sentence of the owner verbatim,
with the lines the agent read from it: the name as she said it, the
quantity, the unit and the money. It writes a draft and nothing else — the
book of the business does not move until the owner says yes — and it answers
with the three parts of the answer of Paca: what it understood, what it did
not understand and what it still needs. The money is never inferred: a
price the agent filled is refused, and a line the owner did not price is
what the draft still needs. The quantity and its unit travel together, and
when the owner named both the money that was paid and the price of the
unit, the two must agree. The day is one of the three things Paca may fill
by herself, so a draft that names no day is the day of today and the answer
says so. A draft expires in seven days if the owner says nothing.

**The confirmation writes the purchase and moves the money.** The tool
`confirm_purchase` takes the business, the draft, the revision the owner
saw and the word she used to confirm it. The account that pays is optional:
the money comes from the cash of the business, and from the account the
owner names when she paid with another one. The purchase, its lines and the
movement of the money are written together or not at all: a book that holds
a purchase and no movement of its money lies about what the owner bought.
The confirmation is refused when the draft expired, when the draft changed
under the eyes of the owner, and when the draft still needs the money of one
of its lines: the money is never inferred. A refused confirmation leaves
nothing behind — the draft stays as it was and no purchase is written. The
answer carries the purchase that was written and the three parts of the
answer of Paca: what it recorded, what it did not understand and what it
still needs. The answer names the names the catalog of the business did not
know: a name the catalog does not know is not an error, it is recorded as
the owner said it, and the agent tells her that the name is new.

**The pending drafts can be dropped and read back.** The tool `cancel_draft`
takes the business and the draft, and it drops the draft the owner does not
want: the draft moves to the state `cancelled` and it never waits again. A
draft that is confirmed already and a draft that is cancelled already are
refused — they do not wait any more — and so is a draft that expired: the
price of a market does not wait. The tool `read_drafts` takes the business
and answers with the drafts that still wait for the yes of the owner, each
one with what it still needs, what it was not sure about and what PACA
filled by herself: a lost conversation is recovered from the book of the
business and not from the memory of the model. The day is optional and it
means the day of today when it is empty: the day decides which drafts are
still pending, because the expiry is computed when the draft is read. Both
tools change nothing in the book.

**The catalog of the names begins empty.** The tool `match_items` takes the
business and the name as the owner said it, and it answers with the status
of that name: `exact` when one ingredient answers it, `ambiguous` when
several answer it — with the options, so the owner chooses with one word —
and `unknown` when none of them does. A name the catalog does not know is
not an error: the answer says that it is new, and the confirmation of the
purchase records it as she said it. The tool writes nothing: the catalog
grows with the confirmation of a purchase, never with a read.

**The failures are answered in words the agent can act on.** A document that
is not a number, a phone without a country code, an address that is not an
address, a document that is already registered, an email that belongs to
somebody else, a move the map does not allow, an observation that does
not fit the status, a name that is missing, a business that is already
there, an account whose name is taken inside its business: each one says
what happened and what to do about it.

The rules themselves live in the [domain](../../packages/domain): the cleaning of
the phone, the check of the email, the map of the states and the rule of the
observation, the rule of the complete onboarding, the rule of the personal
business and the rule of the debt. The tools only carry them to the agent.

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
