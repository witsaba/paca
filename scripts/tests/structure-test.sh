#!/usr/bin/env bash
#
# The tests of the structure of the monorepository.
#
# They assert the two promises that the split of the library relies on, and
# that nothing keeps them:
#
#   1. every module of the repository resolves its own graph on its own.
#      The workspace of the repository resolves the modules of the tree, but
#      a replace of a dependency is ignored: a module that depends on a
#      sibling needs its own replace, or it cannot resolve itself outside
#      the workspace.
#
#   2. every relative link of the documentation points at a file that
#      exists. A link that points at nothing is a lie the reader finds out
#      about only when the link is followed.
#
# The script has no dependency besides a POSIX shell, the Go toolchain and
# the usual system tools: there is no test framework to install.
#
# Usage: scripts/tests/structure-test.sh

set -u

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
REPO_ROOT=$(dirname "$(dirname "$SCRIPT_DIR")")

# ---------------------------------------------------------------------------
# The state of the run
# ---------------------------------------------------------------------------

ASSERTIONS=0
FAILURES=0

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

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"
  ASSERTIONS=$((ASSERTIONS + 1))
  case "$haystack" in
    *"$needle"*) fail "$message: [$needle] was found" ;;
    *) ;;
  esac
}

assert_file_exists() {
  local path="$1"
  local message="$2"
  ASSERTIONS=$((ASSERTIONS + 1))
  if [ ! -e "$path" ]; then
    fail "$message: [$path] does not exist"
  fi
}

# ---------------------------------------------------------------------------
# The modules
# ---------------------------------------------------------------------------

# The modules of the repository, one per line.
modules_of_the_repository() {
  find "$REPO_ROOT" -name go.mod -not -path '*/.git/*' -not -path '*/tools/*' |
    sed "s|^$REPO_ROOT/||; s|/go.mod$||" |
    sort
}

# test_every_module_resolves_its_own_graph asserts that every module
# resolves its graph without the workspace of the repository.
#
# The workspace hides the missing replace: it resolves the modules of the
# tree by itself. A tool that works on a module alone — `go list -m all`,
# `go get`, `go mod verify` — does not, and it fails on a module that
# depends on a sibling without carrying its own replace.
test_every_module_resolves_its_own_graph() {
  local module
  while IFS= read -r module; do
    local output status
    output=$(cd "$REPO_ROOT/$module" && GOWORK=off go list -m all 2>&1)
    status=$?
    # The command prints the graph of the module; the status is the answer:
    # it is not zero when the module cannot resolve one of its
    # dependencies without the workspace.
    assert_equals "0" "$status" "the module $module resolves its own graph"
    assert_not_contains "$output" "Could not read from remote repository" \
      "the module $module resolves its own graph without fetching a sibling"
  done < <(modules_of_the_repository)
}

# ---------------------------------------------------------------------------
# The documentation
# ---------------------------------------------------------------------------

# The documents of the repository that hold links.
documents_of_the_repository() {
  find "$REPO_ROOT" -name 'README.md' -not -path '*/.git/*' -not -path '*/node_modules/*' |
    sed "s|^$REPO_ROOT/||" |
    sort
}

# test_every_relative_link_points_at_a_file asserts that every relative
# link of the documentation points at something that exists.
test_every_relative_link_points_at_a_file() {
  local document
  while IFS= read -r document; do
    local directory
    directory=$(dirname "$REPO_ROOT/$document")

    # The links of a document: the target between the parentheses of a
    # markdown link, and the target of a reference to a file of the tree.
    local links
    links=$(grep -oE '\]\([^)]+\)' "$REPO_ROOT/$document" |
      sed 's/^](//; s/)$//' |
      grep -vE '^(https?:|mailto:|#)' || true)

    local link
    while IFS= read -r link; do
      [ -n "$link" ] || continue
      # The anchor of a link to a file of the tree is not part of the path.
      local target="${link%%#*}"
      [ -n "$target" ] || continue
      assert_file_exists "$directory/$target" \
        "the link [$link] of [$document] points at a file that exists"
    done <<EOF
$links
EOF
  done < <(documents_of_the_repository)
}

# ---------------------------------------------------------------------------
# The run
# ---------------------------------------------------------------------------

test_every_module_resolves_its_own_graph
test_every_relative_link_points_at_a_file

printf '%s assertion(s), %s failure(s)\n' "$ASSERTIONS" "$FAILURES"

if [ "$FAILURES" -ne 0 ]; then
  exit 1
fi

exit 0
