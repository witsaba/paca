#!/usr/bin/env bash
#
# The differential check of the slice of the onboarding.
#
# The change touches the adapter of the people, and the four tools of the
# people that the request did not name use it. The check runs the tests of
# the library and of the service against the previous code and against the
# change, and prints the two results side by side.
#
# Every module is run twice: the unit tests, which hold the tests of the
# adapters and of the tools, and the container tests, which hold the tests
# of the tables.
#
# The check is a gate, not a report: it fails when a suite fails, and it
# fails when a package that passed against the previous code no longer
# passes against the change. A difference nobody asked for is a regression.
#
# The previous code is a copy of the base commit in a scratch directory: it
# is never committed, and it never touches the branch.
#
# Usage: scripts/tests/differential-onboarding.sh <base-commit>

set -euo pipefail

BASE="${1:-}"
if [ -z "${BASE}" ]; then
  printf 'the differential check needs the base commit\n' >&2
  exit 1
fi

REPO_ROOT=$(CDPATH='' cd -- "$(dirname -- "$0")/../.." && pwd -P)
SCRATCH=$(mktemp -d)
trap 'rm -rf "${SCRATCH}"' EXIT

printf 'the differential check of %s against the working tree\n' "${BASE}"
printf '\n'

git -C "${REPO_ROOT}" archive "${BASE}" | tar -x -C "${SCRATCH}"

# The modules whose tests the check runs.
MODULES=(packages/database apps/mcp-server)

# The summary of a suite: one line per package, the name of the package and
# whether it passed. It is what the two runs are compared on.
run_suite() {
  local root="$1"
  local report="$2"
  local module="$3"
  local selector="$4"
  local tag="$5"
  local label="$6"

  local output
  local failed=0
  output="${report}/$(printf '%s' "${module}-${label}" | tr '/' '-').log"

  if [ "${tag}" = "integration" ]; then
    if ! (cd "${root}/${module}" && go test -count=1 -tags=integration -timeout=15m "${selector}" >"${output}" 2>&1); then
      failed=1
    fi
  else
    if ! (cd "${root}/${module}" && go test -count=1 -timeout=15m "${selector}" >"${output}" 2>&1); then
      failed=1
    fi
  fi

  grep -E '^(ok|FAIL)[[:space:]]' "${output}" | head -8 || true
  awk '/^(ok|FAIL)[[:space:]]/ { print $1, $2 }' "${output}" >>"${report}/summary"
  printf '%s' "${failed}" >>"${report}/failed"
}

OLD_REPORT="${SCRATCH}/old"
NEW_REPORT="${SCRATCH}/new"
mkdir -p "${OLD_REPORT}" "${NEW_REPORT}"
: >"${OLD_REPORT}/summary"
: >"${NEW_REPORT}/summary"
: >"${OLD_REPORT}/failed"
: >"${NEW_REPORT}/failed"

run_suites() {
  local root="$1"
  local report="$2"
  local label="$3"

  for module in "${MODULES[@]}"; do
    printf '=== %s, the unit tests of %s\n' "${label}" "${module}"
    run_suite "${root}" "${report}" "${module}" './...' 'unit' "${label}-unit"
    printf '\n'
    printf '=== %s, the container tests of %s\n' "${label}" "${module}"
    run_suite "${root}" "${report}" "${module}" './test/...' 'integration' "${label}-integration"
    printf '\n'
  done
}

run_suites "${SCRATCH}" "${OLD_REPORT}" 'the previous code'
run_suites "${REPO_ROOT}" "${NEW_REPORT}" 'the change'

# The gate: the change must pass every suite, and every package that passed
# against the previous code must still pass.
status=0

if grep -q '1' "${NEW_REPORT}/failed"; then
  printf 'a suite of the change failed\n' >&2
  status=1
fi

if grep -q '^FAIL ' "${NEW_REPORT}/summary"; then
  printf 'a package of the change failed\n' >&2
  status=1
fi

while read -r outcome package; do
  [ "${outcome}" = "ok" ] || continue
  if ! grep -qx "ok ${package}" "${NEW_REPORT}/summary"; then
    printf 'the package %s passed against the previous code and does not pass against the change\n' "${package}" >&2
    status=1
  fi
done <"${OLD_REPORT}/summary"

if [ "${status}" -ne 0 ]; then
  printf 'the differential check failed\n' >&2
  exit "${status}"
fi

printf 'the differential check passed: every package that passed against the previous code still passes against the change\n'
