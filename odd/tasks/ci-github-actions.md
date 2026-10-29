# CI — the continuous integration of PACA on GitHub Actions

The GitHub Actions workflow of the monorepository: the quality gate of the
MCP server, the smoke test of the PostgreSQL stack and the shellcheck of the
shell scripts, plus the root Makefile that dispatches them.

## Specs

- **S1**: The user's request, verbatim: "let work in a new git worktree, analyze the current database and app mcp added and add the CI for github."

- **S2**: The scope of the workflow, the option the user chose, verbatim: "Gate + DB + shellcheck"

  ```
  jobs:
    gate      -> make -C apps/mcp-server all
    database  -> docker compose build + db smoke test
    shellcheck-> shellcheck db/postgres/scripts/*.sh
  ```

- **S3**: How CI invokes the stacks, the option the user chose, verbatim: "Add a root dispatching Makefile"

  ```
  # Makefile (root)
  all: gate-apps gate-db
  gate-apps: ; $(MAKE) -C apps/mcp-server all
  gate-db:   ; $(MAKE) -C db smoke
  ```

- **S4**: The container tests of the MCP server, the option the user chose, verbatim: "No, keep CI on unit level"

  ```
  gate: fmt vet lint build test race cover vuln
  (no testcontainers job)
  ```

- **S5**: The branch of the work, per the branch-pr convention: `ci/add-github-actions`, in the worktree `/Users/braejan/workspace/witsaba/repositories/paca-worktrees/feat-ci-github-actions`, created from `main` at `780f5c7`.

## Tasks

| ID  | What | Links | Route | Commit |
|-----|------|-------|-------|--------|
| T1  | The feature document and the root dispatching Makefile (help, gate, shellcheck, db-up, db-smoke, db-down, all) | S1, S3 | inline | `521a5c9` (the document), `0351e90` (the Makefile) |
| T2  | The CI workflow `.github/workflows/ci.yml`: the gate job, the database job, the shellcheck job | S2, S4 | inline | `ad37931` |
| T3  | The documentation: the root README describes the root Makefile and the CI | S2, S3 | inline | `bcc7217` |
| T4  | The verification of the workflow: `make gate`, `make db-smoke` and `make shellcheck` run green locally | S2, S3, S4 | inline | (the evidence is in L9; no code) |
| T5  | The pull request: issue #13 updated to the current baseline, PR opened with `Closes #13` and the `type:chore` label | S1, S2 | inline | PR #20 (the deliverable) |

## Log

- **L1** (2026-10-26): "let work in a new git worktree, analyze the current database and app mcp added and add the CI for github."
- **L2** (2026-10-26): Scope, answered by the user: "Gate + DB + shellcheck" — the quality gate, the database smoke test and the shellcheck. No PR validation workflow (no `status:approved` label, no approval gate in the repository).
- **L3** (2026-10-26): Invocation, answered by the user: "Add a root dispatching Makefile" — the root dispatches, it holds no stack-specific logic.
- **L4** (2026-10-26): Container tests, answered by the user: "No, keep CI on unit level" — the testcontainers suite stays a local gate.
- **L5** (2026-10-26): Analysis of the baseline: `main` at `780f5c7` has no `.github`; `apps/mcp-server/Makefile` exposes `all` (fmt-check, vet, lint, build, test, race, cover, vuln) with an 80% coverage floor; `db/Makefile` is the umbrella of the databases and `db/postgres/Makefile` owns `up`/`smoke`; `db/postgres/scripts/smoke-test.sh` asserts the database guarantees and needs the stack running, `db/postgres/.env` and `nc` on the host; `go.work` pins Go 1.27.3 and `tools/go.mod` pins golangci-lint v2.15.0 and govulncheck v1.8.0.
- **L6** (2026-10-26): The branch `origin/feat/ci-github-actions` is stale: it forks from `bd7a378` (before the blank-slate reset) and still carries the deleted finance-manager tree. It is a style reference only; the workflow is written from scratch on top of the current `main`. The branch was deleted later: see L11.
- **L7** (2026-10-26): Issue #13 tracks the CI, but its body describes the pre-reset baseline (five modules, the sqlite schema verifier, a 75% per-file coverage floor). It is updated to the current baseline before the pull request closes it.
- **L8** (2026-10-26): `actions/checkout@v7` and `actions/setup-go@v7` exist (checked against the GitHub API on 2026-10-26); the cache dependency paths are `go.work.sum`, `apps/mcp-server/go.sum` and `tools/go.sum`. `setup-go` reads the `go` directive of a `go.work` file, so the version of the workflow is the one of the workspace.
- **L9** (2026-10-26): The verification of the slice, run locally in the worktree: `make gate` is green (fmt-check, vet, lint with 0 issues, build, test, race, the coverage floor at 85.0% over the 80% threshold, govulncheck with no vulnerabilities); `make db-up` builds the image of the PostgreSQL 18.6 stack with pgvector and starts the cluster healthy, and `make db-smoke` passes its 23 assertions (UTF-8, UTC, UUID, vector search, checksums, no published port, cross-container reachability); `make shellcheck` passes over the shell script of the stack. The workflow is parsed and its jobs and triggers checked.
- **L10** (2026-10-26): Issue #13 was updated to the current baseline (its body described the pre-reset repository) and the pull request #20 was opened with `Closes #13` and the `type:chore` label.
- **L11** (2026-10-29): The stale branch `origin/feat/ci-github-actions` was deleted from the origin. It held nothing that `main` does not hold: the pull request #20 was merged as the squash `2116666`, and the workflow of the repository is the one of that pull request. The seven commits of the deleted branch that were not in `main` carried the tree of the pre-reset repository — the `apps/cli` and `apps/mcp` trees that the blank-slate reset removed — and none of it was added back. The identifiers of the work units of this slice live in the head of the pull request, which the server keeps, and not in the history of `main`.
