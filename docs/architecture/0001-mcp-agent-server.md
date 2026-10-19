# ADR 0001 — PACA MCP Agent Server Architecture

- **Status:** Accepted
- **Date:** 2026-10-19
- **Deciders:** project owner (single user)
- **Supersedes:** none formally; supersedes the *forward-looking* parts of
  [`docs/architecture/backend-scaffold.md`](backend-scaffold.md) for the
  agent-facing surface. The earlier scaffold doc stays as the literal record
  of how the CLI was bootstrapped.

---

## Context

PACA is a personal finance manager for the Colombian market. The user wants
to replace the daily Google Timesheet with in-LLM-agent workflows that record
expenses, income, debt, and (later) custom notifications plus an "expense map"
view that shows where the money actually went.

Foundations already merged into the repository:

- **SQLite schema** — [`packages/db/migrations/001_initial_schema.sql`](../../packages/db/migrations/001_initial_schema.sql)
  Five STRICT tables (`contacts`, `accounts`, `loans`, `movements`,
  `loan_payments`), COP integer money, ISO `YYYY-MM-DD` dates, single-table
  inheritance with a `kind` discriminator, 57-check verifier
  ([`packages/db/tools/verify_schema.sh`](../../packages/db/tools/verify_schema.sh)).
- **Go backend scaffold** — [`apps/cli/cmd/paca`](../../apps/cli/) with
  hexagonal layers, now shared as the module
  [`packages/core`](../../packages/core/README.md): `packages/core/domain`,
  `packages/core/ports`, `packages/core/application`, and a
  `packages/core/adapters/sqlite` slot reserved for the SQLite repository.
  Driver: `modernc.org/sqlite` (pure Go, no CGO).
- **Settled decisions** in
  [`odd/tasks/deferred-decisions.md`](../../odd/tasks/deferred-decisions.md):
  single-user, local machine, no `user_id` columns.

What is missing is an **agent-facing surface**. The user wants to expose the
system to any local LLM agent that speaks MCP, instead of building a human
UI. The repository needs one ADR that captures the architecture and the
cross-cutting decisions the implementation will build on, so feature docs and
PRs do not re-litigate them.

---

## Decision Drivers

- **Single user, personal finance** — high-impact errors (lost money records),
  low concurrent load, no horizontal scaling.
- **Agent-first** — tools exposed to LLM agents via MCP; no human UI planned
  for the first iteration.
- **Internal-network deployment** — the server is reachable from multiple
  devices on the user's trusted LAN, not only loopback. This adds auth and
  transport-security responsibilities that a loopback-only deployment would
  not need.
- **Local storage** — SQLite, single binary, no separate DB process.
- **Speed** — ship MVP1 in tight, reviewable slices.
- **Foundations already in place** — schema and hexagonal layers are merged.
  The goal of this ADR is to extend, not to redo.

---

## Considered Options

### Option A — MCP server reusing the CLI's hexagonal services

A second binary, `apps/mcp/cmd/paca-mcp`, exposes MCP tools. Each tool is a
thin wrapper that calls `packages/core/application` services. The SQLite adapter
sits in `packages/core/adapters/sqlite` and is shared by both binaries.

### Option B — MCP server + gRPC backend

Two binaries (MCP and gRPC), shared proto schemas. The MCP server translates
JSON-RPC into gRPC calls. Both CI and Go tooling work harder.

### Option C — MCP server + REST/HTTP backend

Two binaries, JSON HTTP. Simpler than gRPC, still two processes and a second
wire format.

### Decision

**Option A.** The MCP server is a new composition root over the same
`internal/` packages the CLI already uses.

**Why over B:** no current consumer speaks gRPC. Agents speak MCP. Adding a
proto layer buys typed contracts across two processes whose only consumer is
an LLM over JSON-RPC. The win does not pay for the schema, codegen, and
second-binary overhead.

**Why over C:** untyped JSON contracts drift; gRPC's typed contract was the
best argument for B and C does not have it. C is also two binaries without
the architectural payoff.

---

## Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                         LLM Agent Host                             │
│       Claude Desktop · Cursor · pi · future local agents           │
└───────────────────────────┬────────────────────────────────────────┘
                            │  MCP over HTTP+SSE (JSON-RPC 2.0)
                            │  Bearer token when bind ≠ loopback
                            ▼
┌────────────────────────────────────────────────────────────────────┐
│   apps/mcp/cmd/paca-mcp        (NEW binary, ~200 lines of wiring)  │
│   • `modelcontextprotocol/go-sdk` `StreamableHTTPServer`           │
│   • stdlib `net/http` lifecycle; `--bind`/`--port`/`--auth-token`  │
│   • one `Server`, one `AddTool(...)` per MCP tool                  │
│   • thin: parse → call service → format response                   │
└───────────────────────────┬────────────────────────────────────────┘
                            │   in-process Go calls
                            ▼
┌────────────────────────────────────────────────────────────────────┐
│   packages/core/application/*  (scaffolded in PR #5, moved in S00) │
│   RecordExpense · RecordIncome · RecordTransfer · RecordLoan · …   │
│   GetBalance · ExpensesByCategory · OutstandingDebts · …           │
└───────────────────────────┬────────────────────────────────────────┘
                            │   ports
                            ▼
┌────────────────────────────────────────────────────────────────────┐
│   packages/core/adapters/sqlite (NEW — the adapter slot from #5)   │
│   modernc.org/sqlite · migrations under packages/db/migrations/    │
└────────────────────────────────────────────────────────────────────┘

        ┌──────────────────────────────────────────────┐
        │   apps/cli/cmd/paca   (existing, untouched)  │
        │   composes the same `packages/core/application`│
        └──────────────────────────────────────────────┘
```

### Repository Layout

The implementation lands at:

- `apps/mcp/cmd/paca-mcp/main.go` — composition root: `wire(slog, db, app) → mcp.Server`
- `apps/mcp/internal/tools/*.go` — one file per MCP tool group (money, debt, setup, health)
- `packages/core/adapters/sqlite/` — concrete repository over the existing schema
- `packages/db/` — a Go module that publishes the migrations with `go:embed`, so
  that the adapter depends on the schema instead of on a copy of it (amendment A8)
- `packages/db/migrations/002_*.sql` — new tables introduced by this ADR (`audit_log`, `idempotency_keys`)
- `packages/db/tools/verify_schema.sh` — extended with the same discipline as `001`

The CLI command at `apps/cli/` stays untouched. Both binaries are composition
roots over the same domain code.

---

## Tools (MVP1 — final list)

The tools an LLM calls. Grouped by responsibility. Optional arguments are
marked `?`. Every write tool accepts an optional `idempotency_key` and an
optional `actor` override (server defaults from the `X-PACA-Actor` header).

### Cross-cutting

| Tool | Purpose |
|---|---|
| `health` | DB open, `PRAGMA user_version`, server uptime. |

### Money — writes

| Tool | Purpose |
|---|---|
| `record_expense` | Insert an `expense` movement. |
| `record_income` | Insert an `income` movement. |
| `record_transfer` | Insert paired `transfer_out` + `transfer_in` movements. |
| `update_movement` | Correct fields on a past movement. |
| `delete_movement` | Remove a movement (soft via notes flag in MVP1). |

### Money — reads

| Tool | Purpose |
|---|---|
| `get_balance` | Per-account balances and grand total. |
| `expenses_by_category` | Aggregated spend per `category` over a period — answers *"where the money goes"*. |
| `income_vs_expenses` | Period buckets of income, expense, net. |
| `cashflow_summary` | Same as above with `group_by` (day/week/month/project). |
| `list_recent_movements` | Recent activity for an account or globally. |

### Debt — writes

| Tool | Purpose |
|---|---|
| `record_loan` | Insert a loan (`borrowed` or `lent`, `informal` or `formal`). |
| `record_loan_payment` | Mark a `loan_payments` row paid and link the movement. |
| `update_loan` | Correct a loan's metadata. |
| `cancel_loan` | Mark a loan `cancelled` (never deleted). |

### Debt — reads

| Tool | Purpose |
|---|---|
| `outstanding_debts` | Per-contact lent and borrowed totals. |
| `upcoming_loan_payments` | Payments due within `days_ahead`. |
| `loan_schedule` | Full schedule for one loan. |

### Setup — writes

| Tool | Purpose |
|---|---|
| `add_account` | Insert an `accounts` row, choosing `kind = nequi | bank | cash`. |
| `add_contact` | Insert a `contacts` row, `kind = person | entity`. |
| `add_category` | Record a known category label. |

### Setup — reads

| Tool | Purpose |
|---|---|
| `list_accounts` | All accounts; `active_only` filter. |
| `list_contacts` | All contacts; `kind` filter. |
| `list_categories` | Known category labels. |
| `list_active_loans` | Loans with `status = 'active'`. |

**Total: ~22 tools.**

---

## Cross-cutting Concerns

Each section below corresponds to a non-negotiable behaviour the
implementation must satisfy. The feature document
[`odd/tasks/mcp-server-mvp1.md`](../../odd/tasks/) — drafted after this ADR
is merged — splits these into reviewable tasks with one focused test each.

### C1. Transport — HTTP with SSE (Streamable HTTP) on the internal network

- Bind address via `--bind` (`127.0.0.1` default).
- Port via `--port` (`7333` default).
- Loopback bind → plain HTTP; auth is **optional** unless `--require-auth` is set.
- Non-loopback bind → **TLS required**; cert and key from `--tls-cert` / `--tls-key`.
  The server refuses to start if either flag is missing while bound to a
  non-loopback address.
- Implements `StreamableHTTPServer` from
  [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).
- Exposes a separate `/healthz` endpoint for liveness (not part of the MCP
  surface).

### C2. Authentication

- Bearer token via `Authorization: Bearer <token>`.
- Token supplied by `--auth-token`. The server **fails to start** when
  `--bind` is non-loopback and no token is set.
- Token rotation in MVP1 is "restart with a new flag" — no graceful
  revocation. Documented as a limitation.
- mTLS is deferred to a future ADR.

### C3. Idempotency for writes

- Every write tool accepts an optional `idempotency_key` (UUID v4 recommended).
- A new `idempotency_keys` table records:
  `(actor, key, tool, request_hash, output_json, created_at)` with a `TTL` of 7 days.
- A repeated `(actor, key)` returns the original output with
  `duplicate_request: true`; **no second write occurs**.
- Without a key, the call proceeds normally. The agent bears the retry risk.

### C4. Audit log

- A new `audit_log` table records every write:
  `id, tool, input_json, output_json, actor, occurred_at, idempotency_key?`.
- Reads are not audited in MVP1.
- Future: a filtered `audit_log` reader as a tool (out of scope).

### C5. Actor identification

- The MCP request header `X-PACA-Actor` identifies the calling agent
  (e.g., `claude-desktop`, `cursor`, `pi`, `cli`).
- Falls back to `agent:<tool_name>` when missing.
- The actor propagates through `packages/core/application` into
  `audit_log.actor` and `idempotency_keys.actor`.

### C6. Amount and currency normalization

- Tool inputs accept decimal strings (`"1500000.50"`) or integer strings
  (`"1500000"`).
- The service rounds **half-up** to integer COP at the service boundary.
  The rounded value is mirrored in the response so the agent can confirm.
- Inputs with more than 4 decimal places or non-finite/non-numeric values
  return `validation_failed` and are not written.
- Currency: when the referenced `account.currency ≠ 'COP'`, the tool
  returns the typed error `currency_not_supported_in_mvp1` with the
  account id and observed currency. **No silent conversion, no FX.**

### C7. Error model

A closed set of stable error codes that an agent can branch on without
parsing prose:

| Code | Meaning |
|---|---|
| `validation_failed` | Argument shape, decimal precision, date format, amount > 0, etc. |
| `not_found` | The referenced id does not exist (movement, account, contact, loan). |
| `duplicate_request` | Idempotency key already used; the original output is returned. |
| `currency_not_supported_in_mvp1` | Account currency is not COP. |
| `account_inactive` | Operation targets a closed account. |
| `loop_required_for_unsafe_call` | Reserved for future calls that the server refuses over LAN. |
| `internal_error` | Anything unexpected; details do not leak DB internals. |

Responses include `code` and a human-readable `detail`; no DB error strings,
file paths, or stack frames in `detail`.

### C8. Schema migrations with a running server

- `paca-mcp` holds the SQLite file open.
- Migration procedure documented: **stop server → apply migration → start server**.
  SQLite's write lock makes the brief window safe.
- WAL mode and online migrations are out of scope for MVP1.

### C9. Backups

- The single-file SQLite DB is a single point of failure.
- The ADR does **not** bake a backup binary in MVP1.
- The user is responsible for periodic snapshots (`cp` or `sqlite3 .backup`).
- Future: a `paca backup` command streamed to a target file (out of scope).

### C10. CLI parity

- The CLI composes `packages/core/application` services **directly**.
- The CLI does **not** call the HTTP MCP server.
- Both binaries are independent composition roots over the same domain code.

### C11. Testing strategy

- Every cross-cutting concern (C1–C10) has at least one focused test.
- Every listed tool has:
  1. a service-layer unit test, and
  2. an integration test using the MCP client SDK over `httptest`.
- The 57-check verifier in
  [`packages/db/tools/verify_schema.sh`](../../packages/db/tools/verify_schema.sh)
  keeps passing. New tables (`audit_log`, `idempotency_keys`) extend the
  verifier via migration `002_*.sql`.

---

## Amendments

The feature document [`odd/tasks/mcp-server-mvp1.md`](../../odd/tasks/mcp-server-mvp1.md)
records the amendments this ADR needs, each applied as a one-line edit in the
session that first needs it.

| Amendment | Change | Applied in |
| --- | --- | --- |
| A1 | the tool count is 25, not the 22 the prose and the acceptance check claim | pending — S05 |
| A2 | the shared core moves from `apps/cli/internal/` to the module `packages/core`, because Go's `internal` rule hides `apps/cli/internal/...` from `apps/mcp` | **applied — S00** |
| A3 | `cashflow_summary` drops `project` in MVP1 | pending — S20 |
| A4 | `income_vs_expenses` and `cashflow_summary` are disambiguated | pending — S19 |
| A5 | `delete_movement` is a hard delete, with the audit log as the recovery trail | pending — S16 |
| A6 | `add_category` and `list_categories` need a `categories` table | pending — S08 |
| A9 | the migrations are published by a Go module of their own, `packages/db`, because `go:embed` cannot cross a module boundary | **applied — S01** |
| A10 | a uniqueness violation is a `validation_failed`; `duplicate_request` is reserved for the idempotency key | **applied — S01** |

The roadmap numbers A7 and A8 are consequences of the ADR and not edits of it,
so this table has no row for them.

---

## Validation

This ADR's decisions will be verified as each cross-cutting concern and each
tool is implemented. The feature document
[`odd/tasks/mcp-server-mvp1.md`](../../odd/tasks/) — to be drafted after this
ADR is merged — records the S# per concern and tool, the route, and the
verification per task.

Acceptance check at MVP1 close:

- All 22 MVP1 tools listed above pass their service-layer and integration tests.
- All 11 cross-cutting concerns (C1–C11) have at least one passing test.
- All 57 existing schema checks still pass.
- A new migration `002_*.sql` extends the verifier with checks for the two
  new tables.

---

## Deferred (explicitly out of MVP1)

Tracked here so the ADR's shape leaves room and so future ADRs do not
re-litigate the boundaries.

- **MVP2 — Time tracking.** New `time_entries` (+ `projects`) tables; tools
  `start_timer`, `stop_timer`, `cancel_timer`, `log_time_entry`,
  `timesheet(period, project?)`. Same SQLite, same MCP surface, no schema
  impact on money or debt.
- **MVP2 — Bulk import.** `import_csv_transactions(path, format?)` for
  migrating from Google Timesheet exports.
- **MVP3 — Notifications.** MCP has no native user-facing notification
  primitive. A `paca-scheduler` sidecar plus `set_notification_rule` /
  `list_notification_rules` tools would carry this; ntfy/email/webhooks
  are evaluation topics.
- **MVP3 — Maps.** Requires `lat`/`lon` on movements or a separate
  `expense_locations` table; tool returns either a static image URL or
  GeoJSON. Requires picking a map provider (OSM / Google / Mapbox).
- **MVP3 — Multi-currency.** Allow non-COP accounts in tool inputs; the
  typed error today is the placeholder.
- **MVP3 — Prometheus metrics.** Optional `--metrics-addr` exporting
  request count, write latency, idempotency hit ratio.

---

## References

- [`docs/architecture/backend-scaffold.md`](backend-scaffold.md) — the
  earlier, informal decision record (CLI scaffold).
- [`packages/db/migrations/001_initial_schema.sql`](../../packages/db/migrations/001_initial_schema.sql)
  — the schema.
- [`packages/db/tools/verify_schema.sh`](../../packages/db/tools/verify_schema.sh)
  — the 57-check schema verifier.
- [`odd/tasks/`](../../odd/tasks/) — feature documents that track
  implementation work.
- PR #3 — SQLite schema.
- PR #5 — Go backend scaffold.
- MCP spec — `https://modelcontextprotocol.io`.
- [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)
  — official MCP Go SDK.
