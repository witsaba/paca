# PACA — the onboarding that opens the personal business of the person

Feature document for the second slice of PACA: the person arrives at Paca, Paca
onboards them, and the person leaves with a **Personal Business** and a book
that tracks every income and outcome. The work is divided in three slices, one
per session, and this document holds the whole: the specs, the tasks of the
three slices and the log of the decisions.

- **Project:** `packages/domain`, `packages/database`, `apps/mcp-server`, `hermes`.
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-worktrees/onboarding-personal-business`, branch `feat/onboarding-personal-business`.
- **Scope:** the onboarding of the person, the personal business, its accounts and its book. No second business, no multi-currency, no budgets, no recurring entries, no attachments, no investments, no taxes.

---

## Specs

Each spec is the user's own wording, quoted verbatim, except the ones marked
*(decided)*, which record a choice the user took during the discussion.

### S1 — The onboarding comes first

> let start with a new feature. Paca is for now, for OPB (one person bussiness) I wanna build a proper onboarding. When the user talk to Paca, must to first give the onboarding way, for complete the user information.

### S2 — The person without a business

> What about the users that doens't have a business yet? Very good questions. I decided to bring to the user the `Personal Business` which allow to the user the manage the own personal finantials. So is needed to explore and designs a very simple database model table for allowing to the user handle the own finantials this is the `default` business.

### S3 — Only the personal business, for now

> For now dont work in the way of creating new business, only the ownboarding and the default personal business.

### S4 — A book, not a log of expenses

> This must to allow to track any income/outcome and have a contable book personal.

### S5 — The accounts table

> Help me with a doubt a got, shall I add too a `accounts` table for personal contability? Explore it and keep it the most simple as posible.

### S6 — The debts

> Is important to add loans, example, a personal loan with a friend, a product that was recieved for paying later.

### S7 — Two MCP servers

> I see here something that I was talking in the past. Maybe is now the moment to have a ledger-mcp-server and user-mcp-server. Review this based in a proper Hexagonal Architecture, Domain Driven Design and SOLID principles in order to have a proper separation.

### S8 — Three sessions

> given this, I consider that is too much for just one session, so, lets divide this in three session in order to achieve the entire onboarding. Explore again and tell me how do you consider this three slices onboarding division.

### S9 — What the onboarding requires

> 1. The user name, the user id required. I agreed. Paca must to persuade the user in order to get the email, phone.

### S10 — The identity of the book, and the collision that is coming

> 2. yes to the ledger mode, the personal account must to be the user id, but think how to avoid colissions for the future, example I wanna open a bakery, but is a personal business associated to my own ID too. I know for now not a problem but annotate this.

### S11 — The default accounts

> 3. Yes, create at the end of the onboarding and a common income/expense categories

### S12 — The tools name the business

> 4. Do the ledger tools take and explicit business_id.

### S13 — The book starts at zero

> For 2, start with zero, is a lie offcourse, but is the user whom have to talk with paca and start to feed the information.

### S14 — The worktree, and no duplicated code

> remember to working in a git worktree apart. I dont know if posible to avoid the duplicated code in a central package and reuse as much as posible.

### S15 — The two contexts, one process *(decided)*

The contexts are separated in the code and not in the processes, for now:

| Context | Its models and rules | Its use cases and tools |
|---|---|---|
| Identity | `packages/domain/user` | `apps/mcp-server/internal/application/users.go`, `adapters/inbound/mcpserver/users.go` |
| Ledger | `packages/domain/business`, `packages/domain/account`, `packages/domain/ledger` | `apps/mcp-server/internal/application/ledger.go`, `adapters/inbound/mcpserver/ledger.go` |

The orchestration of the onboarding is a process manager over the two contexts and
lives in the application layer (`internal/application/onboarding.go`). Neither
context calls the rules of the other.

**Why:** the schema has one writer (`apps/db-admin`), the domain library is one
module, and the persistence is one connection stack (`packages/database`). A
second process would not split the domain nor the schema; it would buy a second
container over the same database, and it would duplicate the MCP assembly, the
health handler, the configuration and the composition root.

**The trigger for the split**, when it comes: the ledger gets a lifecycle of its
own (statement imports, recurring jobs, an independent release) or a risk
isolation of its own. The extraction is mechanical, because the seam is already
there — the ports of each context, the models in the domain and the queries in
the library.

### S16 — The central packages *(decided)*

The duplicated code is avoided by the central packages that already exist, each
with its own rule:

| Concern | Central package | Rule |
|---|---|---|
| The models, the rules and the ports | `packages/domain` | imports nothing but the standard library; one owner per concept |
| The queries, the migrations and the repositories | `packages/database` | implements the ports of the domain; one connection stack; one writer of the schema |
| The infrastructure every service shares | `packages/platform` | imports nothing of the monorepository but itself |

The duplication that exists today is the health handler of the two services, and
it is paid in the first task of the second slice (T8): a shared HTTP adapter
module, with the same relation to the domain that `packages/database` has.

### S17 — The model of the book *(decided)*

The book belongs to a **business**, and the business has an identity of its own.
The document ID of the person is the **owner** of the business, never its key:

```sql
CREATE TABLE businesses (
    id        BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id  BIGINT NOT NULL REFERENCES users (id),
    kind      TEXT   NOT NULL,
    name      TEXT   NOT NULL,
    currency  TEXT   NOT NULL DEFAULT 'COP',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT businesses_kind_check CHECK (kind IN ('personal', 'business'))
);

CREATE UNIQUE INDEX businesses_one_personal_per_owner
    ON businesses (owner_id) WHERE kind = 'personal';
```

The personal business of a person is found by `(owner_id, kind = 'personal')`,
and the partial unique index makes "exactly one" a rule of the table. The
bakery of the future is a second row of the same table, with its own identity
and its own accounts: **no collision is possible, because the key space is the
businesses, never the people.**

The account and the movement follow:

```sql
CREATE TABLE accounts (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES businesses (id),
    kind        TEXT   NOT NULL,
    name        TEXT   NOT NULL,
    counterparty TEXT  NULL,
    due_date    DATE   NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT accounts_kind_check CHECK (kind IN ('asset', 'liability', 'income', 'expense', 'equity')),
    CONSTRAINT accounts_name_unique UNIQUE (business_id, name)
);
```

The kinds of an account are the five of the accounting, and the name of an
account is the name the person reads in their book. **The names are in Spanish,
because the person reads them; the identifiers are in English, because the code
is read by the team.**

### S18 — The debt is an account, never a table *(decided)*

A loan with a friend is a liability account with a counterparty and a due date.
There is no `loans` table, and there never will be one:

| The life of the money | The movement |
|---|---|
| "A friend lends me 500" | debit `cash`, credit `liability:préstamo:<amigo>` |
| "I pay the friend 100" | debit `liability:préstamo:<amigo>`, credit `cash` |
| "A product I will pay later" | debit `expense:producto`, credit `liability:por pagar:<proveedor>` |
| "The interest the bank charges" | debit `expense:intereses`, credit `liability:préstamo` |

The balance of a liability account is what the person still owes. Every
payment, every due date and every interest is an ordinary movement of the book.

### S19 — The movement is two accounts and one amount *(decided)*

Every movement of the book is money that moves from a debit account to a credit
account, and the amount is positive. The sum of the two sides is equal by the
shape of the row itself: the invariant of the double entry is held by the model,
and not by a trigger of the database.

The balance of an account is `débitos − créditos`, with the sign flipped for
`liability`, `equity` and `income`. The result reads "what it means to the
person" in every kind: `cash` in $500 means *I have it*, `liability` in $500
means *I owe it*, `income` in 500 means *I earned it*, `expense` in 500 means
*I spent it*.

### S20 — The book starts at zero *(decided)*

The onboarding does not ask for the opening balance: the book starts at zero,
and the person feeds it. The first entries are the ones the person tells to
Paca, and the balance of the book is the truth of what has been fed into it.

### S21 — The names of the default accounts *(decided)*

The default accounts of the personal business are created at the end of the
onboarding, and they are owned by the domain, so the tool contract derives from
them and never restates them:

| Kind | The names |
|---|---|
| `asset` | `Efectivo`, `Banco` |
| `income` | `Salario`, `Otros ingresos` |
| `expense` | `Vivienda`, `Alimentación`, `Transporte`, `Servicios`, `Salud`, `Educación`, `Ocio`, `Otros gastos` |

A debt is not seeded: it is created when it happens.

### S22 — The rules of the complete onboarding *(decided)*

The document ID and the name are **required**. The phone and the email stay
optional, and Paca keeps persuading the person for them, kindly, and accepts
that they refuse.

### S23 — One call, one transaction *(decided)*

The onboarding is complete in **one** use case, in **one** transaction: the data
is validated, the personal business is created, its accounts are created, and
the person moves to `active`. The agent makes one call, and it never leaves a
half-onboarded person behind.

---

## Tasks

One line per task: the identifier, the specs it answers, the route and the
commit that holds it. The routes are: **inline** (the orchestrator writes it)
and **worker** (a bounded writer reads this document and writes the unit).

### Slice 1 — Paca welcomes the person and opens their personal business

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T1 | S2, S10, S11, S17 | The domain of the business and the account: the models, the kinds, the rules, the ports of the storage, and the default accounts of the personal business | inline | `d7a2cfd` — `feat(domain): the business and the account of the personal book` |
| T2 | S9, S13, S22 | The rule of the complete onboarding in the domain of the person: the name is required, the phone and the email are optional, and the completeness of an onboarding is a value that cannot be built broken | inline | `773174d` — `feat(domain): the rule of the complete onboarding` |
| T3 | S10, S17 | The tables of the business and the account: the migration, the queries and the repositories that implement the ports of the domain | inline | `35feb29` — `feat(database): the tables of the business and the account` |
| T4 | S1, S9, S12, S21, S23 | The use case that completes the onboarding and the tool the agent calls: one call, one transaction, the business and its accounts created | inline | `52207f3` — `feat(mcp-server): the tool that completes the onboarding` |
| T5 | S1, S13, S20 | The agent of PACA reaches the MCP server of PACA: the configuration of the agent names it, and the way of the onboarding is in the soul of Paca | inline | `9aba60e` — `feat(agent): the agent of PACA reaches the MCP server of PACA` |
| T6 | S1, S2, S9, S13 | The documentation of the new packages, the new ports and the new tools, and the rules of the structure that pin them | inline | this commit — `docs: the onboarding, the business and the account` |

### Slice 2 — The book moves money

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T7 | S4, S5, S19 | The domain of the movement: the entry, its rules, and the balance of an account | worker | `feat(domain): the movement and the balance of the book` |
| T8 | S16 | The shared HTTP adapter of the services: the health handler of the two services stops being a copy, and the debt the structure test records is paid | worker | `refactor: the shared HTTP adapter of the services` |
| T9 | S4, S19 | The table of the movements, the queries and the repository | worker | `feat(database): the table of the movements` |
| T10 | S4, S12 | The use case and the tool that record a movement, with the descriptions derived from the domain | worker | `feat(mcp-server): the tool that records a movement` |
| T11 | S4 | The tools that read the book: the libro diario (the movements) and the libro mayor (the balance of every account) | worker | `feat(mcp-server): the tools that read the book` |

### Slice 3 — The debts and the closing of the month

| Task | Specs | What it delivers | Route | Commit |
|---|---|---|---|---|
| T12 | S6, S18 | The domain of the debt: a liability account with a counterparty and a due date | worker | `feat(domain): the debt of the personal book` |
| T13 | S6, S18 | The tools of the debts: what the person owes, to whom, and when it falls due, and the payment of a debt | worker | `feat(mcp-server): the tools of the debts` |
| T14 | S4, S6 | The closing of the month: the income, the expenses, the net and the debts that are still open | worker | `feat(mcp-server): the closing of the month` |
| T15 | S1, S4, S6 | The proof of the whole, end to end, on the channel of Paca | inline | `test: the proof of the onboarding and of the book` |

---

## Log

**L1** (2026-11-04): "let start with a new feature. Paca is for now, for OPB (one person bussiness) I wanna build a proper onboarding. When the user talk to Paca, must to first give the onboarding way, for complete the user information. What about the users that doens't have a business yet? Very good questions. I decided to bring to the user the `Personal Business` which allow to the user the manage the own personal finantials. So is needed to explore and designs a very simple database model table for allowing to the user handle the own finantials this is the `default` business. For now dont work in the way of creating new business, only the ownboarding and the default personal business. This must to allow to track any income/outcome and have a contable book personal. Help me with a doubt a got, shall I add too a `accounts` table for personal contability? Explore it and keep it the most simple as posible. With this, the Paca will be a functional agent allowing to the user to onboard to Paca, and immediately handle the personal user contability. Is important to add loans, example, a personal loan with a friend, a product that was recieved for paying later. Let start only exploring. You can too goes to the web and research for a better context."

**L2** (2026-11-04): The exploration answered the doubt of the `accounts` table: yes, and it replaces the categories table and the loans table. Three tables, and the loan is a liability account. The exploration of the repository found three gaps: nothing states what a complete onboarding is, there is no business at all, and the agent does not yet reach the MCP server of PACA.

**L3** (2026-11-04): The maintainer answered the four questions: the name and the document ID are required, the phone and the email are optional and Paca persuades for them; the double entry is accepted, and the identity of the book must not collide with the future bakery; the default categories are created at the end of the onboarding; the ledger tools take an explicit `business_id`.

**L4** (2026-11-04): The maintainer proposed two MCP servers and asked for the review against hexagonal architecture, DDD and SOLID, and for the division of the work in three sessions. The review answered: the contexts are separated in the code, and the processes are separated later, with the trigger stated (S15).

**L5** (2026-11-04): The maintainer agreed with the recommendations, kept the opening balance at zero ("start with zero, is a lie offcourse, but is the user whom have to talk with paca and start to feed the information"), asked for the work in its own worktree, and asked whether the duplicated code could live in a central package. The worktree is created; the answer about the central packages is S16.

**L6** (2026-11-04): The decisions of the maintainer, carried into the specs: the names of the default accounts in Spanish and the identifiers in English (S21), the document ID and the name required (S22), one call and one transaction (S23), the opening balance at zero (S20).

**L7** (2026-11-04): The unknown of T5 is settled against the **pinned** image (`nousresearch/hermes-agent:stable@sha256:9c5561…`): the runtime supports an HTTP MCP server, with the key `url` in the entry of `mcp_servers` (`hermes mcp add <name> --url "http://…/mcp"`, and the transport is detected as HTTP). The agent container is already on the `internal` network of the monorepository — the compose file of the stack says so — and the MCP server of PACA publishes `8081` on that network without publishing a port to the host. So T5 is a configuration entry and nothing else. The tools of the server are registered by the runtime with the prefix `mcp_paca_`, which is the name Paca calls them by.

**L8** (2026-11-04): The slice 1 is delivered in six work units on the branch `feat/onboarding-personal-business`, in the worktree `/Users/braejan/workspace/witsaba/repositories/paca-worktrees/onboarding-personal-business`: the domain of the book (`d7a2cfd`), the rule of the complete onboarding (`773174d`), the tables and the adapters (`35feb29`), the tool that completes the onboarding (`52207f3`), the wiring of the agent (`9aba60e`), and the documentation (this commit). The gate of the three modules is green, their container tests are green, and the rules of the structure of the repository pass with 425 assertions.

**L9** (2026-11-04): Two decisions the work took on its own, recorded here so the reviewer reads them: the **unit of work** (`store.UnitOfWork`) is the port of a transaction of the storage, and the transaction travels in the context, so the repositories of the library join the transaction of the work that calls them; and the **tool is safe to call again**, because the person that is already registered is completed and not registered again, the business that is already open is kept, and the accounts that are already there are read and not written again.

**L10** (2026-11-04): The `full_name` of the people stays nullable in the table: the rows that exist may carry no name, and the backfill that would close that gap is not decided yet. The domain refuses the registration without a name, and the table keeps tolerating what it holds.
