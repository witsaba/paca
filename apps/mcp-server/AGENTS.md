# AGENTS.md — `apps/mcp-server`

The MCP server of the monorepository: it speaks the Model Context Protocol over the
official Go SDK, answers a health endpoint and exposes three tools. The contract of the
root applies here and is never relaxed. This file holds the specifics of the service.

## What this service is

- A **hexagonal application**: the rules of the domain live in the middle, the protocol
  lives at the edge, and the composition root is the only place that knows every layer.
- **Private**: it publishes no port. Only the other containers of the internal network
  reach it.
- A **client of the shared libraries**: the domain of the person comes from
  `packages/domain`, the pool and the queries from `packages/database`. The service never
  writes a row of its own and never migrates.

## The layers of this service

```
cmd/paca-mcp-server/          the composition root
internal/
  application/                 the use cases: the rules of the service
  ports/                      the contracts between the layers
  adapters/
    inbound/http/              the HTTP adapter: the health endpoint
    inbound/mcpserver/         the MCP adapter: the official Go SDK
    outbound/serving/          the checker of the MCP transport
    outbound/database/         the checker of the database
  platform/config/             the configuration of this service: the only
                               infrastructure the service owns
  bootstrap/                   the wiring of every layer
```

The domain knows nothing, the application knows the domain and the ports, the adapters
know the ports, and only the composition root knows the adapters.

The models live in the shared modules: the person and the health in
`packages/domain`, the clock, the logs, the telemetry and the server in `packages/platform`.
A service that has a model of its own keeps it in `internal/domain/`; a model that more
than one service needs lives in `packages/domain`, and it is never copied.

## The rules that are specific to this service

- **The tool contract derives from the domain.** The list of the statuses a tool accepts,
  and the description that tells the agent which moves are allowed, come from the domain
  (`packages/domain/user`). A literal that restates a rule of the domain is a defect: the
  two drift the first time the domain changes.
- **The description of a tool is a contract with an agent.** It says what the tool needs,
  what it does with what it receives, and what it must not do. It never promises a
  validation the tool does not perform.
- **Every error the agent can see has a message in the adapter.** The domain owns the
  vocabulary of the errors; the adapter owns the words the agent reads. An error without
  a case reaches the agent as a Go string.
- **The database is optional, and the service says so in words.** With no database
  configured, the three tools are still there and every call answers that the database is
  not configured. The absence of the repository is expressed by the absence of the value,
  never by a nil pointer inside an interface.
- **The service never migrates.** The schema has one writer, and it is `db-admin`.

## The shape of a change here

1. A rule of the person changes in `packages/domain`, with its test.
2. The use case calls the rule; it does not re-implement it.
3. The adapter maps the outcome of the use case onto the protocol, and adds the case of
   the error if the error is new.
4. The description and the schema of the tool are derived, not re-typed.
5. The gate of the service is green: `make -C apps/mcp-server all`, and the container
   tests: `make apps-integration`.

## The proof

The unit tests exercise the use cases and the protocol through their public interfaces,
with a real MCP client session. The container tests exercise the service over the
network, against a real PostgreSQL.
