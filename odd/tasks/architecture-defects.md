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

## Tasks

| ID  | What | Links | Route | Commit |
|-----|------|-------|-------|--------|
| T1  | The feature document of the slice | S1 | inline | (this document) |
| T2  | The identity of the email address belongs to the domain: the address is stored in its canonical form, and the rows that exist are brought to that form by a migration of the owner of the schema | S2, S3, S4, S10 | inline | (the fix) |
| T3  | One capability, one door: the library that owns the database publishes the only handle of PostgreSQL, and both services ask the database through it | S5, S4 | inline | (the fix) |
| T4  | The port of the users is narrow: the method that no use case calls leaves the port, and the adapter keeps the helper it needs for itself | S6 | inline | (the fix) |
| T5  | The environment of the monorepository is read the same way everywhere: the parsers and the level of the logs live in the platform, and neither service writes them twice | S7, S8 | inline | (the fix) |
| T6  | The configuration of the tracing belongs to the platform: the names, the shape, the reading and the rules live there, and both services read the same one | S7, S8, S9 | inline | (the fix) |
| T7  | The evidence: the gate of the repository is green from nothing, and the review that the maintainer asked for says whether a defect is still open | S1 | inline | (the evidence is in L2) |

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

- **L2** (2026-10-30): The evidence of the slice, and the review that the
  maintainer asked for. (pending)
