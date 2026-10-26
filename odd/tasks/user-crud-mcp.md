# PACA — the CRUD of the user and the three tools of the MCP server

Feature document for the first slice that writes data: the core of the
monorepository grows the repository of the users, and the MCP server grows the
three tools that an agent uses to onboard a person, to move them to another
status and to correct their data.

- **Project:** `packages/core` and `apps/mcp-server`.
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-users`, branch `feat/user-crud`.
- **Scope:** the repository of the users in the core, the new database guards on the table `users`, and the three MCP tools. No delete, no list, no read tool, no REST endpoint, no worker, no authentication.

---

## Specs

Each spec is the user's own wording, quoted verbatim, except the ones marked
*(decided)*, which record a choice the user took during the discussion.

### S1 — The core must allow CRUD, the MCP server must grow the tools

> let create the feature for add de functionality for users. The core most to allow to CRUD operations and the mcp server must to have the next functionalities in the mcp:

### S2 — The onboarding tool

> Allow to create a new user with default onboarding status, the MCP tool must to indicate to the agents that the user ID MUST to be bring it for the user, the name is optional but the agent must to be kind with the user trying that the user give it a name,  the tool must to be oriented to the onboarding, not some a create_new_user or something like that.

### S3 — The phone and the email

> Same for the phone and email, put in the instructions that the phone must to include the country indicatif. In the MCP server must to clean the phone number example son +57 321-253-9984 must to be cleaned and save it in 573212539984. The same for the email, the MCP tool must to validate the proper email if bring it by the user but put in the tool description that the agent must to try to get all the information. the MCP ensure the created_at and updated_at dates.

### S4 — The tool of the status

> A tool for promoting the status to active, use a map for a state machine that allows from onboarding to active, from inactive to active or blocked, from onboarding to blocked too, from active to inactive etc.

### S5 — The tool of the data

> a tool for updating data like name, phone and email the field that doesn't be updated is empty.

### S6 — The direct writes of the core

> for the core, try directly to insert in order to do not query and then insert. Direct insert, if the record exists and try to insert this give a primary key error. Similar for update, the update must to directly try to update if zero records update.

### S7 — The unique validations, in the database

> Finally, take this feature too to add unique validations for the email, and the phone when is not null. This directly in the database.

### S8 — A new worktree, no assumptions

> WOrk in a new git  worktree. Do not assume anything, instead asking to me with your recommendations and why instead assuming anything.

### S9 — The names of the three tools *(decided)*

The user named them: `onboard_user`, `change_user_status` and
`update_user_contact_data`.

### S10 — The state machine, complete *(decided)*

The map of the transitions is in the core, and it is the one the user picked:

```
onboarding ──▶ active      active     ──▶ inactive   inactive   ──▶ active
onboarding ──▶ inactive    active     ──▶ blocked    inactive   ──▶ blocked
onboarding ──▶ blocked     blocked    ──▶ active     blocked    ──▶ inactive
```

No transition returns to `onboarding`: the state of a person that has just
been created is the entry of the machine, never a destination.

### S11 — The observation is mandatory for the two closed states *(decided)*

> good point, I almost forget it, the observations must to be mandatory when inactive or blocked. and empty when onboarding or active.

The rule is in the core **and** in the database:

- `inactive` and `blocked` carry a non-empty observation.
- `onboarding` and `active` carry none.

The database holds it as a CHECK on the table, so no writer breaks it. The
MCP instructions hold it too, in words, so the agent knows it before the call
fails:

> BTW the observation status check is in the database but the MCP instructions must to be clear with that too.

### S12 — The phone is cleaned by the core, and it carries the country code *(decided)*

The cleaning happens in the core — `packages/core/domain/user` — so the MCP
server, the future REST service and the worker all store the same number:

| input | stored |
| --- | --- |
| `+57 321-253-9984` | `573212539984` |
| `+57 (321) 253-9984` | `573212539984` |
| `573212539984` | `573212539984` |
| `3212539984` | refused: it carries no country code |
| `+57 321-253-99` | refused: too short |

The stored number holds digits only, between 11 and 15 of them. Eleven is the
shortest international number — a country code and a national number of ten —
so a national number written without its country code is refused.

### S13 — The email is validated *(decided)*

The address is trimmed and must look like an address. The database holds the
same rule as a CHECK.

### S14 — The database guards *(decided)*

```sql
CREATE UNIQUE INDEX users_email_key ON users (lower(email)) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX users_phone_key ON users (phone)        WHERE phone IS NOT NULL;

ALTER TABLE users ADD CONSTRAINT users_phone_format_check
    CHECK (phone IS NULL OR phone ~ '^[0-9]{11,15}$');
ALTER TABLE users ADD CONSTRAINT users_email_format_check
    CHECK (email IS NULL OR email ~ '^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$');
ALTER TABLE users ADD CONSTRAINT users_optional_not_empty_check
    CHECK ((full_name IS NULL OR full_name <> '')
       AND (phone    IS NULL OR phone    <> '')
       AND (email    IS NULL OR email    <> ''));
ALTER TABLE users ADD CONSTRAINT users_status_observation_check
    CHECK ((status IN ('inactive', 'blocked')
            AND status_observations IS NOT NULL AND status_observations <> '')
        OR (status IN ('onboarding', 'active')
            AND status_observations IS NULL));
```

The email is compared without case, because `Foo@x.com` and `foo@x.com` are
the same mailbox.

### S15 — The operations of the core *(decided)*

```
Create(ctx, id, details) (user, error)
GetByID(ctx, id) (user, error)
UpdateDetails(ctx, id, details) (user, error)
ChangeStatus(ctx, id, status, observation) (user, error)
```

No delete: a person that must go is `blocked`. No list: nothing consumes it
yet.

### S16 — The three tools, and only three *(decided)*

The MCP server exposes `onboard_user`, `change_user_status` and
`update_user_contact_data`. It exposes no read tool: the agent works with
what the two write tools answer.

### S17 — The empty field is a field that does not change *(decided)*

> I wanna say that the MCP recieve for example the tool call for update the name and the email, and the field for the phone is emtpy so in database must to only update these two, if recieve all , all trhee must to be updated. do you got it?

An empty field is left as it is. A field that is not given cannot be cleared:
the tools set a value or change it, and nothing more.

### S18 — The timestamps belong to the database *(decided)*

`created_at` and `updated_at` are filled by the database. The UPDATE sets
`updated_at = now()` inside the statement, so the clock of the database is the
only clock that writes a row.

### S19 — The status change reads the current row *(decided)*

The map of the transitions lives in the core, so the core has to know the
current status to validate the move. The core reads the row, validates the
move, and then issues **one** UPDATE that also requires the status to be
unchanged:

```sql
UPDATE users
   SET status = $2, status_observations = $3, updated_at = now()
 WHERE id = $1 AND status = $4
```

Zero rows updated means the row is gone or it changed at the same time.

### S20 — The document ID *(decided)*

The ID is the Colombian national document ID, the caller supplies it and
nothing generates it. The tool receives it as a string and the core keeps its
digits, so `1.234.567` and `1234567` are the same person.

---

## Tasks

| ID | Task | Specs | Route | Commit |
| --- | --- | --- | --- | --- |
| T1 | The domain of the user: the repository port, the map of the transitions, the cleaning of the phone and the email, the rules of the observation | S10, S11, S12, S13, S15, S17, S20 | inline | `6e9763a`, `37e73ad` |
| T2 | The schema: the migration with the unique indexes and the CHECKs, and the generated code | S14 | inline | `0ac9c5f` |
| T3 | The adapter of the core: the repository over PostgreSQL, with the direct insert and the direct update | S6, S15, S18, S19 | inline | `d16e015`, `e0eddee` |
| T4 | The MCP server: the use cases, the three tools, their instructions and the wiring | S2, S3, S4, S5, S9, S11, S16, S17 | inline | `3d04e34`, `36eaf9b`, `b28e335` |
| T5 | The documentation of both modules and the evidence of the slice | S1, S8 | inline | `ca12e66`, `f475d4b`, `caee459`, `ec45b6e`, `79fc926`, this close |

Work units: W1 the domain, W2 the schema and the adapter, W3 the tools, W4 the
documentation and the close.

**Delivery strategy:** the branch carries the slices, one commit each, and the
slice boundary is recorded here. Every slice stays under the ~400 authored
lines, so the review reads one slice at a time. The pull request is opened as
a chain, one pull request per slice, and the last one closes the feature.

---

## Log

### L1 — The original request

> let create the feature for add de functionality for users. The core most to allow to CRUD operations and the mcp server must to have the next functionalities in the mcp:
> - Allow to create a new user with default onboarding status, the MCP tool must to indicate to the agents that the user ID MUST to be bring it for the user, the name is optional but the agent must to be kind with the user trying that the user give it a name,  the tool must to be oriented to the onboarding, not some a create_new_user or something like that. Same for the phone and email, put in the instructions that the phone must to include the country indicatif. In the MCP server must to clean the phone number example son +57 321-253-9984 must to be cleaned and save it in 573212539984. The same for the email, the MCP tool must to validate the proper email if bring it by the user but put in the tool description that the agent must to try to get all the information. the MCP ensure the created_at and updated_at dates.
> - A tool for promoting the status to active, use a map for a state machine that allows from onboarding to active, from inactive to active or blocked, from onboarding to blocked too, from active to inactive etc.
> - a tool for updating data like name, phone and email the field that doesn't be updated is empty.
>
> for the core, try directly to insert in order to do not query and then insert. Direct insert, if the record exists and try to insert this give a primary key error. Similar for update, the update must to directly try to update if zero records update.
>
> Finally, take this feature too to add unique validations for the email, and the phone when is not null. This directly in the database.
>
> WOrk in a new git  worktree. Do not assume anything, instead asking to me with your recommendations and why instead assuming anything.

### L2 — The names of the tools

The user named them: `onboard_user`, `change_user_status` and
`update_user_contact_data`.

### L3 — The observation

> good point, I almost forget it, the observations must to be mandatory when inactive or blocked. and empty when onboarding or active.

And later:

> BTW the observation status check is in the database but the MCP instructions must to be clear with that too.

### L4 — The empty field

> I wanna say that the MCP recieve for example the tool call for update the name and the email, and the field for the phone is emtpy so in database must to only update these two, if recieve all , all trhee must to be updated. do you got it?

### L5 — The decisions of the discussion

| Question | The user's answer |
| --- | --- |
| The names of the tools | `onboard_user`, `change_user_status`, `update_user_contact_data` |
| The transitions | The full map, and never back to `onboarding` |
| The phone | Cleaned in the core: digits only, and the country code is required |
| The database guards | The unique indexes **and** the format CHECKs |
| The operations of the core | Create, GetByID, UpdateDetails, ChangeStatus |
| The read tool of the MCP | No: only the three tools |
| The timestamps | The database, inside the UPDATE statement |
| The empty field | Not updated; and a value can never be cleared |
| The transitions, where they live | In the core only; the database keeps the CHECK on the four values |
| The observation, where it lives | Also in the database, with a CHECK |
| The status change, how it reads the current row | Read the row, validate the move, one direct UPDATE |

### L6 — The decisions of the implementation

- **The phone needs eleven digits, not ten.** The bound of the specification
  was ten to fifteen, and ten is exactly the length of a Colombian number
  without its country code: the number the rule had to refuse. The bound is
  eleven to fifteen, and the database holds the same bound.
- **The email is compared without case.** The unique index is on
  `lower(email)`: `Foo@x.com` and `foo@x.com` are the same mailbox.
- **The observation is written as NULL, never as an empty string.** The
  domain writes NULL for a value that is not set, and the CHECK of the table
  asks for exactly that.
- **The core refuses an observation that does not fit the status before it
  touches the database.** The rules of the domain do not depend on the
  storage, and a caller that breaks them hears about it first.
- **The tools are registered even without a database.** The list of the
  tools is a property of the transport, and a call without a database
  answers that the database is not there.
- **The coverage gate of the module lost the file of the repository.** Its
  queries run against PostgreSQL, so the container tests are what measure
  it; the unit gate measures the rest. The rule of the module is unchanged:
  the gate of a database-bound module is split in two.
- **The document ID keeps its digits.** `1.234.567` and `1234567` are the
  same document, and the core refuses a document that carries anything else.

### L7 — What the delegated worker found, and what was decided

The container suite of the repository was written by a delegated worker, and
it reported three findings. They were decided like this:

1. **The coverage of the container tests never reached the adapter.** The
   target `test-integration` measured the coverage of the packages of the
   test run, and the adapter was never attributed. It measures with
   `-coverpkg` now, so the two numbers — the unit gate and the container
   gate — are the coverage of the module.
2. **The order of the two rules of the change of the status decides which
   error the caller sees.** The observation is checked against the target
   status before the map of the states is consulted, so a move to a status
   that takes no reason is refused with the reason and not with the map.
   The rule was left as it is: the schema of the tool lists the three
   statuses the user may move to, so the overlap never reaches an agent.
3. **The adapter wrote the data as it received it.** It cleans it now, with
   the rules of the domain, so the port holds whatever the caller sends and
   the error of a bad number is the error of the rule.

Two things are known and open:

- The branch of the adapter that reports a user that moved at the same time
  (`ErrConcurrentChange`) has no container test: it needs a writer that
  changes the row between the read and the write, and forcing that
  deterministically would need a hook the adapter does not have.
- The harness of the container suite waited for the database only once, and
  the container opens the port before the server accepts the connections:
  the suite failed in about eight percent of its runs before the wait was
  added. It is green in twenty-five consecutive runs after it.

### L8 — The evidence of the slice

| Check | Result |
| --- | --- |
| `make gate` (every application and every package) | Green: the linter with no issue, the analyzers, the build, the tests, the race detector, the coverage at 96.8% against the 80% floor, and no vulnerability. |
| `make packages-integration` (the container tests) | Green: 14 tests and 30 sub-tests against a real PostgreSQL, and the coverage of the adapter and of the domain at 67.7%. |
| The migration against the running database of the stack | Green: the version 2 was applied to the database that already carried the version 1, the indexes and the CHECK constraints are in place, a valid row is accepted, and a phone with its punctuation is refused. |
| The tools, through the protocol | Green: the three tools are listed with their descriptions and their schema, the calls register a person, move a user and write the data of a user, and every failure is answered in words the agent can act on. |

**Risk:** none. The migration is additive and reversible, the change of the
contract of the table was the request itself, and the container tests prove
the new rules against a real database.

