#!/usr/bin/env bash
#
# The tests of the keeper of the extension.
#
# The keeper decides whether the pinned version of pgvector is behind the
# latest release. The decision is a pure function of the two versions it
# receives, so the tests prove it without a network: they run the script
# with a matrix of versions and assert what it prints and the exit code.
#
# Usage: scripts/tests/pgvector-release-test.sh

set -u

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
REPO_ROOT=$(dirname "$(dirname "$SCRIPT_DIR")")
KEEPER="$REPO_ROOT/scripts/ci/pgvector-release.sh"

# ---------------------------------------------------------------------------
# The state of the run
# ---------------------------------------------------------------------------

ASSERTIONS=0
FAILURES=0
STATUS=0
OUTPUT=''
ERROR=''

# ---------------------------------------------------------------------------
# The assertions
# ---------------------------------------------------------------------------

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

# Runs the keeper with the arguments and keeps its output and exit code.
run_keeper() {
  OUTPUT=$("$KEEPER" "$@" 2>/tmp/pgvector-release-test.err)
  STATUS=$?
  ERROR=$(cat /tmp/pgvector-release-test.err)
}

# ---------------------------------------------------------------------------
# The decision
# ---------------------------------------------------------------------------

printf '==> the decision\n'

# The pinned version is the latest one: nothing to do.
run_keeper 0.8.7 0.8.7
assert_equals 0 "$STATUS" "the same version is current"
assert_equals "current" "$OUTPUT" "the same version is current"

# A newer patch, minor or major release means the pin is behind.
run_keeper 0.8.7 0.8.8
assert_equals 0 "$STATUS" "a newer patch is behind"
assert_equals "behind" "$OUTPUT" "a newer patch is behind"

run_keeper 0.8.7 0.9.0
assert_equals 0 "$STATUS" "a newer minor is behind"
assert_equals "behind" "$OUTPUT" "a newer minor is behind"

run_keeper 0.8.7 1.0.0
assert_equals 0 "$STATUS" "a newer major is behind"
assert_equals "behind" "$OUTPUT" "a newer major is behind"

# A pin that is ahead of the latest release is not behind.
run_keeper 0.8.10 0.8.9
assert_equals 0 "$STATUS" "a newer pin is current"
assert_equals "current" "$OUTPUT" "a newer pin is current"

# The components are numbers, not text: 0.8.10 is newer than 0.8.9.
run_keeper 0.8.9 0.8.10
assert_equals 0 "$STATUS" "the components are compared as numbers"
assert_equals "behind" "$OUTPUT" "the components are compared as numbers"

# The tag of the release carries a `v`; the pin does not. Both are versions.
run_keeper v0.8.7 v0.8.8
assert_equals 0 "$STATUS" "the v prefix is accepted"
assert_equals "behind" "$OUTPUT" "the v prefix is accepted"

# A version with fewer components is padded with zeros.
run_keeper 0.8 0.8.0
assert_equals 0 "$STATUS" "a shorter version is padded"
assert_equals "current" "$OUTPUT" "a shorter version is padded"

run_keeper 0.8 0.8.1
assert_equals 0 "$STATUS" "a shorter version is padded before the comparison"
assert_equals "behind" "$OUTPUT" "a shorter version is padded before the comparison"

run_keeper 0.8.7 0.8.7.1
assert_equals 0 "$STATUS" "a longer version is padded"
assert_equals "behind" "$OUTPUT" "a longer version is padded"

# ---------------------------------------------------------------------------
# The arguments
# ---------------------------------------------------------------------------

printf '==> the arguments\n'

# A value that is not a version is refused, and the refusal is not a
# decision: the exit code says so.
run_keeper 0.8.x 0.8.8
assert_equals 2 "$STATUS" "a pinned value that is not a version is refused"
assert_contains "$ERROR" "not a version" "the refusal names the value"

run_keeper 0.8.7 latest
assert_equals 2 "$STATUS" "a latest value that is not a version is refused"
assert_contains "$ERROR" "not a version" "the refusal names the value"

run_keeper 0.8.7
assert_equals 2 "$STATUS" "a missing argument is refused"
assert_contains "$ERROR" "usage" "the refusal says how to call the keeper"

# ---------------------------------------------------------------------------
# The verdict
# ---------------------------------------------------------------------------

printf '%s assertions, %s failure(s)\n' "$ASSERTIONS" "$FAILURES"

if [ "$FAILURES" -ne 0 ]; then
  printf 'the keeper of the extension does not decide what the matrix describes.\n' >&2
  exit 1
fi

printf 'the keeper of the extension decides what the matrix describes.\n'
