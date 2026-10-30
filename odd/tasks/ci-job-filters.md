# CI — the job filters of the continuous integration

The slice of the continuous integration that runs only the checks a change
can affect: the five jobs of the workflow stay the five checks they are,
and each one runs only when the files of the change can change what it
proves.

## Specs

- **S1**: The request of the maintainer, verbatim: "check the latest observations about CI, ensure the main is updated before starting. Open a new git worktree for add the CI properly in order to only run the modifyed pipelines."

- **S2**: The objection that opened the slice, verbatim: "The problem I saw is not needed to ran any CI due to no software was change. The next session must to be fix the CI in order to only run the proper CI that was modified or even avoid running if not needed."

- **S3**: The claim to prove, verbatim: "a job runs **only** when the change can affect what it proves."

- **S4**: The rule of the repository that must keep holding, verbatim: "A pull request is green when every job is green."

- **S5**: The branch of the work and its worktree, per the practice of the repository: `ci/job-filters`, in the worktree `/Users/braejan/workspace/witsaba/repositories/paca-worktrees/ci-job-filters`, created from `main` at `2ae0593`.

## Tasks

| ID  | What | Links | Route | Commit |
|-----|------|-------|-------|--------|
| T1  | The feature document, the rules of the filters (`scripts/ci/job-filters.txt`) and the engine that reads them (`scripts/ci/changed-jobs.sh`) | S1, S2, S3 | inline | `1092a38` (the document), `e2f82db` (the rules and the engine) |
| T2  | The differential test of the filters (`scripts/tests/ci-filters-test.sh`) and the target of the repository that runs it (`make ci-filters-test`) | S3, S4 | inline | `81f5b23` |
| T3  | The workflow: the job `changes` computes the changed files, and each one of the five jobs runs only when it is needed | S2, S3, S4 | inline | `cf55f68` |
| T4  | The documentation: the README describes the job `changes` and the rules of the filters | S3 | inline | `d48e320` |
| T5  | The evidence: the differential test over the matrix of changes, the shellcheck of the new scripts and the tests of the structure of the repository | S3, S4 | inline | `5536cfa` (the evidence is in L2) |

The slice was merged as a squash (`bc06263`), so the history of `main` holds one commit for the
whole feature. The identifiers of the work units are not in the history of `main`: they live in the
pull request #39, whose head the server keeps. The record of what happened is the pull request
itself.

## Log

- **L1** (2026-10-30): The plan that the last session left behind cannot be executed as it was written. GitHub Actions filters the files of a run **only at the level of the workflow**: the syntax of the workflow has `on.<event>.paths` and no `jobs.<job_id>.paths`, and the README of `dorny/paths-filter` says it in one sentence: "GitHub Actions built-in path filters don't allow this because they don't work on a level of individual jobs or steps." The shape of the slice is therefore the other one: a job `changes` that computes the files of the change and the rules of the repository decide which jobs need them. The rule of the filter is not a comment of the workflow: it lives in `scripts/ci/job-filters.txt`, the engine reads it, and the tests of the filters prove it against a matrix of changes.
- **L2** (2026-10-30): The evidence of the slice, run locally in the worktree: `make ci-filters-test` proves the matrix of changes (a source of Go, a module, a documentation, a shell script, a compose file, a manifest of the environment, a document of `odd/tasks/`, the workflow itself) against the jobs it needs — 36 assertions, 0 failures; `make shellcheck` passes over the seven scripts of the repository; `make structure-test` keeps its 224 assertions green; the gate of the repository (`make all`) passes from nothing and leaves no container behind. The engine was also run against the real difference of the branch (`main..HEAD`) and it decides that every job runs: the change of the workflow is the machinery of the filters. The cost that the slice removes is measured: a change of four documents under `odd/tasks/` ran five jobs and 3 m 42 s of runner time in the pull request #37, and it runs one job now.
- **L3** (2026-10-30): The shape of the slice and the deviation it carries. The slice is five work units and 695 authored lines; every work unit is under the 400-line budget, the feature as a whole is over it, and it lands as one pull request with the work units as the review path — the same deviation the previous big slices recorded. The rules are deliberately coarse: a path that is missing from the rules skips a proof that the change needed, and a path that is too wide only costs a run that was not needed. The rules are the ones of the repository, in `scripts/ci/job-filters.txt`, and the tests of the filters read them, so a rule is a change of the code and never a comment of the workflow.
- **L4** (2026-10-30): The gate of the worktree failed on its first run, and the cause was not the slice: the named volume `paca-postgres-data` is shared by every worktree of the machine, and the cluster it holds was initialized with the credentials of the `.env` of the main worktree (`POSTGRES_DB=paca-pg`). The installer of the worktree wrote its own defaults (`POSTGRES_DB=paca`), so the smoke test found no database. The `.env` files of the worktree were aligned with the ones of the main worktree, and the gate passed. The lesson is for the next worktree: the credentials of a stack live with the stack, and a new worktree inherits the ones of the machine before it starts a container.
- **L5** (2026-10-30): The slice was merged as the squash `bc06263` on 2026-10-29, and the pull request #39 is closed. A squash merge leaves the history of `main` with one commit for a whole feature, so the identifiers of the work units of the table above are not in `main`: they live in the head of the pull request #39, which the server keeps, and the branch `ci/job-filters` and its worktree are gone. The work unit of the evidence, `5536cfa`, is the one that closed this table. The workflow of the merge runs the whole gate on purpose: the change of the workflow is the machinery of the filters, and a change of it proves every job. This record is the first change that proves the filters are worth their cost: it touches a document under `odd/tasks/` and nothing else, so the rules of `scripts/ci/job-filters.txt` decide that the job `shellcheck` runs and the other four are skipped.
