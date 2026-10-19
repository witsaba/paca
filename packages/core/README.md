# `packages/core` — the shared core of PACA

The heart of PACA, shared by every delivery mechanism. Today that means the
`paca` CLI and, from session S02 on, the MCP server: two binaries, one set of
rules.

This is a Go module of the workspace: `github.com/witsaba/paca/packages/core`.

## Layout

```
packages/core/
├── core.go           the root package: the identity of the product
├── domain/           entities, value objects, invariants, domain services
├── ports/            the interfaces of both boundaries
├── application/      the use cases: orchestrate the domain through the ports
└── adapters/
    └── sqlite/       the concrete repository over the schema of packages/db
```

The dependency rule is inward only: `application` → `ports` → `domain`. The
domain imports nothing from the project and touches no I/O, so a rule that
lives in the domain behaves the same in the CLI, in the MCP server, and in any
future client. Each package documents its own rule in its `doc.go`.

The adapter is the one place that talks to the storage. It implements the
repository ports of `packages/core/ports` over the schema that
[`packages/db`](../../packages/db) publishes, and it turns every failure of
the driver into a typed error of the domain: the message of the driver never
reaches the detail that travels to an agent.

## What the adapter holds

- **The connection.** The pure-Go driver `modernc.org/sqlite`, one connection
  per pool, and `PRAGMA foreign_keys = ON` on every connection. The pragma is
  set on the connection and never on the schema.
- **The error contract.** The seven stable codes of the C7 taxonomy, and the
  rule that a constraint violation is a validation failure, a missing row is a
  not-found failure, and anything else is an internal error that says nothing.
- **The migration runner.** It applies the migrations of `packages/db` in the
  order of their number, reads the version of the schema from
  `PRAGMA user_version`, and stops at the first error. A migration added
  later is applied by the same runner, without a change to it.
- **The repositories.** One per table of the schema, each one implementing the
  port of the same name.

## Why a module of its own

The layers used to live under `apps/cli/internal/`. Go's `internal` visibility
rule scopes `apps/cli/internal/...` to `apps/cli/...` alone, so the MCP server
could never import them. A module of its own, imported by both binaries, is the
layout that makes the shared core possible. The move is recorded as amendment
A2 in
[`docs/architecture/0001-mcp-agent-server.md`](../../docs/architecture/0001-mcp-agent-server.md).

## Who depends on it

- [`apps/cli`](../../apps/cli/README.md) — the `paca` command.
- [`apps/mcp`](../mcp) — the `paca-mcp` MCP server (session S02).
- [`packages/db`](../../packages/db) — the schema the adapter reads. The core
  depends on the schema, and not the other way around.

Both binaries are composition roots: they wire the services and never own a
business rule.
