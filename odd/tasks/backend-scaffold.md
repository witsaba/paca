# Feature: Backend Scaffold (Go, Hexagonal)

Prepares the PACA backend so application work can start later: a Go workspace, a hexagonal
`paca` CLI application exposing only `--help`, a repository-wide Makefile with the
standard quality targets, and the toolchain pinned as Go tool dependencies.

## Specs

### S1 — Scope of this session

> "This session is for initializing the project."

> "DO NOT ADD ANY DATABASE MIGRATION YET, only preparing the scafollding and a Paca CLI
> header application with the --help with only the --help command."

Scaffolding only. **No** database migration, **no** schema change, **no** SQLite driver,
**no** persistence adapter, **no** application feature. `packages/db/migrations/` is not
touched by this feature.

### S2 — Language and toolchain

> "Golang is the main back choosen by me for continue with implementing the application."

Go is the backend language. The repository becomes a Go workspace. The toolchain version
used is the one installed and verified on the machine: `go1.26.3 darwin/arm64`. The
`go` directive in every module and in `go.work` pins that version, so a contributor on an
older toolchain gets a clear "requires go >= 1.26.3" error instead of a silent mismatch.

### S3 — Hexagonal architecture

> "Use hexagonal architecture and add all the needed for start later."

The `paca` application follows ports-and-adapters. The dependency rule is inward only:
`cmd` -> `adapters` -> `application` -> `ports` -> `domain`, and `domain` imports nothing
from the project. Concretely:

| Layer | Package | Holds |
| --- | --- | --- |
| Entry point | `cmd/paca` | composition root only: build the command, execute it, exit |
| Driving adapter | `internal/adapters/cli` | the cobra command tree — the delivery mechanism |
| Application | `internal/application` | use cases; orchestrate the domain through ports |
| Ports | `internal/ports` | interfaces: driving (input) and driven (output) boundaries |
| Domain | `internal/domain` | entities, value objects, invariants, domain services |

Each package ships a `doc.go` stating its role and its dependency rule, so the scaffold
documents the boundary it is meant to keep. The domain package must never import an
adapter, an application package, or any I/O library; the application package depends on
ports and domain only.

### S4 — Makefile with the standard targets

> "Focus on build a proper Makefile with options for download dependencies, build, test,
> test with coverage, check the race conditions, check the vulns and web research about the
> standar industries."

One repository-wide Makefile at the root, the single entry point for the developer
workflow. Required targets, all working against the Go workspace:

| Target | Command it wraps | Purpose |
| --- | --- | --- |
| `deps` | `go mod download` per workspace module | download dependencies |
| `build` | `go build` per module | compile every binary into `bin/` |
| `test` | `go test ./...` | run the test suite |
| `cover` | `go test -coverprofile` | test with coverage, per-package and total |
| `race` | `go test -race ./...` | detect data races |
| `vuln` | `govulncheck ./...` | scan reachable dependencies for known vulnerabilities |

Industry-standard targets added on the same evidence: `tidy` (`go mod tidy` per module),
`lint` (`golangci-lint run`), `fmt` / `fmt-check` (`gofmt`), `all` (the full gate, used by
CI), `run` (execute the CLI through `go run`), `clean`, and `help` as the default target.
`make` with no arguments prints the target list.

The tools are **not** expected to be installed globally. `golangci-lint` and
`govulncheck` are declared as Go tool dependencies (Go 1.24+ `tool` directive) in a
dedicated `tools` module, so the exact tool version is pinned in `go.mod`/`go.sum` and
`go tool <name>` runs it. A developer needs only the Go toolchain.

### S5 — The `paca` CLI header application

> "Prepare it to create later a CLI, do not focus in the CLI yet but is good to prepare the
> way."

> "a Paca CLI header application with the --help with only the --help command."

The `paca` binary exists as a header: a cobra root command with a name, a short
description, a long description, and **no subcommands and no other flags**. `--help` is
the only supported command. Invoking `paca` with no arguments prints the help text and
exits `0`. The CLI is a driving adapter, so the future commands attach as subcommands of
the same root command without touching the domain or the application layer.

### S6 — Industry standards, researched not assumed

> "web research about the standar industries"

The layout, the toolchain pinning, the linter, and the vulnerability scanner follow the
published Go guidance, not invention. The research and its sources are recorded in
`docs/architecture/backend-scaffold.md`:

- `cmd/` + `internal/` module layout — go.dev, *Organizing a Go module*.
- Tool dependencies via the `tool` directive — Go 1.24+ feature, removes the global
  install requirement.
- `golangci-lint` v2 as the linter, `govulncheck` as the vulnerability scanner — the two
  tools the Go project itself documents.
- `go.work` for a multi-module monorepo.

### S7 — Work in a git worktree

> "Work as usual in a git worktree and this is the purpose."

All work happens in a git worktree of the same clone, on the branch
`feature/backend-go-scaffold`, at
`/Users/braejan/workspace/witsaba/repositories/paca-worktrees/backend-go-scaffold`.
`main` is not modified. The branch is pushed and a pull request against `main` is opened
only if the user asks for it in this session.

### S8 — Monorepo integration

The repository keeps its existing shape: `apps/` holds runnable applications,
`packages/` holds shared libraries, `tools/` holds repository-wide developer tooling.
The Go workspace ties them together, so `make test` covers every module. The root
`README.md` and the `apps/README.md` are updated to describe the new layout, and
`packages/db` stays exactly as it is — schema only, no Go code yet.

## Tasks

| ID | Specs | Route | Commit |
| --- | --- | --- | --- |
| T1 | S2, S6, S8 | inline — Go workspace, `tools` module with pinned tool dependencies | `fd6f273` build(go): add Go workspace and pinned tool dependencies |
| T2 | S3 | inline — hexagonal package skeleton with `doc.go` boundary contracts | `fefdb26` feat(cli): scaffold hexagonal packages for the paca app |
| T3 | S5 | inline — test-first cobra root command exposing only `--help` | `90ba264` feat(cli): add paca root command with help-only surface |
| T4 | S4 | inline — repository-wide Makefile and `golangci-lint` configuration | `4fe8c32` build(make): add repository-wide Makefile and lint config |
| T5 | S6, S8 | inline — architecture note, app README, layout updates | `cb7276c` docs: document the Go backend scaffold and update repo layout |
| T6 | S1, S4 | inline — manifests tidied, full verification gate | `ae5b9d5` build(go): tidy the module manifests |

## Log

### L1 — Origin (verbatim)

> let start with preparing the backend app. Let use Golang. So your mission rn is to explore
> and prepare the backend project to run go. Golang is the main back choosen by me for
> continue with implementing the application. This session is for initializing the project.
> Use hexagonal architecture and add all the needed for start later. Focus on build a proper
> Makefile with options for download dependencies, build, test, test with coverage, check the
> race conditions, check the vulns and web research about the standar industries. Prepare it
> to create later a CLI, do not focus in the CLI yet but is good to prepare the way. Work as
> usual in a git worktree and this is the purpose. DO NOT ADD ANY DATABASE MIGRATION YET,
> only preparing the scafollding and a Paca CLI header application with the --help with only
> the --help command.

### L2 — Decisions taken during implementation

- **Multi-module workspace over a single root module.** The repository is a monorepo whose
  `apps/` and `packages/` are meant to hold independently versioned units. `go.work` at the
  root with one module per application keeps that boundary real; a single root module would
  erase it before any code exists.
- **Tools in a dedicated `tools` module.** `golangci-lint` pulls ~150 transitive modules.
  Keeping them in `tools/go.mod` keeps the application modules free of linter dependencies,
  and `go tool` still resolves them from the workspace.
- **cobra as the CLI framework.** The de facto standard for Go CLIs; it provides the
  `--help` output, the command tree that S5 requires later, and the flag parsing that a
  finance CLI will need. `urfave/cli` was rejected as the less common choice.
- **CI workflow included.** The user asked for the standard industry quality gates; a
  single GitHub Actions workflow that runs `make all` is how those gates are actually
  enforced, and it adds no product surface. It shipped inside the documentation commit
  (`cb7276c`) and is **held back from the push** by `cfab536`: GitHub refuses to create
  a workflow file from a push whose token lacks the `workflow` scope, and the token of
  this environment does not carry it. The file is preserved in the history of the
  branch and is restored by one follow-up commit once the scope is granted
  (`gh auth refresh -h github.com -s workflow`).

### L3 — Verified outcome

Every check below was run in the worktree and observed, not assumed:

| Check | Command | Result |
| --- | --- | --- |
| Build | `make build` | every module compiles; `bin/paca` produced |
| Help output | `./bin/paca --help` | prints the header, exit `0` |
| No arguments | `./bin/paca` | prints the same header, exit `0` |
| Unknown command | `./bin/paca accounts` | `Error: unknown command "accounts" for "paca"`, exit `1` |
| Test suite | `make test` | `ok github.com/witsaba/paca/apps/cli/internal/adapters/cli` |
| Coverage | `make cover` | `internal/adapters/cli` at 100.0%; `cmd/paca` at 0.0% by design (see the architecture note) |
| Race detector | `make race` | no race reported |
| Vulnerabilities | `make vuln` | `No vulnerabilities found` in both modules |
| Lint | `make lint` | `0 issues` in both modules |
| Formatting | `make fmt-check` | no unformatted file |
| Manifests | `make tidy` | idempotent: a second run leaves no diff |
| Full gate | `make all` | exit `0` |
| Schema untouched | `bash packages/db/tools/verify_schema.sh` | `passed: 57, failed: 0` — the migration was not touched by this feature |

The test for the `--help` rule was observed RED before the implementation
(`undefined: cli.NewRootCommand`, build failure) and GREEN after it.

### L4 — Not done, by design

- No database migration and no SQLite driver: S1 forbids it. The driver choice
  remains the open question in the root README.
- No CLI subcommand beyond the header: S5 limits the surface to `--help`.
- The branch is pushed and the pull request is opened against `main` on the user's
  request, linking issue #4.
- **The CI workflow is not in the pushed branch.** GitHub rejects a push that creates
  a workflow file when the token lacks the `workflow` scope, which the token of this
  environment does not carry. The workflow is authored and kept in the branch history;
  it is restored by a single follow-up commit once the scope is granted, and the pull
  request carries no automated checks until then. The local gate (`make all`) is the
  proof in the meantime.
