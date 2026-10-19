# tools/

Reserved for repository-wide developer tooling: scripts that operate across the
monorepo rather than inside a single package.

Typical contents:

- `scripts/` — one-off maintenance and release helpers;
- `checks/` — repository-wide quality gates that run across every package.

Nothing here yet. The one tool PACA already has is package-scoped rather than
repository-wide, so it lives with the package it verifies:
[`packages/db/tools/verify_schema.sh`](../packages/db/tools/verify_schema.sh).
