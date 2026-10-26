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
| T1  | The feature document and the root dispatching Makefile (help, gate, shellcheck, db-up, db-smoke, db-down, all) | S1, S3 | inline | `feat(build): the root Makefile that dispatches to the stacks of the monorepository` |
| T2  | The CI workflow `.github/workflows/ci.yml`: the gate job, the database job, the shellcheck job | S2, S4 | inline | `ci(workflows): the continuous integration of PACA — the gate, the database and the shellcheck` |
| T3  | The documentation: the root README describes the root Makefile and the CI | S2, S3 | inline | `docs(readme): the quality gate of the monorepository and its continuous integration` |
| T4  | The verification of the workflow: `make gate`, `make db-smoke` and `make shellcheck` run green locally | S2, S3, S4 | inline | (folded into T1-T3) |
| T5  | The pull request: issue #13 updated to the current baseline, PR opened with `Closes #13` and the `type:chore` label | S1, S2 | inline | (the PR is the deliverable) |

## Log

- **L1** (2026-10-26): "let work in a new git worktree, analyze the current database and app mcp added and add the CI for github."
- **L2** (2026-10-26): Scope, answered by the user: "Gate + DB + shellcheck" — the quality gate, the database smoke test and the shellcheck. No PR validation workflow (no `status:approved` label, no approval gate in the repository).
- **L3** (2026-10-26): Invocation, answered by the user: "Add a root dispatching Makefile" — the root dispatches, it holds no stack-specific logic.
- **L4** (2026-10-26): Container tests, answered by the user: "No, keep CI on unit level" — the testcontainers suite stays a local gate.
- **L5** (2026-10-26): Analysis of the baseline: `main` at `780f5c7` has no `.github`; `apps/mcp-server/Makefile` exposes `all` (fmt-check, vet, lint, build, test, race, cover, vuln) with an 80% coverage floor; `db/Makefile` is the umbrella of the databases and `db/postgres/Makefile` owns `up`/`smoke`; `db/postgres/scripts/smoke-test.sh` asserts the database guarantees and needs the stack running, `db/postgres/.env` and `nc` on the host; `go.work` pins Go 1.27.3 and `tools/go.mod` pins golangci-lint v2.15.0 and govulncheck v1.8.0.
- **L6** (2026-10-26): The branch `origin/feat/ci-github-actions` is stale: it forks from `bd7a378` (before the blank-slate reset) and still carries the deleted finance-manager tree. It is a style reference only; the workflow is written from scratch on top of the current `main`.
- **L7** (2026-10-26): Issue #13 tracks the CI, but its body describes the pre-reset baseline (five modules, the sqlite schema verifier, a 75% per-file coverage floor). It is updated to the current baseline before the pull request closes it.
- **L8** (2026-10-26): `actions/checkout@v7` and `actions/setup-go@v7` exist (checked against the GitHub API on 2026-10-26); the cache dependency paths are `go.work.sum`, `apps/mcp-server/go.sum` and `tools/go.sum`.
