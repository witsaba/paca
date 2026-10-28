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

## The contract of the agents

[`AGENTS.md`](AGENTS.md) holds the contract of the architecture: the layers and the
direction of the dependencies, the ownership of the rules of the domain, the owner of
every model, and the checks of SOLID. It is written for a coding agent, and it is the
canonical entry of the instructions of the repository. The entry file of each module
(`packages/*/AGENTS.md`, `apps/*/AGENTS.md`) adds the specifics of that module.

## What is in the repository now

| Directory          | What it is                                        | README |
|--------------------|---------------------------------------------------|--------|
| `apps/mcp-server/` | The MCP server: the Model Context Protocol over the official Go SDK, with a health endpoint and three tools for the people who use PACA. | [apps/mcp-server/README.md](apps/mcp-server/README.md) |
| `apps/db-admin/`   | The database admin: the single writer of the schema. It applies the pending migrations when it starts, then stays alive with one health endpoint. | [apps/db-admin/README.md](apps/db-admin/README.md) |
| `packages/database/` | The database library: the pool, the queries, the migrations and the repositories that write the rows. A library, with no port and no container. | [packages/database/README.md](packages/database/README.md) |
| `packages/domain/` | The domain: the models of the people and the port of their storage. A library that imports nothing but the standard library. | [packages/domain/README.md](packages/domain/README.md) |
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

The root `Makefile` drives the same project, and the targets of the whole
stack are the ones the continuous integration calls:

```sh
make stack-up        # start every stack and wait until the whole project is healthy
make stack-schema    # assert that the schema was written by the service that owns it
make stack-down      # stop every container of the project, keeping the named volumes
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
| `8082`| Database admin| The containers of `paca_internal`, only.            |
| `4317`| Jaeger (OTLP) | The containers of `paca_internal`, only.            |
| `4318`| Jaeger (OTLP) | The containers of `paca_internal`, only.            |
| `16686`| Jaeger UI    | The loopback interface of the host, only.           |
| `8080`| *reserved*    | For the REST backend.                               |
| `50051`| *reserved*   | For the gRPC backend.                               |

The MCP server publishes no port at all. It binds the interface of its own
container and sits on `paca_internal`, so the other containers of the stack
reach it and nothing else does. The health endpoint shares that surface.

The database admin publishes no port either. Its only route is the health
endpoint, and it serves the same purpose: the stack asks the container
whether it is alive, from inside the container.

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

## The schema of the database

The schema has **one writer**, and it is the `db-admin` service. It applies
the pending migrations of `packages/database/migrations/` when it starts, and it
says so in its log. A service never migrates on boot: a service that
migrates on boot races with the other replicas of itself.

The service runs in the container, on the internal network, and it needs the
credentials of the PostgreSQL stack to reach the database. The credentials
live with the stack that owns them (`db/postgres/.env`); the services that
talk to the database load that file and assemble the connection string from
it.

## The quality gate

Every Go module brings its own Makefile, with the same targets: the format
check, the analyzers, the linter, the build, the tests, the race detector,
the coverage gate, the benchmarks and the vulnerability scan.

```sh
make -C apps/mcp-server all
make -C packages/domain all
make -C packages/database all
```

The tools are pinned in `tools/go.mod` and run with `go tool`, so every
machine and every container uses the same versions.

## The gate of the repository

`make all` is the one command that proves the repository: it runs every
check the continuous integration runs, and it runs them **from nothing** —
no image, no container, no volume and no `.env` are needed.

```sh
make all
```

It writes the environment it needs, it runs the quality gate of every
application and every package, the container tests of the packages, the
container tests of the applications, the shellcheck of the scripts, the
tests of the installer of the environment, the smoke test of every
database, the smoke test of the collector of the traces, and finally the
whole stack: it comes up, the schema is asserted, and it goes down.

A step that fails stops the gate, and the exit code says so. The gate
leaves no container behind: the containers it started are stopped when it
ends, whether it passed or failed.

The last step of the gate is the proof of the stack. `make stack-up`
brings the whole monorepository up and waits until it is healthy, and
`make stack-schema` asserts, against the running project, that every
container is running and healthy, that the writer of the schema was not
restarted (it waits for its database instead of exiting and being
restarted), that every migration of `packages/database/migrations` is applied,
and that the table `users` exists. The script behind it is
[`scripts/stack-schema.sh`](scripts/stack-schema.sh).

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
make stack-up           # the whole monorepository, up and healthy
make stack-schema       # the schema of the database was written by the service that owns it
make stack-down         # stop every container of the project, keeping the named volumes
make all                # the full gate of the repository, from nothing
```

The workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on
every push to `main` and on every pull request, with five jobs:

| Job            | What it verifies                                                    |
|----------------|---------------------------------------------------------------------|
| `gate`         | fmt-check, vet, lint, build, test, race, the coverage floor and the vulnerability scan of every application and every package, and then the container tests of the shared libraries: the rules of the tables, asserted against a real PostgreSQL. |
| `database`     | the image of the stack builds, the cluster starts, and the smoke test asserts UTF-8, UTC, UUID, vector search, checksums and isolation. |
| `observability`| the collector of the traces starts, and its query API answers.       |
| `stack`        | the whole monorepository comes up from nothing, and the schema of the database is there: it was written by the service that owns it. |
| `shellcheck`   | the shell scripts of the repository pass shellcheck.                  |

The container tests of the MCP server (`make -C apps/mcp-server
test-integration`) stay a local gate: they need Docker and build the image
of the server, so they are not part of the workflow.
