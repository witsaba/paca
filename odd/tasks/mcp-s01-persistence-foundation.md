# S01 — Persistence foundation

Session S01 of the
[PACA MCP server MVP1 roadmap](mcp-server-mvp1.md): the real SQLite adapter,
the migration runner, and the typed error taxonomy the rest of the spine
builds on.

- **Roadmap entry:** S01 — Persistence foundation (Phase 0 — The spine).
- **Branch:** `feat/mcp-s01-persistence-foundation`, cut from an up-to-date `main`.
- **Done when:** `make all` is green, and a migration `002` can be added and
  applied without touching the runner.

---

## Specs

Each spec is quoted verbatim. `R#` quotes the roadmap; `U#` quotes the user.

- **R1** — Goal: "**Goal:** a real SQLite adapter."
- **R2** — Spec: "**Spec:** ADR §Architecture, C8, C11."
- **R3** — Change: "`packages/core/adapters/sqlite` implements the repository
  interfaces declared in `packages/core/ports`. The driver is
  `modernc.org/sqlite` (pure Go, no CGO — settled in ADR 0001). Every
  connection sets `PRAGMA foreign_keys = ON` (per-connection, never persisted
  by the migration). A migration runner applies
  `packages/db/migrations/*.sql` in order, checks `PRAGMA user_version`, and
  aborts on the first error (the `-bail` discipline of migration `001`).
  SQLite constraint violations are translated into typed domain errors, never
  leaked as raw driver strings."
- **R4** — Tests: "**Tests:** every repository method against a temporary
  database; a foreign-key enforcement test; a runner test that refuses a
  second run of migration `001`; a test that a constraint violation maps to a
  typed error and not to `internal_error`."
- **R5** — Done when: "**Done when:** `make all` is green, and a migration
  `002` can be added and applied without touching the runner."
- **R6** — House rule: "The commit is a reviewable slice of roughly 150–400
  authored changed lines, which is the house limit."
- **R7** — Session protocol: "**Write the test first.** One RED test per rule
  the tool's entry names: the happy path, every named error, and the boundary
  cases. The happy path goes RED before the implementation exists."
- **R8** — ADR C7: the stable error codes are `validation_failed`,
  `not_found`, `duplicate_request`, `currency_not_supported_in_mvp1`,
  `account_inactive`, `loop_required_for_unsafe_call`, `internal_error`.
  "Responses include `code` and a human-readable `detail`; no DB error
  strings, file paths, or stack frames in `detail`."
- **R9** — ADR C8: "`paca-mcp` holds the SQLite file open. Migration procedure
  documented: **stop server → apply migration → start server**. SQLite's write
  lock makes the brief window safe. WAL mode and online migrations are out of
  scope for MVP1."
- **R10** — ADR C11: "The 57-check verifier in
  [`packages/db/tools/verify_schema.sh`](../../packages/db/tools/verify_schema.sh)
  keeps passing."

### Decisions of this session (user)

- **U1** — Scope: "Full S01, split into work-unit commits (recommended)" —
  one roadmap session, one branch, several reviewable commits: foundation +
  runner, then one commit per table (accounts, contacts, loans, movements,
  loan_payments). Honors the roadmap scope and the 400-line limit.
- **U2** — Migrations source: "packages/db becomes a Go module that embeds the
  SQL (rec.)" — new module `github.com/witsaba/paca/packages/db` with
  `go:embed` over `migrations/*.sql`, exposing an `embed.FS`;
  `packages/core` requires it with a `replace`, mirroring the S00 pattern. The
  SQL stays exactly where the ADR and roadmap say it belongs; `go.work` gains
  a fourth module.
- **U3** — Domain errors: "Full C7 taxonomy in S01" — S01 implements every
  stable code (R8) with the detail rules. **Consequence:** roadmap session
  S03 shrinks to the money value object, the response envelope, and actor
  resolution; the taxonomy is delivered here.

---

## Delivery strategy

The branch carries **one work-unit commit per task** (T1–T13), not one
oversized commit. The house limit is roughly 400 authored changed lines per
reviewable slice, and the scope of S01 is decided by the user (U1): the full
persistence foundation, split into work units so every commit stays reviewable
on its own.

- **Slice boundary:** the branch is past a single 400-line slice. It is a
  stack of reviewable commits, each one a behavior unit with its tests
  alongside it, in the same shape as the S00 branch (a code slice and a
  documentation slice).
- **Pull request:** the branch is over 400 lines, so it is opened as a
  **chained PR** — one PR per work unit, stacked on the branch — unless the
  user asks for a single PR instead. The decision belongs to the user.
- **Merge order:** the commits merge in order; the last one closes the
  session. A commit is never half-merged.

---

## Tasks

| ID | Task | Spec | Route | Commit |
|---|---|---|---|---|
| T1 | Branch `feat/mcp-s01-persistence-foundation` from an up-to-date `main`; commit the pending `README.md` deferred-decisions edit from the previous session as its own documentation slice | R6 | parent | `0f59dd6` |
| T2 | New module `packages/db` (`github.com/witsaba/paca/packages/db`) with `go:embed` over `migrations/*.sql`; `go.work` and the Makefile module list pick it up. The `require` + `replace` from `packages/core` lands with T5, in the same commit as the import that makes it real | R3, U2 | parent | `e136591` |
| T3 | Full C7 error taxonomy in `packages/core/domain`: typed errors, the seven stable codes, and a `detail` that never carries DB internals, file paths, or stack frames | R8, U3 | parent | `490a31f` |
| T4 | SQLite adapter foundation in `packages/core/adapters/sqlite`: open/close, `PRAGMA foreign_keys = ON` per connection, and the translation of SQLite constraint violations into typed domain errors | R3, R4 | parent | `de926af` |
| T5 | Migration runner: applies the embedded `*.sql` in numeric order, reads and checks `PRAGMA user_version`, aborts on the first error; the "a migration `002` applies without touching the runner" proof uses a fixture, because the real `002` belongs to S04 | R3, R4, R5 | parent | `6b70ba7` |
| T6 | Domain entities for the five tables and the repository port interfaces in `packages/core/ports`; the sign of a kind of movement, which is a rule of the business and not of the storage | R3 | parent | `036b31d`, `73cdaaf` |
| T7 | `accounts` repository: every method against a temporary database | R4 | parent (the reference of the other four) | `59a0b1b` |
| T8 | `contacts` repository: every method against a temporary database | R4 | worker | `edd0a65` |
| T9 | `loans` repository: every method against a temporary database | R4 | worker | `6c8111f` |
| T10 | `movements` repository: every method against a temporary database | R4 | worker | `7a4d357` |
| T11 | `loan_payments` repository: every method against a temporary database | R4 | worker | `d08a07f` |
| T12 | Verification: `make all` green over the four modules, the 57-check verifier green, and a foreign-key enforcement test proving `PRAGMA foreign_keys = ON` is per connection | R4, R10 | parent | — (no code) |
| T13 | Documentation: roadmap entry S01 → Done with the commit hash, `## Log` gains the session entry and the decisions of U1–U3; `packages/core/README.md`, `packages/db/README.md` and the ADR amendments table updated | R5 | parent | `16e4401`, and the closing slice of the branch |

---

## Log

### L1 — Origin (verbatim)

> check in engram about the S00 which was done in the previous session and give
> a recap the next one to do

> let start with the exploration

### L2 — Findings that shaped this session

- `packages/core/domain`, `packages/core/ports` and `packages/core/application`
  held only a `doc.go`. S01 therefore *creates* the entities, the ports and the
  adapter; it does not implement interfaces that already exist.
- Full CRUD over the five tables is roughly 900–1200 authored lines, against a
  house limit of about 400 per commit (R6). Hence the work-unit split of U1.
- `packages/db/migrations` sits outside the `packages/core` module, and
  `go:embed` cannot cross a module boundary. Hence the `packages/db` module
  of U2.
- Migration `001` manages its own transaction (`PRAGMA foreign_keys = ON;`
  before `BEGIN IMMEDIATE;`, `PRAGMA user_version = 1;` inside it,
  `COMMIT;` at the end). The runner must not wrap it: it executes the file as
  a script and reads `user_version` back.
- Migration `001` carries the emptiness precheck (a temporary table and a
  `RAISE(ROLLBACK)` trigger), so a second run rolls the whole file back. That
  is the "refuses a second run" test of R4.
- The real migration `002` belongs to S04 (`audit_log`, `idempotency_keys`, and
  `categories` under decision O1). The R5 proof in S01 uses a fixture
  migration, not the real `002`.
- `modernc.org/sqlite` is available on the module proxy; the pure-Go driver
  needs no CGO, and `make all` scans its dependency tree with `govulncheck`.
- The working tree carried an uncommitted `README.md` edit from the previous
  session (the settled driver and single-user deferred decisions). T1 commits
  it as its own documentation slice.

### L3 — The decisions of this session (verbatim)

1. **Scope S01** — "Full S01, split into work-unit commits (recommended)".
2. **Migrations source** — "packages/db becomes a Go module that embeds the
   SQL (rec.)".
3. **Domain errors** — "Full C7 taxonomy in S01".

### L3b — The delivery of the session

- Branch `feat/mcp-s01-persistence-foundation`, pushed to `origin`.
- Issue [#9](https://github.com/witsaba/paca/issues/9) (`enhancement`), pull
  request [#10](https://github.com/witsaba/paca/pull/10) (`type:feature`).
- The branch carries 16 work-unit commits, and the pull request is opened as
  one: the review map is the commit list. 40 files, 5,187 insertions, 86
  deletions.

### L4 — Evidence

- Roadmap S01 entry, quoted in R1–R5: `odd/tasks/mcp-server-mvp1.md`.
- ADR C7, C8, C11: `docs/architecture/0001-mcp-agent-server.md`.
- Schema and the emptiness precheck: `packages/db/migrations/001_initial_schema.sql`.
- The 57-check verifier: `packages/db/tools/verify_schema.sh`.

### L5 — Findings of the implementation

- **The `holder` column of `accounts` can never hold a value.** The
  `accounts_shape` CHECK constraint requires `holder IS NULL` for every kind
  of account, while the reference of the repository documents the column as a
  field of a Nequi account. The constraint is the one that is enforced, so
  the domain does not model the column and the repository does not write it.
  The two documents of the repository disagree, and the disagreement is
  reported to the user: the fix belongs to the schema or to the reference,
  and it has to be taken before the migration `002` of session S04.
- **The runner is idempotent, and the migration is the one that refuses.**
  The runner applies only the migrations the database has not applied, so
  running it twice changes nothing. The initial schema, run against a
  database that already holds tables, refuses itself and leaves the database
  exactly as it was. Both properties are tested.
- **The order of the migrations belongs to the runner**, and not to the module
  that publishes the files: the runner has to order the migrations of any
  filesystem, because the proof that a migration added later applies without
  a change to the runner is a fixture, and not a file of the repository.
- **The sign of a kind of movement is a rule of the business.** The amount of
  a movement is always positive, and the kind says which way the money went.
  The sign lives in the domain, so the balance of an account, the total of a
  period and the total of a category all agree.

### L6 — Evidence of the verification

- `make all` exits 0 over the four modules of the workspace (`apps/cli`,
  `packages/core`, `packages/db`, `tools`): `fmt-check`, `vet`, `lint` (0
  issues), `build`, `test`, `race` and `vuln` (0 vulnerabilities in the code).
- The 57-check verifier of the schema passes: 57 passed, 0 failed.
- The CLI is untouched: `paca --help`, `paca` with no arguments and
  `paca accounts` behave exactly as before the session.
- The runner tests cover the four rules of R4: the schema applies, the
  initial schema is not applied twice, a migration added later applies
  without a change to the runner, and a constraint violation maps to a typed
  error and not to `internal_error`.
