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
└── application/      the use cases: orchestrate the domain through the ports
```

The dependency rule is inward only: `application` → `ports` → `domain`. The
domain imports nothing from the project and touches no I/O, so a rule that
lives in the domain behaves the same in the CLI, in the MCP server, and in any
future client. Each package documents its own rule in its `doc.go`.

The SQLite adapter lands in `packages/core/adapters/sqlite` in session S01.

## Why a module of its own

The layers used to live under `apps/cli/internal/`. Go's `internal` visibility
rule scopes `apps/cli/internal/...` to `apps/cli/...` alone, so the MCP server
could never import them. A module of its own, imported by both binaries, is the
layout that makes the shared core possible. The move is recorded as amendment
A2 in
[`docs/architecture/0001-mcp-agent-server.md`](../../docs/architecture/0001-mcp-agent-server.md).

## Who depends on it

- [`apps/cli`](../../apps/cli/README.md) — the `paca` command.
- `apps/mcp` — the `paca-mcp` MCP server (session S02).

Both binaries are composition roots: they wire the services and never own a
business rule.
