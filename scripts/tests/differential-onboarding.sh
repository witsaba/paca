#!/usr/bin/env bash
#
# The differential check of the slice of the onboarding.
#
# The change touches the adapter of the people, and the four tools of the
# people that the request did not name use it. The check runs the container
# tests of the library and of the service against the previous code and
# against the change, and prints the two results side by side.
#
# The previous code is a copy of the base commit in a scratch directory: it
# is never committed, and it never touches the branch.
#
# Usage: scripts/tests/differential-onboarding.sh <base-commit>

set -u

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

run_suite() {
  local label="$1"
  local root="$2"

  printf '=== %s\n' "${label}"

  (cd "${root}/packages/database" && go test -count=1 -tags=integration -timeout=15m ./test/... 2>&1 |
    grep -E '^(ok|FAIL|---)' | head -5)
  (cd "${root}/apps/mcp-server" && go test -count=1 -tags=integration -timeout=15m ./test/... 2>&1 |
    grep -E '^(ok|FAIL|---)' | head -6)

  printf '\n'
}

run_suite "the previous code" "${SCRATCH}"
run_suite "the change" "${REPO_ROOT}"
