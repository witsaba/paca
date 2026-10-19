# PACA — MCP Server MVP1 roadmap

Feature document for the MCP agent server described by
[`docs/architecture/0001-mcp-agent-server.md`](../../docs/architecture/0001-mcp-agent-server.md).

The ADR fixes the architecture. This document fixes the **order of work**: every
tool of MVP1 is delivered in its own session, on its own branch, on its own
reviewable commit, so that the feature can be built one functionality at a
time and stopped at any point with something that works.

- **Sessions:** 30 — 5 spine sessions (the foundation every tool needs) + 25 tool sessions.
- **Canonical tool list:** ADR 0001 §"Tools (MVP1 — final list)", quoted verbatim in the Specs below.
- **Delivery:** one branch `feat/mcp-<slug>` and one Conventional Commit per session,
  merged into `main` as it lands. A session is never half-merged.

---

## How to use this roadmap

### The unit of work

One session = one session of this roadmap = one tool (or one spine item) = one
branch = one commit. The commit is a reviewable slice of roughly 150–400
authored changed lines, which is the house limit.

### The protocol of a session

1. **Read** this document from the top down to the `## Log` section. The `## Log`
   holds the user's original request and every correction since; it is part of
   the specification, not a diary.
2. **Confirm** the session's entry against the ADR. The ADR wins on architecture;
   this document wins on order and on the per-tool detail.
3. **Branch** `feat/mcp-<tool-slug>` from an up-to-date `main`.
4. **Write the test first.** One RED test per rule the tool's entry names:
   the happy path, every named error, and the boundary cases. The happy path
   goes RED before the implementation exists.
5. **Implement** the service in `packages/core/application`, the repository
   method in `packages/core/adapters/sqlite`, and the thin MCP wrapper in
   `apps/mcp/internal/tools`. The wrapper parses, calls the service, and
   formats the response — it holds no business rule.
6. **Verify** with `make all` from the repository root, and with the session's
   own focused test. Both must be green.
7. **Commit** the session's slice with the message format below.
8. **Update** this document: the session's entry moves to *Done* with its commit
   hash, and the `## Log` gains one entry per session.

### The commit message format

```
feat(mcp): <tool-name> — <one-line outcome>

Spec: odd/tasks/mcp-server-mvp1.md, S<n>
ADR:  docs/architecture/0001-mcp-agent-server.md
```

### Rules that apply to every session

- **C7 error taxonomy.** Every failure the agent can observe is one of the
  stable codes. No new prose-only errors.
- **C6 money.** Amounts enter as decimal strings and leave as integer COP,
  rounded half-up. The rounded value is echoed in the response.
- **C3/C4/C5 spine.** Every write is idempotent-key aware, audited, and
  attributed to an actor — for free, because the spine does it. A tool never
  implements those three itself.
- **CLI parity (C10).** The CLI never calls the MCP server. Both compose
  `packages/core/application` directly. A service added for a tool is
  immediately available to the CLI; wiring the CLI to it is a separate,
  optional session.
- **One tool per session.** A session that finds it needs two tools splits
  itself and leaves the second for the next session.

---

## Roadmap at a glance

| Session | ID | Title | ADR tool | Depends on |
|---|---|---|---|---|
| S00 | — | Core module extraction | — | — |
| S01 | — | Persistence foundation (SQLite adapter + migration runner) | — | S00 |
| S02 | — | MCP server shell (transport, TLS, auth, `/healthz`) | — | S01 |
| S03 | — | Tool contract layer (errors, money, actor, response shape) | — | S02 |
| S04 | — | Idempotency + audit + migration `002` | — | S03 |
| S05 | T01 | `health` | Cross-cutting | S04 |
| S06 | T19 | `add_account` | Setup — writes | S05 |
| S07 | T20 | `add_contact` | Setup — writes | S06 |
| S08 | T21 | `add_category` | Setup — writes | S07 |
| S09 | T22 | `list_accounts` | Setup — reads | S06 |
| S10 | T23 | `list_contacts` | Setup — reads | S07 |
| S11 | T24 | `list_categories` | Setup — reads | S08 |
| S12 | T02 | `record_expense` | Money — writes | S09 |
| S13 | T03 | `record_income` | Money — writes | S12 |
| S14 | T04 | `record_transfer` | Money — writes | S13 |
| S15 | T05 | `update_movement` | Money — writes | S12 |
| S16 | T06 | `delete_movement` | Money — writes | S15 |
| S17 | T07 | `get_balance` | Money — reads | S14, S16 |
| S18 | T08 | `expenses_by_category` | Money — reads | S12 |
| S19 | T09 | `income_vs_expenses` | Money — reads | S13 |
| S20 | T10 | `cashflow_summary` | Money — reads | S19 |
| S21 | T11 | `list_recent_movements` | Money — reads | S12 |
| S22 | T12 | `record_loan` | Debt — writes | S07, S09 |
| S23 | T13 | `record_loan_payment` | Debt — writes | S22 |
| S24 | T14 | `update_loan` | Debt — writes | S22 |
| S25 | T15 | `cancel_loan` | Debt — writes | S22 |
| S26 | T16 | `outstanding_debts` | Debt — reads | S22 |
| S27 | T17 | `upcoming_loan_payments` | Debt — reads | S23 |
| S28 | T18 | `loan_schedule` | Debt — reads | S22 |
| S29 | T25 | `list_active_loans` | Setup — reads | S26 |

The build order is dependency order, not the ADR's presentation order: the
setup writes come first because every money and debt tool needs an account and
a contact to exist, and `health` comes first because it is the cheapest
end-to-end proof that the spine works.

---

## Specs

Each spec is the ADR's own wording, quoted verbatim. The session entry expands
it into inputs, outputs, rules, and tests.

- **S1** — `health` — "DB open, `PRAGMA user_version`, server uptime."
- **S2** — `record_expense` — "Insert an `expense` movement."
- **S3** — `record_income` — "Insert an `income` movement."
- **S4** — `record_transfer` — "Insert paired `transfer_out` + `transfer_in` movements."
- **S5** — `update_movement` — "Correct fields on a past movement."
- **S6** — `delete_movement` — "Remove a movement (soft via notes flag in MVP1)."
- **S7** — `get_balance` — "Per-account balances and grand total."
- **S8** — `expenses_by_category` — "Aggregated spend per `category` over a period — answers *"where the money goes"*."
- **S9** — `income_vs_expenses` — "Period buckets of income, expense, net."
- **S10** — `cashflow_summary` — "Same as above with `group_by` (day/week/month/project)."
- **S11** — `list_recent_movements` — "Recent activity for an account or globally."
- **S12** — `record_loan` — "Insert a loan (`borrowed` or `lent`, `informal` or `formal`)."
- **S13** — `record_loan_payment` — "Mark a `loan_payments` row paid and link the movement."
- **S14** — `update_loan` — "Correct a loan's metadata."
- **S15** — `cancel_loan` — "Mark a loan `cancelled` (never deleted)."
- **S16** — `outstanding_debts` — "Per-contact lent and borrowed totals."
- **S17** — `upcoming_loan_payments` — "Payments due within `days_ahead`."
- **S18** — `loan_schedule` — "Full schedule for one loan."
- **S19** — `add_account` — "Insert an `accounts` row, choosing `kind = nequi | bank | cash`."
- **S20** — `add_contact` — "Insert a `contacts` row, `kind = person | entity`."
- **S21** — `add_category` — "Record a known category label."
- **S22** — `list_accounts` — "All accounts; `active_only` filter."
- **S23** — `list_contacts` — "All contacts; `kind` filter."
- **S24** — `list_categories` — "Known category labels."
- **S25** — `list_active_loans` — "Loans with `status = 'active'`."

---

## ADR amendments this roadmap requires

The ADR and the merged schema disagree in six places. Each is resolved here
and must be applied to the ADR as a one-line edit (an "amendment session",
folded into the session that first needs it).

- **A1 — the tool count is 25, not 22.** The ADR's own tables list 25 tools
  (1 + 5 + 5 + 4 + 3 + 3 + 4). The prose says "~22 tools" and the acceptance
  check says "All 22 MVP1 tools". The tables are the requirement; the number
  is the error. Fix the prose and the acceptance check.
- **A2 — the shared core cannot live under `apps/cli/internal/`.** Go's
  `internal` visibility rule scopes `apps/cli/internal/...` to `apps/cli/...`
  only, so `apps/mcp` cannot import the application services the ADR says both
  binaries share. The shared core moves to a new module, `packages/core`
  (`github.com/witsaba/paca/packages/core`), holding `domain`, `ports`,
  `application`, and `adapters/sqlite`. Both `apps/cli` and `apps/mcp` require
  it. The ADR's paths become `packages/core/application` and
  `packages/core/adapters/sqlite`.
- **A3 — `cashflow_summary` loses `project` in MVP1.** There is no project
  column anywhere in the schema; `projects` arrive with MVP2 time tracking.
  MVP1 supports `group_by` = `day | week | month`; `project` is rejected with
  `validation_failed` and says so in the detail.
- **A4 — `income_vs_expenses` and `cashflow_summary` are disambiguated.**
  `income_vs_expenses` returns the period totals (income, expense, net, count).
  `cashflow_summary` returns the bucketed series. The ADR describes both as
  period buckets, which would make them the same tool.
- **A5 — `delete_movement` is a hard delete.** The ADR's "soft via notes flag"
  is not a soft delete: it leaves the row in place, in every balance, forever.
  MVP1 deletes the row and keeps the recovery trail in `audit_log` (C4), which
  already stores the full before/after of every write. See open decision O2.
- **A6 — `add_category` and `list_categories` need a table.** `movements.category`
  is free text; there is no catalogue to write to or read from. Migration
  `002` adds a `categories` table. See open decision O1.

Two smaller consequences, not amendments but facts the sessions must respect:

- **A7 — no uniqueness code exists.** The C7 taxonomy has no code for a real
  uniqueness violation, and `duplicate_request` is reserved for idempotency.
  MVP1 maps uniqueness violations (duplicate NIT, duplicate Nequi phone,
  duplicate bank account) to `validation_failed` with a specific detail. The
  gap is recorded for a future ADR.
- **A8 — the CLI stays untouched.** C10 says the CLI composes the services
  directly. No MVP1 session changes the CLI's behaviour; the CLI keeps
  shipping `--help` only. Wiring the CLI to the new services is a separate
  feature after MVP1.

---

## Open decisions

These three are product decisions the sessions cannot make on their own. Each
has a default, and the roadmap proceeds on that default unless the user
overrides it.

- **O1 — Is there a curated category catalogue?** (blocks S08, S11; affects migration `002`)
  - *Default:* yes. Migration `002` adds `categories (id, name, created_at)`
    with a unique name. `add_category` seeds it, `list_categories` reads it.
    `movements.category` stays free text; `expenses_by_category` groups by the
    stored value and reports the share that is not in the catalogue.
  - *Alternative:* drop the catalogue — `list_categories` returns the distinct
    `movements.category` values seen so far, and `add_category` is removed.
    That is 23 tools, not 25.
- **O2 — Does `delete_movement` delete the row or mark it?** (blocks S16; affects migration `002`)
  - *Default:* hard delete, with the audit log as the recovery trail (A5).
  - *Alternative:* a `deleted_at` column on `movements` and a soft delete. Every
    balance, report, and list query then has to filter on it, and the
    `movements` partial index has to change. Higher cost, cleaner semantics.
- **O3 — Does `record_loan` move the cash?** (blocks S22, S23)
  - *Default:* no, unless asked. `record_loan` records the contract. An
    optional `initial_movement` (`account_id`, `amount`, `occurred_on`)
    creates the paired `loan_lent` / `loan_received` movement in the same
    transaction, linked by `loan_id`.
  - *Alternative:* the cash movement is mandatory, which makes a loan that was
    agreed verbally impossible to record, and an agreement whose money has not
    moved yet impossible to represent.

---

## Tasks

### Phase 0 — The spine

Five sessions. No tool is possible before S04 finishes, so this phase is
built first and merged as a unit of five reviewable commits.

#### S00 — Core module extraction

- **Status:** Done — commit `88032ee` on `feat/mcp-s00-core-extraction`.
- **Spec:** A2. **Goal:** make the shared core importable by both binaries.
- **Change:** new module `packages/core` (`github.com/witsaba/paca/packages/core`)
  with `domain/`, `ports/`, `application/` moved from `apps/cli/internal/`;
  `apps/cli` requires it and its imports are updated; `go.work` gains the new
  module; `apps/cli/internal/adapters/cli` stays where it is.
- **No behaviour change.** The CLI must still print the same help and reject
  the same arguments.
- **Tests:** the existing `apps/cli` tests move with the packages and still pass.
- **Done when:** `make all` is green and `apps/cli` and `packages/core` are
  both listed by `make help`.

#### S01 — Persistence foundation

- **Spec:** ADR §Architecture, C8, C11. **Goal:** a real SQLite adapter.
- **Change:** `packages/core/adapters/sqlite` implements the repository
  interfaces declared in `packages/core/ports`. The driver is
  `modernc.org/sqlite` (pure Go, no CGO — settled in ADR 0001). Every
  connection sets `PRAGMA foreign_keys = ON` (per-connection, never persisted
  by the migration). A migration runner applies
  `packages/db/migrations/*.sql` in order, checks `PRAGMA user_version`, and
  aborts on the first error (the `-bail` discipline of migration `001`).
  SQLite constraint violations are translated into typed domain errors, never
  leaked as raw driver strings.
- **Tests:** every repository method against a temporary database; a
  foreign-key enforcement test; a runner test that refuses a second run of
  migration `001`; a test that a constraint violation maps to a typed error
  and not to `internal_error`.
- **Done when:** `make all` is green, and a migration `002` can be added and
  applied without touching the runner.

#### S02 — MCP server shell

- **Spec:** C1, C2. **Goal:** the server that will carry the tools.
- **Change:** new module `apps/mcp` with the binary `apps/mcp/cmd/paca-mcp`.
  Flags: `--bind` (default `127.0.0.1`), `--port` (default `7333`),
  `--auth-token`, `--tls-cert`, `--tls-key`, `--db`, `--require-auth`. The
  transport is Streamable HTTP from the official MCP Go SDK
  (`github.com/modelcontextprotocol/go-sdk`; the exact API is confirmed
  against the pinned version at implementation time). A `/healthz` endpoint
  answers liveness and is not part of the MCP surface. The server **refuses to
  start** when `--bind` is not loopback and `--auth-token` is missing, or when
  `--bind` is not loopback and TLS material is missing. A bearer token is
  checked on every MCP request. Startup logs never print the token.
- **Tests:** starts and serves; refuses to start without a token on a
  non-loopback bind; refuses to start without TLS on a non-loopback bind;
  rejects a request with a wrong or missing token; `/healthz` answers 200.
- **Done when:** `make all` is green and the binary serves an empty MCP
  session over `httptest`.

#### S03 — Tool contract layer

- **Spec:** C5, C6, C7, C10. **Goal:** the shared shape every tool uses.
- **Change:** in `packages/core/domain` — the error taxonomy as typed errors
  with stable codes (`validation_failed`, `not_found`, `duplicate_request`,
  `currency_not_supported_in_mvp1`, `account_inactive`,
  `loop_required_for_unsafe_call`, `internal_error`) and a `detail` that never
  carries DB internals, file paths, or stack frames; the money value object
  that parses a decimal string, rejects more than 4 decimal places or a
  non-positive amount, and rounds half-up to integer COP. In
  `apps/mcp/internal/tools` — the shared response envelope
  (`{ "ok": bool, "data": ..., "error": { "code", "detail" } }`), the actor
  resolution from the `X-PACA-Actor` header with the `agent:<tool_name>`
  fallback (C5), and the helper that registers a tool with the SDK and
  converts a domain error into the envelope.
- **Tests:** the money parser (valid, 4 decimals, 5 decimals, zero, negative,
  non-numeric, exponent form); the rounding of `0.005` to `1` and `-0.005` to
  `0` under half-up; every error code mapping to the envelope; the actor
  fallback when the header is absent.
- **Done when:** `make all` is green and a tool can be written without
  touching the error or money code.

#### S04 — Idempotency, audit, and migration `002`

- **Spec:** C3, C4, C8, C11. **Goal:** every write is safe to retry and
  attributable.
- **Change:** migration `packages/db/migrations/002_agent_tables.sql` adds
  `audit_log` (`id, tool, input_json, output_json, actor, occurred_at,
  idempotency_key?`) and `idempotency_keys` (`actor, key, tool, request_hash,
  output_json, created_at`, unique on `(actor, key)`), plus the `categories`
  table if O1 stays at its default. The write path wraps every write tool:
  hash the request, look the key up, return the stored output with
  `duplicate_request: true` when it was seen before, and write the audit row
  with the before/after state. The 7-day TTL is enforced by pruning expired
  keys on write. The verifier `packages/db/tools/verify_schema.sh` gains
  checks for the new tables, and the existing 57 checks keep passing.
- **Tests:** the same key twice returns the original output and does not write
  twice; a different key writes again; a different actor with the same key
  writes again; every write produces exactly one audit row; an expired key
  writes again; the verifier passes.
- **Done when:** `make all` is green and the verifier passes with the new
  checks.

---

### Phase 1 — The setup tools

Six sessions. The write tools come before the read tools: a read of an empty
table is a weaker proof than a write followed by a read.

#### S05 — `health` (T01)

- **Spec:** S1. **Goal:** the end-to-end proof that the spine works.
- **Input:** none.
- **Output:** `{ "status": "ok", "schema_version": 1, "uptime_seconds": 42 }`.
- **Rules:** the database is opened and reachable; `PRAGMA user_version` is
  reported as the schema version; the uptime is measured from process start.
  A database that cannot be opened fails with `internal_error` and a detail
  that does not name the file.
- **Files:** `packages/core/application/health.go`,
  `packages/core/adapters/sqlite/health.go`, `apps/mcp/internal/tools/health.go`.
- **Tests:** the service test against a temporary database; the integration
  test over `httptest` with the MCP client SDK, asserting the three fields.
- **Done when:** the tool is listed by an MCP client, `make all` is green.

#### S06 — `add_account` (T19)

- **Spec:** S19. **Goal:** create an account.
- **Input:** `kind` (`nequi` | `bank` | `cash`), `name`, `opened_on`
  (YYYY-MM-DD), `currency` (default `COP`), `phone?`, `holder?`, `bank_code?`,
  `account_number?`, `account_type?` (`ahorro` | `corriente`), `notes?`,
  `idempotency_key?`, `actor?`.
- **Output:** `{ "account_id": 1, "kind": "nequi", "name": "Nequi", "currency": "COP" }`.
- **Rules:** the subtype columns must match the kind exactly, or
  `validation_failed` — a `nequi` account with a `bank_code` is refused, as is
  a `nequi` account without a `phone`. A non-`COP` currency is accepted at
  creation time (the account exists) but every money tool refuses to use it
  with `currency_not_supported_in_mvp1` (C6). A duplicate Nequi phone or
  duplicate bank account is `validation_failed` (A7). A malformed `opened_on`
  is `validation_failed`.
- **Files:** `packages/core/application/add_account.go`,
  `packages/core/adapters/sqlite/accounts.go`, `apps/mcp/internal/tools/setup.go`.
- **Tests:** one per kind; each kind with a mismatched subtype column; missing
  required subtype field; duplicate phone; duplicate bank account; bad date;
  bad currency format. Every refused case leaves the `accounts` row count
  unchanged.
- **Done when:** `make all` is green and a client can create and list an account.

#### S07 — `add_contact` (T20)

- **Spec:** S20. **Goal:** create a contact.
- **Input:** `kind` (`person` | `entity`), `name`, `phone`, `id_document?`,
  `tax_id?`, `legal_name?`, `notes?`, `idempotency_key?`, `actor?`.
- **Output:** `{ "contact_id": 1, "kind": "person", "name": "María" }`.
- **Rules:** a `person` requires `id_document` and forbids `tax_id` and
  `legal_name`; an `entity` requires `tax_id` and `legal_name` and forbids
  `id_document` — the schema's `contacts_shape` CHECK states this, and the
  service states it first with a readable detail. A duplicate `tax_id` is
  `validation_failed` (A7). A repeated `id_document` is **allowed**: the
  schema deliberately does not constrain it, and the service must not invent
  the constraint.
- **Files:** `packages/core/application/add_contact.go`,
  `packages/core/adapters/sqlite/contacts.go`.
- **Tests:** person; entity; person missing `id_document`; entity missing
  `tax_id`; entity missing `legal_name`; person with `tax_id`; duplicate
  `tax_id`; repeated `id_document` accepted. Every refused case leaves the
  `contacts` row count unchanged.
- **Done when:** `make all` is green.

#### S08 — `add_category` (T21)

- **Spec:** S21. **Goal:** record a known category label.
- **Input:** `name`, `idempotency_key?`, `actor?`.
- **Output:** `{ "category_id": 1, "name": "groceries" }`.
- **Rules:** the name is trimmed and lower-cased. A repeated name returns the
  existing category with `duplicate_request: false` and a `created: false`
  flag, rather than failing — the agent asked for a label that already
  exists. An empty name is `validation_failed`.
- **Depends on:** O1 (default: the `categories` table from migration `002`).
- **Tests:** creates; repeats; empty name; trims and lower-cases.
- **Done when:** `make all` is green.

#### S09 — `list_accounts` (T22)

- **Spec:** S22. **Goal:** list the accounts.
- **Input:** `active_only?` (default `false`).
- **Output:** `{ "accounts": [ { "id", "kind", "name", "currency", "opened_on",
  "closed_on", "is_active", "phone", "holder", "bank_code", "account_number",
  "account_type", "notes" } ], "count": 3 }`.
- **Rules:** ordered by `name`, then `id`. `active_only` filters on
  `is_active = 1`. A closed account is returned by default, with its
  `closed_on` set, so the agent can see history. The subtype fields are
  `null` when they do not apply to the kind.
- **Files:** `packages/core/application/list_accounts.go`,
  `packages/core/adapters/sqlite/accounts.go`.
- **Tests:** empty database; three accounts of the three kinds; `active_only`
  filters; ordering; `closed_on` populated for a closed account.
- **Done when:** `make all` is green.

#### S10 — `list_contacts` (T23)

- **Spec:** S23. **Goal:** list the contacts.
- **Input:** `kind?` (`person` | `entity`).
- **Output:** `{ "contacts": [ { "id", "kind", "name", "phone", "id_document",
  "tax_id", "legal_name", "notes" } ], "count": 2 }`.
- **Rules:** ordered by `name`, then `id`. The `kind` filter is optional; an
  unknown `kind` value is `validation_failed`.
- **Files:** `packages/core/application/list_contacts.go`,
  `packages/core/adapters/sqlite/contacts.go`.
- **Tests:** empty; both kinds; the `kind` filter; an unknown `kind`; ordering.
- **Done when:** `make all` is green.

#### S11 — `list_categories` (T24)

- **Spec:** S24. **Goal:** list the known category labels.
- **Input:** `with_usage?` (default `false`).
- **Output:** `{ "categories": [ { "id", "name", "created_at", "usage_count"? } ],
  "count": 5 }`.
- **Rules:** ordered by `name`. With `with_usage`, `usage_count` is the number
  of movements that carry the exact label. If O1 was decided as "no
  catalogue", this tool instead returns the distinct `movements.category`
  values and `add_category` does not exist.
- **Files:** `packages/core/application/list_categories.go`,
  `packages/core/adapters/sqlite/categories.go`.
- **Tests:** empty; several labels; `with_usage` counts the movements; a
  label with no movements reports zero.
- **Done when:** `make all` is green.

---

### Phase 2 — The money writes

#### S12 — `record_expense` (T02)

- **Spec:** S2. **Goal:** record money going out.
- **Input:** `account_id`, `amount` (decimal string), `occurred_on`
  (YYYY-MM-DD), `category?`, `counterparty_contact_id?`, `notes?`,
  `idempotency_key?`, `actor?`.
- **Output:** `{ "movement_id": 12, "kind": "expense", "amount_cop": 15000,
  "occurred_on": "2026-10-19" }`.
- **Rules:** an unknown account is `not_found`; a closed account is
  `account_inactive`; a non-`COP` account is
  `currency_not_supported_in_mvp1`; an amount ≤ 0, more than 4 decimal
  places, or a non-numeric amount is `validation_failed`; a date that is not
  a real ISO date is `validation_failed`. The amount is rounded half-up to
  integer COP and the rounded value is echoed. An unknown
  `counterparty_contact_id` is `not_found`. The `category` is stored trimmed
  and lower-cased.
- **Writes:** one `movements` row with `kind = 'expense'`; the audit row and
  the idempotency row come from the spine.
- **Files:** `packages/core/application/record_expense.go`,
  `packages/core/adapters/sqlite/movements.go`, `apps/mcp/internal/tools/money.go`.
- **Tests:** the happy path with a decimal amount (`"15000.50"` → `15001`);
  the boundary at 4 decimals; 5 decimals refused; zero and negative refused;
  a non-numeric amount refused; a bad date refused; an unknown account; a
  closed account; a non-COP account; an unknown counterparty. Every refused
  case leaves the `movements` row count unchanged and writes no audit row.
- **Done when:** `make all` is green and a client can record an expense that
  `get_balance` later reflects.

#### S13 — `record_income` (T03)

- **Spec:** S3. **Goal:** record money coming in.
- **Input, output, rules:** as `record_expense`, with `kind = 'income'`.
  The service is shared with `record_expense`; the two tools differ only in
  the movement kind they write. Both are separate tools and separate sessions.
- **Files:** `packages/core/application/record_income.go` (or the shared
  movement service from S12), `apps/mcp/internal/tools/money.go`.
- **Tests:** the happy path; the same refusal set as S12, including the
  non-COP account.
- **Done when:** `make all` is green.

#### S14 — `record_transfer` (T04)

- **Spec:** S4. **Goal:** move money between two of my own accounts.
- **Input:** `from_account_id`, `to_account_id`, `amount` (decimal string),
  `occurred_on`, `category?`, `notes?`, `idempotency_key?`, `actor?`.
- **Output:** `{ "from_movement_id": 30, "to_movement_id": 31, "kind":
  "transfer", "amount_cop": 50000, "occurred_on": "2026-10-19" }`.
- **Rules:** both accounts must exist (`not_found`), be active
  (`account_inactive`), and be `COP` (`currency_not_supported_in_mvp1`).
  `from_account_id == to_account_id` is `validation_failed`. The two
  movements are written in **one transaction**: a failure writes neither. The
  two amounts are equal by construction.
- **Writes:** two `movements` rows, `transfer_out` and `transfer_in`.
- **Files:** `packages/core/application/record_transfer.go`,
  `packages/core/adapters/sqlite/movements.go`.
- **Tests:** the happy path; the same account twice refused; an unknown
  account; a closed account; a non-COP account; a bad amount; a bad date. The
  failure cases leave the `movements` row count unchanged — no half transfer.
- **Done when:** `make all` is green and the two movements cancel in
  `get_balance`.

#### S15 — `update_movement` (T05)

- **Spec:** S5. **Goal:** correct a movement that was recorded wrongly.
- **Input:** `movement_id`, and any of `amount`, `occurred_on`, `category`,
  `counterparty_contact_id`, `notes` to change. `kind` and `account_id` are
  **not** changeable: the kind decides the sign of the cash effect, and
  moving a movement to another account is a transfer. A request to change
  either is `validation_failed`.
- **Output:** `{ "movement_id": 12, "before": { ... }, "after": { ... } }`.
- **Rules:** an unknown movement is `not_found`. The new values are validated
  exactly as in S12: the amount is rounded half-up and echoed, the date must
  be a real ISO date, an unknown counterparty is `not_found`. The audit row
  stores the before and the after.
- **Files:** `packages/core/application/update_movement.go`,
  `packages/core/adapters/sqlite/movements.go`.
- **Tests:** changes one field; changes several; an unknown movement; a bad
  amount; a bad date; `kind` refused; `account_id` refused. Every refused
  case leaves the stored row unchanged.
- **Done when:** `make all` is green.

#### S16 — `delete_movement` (T06)

- **Spec:** S6, amendment A5. **Goal:** remove a movement that should never
  have existed.
- **Input:** `movement_id`, `idempotency_key?`, `actor?`.
- **Output:** `{ "deleted": true, "movement_id": 12 }`.
- **Rules:** an unknown movement is `not_found`. The row is removed (hard
  delete, unless O2 decides otherwise). A movement attached to a loan is
  removable; `loan_payments.movement_id` becomes `null` through the schema's
  `ON DELETE SET NULL`, and that is reported in the output as
  `"unlinked_loan_payment": true`. The audit row keeps the full before-state,
  so the deletion is recoverable from `audit_log`.
- **Depends on:** O2 (default: hard delete).
- **Files:** `packages/core/application/delete_movement.go`,
  `packages/core/adapters/sqlite/movements.go`.
- **Tests:** deletes a plain movement; deletes a movement linked to a loan
  payment and reports the unlink; an unknown movement; a refused case leaves
  the row count unchanged; the audit row records the deleted row.
- **Done when:** `make all` is green.

---

### Phase 3 — The money reads

#### S17 — `get_balance` (T07)

- **Spec:** S7. **Goal:** how much money there is, per account and in total.
- **Input:** `account_id?`, `include_inactive?` (default `false`).
- **Output:** `{ "accounts": [ { "account_id", "name", "kind", "balance_cop" } ],
  "total_cop": 350000 }`.
- **Rules:** the sign of a movement comes from its kind, exactly as the
  schema states: `income` (+), `expense` (−), `transfer_in` (+),
  `transfer_out` (−), `loan_received` (+), `loan_lent` (−), `loan_repayment`
  (−), `loan_recovered` (+). A closed account is excluded unless
  `include_inactive` is set. The grand total is the sum of the returned
  balances.
- **Files:** `packages/core/application/get_balance.go`,
  `packages/core/adapters/sqlite/movements.go`.
- **Tests:** an empty database returns zero balances and a zero total; a
  balance after an expense, an income, and a transfer pair; the sign of every
  one of the eight kinds; a closed account is excluded; `account_id` narrows
  the result.
- **Done when:** `make all` is green and the total equals the sum of the
  per-account balances.

#### S18 — `expenses_by_category` (T08)

- **Spec:** S8. **Goal:** where the money went.
- **Input:** `from` (YYYY-MM-DD), `to` (YYYY-MM-DD), `account_id?`,
  `category?`.
- **Output:** `{ "categories": [ { "category": "groceries", "total_cop": 320000,
  "count": 12, "share": 0.34 } ], "total_cop": 940000, "count": 41 }`.
- **Rules:** the period is inclusive on both ends. Only `expense` movements
  count. The totals are ordered by `total_cop` descending, then by category
  name. Movements with no category are grouped under the label
  `uncategorized`. `share` is the fraction of the period total, rounded to
  two decimals. A period with no expenses returns an empty list and a zero
  total, not an error. `from` after `to` is `validation_failed`.
- **Files:** `packages/core/application/expenses_by_category.go`,
  `packages/core/adapters/sqlite/movements.go`.
- **Tests:** an empty period; one category; several categories ordered by
  total; `uncategorized`; a category filter; an account filter; `from` after
  `to` refused; the period boundaries are inclusive.
- **Done when:** `make all` is green and the query uses the partial index
  `idx_movements_income_expense` (confirmed with `EXPLAIN QUERY PLAN`).

#### S19 — `income_vs_expenses` (T09)

- **Spec:** S9, amendment A4. **Goal:** the period totals.
- **Input:** `from`, `to`, `account_id?`.
- **Output:** `{ "from": "2026-10-01", "to": "2026-10-31", "income_cop": 4200000,
  "expense_cop": 3150000, "net_cop": 1050000, "income_count": 2,
  "expense_count": 37 }`.
- **Rules:** the period is inclusive on both ends. Only `income` and `expense`
  movements count. `net_cop` is `income_cop - expense_cop`, and it may be
  negative. The counts are the number of movements of each kind. The
  bucketed series is **not** part of this tool — that is `cashflow_summary`.
- **Files:** `packages/core/application/income_vs_expenses.go`,
  `packages/core/adapters/sqlite/movements.go`.
- **Tests:** an empty period; income only; expenses only; a negative net; the
  period boundaries are inclusive; `from` after `to` refused; an account
  filter.
- **Done when:** `make all` is green and the net equals income minus expense.

#### S20 — `cashflow_summary` (T10)

- **Spec:** S10, amendments A3 and A4. **Goal:** the movement of money over
  time.
- **Input:** `from`, `to`, `group_by` (`day` | `week` | `month`; default
  `month`), `account_id?`, `category?`.
- **Output:** `{ "group_by": "month", "buckets": [ { "period": "2026-10",
  "income_cop": 4200000, "expense_cop": 3150000, "net_cop": 1050000,
  "count": 39 } ], "total_income_cop": 4200000, "total_expense_cop": 3150000 }`.
- **Rules:** the buckets are ordered by period and are contiguous — a month
  with no movement is a bucket with zeros, not a missing one. `day` buckets on
  `YYYY-MM-DD`. `month` buckets on `YYYY-MM`. `week` buckets on the ISO week,
  `YYYY-Www`, computed in Go from the date rather than by SQLite's `%W`,
  whose week numbering is not ISO. `group_by: "project"` is rejected with
  `validation_failed` and a detail that says the schema has no project
  dimension yet (A3).
- **Files:** `packages/core/application/cashflow_summary.go`,
  `packages/core/adapters/sqlite/movements.go`.
- **Tests:** an empty period; one bucket; several buckets; a gap month is a
  zero bucket; `day`, `week`, and `month` bucketing; `group_by: "project"`
  refused; `from` after `to` refused; the totals equal the sum of the
  buckets.
- **Done when:** `make all` is green.

#### S21 — `list_recent_movements` (T11)

- **Spec:** S11. **Goal:** what happened lately.
- **Input:** `limit?` (default 20, maximum 100), `account_id?`,
  `counterparty_contact_id?`, `kind?`, `from?`, `to?`.
- **Output:** `{ "movements": [ { "id", "account_id", "kind", "amount_cop",
  "occurred_on", "category", "counterparty_contact_id", "loan_id", "notes" } ],
  "count": 20 }`.
- **Rules:** ordered by `occurred_on` descending, then `id` descending, so
  the newest is first and the order is stable within a day. A `limit` above
  100 is `validation_failed`. An unknown `kind` is `validation_failed`.
- **Files:** `packages/core/application/list_recent_movements.go`,
  `packages/core/adapters/sqlite/movements.go`.
- **Tests:** empty; the default limit; a custom limit; the limit ceiling
  refused; the ordering with two movements on the same day; the filters by
  account, counterparty, kind, and period.
- **Done when:** `make all` is green.

---

### Phase 4 — The debt tools

#### S22 — `record_loan` (T12)

- **Spec:** S12. **Goal:** record a debt contract.
- **Input:** `contact_id`, `direction` (`borrowed` | `lent`), `kind`
  (`informal` | `formal`), `principal` (decimal string), `interest_rate_bp?`,
  `granted_on` (YYYY-MM-DD), `due_on?`, `notes?`, `initial_movement?`
  (`account_id`, `amount?`, `occurred_on?`), `idempotency_key?`, `actor?`.
- **Output:** `{ "loan_id": 3, "direction": "lent", "kind": "informal",
  "principal_cop": 200000, "status": "active" }`.
- **Rules:** an unknown contact is `not_found`. The principal is rounded
  half-up to integer COP and must be greater than zero. `due_on` before
  `granted_on` is `validation_failed`. `interest_rate_bp` is a non-negative
  integer of basis points. The loan starts with `status = 'active'`. When
  `initial_movement` is present, the paired movement
  (`loan_lent` for `lent`, `loan_received` for `borrowed`) is written in the
  same transaction, linked by `loan_id`, and the account is validated as in
  S12. See O3.
- **Files:** `packages/core/application/record_loan.go`,
  `packages/core/adapters/sqlite/loans.go`.
- **Tests:** the happy path; an unknown contact; a zero principal; a
  `due_on` before `granted_on`; a negative rate; with and without
  `initial_movement`; the `initial_movement` against an unknown account. The
  refused cases leave the `loans` and `movements` row counts unchanged.
- **Done when:** `make all` is green.

#### S23 — `record_loan_payment` (T13)

- **Spec:** S13. **Goal:** settle one obligation of a loan.
- **Input:** `loan_id`, `seq` (the position in the schedule), `paid_on`
  (YYYY-MM-DD), `amount` (decimal string, the cash that actually moved),
  `account_id?` (omit when the money moved outside the tracked accounts),
  `notes?`, `idempotency_key?`, `actor?`.
- **Output:** `{ "loan_id": 3, "seq": 1, "scheduled_cop": 50000,
  "paid_cop": 50000, "remaining_cop": 0, "settled": true, "movement_id": 44 }`.
- **Rules:** an unknown loan is `not_found`; a `seq` that is not in the
  schedule is `not_found`; an already-paid obligation is `validation_failed`.
  When `account_id` is given, a movement (`loan_repayment` for `borrowed`,
  `loan_recovered` for `lent`) is written in the same transaction and linked
  to the obligation through `loan_payments.movement_id`. The obligation is
  marked paid (`paid_on` set) only when the paid amount is **at least** the
  scheduled amount; a smaller amount is a partial payment, which the schema
  represents by the movement alone, and the response reports
  `"settled": false` with the remaining amount. The loan's `status` becomes
  `settled` when every obligation is paid.
- **Files:** `packages/core/application/record_loan_payment.go`,
  `packages/core/adapters/sqlite/loan_payments.go`.
- **Tests:** a full payment with an account; a full payment without an
  account; a partial payment reports `settled: false`; an unknown loan; an
  unknown `seq`; an already-paid obligation; a bad date; a non-positive
  amount. The refused cases leave the obligation and the movement counts
  unchanged.
- **Done when:** `make all` is green and `outstanding_debts` reflects the
  payment.

#### S24 — `update_loan` (T14)

- **Spec:** S14. **Goal:** correct a loan's metadata.
- **Input:** `loan_id`, and any of `due_on`, `interest_rate_bp`, `notes` to
  change.
- **Output:** `{ "loan_id": 3, "before": { ... }, "after": { ... } }`.
- **Rules:** an unknown loan is `not_found`. `principal`, `direction`, `kind`,
  `contact_id`, and `status` are **not** changeable here: `principal` and
  `direction` change the meaning of the contract, and `status` is owned by
  `cancel_loan` and by the payment flow. A request to change one of them is
  `validation_failed`. A `due_on` before `granted_on` is `validation_failed`.
  The audit row stores the before and the after.
- **Files:** `packages/core/application/update_loan.go`,
  `packages/core/adapters/sqlite/loans.go`.
- **Tests:** changes `due_on`; changes the rate; changes the notes; an
  unknown loan; `principal` refused; `status` refused; `direction` refused; a
  `due_on` before `granted_on` refused. The refused cases leave the stored
  row unchanged.
- **Done when:** `make all` is green.

#### S25 — `cancel_loan` (T15)

- **Spec:** S15. **Goal:** retire a loan without destroying its history.
- **Input:** `loan_id`, `reason?`, `idempotency_key?`, `actor?`.
- **Output:** `{ "loan_id": 3, "status": "cancelled", "previous_status":
  "active" }`.
- **Rules:** an unknown loan is `not_found`. A loan is never deleted: the
  schema restricts deleting a loan that has movements, and the history is
  worth keeping. Cancelling a loan that is already `cancelled` is
  `validation_failed`. A `settled` loan is not cancelled — it is already
  finished, and cancelling it would be a lie. A cancelled loan stops counting
  in `outstanding_debts` and `list_active_loans`. The reason is appended to
  the notes.
- **Files:** `packages/core/application/cancel_loan.go`,
  `packages/core/adapters/sqlite/loans.go`.
- **Tests:** cancels an active loan; cancels a formal loan with movements;
  an unknown loan; an already-cancelled loan refused; a settled loan refused.
  The refused cases leave the stored row unchanged.
- **Done when:** `make all` is green.

#### S26 — `outstanding_debts` (T16)

- **Spec:** S16. **Goal:** who owes whom, right now.
- **Input:** `contact_id?`, `direction?` (`borrowed` | `lent`).
- **Output:** `{ "debts": [ { "contact_id", "contact_name", "direction",
  "loan_id", "principal_cop", "recovered_cop", "outstanding_cop", "due_on" } ],
  "total_outstanding_cop": 300000, "total_lent_cop": 0,
  "total_borrowed_cop": 300000 }`.
- **Rules:** only `active` loans are reported. The outstanding amount is
  `principal - recovered`, where `recovered` sums the linked movements:
  `loan_recovered` for a `lent` loan and `loan_repayment` for a `borrowed`
  loan. This is the reconciliation query the repository README already
  documents. A loan with nothing recovered is fully outstanding. A loan whose
  recovered amount equals or exceeds the principal is not reported. The
  totals cover the loans in the result, and the `contact_id` and `direction`
  filters narrow the result.
- **Files:** `packages/core/application/outstanding_debts.go`,
  `packages/core/adapters/sqlite/loans.go`.
- **Tests:** no loans; a lent loan with nothing recovered; a lent loan with a
  recovery; a borrowed loan with a repayment; a fully recovered loan is not
  reported; a cancelled loan is not reported; the filters; the totals.
- **Done when:** `make all` is green and the numbers match the README query.

#### S27 — `upcoming_loan_payments` (T17)

- **Spec:** S17. **Goal:** what is due soon.
- **Input:** `days_ahead` (default 7, maximum 365), `contact_id?`, `include_overdue?`
  (default `true`).
- **Output:** `{ "payments": [ { "loan_id", "contact_id", "contact_name", "seq",
  "scheduled_cop", "remaining_cop", "due_on", "days_until_due", "overdue" } ],
  "count": 2, "total_due_cop": 150000 }`.
- **Rules:** only obligations with `paid_on IS NULL` are reported. `due_on` is
  between today and today + `days_ahead`. An obligation whose `due_on` is in
  the past is `overdue: true` with a negative `days_until_due`, and is
  included only when `include_overdue` is set. The remaining amount is
  `scheduled - the linked movement amount`, so a partial payment shows what
  is still owed. The obligations are ordered by `due_on`, then by `seq`.
- **Files:** `packages/core/application/upcoming_loan_payments.go`,
  `packages/core/adapters/sqlite/loan_payments.go`.
- **Tests:** nothing due; one due inside the window; one outside the window;
  an overdue obligation; `include_overdue: false`; a partial payment; the
  ordering; the total.
- **Done when:** `make all` is green.

#### S28 — `loan_schedule` (T18)

- **Spec:** S18. **Goal:** the full repayment schedule of one loan.
- **Input:** `loan_id`.
- **Output:** `{ "loan": { "loan_id", "contact_id", "contact_name", "direction",
  "kind", "principal_cop", "interest_rate_bp", "granted_on", "due_on",
  "status" }, "schedule": [ { "seq", "scheduled_cop", "due_on", "paid_on",
  "movement_id", "remaining_cop" } ], "total_scheduled_cop": 200000,
  "total_paid_cop": 50000 }`.
- **Rules:** an unknown loan is `not_found`. The schedule is the recorded
  `loan_payments` rows, ordered by `seq`. A loan with no recorded schedule
  returns an empty schedule and a `schedule_recorded: false` flag — the
  schedule is not invented, and no amortisation is projected. The remaining
  amount per obligation is `scheduled - the linked movement amount`.
- **Files:** `packages/core/application/loan_schedule.go`,
  `packages/core/adapters/sqlite/loan_payments.go`.
- **Tests:** a loan with a schedule; a loan with no schedule; a paid
  obligation; a partial payment; an unknown loan; the totals.
- **Done when:** `make all` is green.

#### S29 — `list_active_loans` (T25)

- **Spec:** S25. **Goal:** the open contracts at a glance.
- **Input:** `contact_id?`, `direction?`, `kind?`.
- **Output:** `{ "loans": [ { "loan_id", "contact_id", "contact_name",
  "direction", "kind", "principal_cop", "outstanding_cop", "interest_rate_bp",
  "granted_on", "due_on", "status" } ], "count": 2 }`.
- **Rules:** only `status = 'active'` loans are returned. The outstanding
  amount is the same reconciliation as S26. The loans are ordered by
  `granted_on` descending, then by `id` descending. The filters narrow the
  result.
- **Files:** `packages/core/application/list_active_loans.go`,
  `packages/core/adapters/sqlite/loans.go`.
- **Tests:** no loans; one active loan; a cancelled loan excluded; a settled
  loan excluded; the outstanding amount; the filters; the ordering.
- **Done when:** `make all` is green and the outstanding amount agrees with
  S26.

---

## Definition of done for MVP1

- All 25 tools are implemented, each with a service-layer unit test and an
  MCP integration test over `httptest` (C11).
- All 11 cross-cutting concerns (C1–C11) have at least one passing test.
- The 57 existing schema checks still pass, and migration `002` adds checks
  for the new tables.
- `make all` is green on `main`.
- The ADR carries the amendments A1–A6.
- The open decisions O1, O2, and O3 are settled and recorded in this
  document's `## Log`.

---

## Log

### L1 — Origin (verbatim)

> list all the functionalities to implement in the mvp 1.

### L2 — The request for a per-session roadmap (verbatim)

> i wanna you add the entire from `health` to `list_active_loans` individually. In order to have a roadmap to achieve. I wanna build functionality per session.

### L3 — Findings that shaped the roadmap

- The ADR's tool tables list 25 tools, not the 22 its prose claims (A1).
- Go's `internal` visibility rule makes the ADR's shared-core layout
  impossible as written: `apps/mcp` cannot import `apps/cli/internal/...`
  (A2).
- The schema has no project dimension, so `cashflow_summary` cannot group by
  project in MVP1 (A3).
- `movements.category` is free text with no catalogue behind it, so
  `add_category` and `list_categories` need a table (A6).
- The ADR's "soft via notes flag" delete leaves the row in every balance
  forever; a hard delete plus the audit log is the honest version (A5).
- The ADR's error taxonomy has no code for a uniqueness violation, and
  `duplicate_request` is reserved for idempotency, so uniqueness violations
  map to `validation_failed` in MVP1 (A7).

### L4 — Defaults taken for the open decisions

Unless the user overrides them, the roadmap proceeds with: a curated
`categories` table (O1), a hard delete with the audit log as the recovery
trail (O2), and a loan that does not move cash unless the caller asks for an
`initial_movement` (O3).

### L5 — Sessions

One entry per completed session, newest last. Each records the branch, the
commit, and the outcome.

- **S00 — Core module extraction.** Branch `feat/mcp-s00-core-extraction`,
  commit `88032ee`, issue
  [#7](https://github.com/witsaba/paca/issues/7), pull request
  [#8](https://github.com/witsaba/paca/pull/8). The layers `domain`, `ports`, and `application` moved from
  `apps/cli/internal/` into the new module `packages/core`
  (`github.com/witsaba/paca/packages/core`); `apps/cli` requires it with a
  `replace` to the local path, and `go.work` lists the module. The product
  identity (`PACA`, the description) moved into the root package of the core, so
  the dependency is real and `go mod tidy` keeps it; the CLI presents the same
  bytes as before, and a test pins them. `make all` is green over the three
  modules, and `make help` lists `apps/cli` and `packages/core`. The ADR
  carries amendment A2 (the path change) and an amendments table that tracks
  A1, A3–A6 to the session that needs each one.

- **Slice boundary of the S00 branch.** The branch carries two commits: the code
  slice `88032ee` (about 65 changed lines of Go and manifests, the rest renames
  of the moved packages) and the documentation slice `5e927d7` (this roadmap
  document, 928 lines, which the ADR links to and which did not exist in the
  repository yet).

### L6 — The decisions of the S00 session

The user answered the four open questions of the session before any code was
written:

1. **Module layout** — a separate module with its own `go.mod`, listed in
   `go.work`.
2. **Scope of the move** — `domain`, `ports`, and `application`; the CLI
   adapter stays in `apps/cli/internal/adapters/cli`.
3. **Linking** — `require` plus a `replace` to `../packages/core`, so the CLI
   module builds on its own. (The path is `../../packages/core`: the
   `replace` is relative to the module's own directory, `apps/cli`.)
4. **Import path** — `github.com/witsaba/paca/packages/core/...`.

One addition was made to keep the new dependency honest: the root package of
the core holds the product identity, and the CLI presents it. Without a real
import, `go mod tidy` would drop the `require` and the shared core would not
be wired to anything.
