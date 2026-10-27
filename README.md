# PACA

The monorepository of PACA: the databases, the services and the tools that
support them, in one repository and one compose project.

## The layout

| Directory     | What it holds                                                              |
|---------------|----------------------------------------------------------------------------|
| `apps/`       | The services, one directory and one Go module each.                        |
| `packages/`   | The libraries the services share, one directory and one Go module each.     |
| `db/`         | The databases, one directory and one stack each.                           |
| `observability/` | The traces: the collector and the backend that keeps them.              |
| `tools/`      | The developer tools, pinned as Go tool dependencies.                        |
| `scripts/`    | The scripts of the repository: the installer of the environment.            |
| `odd/tasks/`  | The feature documents: the specification of every piece of work.           |

The repository is a Go workspace (`go.work`), so the modules are built and
tested against each other, and every module keeps working on its own.

## What is in the repository now

| Directory          | What it is                                        | README |
|--------------------|---------------------------------------------------|--------|
| `apps/mcp-server/` | The MCP server: the Model Context Protocol over the official Go SDK, with a health endpoint and no tool yet. | [apps/mcp-server/README.md](apps/mcp-server/README.md) |
| `packages/core/`  | The core: the models, the repositories and the migrations that the services share. A library, with no port and no container. | [packages/core/README.md](packages/core/README.md) |
| `db/postgres/`     | PostgreSQL 18.6 with pgvector 0.8.7, internal only. | [db/postgres/README.md](db/postgres/README.md) |
| `observability/`   | Jaeger 2 as the OpenTelemetry collector and the trace backend: the traces of the services, kept for two days. | [observability/README.md](observability/README.md) |

## The installation

The stacks need their environment: the credentials of the database, and
whatever a stack declares in its own manifest. Every stack that ships an
`env.example` gets its `.env` written by the installer of the environment:

```sh
make env-init
```

It asks one question per value and offers the default of each one: the value
already configured when the file exists, the value of the manifest otherwise.
The empty answer keeps the default, any other answer replaces it, and `g`
generates a strong password for the secrets.

Running it again keeps what is configured, so a value can be changed at any
time without losing the rest of the file. The secrets are never printed, and
the files that hold them are written for the owner alone.

The installer also works without a terminal, for a scripted installation and
for the continuous integration:

```sh
make env-init ARGS="--yes"       # keep what is configured, generate the rest
make env-init ARGS="--dry-run"    # print what it would write, write nothing
./scripts/setup-env.sh --help     # every option
```

The `.env` files are never committed. The manual way, with an editor, keeps
working: copy the example of the stack and change the values.

## Running the stacks

Every stack keeps its own compose file; the root one includes them all, and
declares the networks they share.

```sh
docker compose up --wait                    # every stack of the monorepository
docker compose up -d --wait mcp-server      # one service of it
docker compose down                         # stop everything
```

A single stack also runs on its own, from its own directory:

```sh
make -C apps/mcp-server compose-up
make -C apps/mcp-server smoke
```

## The networks

| Network         | Attribute        | Purpose                                                                 |
|-----------------|------------------|-------------------------------------------------------------------------|
| `paca_internal` | `internal: true` | Service-to-service traffic. A container on it has **no outbound access at all**. |
| `paca_egress`   | plain bridge     | Outbound traffic for the services that need the internet.                |

A service that only talks to the other services joins `paca_internal` alone.
A service that needs the internet joins `paca_egress` as well.

## The ports

The ports of the monorepository are reserved, so a new service never takes
the port of another one.

| Port  | Service       | Reachable from                                     |
|-------|---------------|----------------------------------------------------|
| `5432`| PostgreSQL    | The containers of `paca_internal`, only.            |
| `8081`| MCP server    | The containers of `paca_internal`, only.            |
| `4317`| Jaeger (OTLP) | The containers of `paca_internal`, only.            |
| `4318`| Jaeger (OTLP) | The containers of `paca_internal`, only.            |
| `16686`| Jaeger UI    | The loopback interface of the host, only.           |
| `8080`| *reserved*    | For the REST backend.                               |
| `50051`| *reserved*   | For the gRPC backend.                               |

The MCP server publishes no port at all. It binds the interface of its own
container and sits on `paca_internal`, so the other containers of the stack
reach it and nothing else does. The health endpoint shares that surface.

The interface of Jaeger is the only exception: it is published on the
loopback interface of the host, and on nothing else. It is a surface for the
developer who reads the traces, not a surface of the network.

## The observability

The services trace themselves with [OpenTelemetry](https://opentelemetry.io)
and send the traces to the collector of the `observability/` stack. The
collector is Jaeger 2, which keeps the traces for two days in an embedded
database and serves the interface that reads them.

```sh
make observability-up      # start the collector and wait until it is healthy
make observability-smoke   # ask the collector whether it answers
make observability-down    # stop it
```

The interface of the traces is at <http://127.0.0.1:16686>.

The logs of the services stay on the standard output, where the runtime of
the containers collects them. Every log record carries the identifiers of
the trace it belongs to, so a line of the log opens the whole request it
comes from.

## The quality gate

Every Go module brings its own Makefile, with the same targets: the format
check, the analyzers, the linter, the build, the tests, the race detector,
the coverage gate, the benchmarks and the vulnerability scan.

```sh
make -C apps/mcp-server all
make -C packages/core all
```

The tools are pinned in `tools/go.mod` and run with `go tool`, so every
machine and every container uses the same versions.

## The continuous integration

The root `Makefile` dispatches to the stacks; it holds no stack-specific
logic. It is the entry point of the continuous integration, so the workflow
never names an application or a database.

```sh
make gate               # the quality gate of every application and every package
make packages-integration  # the container tests of every package
make shellcheck         # the shell scripts of the repository
make db-up              # every database, up and healthy
make db-smoke           # the smoke test of every database
make db-down            # stop every database, keeping the data volumes
make all                # the full gate: the applications, the shellcheck and the databases
```

The workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on
every push to `main` and on every pull request, with four jobs:

| Job            | What it verifies                                                    |
|----------------|---------------------------------------------------------------------|
| `gate`         | fmt-check, vet, lint, build, test, race, the coverage floor and the vulnerability scan of every application and every package, and then the container tests of the shared libraries: the rules of the tables, asserted against a real PostgreSQL. |
| `database`     | the image of the stack builds, the cluster starts, and the smoke test asserts UTF-8, UTC, UUID, vector search, checksums and isolation. |
| `observability`| the collector of the traces starts, and its query API answers.       |
| `shellcheck`   | the shell scripts of the repository pass shellcheck.                  |

The container tests of the MCP server (`make -C apps/mcp-server
test-integration`) stay a local gate: they need Docker and build the image
of the server, so they are not part of the workflow.
