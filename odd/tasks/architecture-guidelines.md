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
| T3   | `AGENTS.md` of the root: the project, the architecture contract, the commands, the workflow   | S2, S5, S6 | inline | _pending_ |
| T4   | The mirror of the contract in Engram                                                          | S4      | inline  | _pending_ |
| T5   | The repair of the repository that is nil inside an interface                                 | S7.1    | inline  | _pending_ |
| T6   | The repair of the tool contract: the schema and the description derive from the domain        | S7.2    | inline  | _pending_ |
| T7   | The repair of the model of the migrations: one owner                                         | S7.3    | inline  | _pending_ |
| T8   | The repair of the duplication: the health domain and the platform layer are shared           | S7.4    | worker  | _pending_ |
| T9   | The repair of the gate: the rule of the layers is pinned by a check                          | S7.5    | inline  | _pending_ |
| T10  | The documentation of the repository and the evidence of the slice                             | S2, S8  | inline  | _pending_ |

Each task is one work unit: one commit, with its tests and its documentation alongside
the behavior it changes.

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
