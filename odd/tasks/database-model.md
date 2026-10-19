# Feature: Database Model (Model B — Single Table Inheritance)

Adds the PACA database schema as a runnable SQLite migration plus the data-team
documentation. Database only: no application code, no application stack, no runtime.

## Specs

### S1 — Scope of the deliverable

The user constrained this feature to exactly two artifacts:

> "add just the datbase script that allows to do the migration, and a README.md documentation
> with the model view and the needed information for the data team. Only this, dont think yet
> in code, software additional. JUST the database. The migration will be later when starting
> working in the software."

Therefore: **one migration script** and **one README**. No ORM, no application source, no
service layer, no test harness, no CI config, no dependency manifest. The migration script is
authored and verified in this feature but **not applied to any PACA database** — the user
states it will be applied later, when software work begins.

### S2 — Database engine and storage model

> "The database choosen is SQLite due to is all that we need so far."

SQLite is the only supported engine. No alternative-engine compatibility layer, no
Postgres-flavored constructs, no ORM-specific type mapping.

### S3 — Key and type conventions

> "Use the most basic datatypes for keys and follow the best practices to have structured data
> for the future."

Concretely, from the measured analysis (see Evidence):

| Rule | Detail |
| --- | --- |
| Surrogate key | `INTEGER PRIMARY KEY` on every table — it is a rowid alias, giving the fastest possible lookup. `INT PRIMARY KEY` is **not** a rowid alias and is not used. `WITHOUT ROWID` is documented as slower for single-column INTEGER primary keys and is **not** used. |
| Type safety | `STRICT` on every table. Type violations raise `SQLITE_CONSTRAINT_DATATYPE` at write time instead of silently coercing. `STRICT` cannot be added to a populated table without a rebuild, so it is decided now. |
| Money | `INTEGER` Colombian pesos. Never floating point. `amount > 0` enforced by CHECK; the sign of a movement is carried by its `kind`, not by the amount. Interest is stored in basis points (`interest_rate_bp`) as an integer. |
| Dates | `TEXT` in ISO-8601. Business dates are `YYYY-MM-DD`; audit timestamps are `YYYY-MM-DDTHH:MM:SS.sssZ` produced by `strftime('%Y-%m-%dT%H:%M:%fZ','now')`. Lexicographic order equals chronological order, so range scans work without a date type. |
| Foreign keys | Declared with an explicit `ON DELETE` policy on every reference. `PRAGMA foreign_keys = ON` must be set on **every** connection — SQLite disables FK enforcement by default. |
| Indexing | Every foreign-key child key gets an index, otherwise a parent delete forces a full scan of the child table. Hot subsets get partial indexes. Report queries get covering indexes. |

### S4 — The inheritance model

> "I usually use the inherit models like in OOP (oriented object programming) in order to
> extend tables. Example if I got a Accounts table, I wanna in a future extend a kind of
> account_nequi where the primary key in this table is the same in the accounts one but here
> I only extend nequi for specifically nequi information, same for contacts or something like
> that."

> "ok, sound good the B model is strong to me."

**Model B (Single Table Inheritance) is the accepted model.** Subtype information lives in
the same table as the base entity, in dedicated nullable columns, gated by a `kind`
discriminator column. A `CHECK` constraint makes invalid states unrepresentable: the columns
of a subtype that do not apply to the active `kind` must be NULL, and the discriminating
column of the active kind must be populated.

**SQLite has no table inheritance.** `CREATE TABLE ... INHERITS` does not exist in SQLite.
The user's OOP shape is therefore expressed as a discriminator plus a `CHECK` constraint, and
the polymorphic shape is surfaced in the documentation as an explicit model view so the data
team can read it the way they think.

The user was shown both candidates with measurements and selected Model B explicitly.

### S5 — Entities in scope

Derived from the project's Engram purpose record and the MVP scope, and confirmed by the
user's own description of their financial setup:

| Entity | Purpose | Kinds |
| --- | --- | --- |
| `accounts` | Where money is held | `nequi`, `bank`, `cash` |
| `contacts` | People and entities transacted with | `person`, `entity` |
| `movements` | Every money event | `income`, `expense`, `transfer_in`, `transfer_out`, `loan_received`, `loan_lent`, `loan_repayment`, `loan_recovered` |
| `loans` | A debt owed to a contact | `informal`, `formal`, plus a `direction` of `borrowed` or `lent` |
| `loan_payments` | The repayment obligations of a loan | — |

The `movements.kind` vocabulary has eight values, chosen so that the sign of the cash effect is
unambiguous and `amount` can always be stored as a positive number. The sign is carried by the
kind, never by the amount:

| kind | cash effect |
| --- | --- |
| `income` | in (+) |
| `expense` | out (-) |
| `transfer_in` | in (+) |
| `transfer_out` | out (-) |
| `loan_received` | in (+) |
| `loan_lent` | out (-) |
| `loan_repayment` | out (-) |
| `loan_recovered` | in (+) |

A contact of kind `person` carries a phone and an identity document (cédula). A contact of
kind `entity` carries a phone, a tax id (NIT) and a legal name. Informal credits come from
natural persons and have no fixed payment schedule, which is why `loans.due_on` is nullable
and `interest_rate_bp` is nullable.

### S5b — Uniqueness policy per kind

Uniqueness is enforced only where the identifier is genuinely unambiguous:

| Identifier | Policy | Reason |
| --- | --- | --- |
| `accounts.phone` for `kind='nequi'` | UNIQUE (partial index) | A Nequi wallet is identified by its phone number. |
| `accounts.(bank_code, account_number)` for `kind='bank'` | UNIQUE (partial index) | A bank account is identified by the pair. |
| `contacts.tax_id` for `kind='entity'` | UNIQUE (partial index) | A NIT identifies exactly one company. |
| `contacts.id_document` for `kind='person'` | indexed, **not** unique | The same person may legitimately appear twice, for example as a service provider and as a lender. |

### S6 — How the migration is applied and tracked

> "add just the datbase script that allows to do the migration"

> "The migration will be later when starting working in the software."

The deliverable is a script that a later migration runner can apply. It must therefore:

- be a single self-contained SQL file, applied with
  `sqlite3 -bail paca.db < packages/db/migrations/001_initial_schema.sql`;
- apply atomically (all DDL in one transaction, so a failure leaves the database untouched);
- record its own version so a future runner can detect what is already applied;
- be safe to apply against an empty database and to fail loudly against a non-empty one;
- **require bail-on-error semantics**, because the sqlite3 CLI continues past an error by
  default and a migration that keeps going can leave a half-applied schema. A future runner
  must abort on the first error and roll the transaction back.

### S6b — Verifiable schema behavior

The schema must be checkable without an application, a language runtime, or any dependency
beyond the sqlite3 CLI. The deliverable includes a verification script that applies the
migration to a throwaway database and asserts the documented behavior: structure, every
CHECK constraint, the uniqueness policy, referential actions, and the guard. It must run
green and must be re-runnable by the data team at any time.

### S7 — Documentation scope

> "a README.md documentation with the model view and the needed information for the data team"

The README must give the data team the model view (how the tables relate, and how the
`kind` discriminator maps to the subtype concept they use), a complete per-table and
per-column reference, the constraints and indexes with their rationale, the conventions, and
the procedure to apply the migration. It must be sufficient to work with the schema without
reading the SQL.

## Tasks

| ID | Specs | Task | Route | Commit |
| --- | --- | --- | --- | --- |
| T1 | S1, S2, S3, S4, S5, S5b, S6 | Author `packages/db/migrations/001_initial_schema.sql` — Model B schema, 5 STRICT tables, CHECK constraints, 12 indexes, FK policies, `RAISE(ROLLBACK)` guard, `user_version` stamping | inline | `d36a6bb` |
| T2 | S7 | Author `README.md` — model view, table and column reference, constraints, indexes, conventions, application procedure | delegated | `0e9ea5d` |
| T3 | S1, S3, S4, S5, S6, S6b | Author `packages/db/tools/verify_schema.sh` — applies the migration to a throwaway database and asserts structure, every CHECK, the uniqueness policy, referential actions and the guard | inline | `254c5ea` |
| T4 | S1, S7 | Author `packages/db/README.md` — package contents, how to apply, how to verify, how to add a migration | delegated | `dad8205` |

The migration, the harness that proves it, and the documentation that describes them are
one behavior, but the work is too large for a single reviewable commit. They are delivered
as a stack of work units, described next.

### Delivery strategy

The feature is larger than a single review slice. The work is committed as a stack of
work units, each of which is a candidate chained PR, so a reviewer can take one slice at
a time instead of reading the whole feature at once.

| Slice | Commit | Contents | Lines |
| --- | --- | --- | --- |
| 1 | `2b4aaba` `chore: ignore SQLite database files and build output` | `.gitignore` | 27 |
| 2 | `d36a6bb` `feat(db): add SQLite schema for the personal finance manager` | the migration | 389 |
| 3 | `254c5ea` `test(db): add schema verification harness` | the harness | 210 |
| 4 | `0e9ea5d` `docs(db): document the database model for the data team` | the root README | 679 |
| 5 | `dad8205` `docs: add package and reserved-directory READMEs` | package and directory READMEs | 105 |
| 6 | `docs(odd): record the database model feature` | this document | 391 |

Slices 1 and 2 are a prerequisite for the rest: the database package does not exist until
they land. Slices 3 to 6 are independent of each other and can be reviewed in any order
once the schema is present.

The root README is deliberately one commit. It is a single reference document, and
splitting it would leave an intermediate state where the data team has half a schema
reference and no way to know which half is current.

**Verification per slice:**

| Slice | Command | Result |
| --- | --- | --- |
| 2, 3, 4, 5, 6 | `bash packages/db/tools/verify_schema.sh` | 57 checks, 0 failures |

**Rollback boundary:** reverting slice 2 removes the schema; slice 3 is the harness that
proves it. Reverting slices 4 and 5 removes only documentation. No slice removes unrelated
work.

## Evidence

All figures below were measured locally, not estimated. Schemas were created, populated
(2,000 accounts, 2,000 contacts, 200,000 movements, ~27 MB per database) and benchmarked on
SQLite 3.54.1, macOS arm64. Timings are the minimum of 50–200 runs after warm cache;
medians were within 10%.

### Read comparison

| Query | Model A (CTI) | Model B (STI) | Winner |
| --- | --- | --- | --- |
| Kind filter, Nequi accounts only | 0.054 ms | 0.005 ms | B ~10x |
| Polymorphic read, 20k rows, 1 subtype | 5.79 ms | 3.92 ms | B |
| Polymorphic read, 20k rows, 2 subtypes | 8.53 ms | 5.11 ms | B |
| Polymorphic read, 20k rows, 4 subtypes | 12.02 ms | 6.89 ms | B |
| Polymorphic read, 20k rows, 8 subtypes | 18.28 ms | 10.03 ms | B |
| Polymorphic read, 20k rows, 16 subtypes | 30.55 ms | 15.59 ms | B |
| Per-account balance (GROUP BY) | 0.05 s | 0.05 s | tie |
| Monthly income/expense report | 0.03 s | 0.03 s | tie |
| Model A with a `UNION ALL` view (mitigation) | 8.92 ms | — | worse than the raw join |

Model A's polymorphic read cost grows linearly with the number of subtypes; Model B stays
flat. This was the deciding factor.

### Write comparison

| Operation | Model A | Model B |
| --- | --- | --- |
| Create account (parent + subtype row, one transaction) | 0.157 ms | 0.151 ms |
| Create contact (parent + subtype row, one transaction) | 0.157 ms | 0.151 ms |

Indistinguishable. The usual argument for single-table inheritance — that the alternative
costs a join on write — does not hold here, so Model B carries no write penalty.

### Integrity comparison

| Rule | Model A (CTI) | Model B (STI) |
| --- | --- | --- |
| Subtype row cannot reference a wrong-kind parent | requires a trigger | CHECK |
| Subtype fields cannot be set on a wrong kind | requires a trigger | CHECK |
| Parent must have exactly one subtype row | **unenforceable** — verified to insert cleanly | not applicable, one row |
| Orphan subtype row | rejected only with FK enforcement on | rejected only with FK enforcement on |
| Uniqueness scoped to one subtype | natural `UNIQUE` | partial index |
| Cross-kind data mixing | impossible by construction | rejected by CHECK |
| Error message accuracy | **masked** — the kind trigger fires before the FK check and reports a kind mismatch for what is really a missing parent | accurate |

### Schema evolution — adding a fourth account type

| | Model A | Model B |
| --- | --- | --- |
| New table | required | not required |
| New trigger | required | not required |
| Parent table rebuild | **required** below SQLite 3.53.0 | required below 3.53.0, avoidable on 3.53.0+ |
| Trigger breakage during rebuild | **yes** — child triggers reference the parent by name and the migration aborts with `no such table: main.accounts` | not applicable |
| Measured rebuild cost, 500 rows | 0.9 ms | 0.9 ms |
| `integrity_check` / `foreign_key_check` after | ok / 0 violations | ok / 0 violations |

### Engine facts, with the version that supports them

| Fact | Version | Source |
| --- | --- | --- |
| No table inheritance; `CREATE TABLE ... INHERITS` does not exist | all | sqlite.org |
| `ALTER TABLE ADD/DROP CONSTRAINT` | **3.53.0** (2026-04-09) | sqlite.org/changes.html |
| `STRICT` tables; types limited to INT, INTEGER, REAL, TEXT, BLOB, ANY | 3.37.0 | sqlite.org/stricttables.html |
| `INTEGER PRIMARY KEY` is a rowid alias; `INT PRIMARY KEY` is not | 3.37.0 | sqlite.org/stricttables.html |
| Foreign keys are **disabled by default**; enable per connection with `PRAGMA foreign_keys = ON` | 3.6.19 | sqlite.org/foreignkeys.html |
| Parent key must be PRIMARY KEY or UNIQUE; child key columns need their own index | 3.6.19 | sqlite.org/foreignkeys.html |
| `WITHOUT ROWID` is faster only for non-integer or composite primary keys | 3.8.2 | sqlite.org/withoutrowid.html |
| Generated columns cannot be part of the PRIMARY KEY | 3.31.0 | sqlite.org/gencol.html |

The local system CLI is **3.51.0** and rejects `ADD CONSTRAINT`; the local Python `sqlite3`
module links **3.54.1** and supports it. A future migration runner must pin and assert a
3.53.0+ floor rather than assume it, and must not rely on the system CLI.

## Log

### L1 — Original request (verbatim)

> Let start a new feature. This feature is about the database model. First at all check in
> engram about the information and purpose of this project and let me know when ready

### L2 — Scope of the database model (verbatim)

> I wanna you focus in the database model. The database choosen is SQLite due to is all that
> we need so far. I need you deep take a perspective of a database engineer and focus in
> building a very fast database model structuring the tables. Use the most basic datatypes
> for keys and follow the best practices to have structured data for the future. I usually use
> the inherit models like in OOP (oriented object programming) in order to extend tables.
> Example if I got a Accounts table, I wanna in a future extend a kind of account_nequi where
> the primary key in this table is the same in the accounts one but here I only extend nequi
> for specifically nequi information, same for contacts or something like that. Your task here
> is analyze this, explore, even search in the web and purpose two models with a pro/cons and
> tradeoff stuff.

### L3 — Read-before-decide (verbatim)

> Do not ask to me yet, show me the tables with all information you collect so far I need
> first read it, then you can ask anything

> resume it in a tabl

### L4 — Model decision (verbatim)

> ok, sound good the B model is strong to me. Now, let work in a git worktree. I need you use
> ODD for adding this feature. add just the datbase script that allows to do the migration,
> and a README.md documentation with the model view and the needed information for the data
> team. Only this, dont think yet in code, software additional. JUST the database. The
> migration will be later when starting working in the software.

### L5 — Decision record

Model B (Single Table Inheritance) selected by the user over Model A (Class Table
Inheritance). The user's original OOP-inheritance request is satisfied at the documentation
level through the model view, and at the schema level through the `kind` discriminator plus
CHECK constraints.

Scope confirmed as database-only: migration script and README. No application code.

Monorepo layout adopted on request: `apps/`, `packages/db/` (migrations and tools), `docs/`,
`odd/`, `tools/`. Only `packages/db/` holds content in this feature; the rest are reserved
placeholders.

### L6 — Findings from implementing and verifying the schema

These were discovered by executing the migration, not by reading about it. They changed the
artifact.

1. **A CHECK constraint cannot detect an invalid value by returning NULL.** The date check
   was first written as `CHECK (strftime('%Y-%m-%d', d) = d)`. SQLite treats a NULL
   comparison as *satisfied*, so a malformed date such as `01/01/2024` was silently accepted
   while `strftime` returned NULL. The check must use `IS` rather than `=`. This was caught
   by the verification harness and fixed; `strftime('%Y-%m-%d', d) IS d` rejects malformed
   input and impossible calendar dates such as `2024-02-30` and `2023-02-29`.
2. **A constraint failure aborts only the statement, and the sqlite3 CLI continues.** The
   first guard relied on a CHECK violation, which would have let the CLI carry on and apply
   the schema anyway. The guard now uses a trigger with `RAISE(ROLLBACK)`, and the
   documented invocation requires `-bail`. This became requirement S6.
3. **A rollback in the middle of the script is not enough on its own.** `RAISE(ROLLBACK)`
   rolls the transaction back, but the CLI keeps going, and the statements that follow then
   run outside any transaction, where each one commits on its own. Verified: without
   `-bail`, the first version of the guard still created all five tables on a non-empty
   database. The emptiness check was therefore moved to the **end** of the transaction, so a
   refusal undoes every change, including the schema version, whether the client aborts on
   the first error or carries on past it.
4. **The schema version must be stamped before the final check, not after.** With the
   version stamped last, a client that continued past the refusal stamped
   `user_version = 1` on a database that had no schema. Verified and fixed; a refused
   migration now leaves `user_version` at `0`.
5. **Foreign key enforcement is per connection.** `PRAGMA foreign_keys = ON` inside the
   migration does not persist to later connections. Every application connection must set
   it. Four apparent FK failures during testing were the test harness, not the schema.
6. **Table order matters for readability, not for SQLite.** `movements` references `loans`,
   so `loans` is created first. SQLite tolerates a forward reference, but the migration
   reads better without one.
7. **Only named constraints can be revised in place.** `ALTER TABLE ... DROP CONSTRAINT`
   (SQLite 3.53.0+) rejects an unnamed inline CHECK with "no such constraint", so the
   `kind` and shape CHECK constraints are now **named** (`accounts_kind`, `accounts_shape`,
   `contacts_kind`, `contacts_shape`, `loans_direction`, `loans_kind`, `movements_kind`).
   Without naming them, adding a new account kind would force a full table rebuild on
   every supported SQLite version, which would defeat the reason for the 3.53.0 floor.
   Verified end to end: a `savings` kind was added with `ALTER TABLE` alone, existing rows
   were preserved, and invalid rows were still rejected.
8. **The `WITHOUT ROWID` and rowid facts hold.** `INTEGER PRIMARY KEY` is a rowid alias and
   `INT PRIMARY KEY` is not, so every surrogate key is declared `INTEGER PRIMARY KEY`.
9. **Query plans confirmed.** With 200,000 movements, the per-account balance query, the
   monthly income/expense report, the account history query, the Nequi phone lookup and the
   loan schedule query all use their intended index with no table scan. Timings: balance
   0.14 s, monthly report 0.02 s, point lookups instant.
10. **`loans.status` is a maintained field, not a derived one.** The outstanding balance is
    derived from `principal` and the repayment schedule; `status` is kept in step by the
    application and reconciled against that derived value.

### L7 — Evidence gathered

- SQLite has no table inheritance; all inheritance patterns are conventions enforced by
  triggers or CHECK constraints.
- Model A polymorphic read scales linearly with subtype count (5.79 ms at one subtype up to
  30.55 ms at sixteen subtypes, on 20k rows). Model B stays flat (3.92 ms up to 15.59 ms).
- Model A kind filter 0.054 ms versus Model B 0.005 ms.
- Model A write 0.157 ms versus Model B 0.151 ms — indistinguishable.
- Model A cannot enforce "parent has exactly one subtype row" — verified that such a row
  inserts cleanly.
- Model A's kind trigger fires before the foreign-key check, masking foreign-key errors with
  a kind-mismatch message.
- Model A's parent rebuild breaks every child trigger, aborting the migration with
  `no such table: main.accounts`.
- SQLite 3.53.0 (2026-04-09) added `ALTER TABLE ADD/DROP CONSTRAINT`, which lets Model B add
  a subtype without a table rebuild. The local CLI is 3.51.0 (lacks it) and the local Python
  `sqlite3` links 3.54.1 (has it).
- `PRAGMA foreign_keys` is off by default; an orphan row was verified to insert successfully
  without it.
- STRICT tables, `INTEGER PRIMARY KEY` as a rowid alias, and the `WITHOUT ROWID` caveat were
  confirmed against sqlite.org.
- The sqlite3 CLI continues past an error by default, and a rollback in the middle of a
  script is not enough to protect against it: the statements that follow run outside the
  transaction. The emptiness check therefore runs last, and the schema version is stamped
  before it, so a refusal leaves the database untouched under both client behaviors.
- Only **named** CHECK constraints can be dropped and replaced with `ALTER TABLE` on SQLite
  3.53.0+. The `kind` and shape constraints are named for that reason.
