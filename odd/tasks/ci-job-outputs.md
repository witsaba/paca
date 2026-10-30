# CI — the outputs of the job that detects the changes reach the jobs

The fix of a defect that made the filters of the continuous integration
silent: the job that detects the changes decided which jobs to run, and the
decision never reached them. Every job of the workflow was skipped, and the
run was green because nothing had failed.

## Specs

- **S1**: The claim the filters make, verbatim, from the document of the slice
  that built them: "a job runs **only** when the change can affect what it
  proves."

- **S2**: The rule of the repository that the defect broke, verbatim: "A pull
  request is green when every job is green, and a skipped job is green: it says
  that the change cannot affect what it proves."

- **S3**: The claim of the workflow, verbatim, from its own header: "A job that
  is skipped is a visible statement that its check was not needed: the workflow
  always reaches a conclusion."

- **S4**: The defect, as the code proves it: the job `changes` declares its
  outputs as `${{ steps.detect.outputs.<job> }}`, and the step that produces
  them carries no `id`. GitHub Actions reads the outputs of a step through its
  id, so every output of the job is empty, and every job that waits for one
  with `if: needs.changes.outputs.<job> == 'true'` is skipped forever.

- **S5**: The branch of the work and its worktree, per the practice of the
  repository: `fix/ci-job-outputs`, in the worktree
  `/Users/braejan/workspace/witsaba/repositories/paca-worktrees/ci-job-outputs`,
  created from `main` at `3ffd79f`.

## Tasks

| ID  | What | Links | Route | Commit |
|-----|------|-------|-------|--------|
| T1  | The feature document of the slice | S1, S2 | inline | `01fec58` (the document) |
| T2  | The workflow: the step that detects the changes carries the id `detect`, the name the job reads its outputs by | S1, S3, S4 | inline | `01fec58` (the fix) |
| T3  | The regression test: the test of the filters proves that every output the workflow reads is produced by a step that carries the id, and that the job declares an output for every job the rules name | S1, S2, S4 | inline | `01fec58` (the test) |
| T4  | The evidence: the test of the filters observes the defect before the fix and passes after it, and the whole gate of the repository is green | S1, S2 | inline | `01fec58` (the evidence is in L2) |

The slice was merged as a squash (`9bc9e50`), so the history of `main` holds one commit for the
whole feature. The identifiers of the work units are not in the history of `main`: they live in the
pull request #43, whose head the server keeps. The record of what happened is the pull request
itself.

## Log

- **L1** (2026-10-30): The defect, found while the record of the merged slice
  was being closed. The session that merged the job filters (the pull request
  #39, the squash `bc06263`) proved its engine locally and never read the
  result of the run it opened: the run of the pull request #41 said
  `the jobs of this run: shellcheck` in the log of the job `changes`, and the
  five jobs of the workflow were all `skipped`. The decision was right and the
  delivery of the decision was broken.

  The cause is one missing line. The job `changes` declares:

  ```yaml
  outputs:
    shellcheck: ${{ steps.detect.outputs.shellcheck }}
  ```

  and the step that produces those outputs carried no `id`. GitHub Actions
  reads what a step produces through the id of the step
  (`steps.<id>.outputs.<name>`), so a step without an id produces outputs that
  no job can read. The five outputs of the job were empty strings, the five
  conditions `if: needs.changes.outputs.<job> == 'true'` were false, and the
  five jobs were skipped. The run was green and nothing had run.

  The defect is worse than a slow run: it is a green that proves nothing. The
  repository had no proof that would have caught it, because the test of the
  filters reads the rules and the jobs of the workflow and never reads the
  outputs that carry the decision from one to the other.

- **L2** (2026-10-30): The evidence of the slice. The regression test observes
  the defect before the fix: against the workflow of `main` it fails with
  `the workflow reads the outputs of the step [detect], and a step of the
  workflow carries that id: [detect] is not one of the lines`, and it passes
  after the fix, with 39 assertions and 0 failures over the matrix of changes.
  The whole gate of the repository is green from nothing: `make shellcheck`
  passes over the seven scripts, `make env-test` keeps its 90 assertions,
  `make structure-test` keeps its 224 assertions and `make ci-filters-test`
  passes its 39. The change touches the workflow and the test of the filters,
  so the rules of `scripts/ci/job-filters.txt` decide that every job of the
  workflow runs in the pull request of this slice: the change of the workflow
  is the machinery of the filters.

- **L3** (2026-10-30): The slice was merged as the squash `9bc9e50`, and the
  pull request #43 is closed. Its run is the first run of the repository that
  proves the filters work: the five jobs of the workflow ran, and the run of
  the merge into `main` runs the whole gate.

  The defect of L1 was invisible in the conclusion of the runs, because a run
  whose jobs are all skipped is green. The conclusion of the jobs is what shows
  it, and it is the conclusion that the repository must read: a run is green
  when every job is green, and a job that is skipped is green because the
  change cannot affect what it proves — a skip that is not a decision is a
  proof of nothing.

  This record is the first change that proves the filters are worth their cost:
  it touches a document under `odd/tasks/` and nothing else, so the rules of
  `scripts/ci/job-filters.txt` decide that the job `shellcheck` runs and the
  other four are skipped.
