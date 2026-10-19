# Deferred: open decisions for a future session

Carried forward from the database model feature. These are not blockers for the schema
that was delivered; they are decisions that must be made before the application code
starts, because both of them change how the migration runner behaves.

Status: **deferred by user request**. The user asked to settle them in a later session.

## D1 — Tech stack and SQLite driver

**The question.** Which language and SQLite driver will the application use?

**Why it matters.** The schema's evolution path depends on the SQLite version the driver
bundles:

| Driver version | How to add a new account kind |
| --- | --- |
| >= 3.53.0 | `ALTER TABLE ... ADD COLUMN` + `DROP/ADD CONSTRAINT`. No table rebuild. |
| < 3.53.0 | Full 12-step table rebuild: create a new table, copy the rows, drop the old one, rename. |

The 3.53.0 path is the reason the `kind` and shape CHECK constraints in
`packages/db/migrations/001_initial_schema.sql` are **named**. An unnamed inline CHECK
cannot be dropped or replaced, so an unnamed constraint would force a rebuild on every
supported SQLite version.

**What to check when this is decided.** The driver's bundled SQLite version, and whether
it can report it. Go (`modernc.org/sqlite`, `mattn/go-sqlite3`) and Node
(`better-sqlite3`, `node:sqlite`) each bundle their own SQLite, so the module version
matters, not just the language. Python's stdlib `sqlite3` linked 3.54.1 on this machine.

**Local facts at the time of writing.** The system `sqlite3` CLI is **3.51.0** and does
*not* support `ALTER TABLE ADD/DROP CONSTRAINT`. The Python `sqlite3` module links
**3.54.1** and does. Creating the schema only needs 3.37.0; the 3.53.0 floor matters
solely for evolving it.

**Default if undecided.** Pin 3.53.0+ and assert the version at runner startup, rather
than depending on the user's ambient SQLite.

## D2 — Single-user or multi-user

**The question.** Will PACA support more than one user?

**Why it matters.** No `user_id` column exists on any of the five tables. The schema
assumes a single user: a Nequi phone number is unique across the whole database, a NIT is
unique across the whole database, and an account name is not scoped to anybody.

**What changes if it becomes multi-user.** Every table gains a `user_id` foreign key, and
every uniqueness rule becomes scoped by it:

| Current rule | Multi-user form |
| --- | --- |
| `accounts.phone` unique where `kind='nequi'` | unique where `kind='nequi'` and `user_id = ?` |
| `accounts.(bank_code, account_number)` unique where `kind='bank'` | scoped by `user_id` |
| `contacts.tax_id` unique where `kind='entity'` | scoped by `user_id` |
| `movements.account_id` -> `accounts.id` | unchanged, but `accounts` is now per user |

**The cost of changing later.** Adding `user_id` to five populated tables and rebuilding
the uniqueness indexes is materially more work than adding it now, before any data
exists. This is the one decision in this list that gets *more* expensive with time, which
is why it is worth settling early rather than late.

**Default if undecided.** Keep the single-user assumption. The current schema is correct
for one user, and the migration runner can add `user_id` later with a migration that
backfills a single owner.

## Neither decision blocks the delivered work

The schema, the migration, the verification harness and the documentation are complete and
verified. Both decisions are recorded in the README's "Open Questions" section, which is
where the data team will look for them.

## Log

### L1 — Origin (verbatim)

> The two opened question and for others session. Push and open a new pr against main.

### L2 — Decision record

The user deferred D1 and D2 to a future session and asked for the branch to be pushed with
a pull request against `main`. The two questions are recorded here so that a later session
picks them up without needing to re-derive them from the conversation.
