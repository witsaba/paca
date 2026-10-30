#!/usr/bin/env bash
#
# The differential test of the job filters of the continuous integration.
#
# The claim the filters make is precise: a job runs only when the change can
# affect what it proves. The test proves it the only way it can be proved:
# a matrix of changes, and the jobs each one needs.
#
# The matrix covers the files of the repository that mean different things:
# a source of Go, a module, a documentation, a shell script, a compose file,
# a manifest of the environment, a document of the specifications and the
# machinery of the filters itself.
#
# The test reads the rules of scripts/ci/job-filters.txt and the workflow of
# .github/workflows/ci.yml, so a job that has no rule, or a rule that names
# a job the workflow does not have, fails the test.
#
# Usage: scripts/tests/ci-filters-test.sh

set -u

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
REPO_ROOT=$(dirname "$(dirname "$SCRIPT_DIR")")

FILTERS="$REPO_ROOT/scripts/ci/job-filters.txt"
ENGINE="$REPO_ROOT/scripts/ci/changed-jobs.sh"
WORKFLOW="$REPO_ROOT/.github/workflows/ci.yml"

# ---------------------------------------------------------------------------
# The state of the run
# ---------------------------------------------------------------------------

ASSERTIONS=0
FAILURES=0

fail() {
  FAILURES=$((FAILURES + 1))
  printf 'not ok - %s\n' "$1" >&2
}

assert_equals() {
  local expected="$1"
  local actual="$2"
  local message="$3"
  ASSERTIONS=$((ASSERTIONS + 1))
  if [ "$expected" != "$actual" ]; then
    fail "$message: expected [$expected], got [$actual]"
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"
  ASSERTIONS=$((ASSERTIONS + 1))
  case "$haystack" in
    *"$needle"*) ;;
    *) fail "$message: [$needle] was not found" ;;
  esac
}

# A line of a list, exactly: a needle that is a part of another line is
# not the line.
assert_has_line() {
  local haystack="$1"
  local needle="$2"
  local message="$3"
  ASSERTIONS=$((ASSERTIONS + 1))
  if ! printf '%s\n' "$haystack" | grep -qx -- "$needle"; then
    fail "$message: [$needle] is not one of the lines"
  fi
}

# ---------------------------------------------------------------------------
# The matrix of the changes
# ---------------------------------------------------------------------------

# Every row is a change of the repository and the jobs it needs: the first
# field is the file that changed, the second one the jobs that must run.
#
# The last row is a change of many files at once: the jobs of a change are
# the union of the jobs of its files.
MATRIX='packages/domain/user/user.go|gate,stack
apps/db-admin/main.go|gate,stack
apps/mcp-server/Dockerfile|gate,shellcheck,stack
apps/mcp-server/compose.yaml|gate,stack
packages/platform/telemetry/telemetry.go|gate,observability,stack
packages/database/go.mod|gate,shellcheck,stack
packages/domain/README.md|gate,shellcheck,stack
packages/domain/AGENTS.md|gate,shellcheck,stack
go.work|gate
go.work.sum|gate
tools/go.mod|gate,shellcheck
tools/go.sum|gate,shellcheck
.golangci.yml|gate
.dockerignore|gate,stack
README.md|shellcheck
AGENTS.md|shellcheck
odd/tasks/ci-job-filters.md|shellcheck
scripts/setup-env.sh|database,shellcheck,stack
scripts/stack-schema.sh|shellcheck,stack
scripts/tests/structure-test.sh|shellcheck
env.example|database,shellcheck,stack
db/postgres/env.example|database,shellcheck,stack
db/postgres/initdb/01-init.sql|database,stack
db/postgres/compose.yaml|database,stack
db/postgres/Dockerfile|database,shellcheck,stack
compose.yaml|database,stack
observability/compose.yaml|observability,stack
.github/workflows/ci.yml|database,gate,observability,shellcheck,stack
scripts/ci/job-filters.txt|database,gate,observability,shellcheck,stack
scripts/ci/changed-jobs.sh|database,gate,observability,shellcheck,stack
Makefile|database,gate,observability,shellcheck,stack
packages/domain/user/user.go;odd/tasks/ci-job-filters.md|gate,shellcheck,stack'

# ---------------------------------------------------------------------------
# The engine
# ---------------------------------------------------------------------------

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

# The jobs the engine decides for a list of files, sorted and joined by a
# comma.
jobs_for() {
  local files="$1"
  local output
  output=$("$ENGINE" --filters "$FILTERS" --files "$files" 2>/dev/null)
  printf '%s\n' "$output" | awk -F= '$2 == "true" { print $1 }' | LC_ALL=C sort | paste -sd, -
}

printf '==> the matrix of the changes\n'
while IFS='|' read -r files expected; do
  [ -n "$files" ] || continue
  # A row may carry many files, separated by a semicolon: the jobs of a
  # change are the union of the jobs of its files.
  printf '%s\n' "$files" | tr ';' '\n' > "$WORK_DIR/changed"
  assert_equals "$expected" "$(jobs_for "$WORK_DIR/changed")" \
    "the change of $files needs the jobs [$expected]"
done <<EOF
$MATRIX
EOF

# ---------------------------------------------------------------------------
# The rules and the workflow agree
# ---------------------------------------------------------------------------

printf '==> the rules and the workflow agree\n'

# The jobs of the workflow: every job of the file, except the one that
# detects the changes, which always runs.
workflow_jobs=$(awk '
  /^jobs:/ { in_jobs = 1; next }
  in_jobs && /^[^[:space:]]/ { in_jobs = 0 }
  in_jobs && /^  [a-zA-Z0-9_-]+:/ {
    job = $1
    sub(/:$/, "", job)
    print job
  }
' "$WORKFLOW" | grep -v '^changes$' | LC_ALL=C sort | paste -sd, -)

# The jobs of the rules: every job the rules name, except the machinery,
# which is the rule of every job.
rules_jobs=$(awk '
  /^[[:space:]]*#/ || /^[[:space:]]*$/ { next }
  { print $1 }
' "$FILTERS" | grep -v '^all$' | LC_ALL=C sort -u | paste -sd, -)

assert_equals "$workflow_jobs" "$rules_jobs" \
  "the rules of the filters name the jobs of the workflow, and no other"

# The machinery of the filters: a change of the workflow itself, of the
# scripts of the continuous integration or of the dispatcher of the targets
# changes what every job proves, so every job runs.
printf '%s\n' '.github/workflows/ci.yml' > "$WORK_DIR/changed"
assert_equals "database,gate,observability,shellcheck,stack" "$(jobs_for "$WORK_DIR/changed")" \
  "a change of the workflow runs every job"

# ---------------------------------------------------------------------------
# The outputs reach the jobs
# ---------------------------------------------------------------------------

printf '==> the outputs reach the jobs\n'

# The ids of the steps of the workflow. A step without an id cannot be read
# by the job that needs what it produces.
workflow_step_ids=$(grep -oE '^[[:space:]]+id:[[:space:]]*[A-Za-z0-9_-]+[[:space:]]*$' "$WORKFLOW" |
  sed -E 's/^[[:space:]]+id:[[:space:]]*//' | LC_ALL=C sort -u)

# The steps the workflow reads the outputs of: `steps.<id>.outputs.<name>`.
workflow_output_steps=$(grep -oE 'steps\.[A-Za-z0-9_-]+\.outputs\.' "$WORKFLOW" |
  sed -E 's/^steps\.([A-Za-z0-9_-]+)\.outputs\.$/\1/' | LC_ALL=C sort -u)

for step in $workflow_output_steps; do
  assert_has_line "$workflow_step_ids" "$step" \
    "the workflow reads the outputs of the step [$step], and a step of the workflow carries that id"
done

# The outputs the job that detects the changes declares.
declared_outputs=$(awk '
  /^    outputs:[[:space:]]*$/ { in_outputs = 1; next }
  in_outputs && /^      [A-Za-z0-9_-]+:/ {
    key = $1
    sub(/:$/, "", key)
    print key
    next
  }
  in_outputs { in_outputs = 0 }
' "$WORKFLOW" | LC_ALL=C sort -u | paste -sd, -)

# The outputs the jobs of the workflow wait for.
consumed_outputs=$(grep -oE 'needs\.changes\.outputs\.[A-Za-z0-9_-]+' "$WORKFLOW" |
  sed -E 's/^needs\.changes\.outputs\.//' | LC_ALL=C sort -u | paste -sd, -)

assert_equals "$declared_outputs" "$consumed_outputs" \
  "the job that detects the changes declares the outputs the jobs of the workflow wait for"

# The jobs the rules name are the jobs the workflow decides: an output that
# the workflow does not declare is an output that never reaches its job, and
# the job that waits for it is skipped forever.
assert_equals "$rules_jobs" "$declared_outputs" \
  "the job that detects the changes declares an output for every job the rules name"

# ---------------------------------------------------------------------------
# The safe default
# ---------------------------------------------------------------------------

printf '==> the safe default\n'

# When the change cannot be read, no proof is lost: every job runs.
missing_base=$("$ENGINE" --filters "$FILTERS" --base 0000000000000000000000000000000000000000 \
  --head HEAD 2>/dev/null | awk -F= '$2 == "true" { print $1 }' | LC_ALL=C sort | paste -sd, -)
assert_equals "database,gate,observability,shellcheck,stack" "$missing_base" \
  "a change that cannot be read runs every job"

# A manual run carries no change: every job runs.
manual=$("$ENGINE" --filters "$FILTERS" --event workflow_dispatch 2>/dev/null |
  awk -F= '$2 == "true" { print $1 }' | LC_ALL=C sort | paste -sd, -)
assert_equals "database,gate,observability,shellcheck,stack" "$manual" \
  "a manual run runs every job"

# ---------------------------------------------------------------------------
# The verdict
# ---------------------------------------------------------------------------

printf '%s assertions, %s failure(s)\n' "$ASSERTIONS" "$FAILURES"

if [ "$FAILURES" -ne 0 ]; then
  printf 'the filters of the continuous integration are not the ones the matrix describes.\n' >&2
  exit 1
fi

printf 'the filters of the continuous integration are the ones the matrix describes.\n'
