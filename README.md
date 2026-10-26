# PACA

The monorepository of PACA: the databases, the services and the tools that
support them, in one repository and one compose project.

## The layout

| Directory     | What it holds                                                              |
|---------------|----------------------------------------------------------------------------|
| `apps/`       | The services, one directory and one Go module each.                        |
| `packages/`   | *Reserved* — the libraries the services will share.                        |
| `db/`         | The databases, one directory and one stack each.                           |
| `tools/`      | The developer tools, pinned as Go tool dependencies.                        |
| `odd/tasks/`  | The feature documents: the specification of every piece of work.           |

The repository is a Go workspace (`go.work`), so the modules are built and
tested against each other, and every module keeps working on its own.

## What is in the repository now

| Directory          | What it is                                        | README |
|--------------------|---------------------------------------------------|--------|
| `apps/mcp-server/` | The MCP server: the Model Context Protocol over the official Go SDK, with a health endpoint and no tool yet. | [apps/mcp-server/README.md](apps/mcp-server/README.md) |
| `db/postgres/`     | PostgreSQL 18.6 with pgvector 0.8.7, internal only. | [db/postgres/README.md](db/postgres/README.md) |

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
| `8080`| *reserved*    | For the REST backend.                               |
| `50051`| *reserved*   | For the gRPC backend.                               |

The MCP server publishes no port at all. It binds the interface of its own
container and sits on `paca_internal`, so the other containers of the stack
reach it and nothing else does. The health endpoint shares that surface.

## The quality gate

Every Go module brings its own Makefile, with the same targets: the format
check, the analyzers, the linter, the build, the tests, the race detector,
the coverage gate, the benchmarks and the vulnerability scan.

```sh
make -C apps/mcp-server all
```

The tools are pinned in `tools/go.mod` and run with `go tool`, so every
machine and every container uses the same versions.
