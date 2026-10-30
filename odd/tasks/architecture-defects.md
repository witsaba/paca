# The four live defects of the architecture audit

The tail of the audit of the architecture (the pull request #35) left four defects
open in the code of PACA. This slice closes them: the identity of the email
address, the capability that two services model in two ways, the dead member of
the port of the users, and the copy of the configuration of the telemetry that
the platform owns.

## Specs

- **S1**: The request of the maintainer, verbatim: "open a new git work tree and
  add the four live defects. Once you finish, review again in order to check
  that there is not more defects to be fixed".

- **S2**: The first defect, verbatim, from the inventory that was read to the
  maintainer: "**Email uniqueness lives only in the table.**
  `00002_users_guards.sql:25` holds `UNIQUE INDEX ... (lower(email))`; the
  domain's `CleanEmail` (`contact.go:90`) trims and validates but never
  normalizes case. `Foo@x.com` and `foo@x.com` are **two distinct domain
  values** for one person. Needs a data migration → its own slice."

- **S3**: The rule of the contract that the defect broke, verbatim: "A rule that
  the **table** also holds (a CHECK, a unique index) is deliberate defense in
  depth: the table guards against the writer that does not know the domain. The
  table never becomes the only holder of a rule."

- **S4**: The rule of the contract that owns the fix, verbatim: "**One owner per
  model, one owner per rule**" and "A concept that more than one module needs
  lives in **one** shared module, and the other modules import it. Two copies of
  the same model are two domains that drift."

- **S5**: The second defect, verbatim, from the same inventory: "**One
  capability, two shapes.** `*postgres.Pool`
  (`mcp-server/.../checker.go:17`) vs `ports.Pinger`
  (`db-admin/.../ports.go:47`)."

- **S6**: The third defect, verbatim, from the same inventory: "**
  `user.Repository.GetByID` is a dead port member.** Only the adapter calls
  itself (`repository.go:177`)." And the rule of the contract that names the
  fix, verbatim: "A port is **narrow**: one interface, one capability, the
  methods the use case actually calls. A method that no use case calls is not a
  method of the port yet."

- **S7**: The fourth defect, verbatim, from the same inventory: "**NEW — the
  telemetry config is duplicated byte-for-byte.** The four parsers
  `parseDuration`/`parseBool`/`parseRatio`/`parseLogLevel`
  (`mcp-server/.../config.go:634-680` ≡ `db-admin/.../config.go:551-597`) and
  the seven `EnvOtel*` constants. `packages/platform` owns the telemetry — this
  is a copy of a shared layer."

- **S8**: The rule of the contract that names the owner of the copy, verbatim:
  "**No copy of a shared layer.** The platform and the shared domain are shared:
  they are extended, never forked. The infrastructure of a service lives in
  `packages/platform`, and the models that more than one service needs live in
  `packages/domain`." And the rule of the module that owns the telemetry,
  verbatim, from `packages/platform/README.md`: "The platform holds the
  infrastructure every service of the monorepository needs and none of them
  should write twice: the clock, the logs, the telemetry, the HTTP server, and
  the health check the container runs."

- **S9**: The rule of the module that owns the telemetry, verbatim, from the
  same file: "It is a **library**. It publishes no port on the network, it
  starts no process, and it reads no environment variable." The configuration
  of the tracing therefore lives in the platform, and the reading of the
  environment stays in the service: the platform receives the reader and never
  reads the environment itself.

- **S10**: The rule of the contract that names the writer of the schema,
  verbatim: "**No second writer of the schema.** One service owns the
  migrations of the database, and it applies them at the start. No service
  migrates on boot." The migration of the identity of the email is written by
  the owner of the schema, in `packages/database/migrations`, and applied by the
  database admin service.

- **S11**: The branch of the work and its worktree, per the practice of the
  repository: `feat/architecture-defects`, in the worktree
  `/Users/braejan/workspace/witsaba/repositories/paca-worktrees/architecture-defects`,
  created from `main` at `c227110`.

- **S12**: The fifth defect, found while the slice was being finished, as the
  code proves it: the three container tests of the MCP server built the image
  of the server inside each test, and the builder deletes the image and
  creates it again with the same name. The failure, verbatim:

  ```
  create container: build image: failed to create an image
  docker.io/paca/mcp-server:integration-test with target sha256:5c6551652717c77b883e5a389b40697204bfe4fddf1f5c01054ab535e3b30dd6
  after deleting the existing one: AlreadyExists: image
  "docker.io/paca/mcp-server:integration-test": already exists
  ```

- **S13**: The sixth defect, found by the review the maintainer asked for, as
  the code proves it: the rule of the gate that says "the application layer
  imports no adapter" forbade only the paths `/internal/adapters/` and
  `/internal/platform/`, so an application layer that imported the adapter of
  the shared library — `packages/database/postgres` — passed the gate with no
  failure.

## Tasks

| ID  | What | Links | Route | Commit |
|-----|------|-------|-------|--------|
| T1  | The feature document of the slice | S1 | inline | `3139c13` (the document) |
| T2  | The identity of the email address belongs to the domain: the address is stored in its canonical form, and the rows that exist are brought to that form by a migration of the owner of the schema | S2, S3, S4, S10 | inline | `011a7d5` (the fix) |
| T3  | One capability, one door: the library that owns the database publishes the only handle of PostgreSQL, and both services ask the database through it | S5, S4 | inline | `415cf5d` (the fix) |
| T4  | The port of the users is narrow: the method that no use case calls leaves the port, and the adapter keeps the helper it needs for itself | S6 | inline | `8ed476c` (the fix) |
| T5  | The environment of the monorepository is read the same way everywhere: the parsers and the level of the logs live in the platform, and neither service writes them twice | S7, S8 | inline | `451855d` (the fix) |
| T6  | The configuration of the tracing belongs to the platform: the names, the shape, the reading and the rules live there, and both services read the same one | S7, S8, S9 | inline | `8b9abca` (the fix) |
| T7  | The image of the server is built once, before the container tests run: the three parallel tests fought over the name of the image and the gate failed about one run in three | S12 | inline | `56350a1` (the fix) |
| T8  | The gate refuses an application that names a concrete adapter: the rule forbade only the adapters of the service, and the adapter of the shared library passed | S13 | inline | `c75bd70` (the fix) |
| T9  | The evidence: the gate of the repository is green from nothing, and the review that the maintainer asked for says what is still open | S1 | inline | (the evidence is in L2, the review in L3) |

The slice was merged as a squash (`134ad00`), so the history of `main` holds one commit for the
whole feature. The identifiers of the work units are not in the history of `main`: they live in the
pull request #47, whose head the server keeps. The record of what happened is the pull request
itself.

## Log

- **L1** (2026-10-30): The request of the maintainer, verbatim: "open a new git
  work tree and add the four live defects. Once you finish, review again in
  order to check that there is not more defects to be fixed".

  The four defects are the ones of the inventory that was read to the
  maintainer after the pull request #45 was merged (the squash `c227110`). The
  inventory is the tail of the audit of the architecture (the pull request #35):
  the ten findings of the audit were fixed by the eleven work units of that
  branch, and four remained open. Each one of them breaks a rule that the
  contract of the repository states.

  The fourth one is new since the audit: the duplication of the configuration of
  the telemetry was not a finding of the audit, and it is a copy of a shared
  layer, which the contract forbids.

  The work lands in the worktree of the branch `feat/architecture-defects`,
  created from `main` at `c227110`. The `.env` of the main worktree is copied
  into it, because the volume of PostgreSQL is shared by the clone and the
  credentials have to be the same.

- **L2** (2026-10-30): The evidence of the slice.

| Check | Result |
| --- | --- |
| `make all` (the gate of the repository, from nothing) | Green: the format, the analyzers, the linter with no issue, the build, the tests, the race detector, the coverage of every module at or above its floor, the vulnerability scan of every module, the container tests of the libraries and of the services, and the stack of the monorepository up and healthy. |
| The migration of the identity, against a real PostgreSQL | Green: the version 3 is applied, the constraint of the canonical form refuses the writer that does not know the domain, the same mailbox in another case is the address of somebody else, and the migration brought the rows that existed to the canonical form. |
| The door to the database | Green: the assertion of the structure refuses the adapters that name a handle, and both services reach the database through the pool of the library. |
| The container tests of the MCP server, six times without the cache | Green, and each run takes a second and a half instead of the half minute the three builds took. |
| The gate of the layers | Green with 337 assertions, against the 331 it proved before: the rule of the application layer is now stated as what the layer may name, and the probe proves it discriminates. |

**Risk:** item 1 — the migration rewrites the rows that exist. It refuses a
mailbox that two people carry, and it fails with it: which of the two rows
keeps the address is a fact about the people, and only a human decides it.

- **L3** (2026-10-30): The review the maintainer asked for: "review again in
  order to check that there is not more defects to be fixed".

  The review read the contract against the code, after the six work units.
  It found that the four defects are closed, and it found what is still open.
  Two of its findings were wrong and were proved wrong: the shared domain and
  the shared platform were never unchecked — the pattern of the gate is
  `././...`, and it reads every package of both modules. The other one was
  right: the rule of the application layer let a concrete adapter of the
  shared library through, and it is closed in T8.

  What is still open, in the order the contract asks for it:

  1. **The one writer of the schema is not pinned.** The code obeys the rule —
     only the database admin applies the migrations — but no assertion says
     so. A second writer would pass the gate.
  2. **The dead API is not pinned, and there is dead API.**
     `MigrateService.Status`, `State.IsUpToDate`, `State.AppliedVersions`,
     `Status.IsHealthy`, `Providers.Propagator` and `SDKServer` have no
     production caller.
  3. **The middleware of the router is written twice.** The two services
     carry the same tracing, the same security headers, the same log of the
     requests, the same recovery of a panic and the same recorder of the
     status. The platform owns the HTTP server, and the middleware belongs
     with it.
  4. **The names of the environment of the database are written twice.**
     `PACA_DATABASE_*` and `POSTGRES_*` are declared in both services, and
     the platform holds the names every service reads the same way.
  5. **The wiring of the telemetry is written twice**, in the two `main`.
  6. **The default branch of the mapping of the errors returns the raw
     error.** A library error that is not in the table reaches the agent as
     a Go string. It is unreachable today, because the composition root
     holds no repository without a database.
  7. **The rule of the domain layer of the applications proves nothing
     today**, because neither application has a domain layer. The rule
     discriminates the moment one appears — the probe proves it — but the
     count of the assertions of the gate does not say so.

  These are the next slices. They are the same shape as this one: one rule
  of the contract, one owner, and an assertion that proves it.

- **L4** (2026-10-30): The slice was merged as the squash `134ad00`, and the
  pull request #47 is closed. The issue #46 is closed with it.

  The history of `main` holds one commit for the whole slice. The
  identifiers of the nine work units are not in it: they live in the head of
  the pull request #47, whose head the server keeps, and the record of what
  happened is the pull request itself.

  The review of L3 is the state of the day it was written. The seven
  findings it lists are still open, and they are the next slices.
