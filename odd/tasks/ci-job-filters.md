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
| T1  | The feature document, the rules of the filters (`scripts/ci/job-filters.txt`) and the engine that reads them (`scripts/ci/changed-jobs.sh`) | S1, S2, S3 | inline | (the work unit) |
| T2  | The differential test of the filters (`scripts/tests/ci-filters-test.sh`) and the target of the repository that runs it (`make ci-filters-test`) | S3, S4 | inline | (the work unit) |
| T3  | The workflow: the job `changes` computes the changed files, and each one of the five jobs runs only when it is needed | S2, S3, S4 | inline | (the work unit) |
| T4  | The documentation: the README describes the job `changes` and the rules of the filters | S3 | inline | (the work unit) |
| T5  | The evidence: the differential test over the matrix of changes, the shellcheck of the new scripts and the tests of the structure of the repository | S3, S4 | inline | (the evidence is in L2) |

## Log

- **L1** (2026-10-30): The plan that the last session left behind cannot be executed as it was written. GitHub Actions filters the files of a run **only at the level of the workflow**: the syntax of the workflow has `on.<event>.paths` and no `jobs.<job_id>.paths`, and the README of `dorny/paths-filter` says it in one sentence: "GitHub Actions built-in path filters don't allow this because they don't work on a level of individual jobs or steps." The shape of the slice is therefore the other one: a job `changes` that computes the files of the change and the rules of the repository decide which jobs need them. The rule of the filter is not a comment of the workflow: it lives in `scripts/ci/job-filters.txt`, the engine reads it, and the tests of the filters prove it against a matrix of changes.
- **L2** (2026-10-30): The evidence of the slice, run locally in the worktree: `make ci-filters-test` proves the matrix of changes (a Go source, a README, a shell script, a compose file, a document of `odd/tasks/`, the workflow itself) against the jobs it needs; `make shellcheck` passes over the two new scripts; `make structure-test` keeps its 223 assertions green. The cost that the slice removes is measured: a change of four documents under `odd/tasks/` ran five jobs and 3 m 42 s of runner time in the pull request #37, and it runs one job now.
