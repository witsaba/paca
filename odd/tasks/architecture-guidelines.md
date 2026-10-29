# PACA — the architecture contract: the agent guidelines and the fixes of the audit

Feature document for the slice that turns the architecture of the monorepository into a
written contract for coding agents (`AGENTS.md`), mirrors it into Engram, and repairs the
inconsistencies the audit of the architecture found: the typed-nil repository that makes
the "no database" answer unreachable, the domain rules that live in the adapter, the model
of the migrations that has two owners, the health domain and the platform layer that are
duplicated per application, and the layer rule that no check of the gate pins.

- **Project:** the whole monorepository (the applications, the shared libraries, the gate).
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-architecture`, branch
  `feat/architecture-guidelines`, based on `main` at `428d4a8`.
- **Scope:** the written contract, its mirror in Engram, and the five repairs of the audit.
  No new table, no new endpoint, no new tool, no change of the observable behavior of the
  services beyond the messages the audit found to be wrong.

---

## Specs

Each spec is the user's own wording, quoted verbatim, except the ones marked *(decided)*,
which record a choice taken with the user.

### S1 — A worktree of its own

> ok open a new git worktree to working on this fixes

The work happens in the worktree
`/Users/braejan/workspace/witsaba/repositories/paca-architecture`, on the branch
`feat/architecture-guidelines`, cut from `main` at `428d4a8`. `main` is not touched.

### S2 — The guidelines of the agents

> add a AGENTS.md (investigate first in a web research for guidelines and good practices)

The file is the canonical entry of the instructions of the repository, at the root. It is
written after a research of the guidelines and of the good practices of the format, and it
follows what that research says: it explains the project before its rules, it keeps one
canonical entry, it uses progressive disclosure, and every command it names exists.

### S3 — The research comes first

> investigate first in a web research for guidelines and good practices

The research of the format is done before the file is written, and its conclusions are
recorded in the log of this document. The file is not a template: it is written from the
repository, and it does not repeat what the tools of the repository already enforce.

### S4 — The mirror in Engram

> ensure to add this to engram too

The contract is saved in Engram as a project observation with a stable topic key, so the
architecture of the monorepository survives the session and the next language of it.

### S5 — The contract is not about Go

> this is pretty important for code maintenance and new feature development agnostic even to Golang, maybe tomorrow is python or rust or another one

The rules of the file are stated in the language of the architecture — hexagonal, DDD,
SOLID — and not in the language of Go. Each rule says what it means in a module boundary,
in a package boundary and in a folder boundary, so the same contract holds when a service
of the monorepository is written in Python or in Rust. The Go specifics of the repository
are a section of their own, and they are the only ones.

### S6 — The architecture, the DDD and the SOLID must be applied properly

> the Architecture, the DDD and SOLID must to be applied properly

The file states, as rules an agent can check, the layering and the direction of the
dependencies, the ownership of the rules of the domain, the ownership of the ports, the
single owner of every model, and the five SOLID principles as they apply to a service of
this monorepository. Each rule says how it is verified in this repository.

### S7 — The repairs of the audit

> working on this fixes

The five repairs the audit found, in the order the audit ranked them:

1. the repository that is nil inside an interface makes `ports.ErrNoDatabase`
   unreachable, and the agent receives an infrastructure message instead of the promised
   one;
2. the tool schema and the description of the tool restate the map of the states of the
   domain instead of deriving it from it, and the error of a concurrent change reaches the
   agent unmapped;
3. the model of the migrations has two owners, the library and the application;
4. the health domain and the platform layer are copied into every application;
5. no check of the gate pins the rule of the layers.

### S8 — The contract must not be a promise

The rules of `AGENTS.md` are only worth writing when the repository can show that it keeps
them. The repairs of S7 are the proof of the contract, and the gate of the repository
grows the checks that pin the rule of the layers (S7.5).

---

## Tasks

| ID   | Task                                                                                          | Specs   | Route   | Commit    |
|------|-----------------------------------------------------------------------------------------------|---------|---------|-----------|
| T1   | The worktree, the branch and the feature document                                             | S1      | inline  | this file |
| T2   | The research of the format of the guidelines of the agents, and its conclusions               | S3      | inline  | this file |
| T3   | `AGENTS.md` of the root: the project, the architecture contract, the commands, the workflow   | S2, S5, S6 | inline | `a8ab4c8` |
| T4   | The mirror of the contract in Engram                                                          | S4      | inline  | Engram 4618 |
| T5   | The repair of the repository that is nil inside an interface                                 | S7.1    | inline  | `645c8c1` |
| T6   | The repair of the tool contract: the schema and the description derive from the domain        | S7.2    | inline  | `d702118` |
| T7   | The repair of the model of the migrations: one owner                                         | S7.3    | inline  | `a8073fc` |
| T8   | The repair of the duplication: the health domain and the platform layer are shared           | S7.4    | inline  | `85d4493` |
| T9   | The repair of the gate: the rule of the layers is pinned by a check                          | S7.5    | inline  | `8b6d3b4` |
| T10  | The documentation of the repository and the evidence of the slice                             | S2, S8  | inline  | this file |

Each task is one work unit: one commit, with its tests and its documentation alongside
the behavior it changes.

### The slice boundary and the delivery strategy

The branch holds **six work units** since `main` (`428d4a8`), and the tooling of the
repository counts them as about **3,200 authored changed lines** — well past the ~400
lines of a reviewable slice. The feature is therefore delivered as a **chain of pull
requests**, one per work unit, in the order of the table above:

| Pull request | Commit                   | What a reviewer reads                                    |
|--------------|--------------------------|----------------------------------------------------------|
| 1            | `a8ab4c8`                | the contract of the agents, and the entry of every module |
| 2            | `645c8c1`                | the promise of the "no database" answer, kept             |
| 3            | `d702118`                | the contract of the status tool, derived from the domain  |
| 4            | `a8073fc`                | the state of the schema, with one owner                   |
| 5            | `8b6d3b4`                | the gate that pins the layers                              |
| 6            | `85d4493`                | the shared platform and the shared health model           |
| 7            | `75f5e79`, `b352a14`, `83e3d72` | the evidence of the slice, and the images that copy every module they import |

The pushes, the pull requests and the merge stay the decision of the person: the branch
holds the work, and nothing is published by the agent. A reviewer who wants the smallest
possible diff reads the pull request of the behavior they care about — each one builds,
each one is green on its own, and each one is a commit that says what it changes.

---

## Log

### L1 — the original request, verbatim

> ok open a new git worktree to working on this fixes, add a AGENTS.md (investigate first in a web research for guidelines and good practices) and ensure to add this to engram too, this is pretty important for code maintenance and new feature development agnostic even to Golang, maybe tomorrow is python or rust or another one, the Architecture, the DDD and SOLID must to be applied properly.

### L2 — the request that the audit answered, verbatim

> recap the latest observations about the SOLID, Hexagonal Architecture and Domain Driven Design in the @apps/ existing so far. I need you explore deep explore and search for any inconsistence about the previous Architecture, designs and principles mention. Is needed ensure that all are really implementing it before continuing adding more features. Let me read your analysis and exploration. DO NOT MAKE CHANGES if find something.

The audit of that request is the evidence of S7. It was read-only: no file of the
repository was changed by it. Its findings were saved in Engram as two observations.

### L3 — the conclusions of the research of the format

The research read the canonical description of the format (`agents.md`, stewarded by the
Agentic AI Foundation under the Linux Foundation) and two guides of good practices
(`farhadeidi/agents-md-guide` and the best practices of `agent-document-reviewer`). What
they say, and what this slice therefore does:

1. **One canonical entry.** The root `AGENTS.md` is the entry of the repository. Other
   entry files point to it; they never copy it. The format is read by Codex, Amp, Jules,
   Cursor, Copilot, Aider, Windsurf, Zed and Factory.
2. **Explain the project before its rules.** The file opens with what PACA is, in the
   terms of its domain, and only then with the rules.
3. **Maximize decision value, not instruction volume.** Every line is loaded on every
   task, so a line is written only when it is specific to the repository, durable, and
   usable by an agent to decide where a change belongs. The generic advice, the style
   rules that the linter already enforces, and the copied documentation are left out.
4. **Progressive disclosure.** The root file stays lean and points to the deeper
   documents; a monorepo carries an entry file per subproject, and the closest one wins.
5. **The commands must be copy-pasteable and must exist.** No command that the
   repository does not have.
6. **Inspect before writing.** The rules of the file are written from the repository, not
   from a template; the audit that preceded the file is what says which rules are true.
7. **It is living documentation.** The file is updated with the code it describes.

### L4 — the shape of the contract, and why it is not about Go

The rules are written as rules of the architecture, and each one is given the boundary
that enforces it in every language: a module boundary (Go), a package boundary (Python),
a crate boundary (Rust) or a folder boundary (any language). A rule that only Go can
enforce is a rule of Go and it says so.

### L5 — the evidence of the audit, in short

The audit of L2 confirmed that the boundary of the modules is real and enforced by the
compiler, that the layering inside the applications holds in every import it could find,
and that the domain module imports nothing but the standard library. It also found the
seven inconsistencies of S7, two of which reach the agent that uses the MCP server.

### L6 — the decisions the repairs required

1. **The absence of a collaborator is the absence of the value.** The library of the
   database no longer returns a `*UserRepository` that is nil inside a `user.Repository`:
   it refuses the nil pool, and the composition root of the MCP server leaves the
   interface nil when there is no database. The promise of the "no database" answer is
   now reachable. *(T5)*
2. **The tool contract is derived, never re-typed.** The states a tool accepts come from
   `user.AllTargets()`, and the description of the allowed moves comes from
   `user.AllStatuses()` and `AllowedTargets()`. The error of a concurrent change is
   mapped into the words of the agent. *(T6)*
3. **The model of a migration has one owner.** It lives in `packages/domain/schema`,
   because two sides need it: the service that writes the schema and the library that
   applies the migrations. The adapter between them maps nothing. The service named the
   dependency in its own `go.mod` — a replace of a dependency is ignored, and the module
   did not build outside the workspace. *(T7)*
4. **The infrastructure is shared; the configuration is not.** The clock, the logs, the
   telemetry, the server and the health check of the container live in `packages/platform`
   now, and the model of the health in `packages/domain/health`. The configuration stays
   with each service: it is the one thing that is really its own, and the two services do
   not read the same environment variables. *(T8)*
5. **The health use case and the health handler stay with each service.** They depend on
   the ports of the service that owns them, and a port belongs to its consumer. Sharing
   them would mean moving the ports into a shared library, which is the ownership rule
   broken in the other direction. The duplication that remains is deliberate, and it is
   the price of the rule. *(T8)*
6. **A layer may be empty.** After the move, neither service has a domain layer of its
   own: every model it needs belongs to more than one service and lives in the shared
   domain. The contract says so, and the gate accepts it: a layer that has no package of
   its own cannot break the rule of the layers. *(T8, T9)*
7. **The gate pins the direction of the dependencies.** The rule is read from the
   compiler itself — the packages of every layer and their real imports — so the
   assertion cannot drift from the code. Four probes were run against the gate before it
   was trusted: a domain that named a driver, a use case that named an adapter, a module
   that forgot to name its dependency, and a layer that named an adapter outside the
   composition root — every one of them was caught. *(T9)*
8. **The image of a service copies every module the service imports.** The move of the
   platform broke the image of both services: the context of the build is the root of the
   repository, and the new module was not copied into it. The container test of the image
   caught it, and the gate now asserts it, so the next module that a service imports
   cannot be forgotten the same way. *(T9)*

### L7 — the evidence of the gate, and the one step that cannot pass on this machine

`make all` was run on the branch. What it proved:

| Step                                | Result                                                              |
|-------------------------------------|---------------------------------------------------------------------|
| the quality gate of the five modules | green: format, analyzers, linter, build, tests, race, coverage, scan |
| the container tests of the packages  | green, against a real PostgreSQL that they create and throw away     |
| the container tests of the applications | green: the image of both services builds and serves                  |
| the shellcheck of the scripts        | green                                                               |
| the tests of the installer          | green                                                               |
| the tests of the structure          | green: 223 assertions                                               |
| **the smoke test of the database**   | **red, and not because of this slice**                              |
| the whole stack                     | **not reached, for the same reason**                                |

The smoke test of the database fails because the volume `paca-postgres-data` of this
machine holds a cluster that **another session initialized**: it carries the database
`paca-pg` and the password of that session, while every `.env` of the tree names the
database `paca` and carries its own password. The smoke test connects to a database that
does not exist, and the stack comes up with `password authentication failed for user
"paca"`.

**The failure is not of this slice.** It was reproduced on `main`, in the worktree of the
repository, without a single file of this slice: `make -C db smoke` fails there with the
same six assertions. The feature document `odd/tasks/core-split.md` records the same
observation, with the same volume, from the slice that split the core.

The fix belongs to the machine and not to the branch: either the volume is
re-initialized — it drops the local data of the development environment, so it is the
decision of the person — or the `.env` of the stack carries the credentials of that
cluster. Nothing of this slice is unproven because of it: the container tests of the
packages and of the applications assert the rules of the schema and of the services
against a real PostgreSQL that they create and throw away.

### L8 — the decision of the machine, and the state of the record

The decision of L7 was taken on the machine, and it was the second one: the
credentials of the cluster of the volume were aligned with the `.env` of the stack.
The volume `paca-postgres-data` was **not** dropped, so the data of the local
development environment is intact.

The step that L7 names as red is green. `make all` now runs the whole gate of the
repository from nothing on this machine and every step passes, the smoke test of the
database and the whole stack included. The table of L7 keeps the state of that day,
as the record of what happened; this entry is the state of today.

Two stale remote branches were also deleted from the origin: `feat/ci-github-actions`
and `feat/observability-tracing`. Both were already merged as squashes
(`2116666` and `33eb9f7`), and the identifiers of their work units live in the
pull requests that merged them, which the server keeps. Nothing that was not in
`main` was added back to it.

The work units of this slice were merged as the squash `d24574d`, so the history of
`main` holds one commit for the whole feature. The identifiers of the table above
live in the pull request #35, whose head the server keeps.
