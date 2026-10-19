# docs/

Reserved for documentation that is neither a package README nor an application README.

Typical contents, as the project grows:

- `architecture/` — how the packages fit together and why;
- `decisions/` — records of the options considered and the choice made, with the
  evidence behind it;
- `guides/` — task-oriented walkthroughs for people working with the data.

Nothing here yet. The documentation that exists today is the root
[README](../README.md), which holds the complete database reference,
[`packages/db/README.md`](../packages/db/README.md) for the schema package, and the
architecture notes under `architecture/`:

- [`architecture/backend-scaffold.md`](architecture/backend-scaffold.md) — the backend
  foundation: the Go workspace, the hexagonal layout, the pinned tools, and the quality
  gate, with the industry sources behind each decision.
- [`architecture/0001-mcp-agent-server.md`](architecture/0001-mcp-agent-server.md) — the
  MCP agent server: the transport, the tools, and the eleven cross-cutting concerns.

The design rationale for the database feature, including the measured comparison behind
the model choice, is tracked in [`odd/tasks/database-model.md`](../odd/tasks/database-model.md).
The design rationale for the backend scaffold is tracked in
[`odd/tasks/backend-scaffold.md`](../odd/tasks/backend-scaffold.md).
