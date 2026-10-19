# CI — the continuous integration of the repository

The feature document of the continuous integration of PACA: the workflow
that runs the gate of the repository on every push and on every pull
request, the coverage floor that the gate enforces, and the composition
root of the CLI that the floor exposed.

- **Branch:** `feat/ci-github-actions`, cut from an up-to-date `main`
  (`bd7a378`), in its own worktree `paca-worktrees/ci-workflow`.
- **Done when:** the workflow runs the gate of the repository, the floor of
  the coverage holds for every file, and `make all` is green.

---

## Specs

Each spec is quoted verbatim. `R#` quotes the repository; `U#` quotes the
user.

- **R1** — The user (verbatim): "let try to solve the CI stuff, i'm almost
  sure i granted the workflow permission to the current gh connection,
  verifyit."
- **R2** — The house rule (coverage): "every new or modified file needs at
  least 75% test coverage, measured per file, statement-weighted".
- **R3** — The gate of the repository (verbatim, from the Makefile): "## all:
  the full quality gate: fmt-check, vet, lint, build, test, race, vuln".
- **R4** — The house rule (worktree): "every session works in its own
  independent git worktree, on its own branch, never in the main checkout".
- **R5** — The house rule (commits): "The commit is a reviewable slice of
  roughly 150–400 authored changed lines, which is the house limit".
- **R6** — The house rule (tests first): "**Write the test first.** One RED
  test per rule the session names".
- **R7** — The verifier of the schema (verbatim, from
  `packages/db/tools/verify_schema.sh`): "Requires: sqlite3 >= 3.37.0
  (STRICT tables)." and "It is a database-level check only. It does not need
  an application, a language runtime, or any dependency beyond the sqlite3
  CLI."
- **R8** — The toolchain of the repository: the workspace and its five
  modules pin `go 1.26.9`, and the linter and the scanner are pinned as tool
  dependencies of the module `tools`, so that a contributor needs nothing
  but the Go toolchain installed.

### The facts of the exploration (2026-10-19)

- The token of the GitHub CLI carries the scope `workflow`:
  `admin:public_key, gist, read:org, repo, workflow` (`gh auth status`). The
  push of a workflow file is therefore allowed, and the user was right.
- The repository has no `.github` directory: no workflow, no PR template, no
  `status:approved` label.
- The gate of the repository is `make all`: fmt-check, vet, lint, build,
  test, race and vuln. Every target discovers the modules with `find`, so a
  new module is picked up without touching the Makefile.
- The coverage of the repository is measured per module by `make cover`, and
  the profiles are merged into `coverage.out`. The measurement of the house
  rule is per file, and the merged profile is what carries it.
- **The gate would fail on the first day.** `apps/cli/cmd/paca/main.go` is
  at 0% of coverage: `main()` cannot be called from a test, and the body of
  the command tree was never moved out of it. The session S02 solved the
  same problem for the root of the MCP server (`apps/mcp/cmd/paca-mcp` is
  at 92.3%), and the same shape is the fix.
- The verifier of the schema needs the `sqlite3` CLI and nothing else, and
  the runner of GitHub ships one.

---

## Tasks

| ID | Task | Spec | Route | Commit |
|---|---|---|---|---|
| T1 | Branch `feat/ci-github-actions` from an up-to-date `main` in its own worktree `paca-worktrees/ci-workflow`; this feature document is the first slice | R4 | parent | `1d84bde` |
| T2 | The workflow `.github/workflows/ci.yml`: the quality gate, the verifier of the schema, and the coverage floor, on every push to `main` and on every pull request | R1, R3, R7, R8 | parent | `fc4de28` |
| T3 | The coverage floor in the Makefile: the target `cover-check` that fails when a file of the repository is below 75% of its statements | R2 | parent | `f7b02df` |
| T4 | The composition root of the CLI made testable: the body of the command tree moves out of `main()` into `execute()`, and its test | R2, R6 | parent | `9b538a8` |
| T5 | The documentation of the repository: the workflow in the layout of the README, and the target in the help of the Makefile | R1 | parent | `9fd418f` |
| T6 | Verification: `make all` green, `make cover-check` green, the workflow validated, and the gate reproduced locally | R3 | parent | — (the closing slice of the branch) |

---

## Log

### L1 — Origin (verbatim)

> let try to solve the CI stuff, i'm almost sure i granted the workflow
> permission to the current gh connection, verifyit.

### L2 — The findings of the exploration

The facts are in the section above: the scope `workflow` is granted, the
repository has no `.github`, the gate of the repository is `make all`, and
the coverage floor would fail on the composition root of the CLI, which is
at 0% because `main()` cannot be called from a test.

### L3 — The decisions of the session

- **The workflow lives on its own branch.** The continuous integration is not
  a session of the roadmap of the MCP server, and it does not touch the
  code the roadmap builds: it has its own branch and its own worktree, cut
  from `main`, so that the pull request of the session S03 carries nothing
  but the session S03.
- **The gate is the gate of the repository.** The workflow runs `make all`,
  the target a contributor runs, and not a list of commands copied into the
  workflow: the two cannot drift apart.
- **The coverage floor is a Makefile target**, and not an `awk` inside the
  workflow: the contributor runs it before the push, and the workflow runs
  the same thing.
- **The floor is a variable.** `COVERAGE_FLOOR` defaults to 75, the house
  rule, and a session can raise it when the repository is ready for it.
- **The runner needs nothing but the toolchain.** The version of Go is the
  one of the workspace, the linter and the scanner are the tool dependencies
  of the module `tools`, and the verifier of the schema needs the `sqlite3`
  CLI of the image.

### L4 — The verification of the session (2026-10-19)

- `make all` is green over the five modules: fmt-check, vet, lint (0 issues),
  build, test, race and vuln (no vulnerability in any module).
- The verifier of the schema passes 57 of 57 checks.
- `make cover-check` is green: every file of the repository is at or above
  75% of its statements, and the lowest file is the root of the CLI at
  87.5%.
- **The floor is not a no-op.** The target was run with a floor of 99%:
  it named the eleven files below it and failed with the status 2.
- The workflow was validated with `actionlint` (v1.7.12): no problem.
- The three commands of the workflow were run in the worktree, exactly as
  the runner runs them.

### L5 — What the session found on the way

- The composition root of the CLI was at 0% of coverage, and the coverage
  floor would have failed on the first day. The fix is the one the session
  S02 applied to the root of the MCP server: the body of the process moves
  out of `main` into a function that a test can call.
- The tree of the README was already behind the repository: it names neither
  `apps/mcp` nor the feature document of the session S02. The session added
  the workflow and left the rest alone: the debt belongs to the sessions
  that own those files, and the pull request of this session carries only
  the continuous integration.
