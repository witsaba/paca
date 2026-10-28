#!/usr/bin/env bash
#
# The tests of the structure of the monorepository.
#
# They assert the promises the architecture of the repository makes, and that
# nothing keeps them:
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
#   3. the layers of every application respect the direction of the
#      dependencies: the domain knows nothing but the standard library, the
#      application knows no adapter, and the adapters know no use case.
#
#   4. the shared domain module knows nothing but the standard library and
#      itself: it is the boundary that makes "the domain knows nothing about
#      the database" a compile error instead of a comment.
#
# The script has no dependency besides a POSIX shell, the Go toolchain and the
# usual system tools: there is no test framework to install.
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

    # The graph is not enough: a module that imports a sibling without
    # naming it in its own go.mod resolves the graph and still refuses to
    # build outside the workspace. The dependencies are resolved here.
    local dependencies
    dependencies=$(cd "$REPO_ROOT/$module" && GOWORK=off go list -deps ./... 2>&1 >/dev/null)
    status=$?
    assert_equals "0" "$status" \
      "the module $module resolves its own dependencies without the workspace"
    assert_not_contains "$dependencies" "updates to go.mod needed" \
      "the module $module names every dependency it imports"
  done < <(modules_of_the_repository)
}

# ---------------------------------------------------------------------------
# The layers
# ---------------------------------------------------------------------------

# The applications of the repository, one per line.
applications_of_the_repository() {
  find "$REPO_ROOT/apps" -mindepth 1 -maxdepth 1 -type d -exec test -f '{}/go.mod' \; -print |
    sed "s|^$REPO_ROOT/||" |
    sort
}

# is_external tells whether an import path leaves the standard library.
#
# The standard library has no dot in the name of its first element; every
# other path — a module, a third-party package — has one.
is_external() {
  case "$1" in
    *.*/*) return 0 ;;
    *) return 1 ;;
  esac
}

# The packages of the standard library that a domain must never name.
#
# They are standard, but they belong to the technology: a model that reaches
# for a driver or for a protocol is a model the technology decides.
FORBIDDEN_IN_THE_DOMAIN="database/sql net/http"

# imports_of_the_layer prints the imports of every package of a layer, one
# package per line: the path of the package, then its imports.
imports_of_the_layer() {
  local module="$1"
  local layer="$2"

  (cd "$REPO_ROOT/$module" && go list -f '{{.ImportPath}} {{join .Imports " "}}' "./$layer/..." 2>/dev/null)
}

# test_the_domain_of_every_application_knows_nothing asserts that the
# domain layer of every application imports nothing but the standard library
# and the domain itself.
#
# The rule of the architecture is that the domain knows nothing about the
# technology. A domain that imports a driver, a table or a protocol is a
# domain that the technology decides.
test_the_domain_of_every_application_knows_nothing() {
  local application
  while IFS= read -r application; do
    local module
    module=$(basename "$application")

    local line
    while IFS= read -r line; do
      [ -n "$line" ] || continue

      local package
      package=${line%% *}

      local imports
      imports=${line#* }

      local imported
      for imported in $imports; do
        is_external "$imported" || continue

        # The domain of an application may name its own domain and the
        # shared domain — the models every service shares. It names nothing
        # else.
        case "$imported" in
          github.com/witsaba/paca/apps/"$module"/internal/domain*) ;;
          github.com/witsaba/paca/packages/domain*) ;;
          *)
            fail "the domain of $module imports nothing but the domain: [$imported]"
            ;;
        esac
        ASSERTIONS=$((ASSERTIONS + 1))
      done

      # The standard library is allowed, the technology inside it is not.
      for imported in $imports; do
        local forbidden
        for forbidden in $FORBIDDEN_IN_THE_DOMAIN; do
          assert_not_contains "$imported" "$forbidden" \
            "the domain of $module names no technology"
        done
      done
    done < <(imports_of_the_layer "$application" internal/domain)
  done < <(applications_of_the_repository)
}

# test_the_application_of_every_application_knows_no_adapter asserts that the
# application layer imports no adapter and no infrastructure.
#
# The use cases speak in the model of the domain and in the ports. The day one
# of them names a concrete technology, the technology decides what the use
# case does.
test_the_application_of_every_application_knows_no_adapter() {
  local application
  while IFS= read -r application; do
    local module
    module=$(basename "$application")

    local line
    while IFS= read -r line; do
      [ -n "$line" ] || continue

      local imports
      imports=${line#* }

      local imported
      for imported in $imports; do
        assert_not_contains "$imported" "/internal/adapters/" \
          "the application of $module imports no adapter: [$imported]"
        assert_not_contains "$imported" "/internal/platform/" \
          "the application of $module imports no infrastructure: [$imported]"
      done
    done < <(imports_of_the_layer "$application" internal/application)
  done < <(applications_of_the_repository)
}

# test_the_adapters_of_every_application_know_no_use_case asserts that the
# adapters import the ports and the domain, and never the use cases.
#
# An adapter that knows the implementation of the use case is an adapter that
# cannot be replaced.
test_the_adapters_of_every_application_know_no_use_case() {
  local application
  while IFS= read -r application; do
    local module
    module=$(basename "$application")

    local line
    while IFS= read -r line; do
      [ -n "$line" ] || continue

      local imports
      imports=${line#* }

      local imported
      for imported in $imports; do
        assert_not_contains "$imported" "/internal/application/" \
          "the adapters of $module know no use case: [$imported]"
      done
    done < <(imports_of_the_layer "$application" internal/adapters)
  done < <(applications_of_the_repository)
}

# test_the_shared_domain_knows_nothing_but_the_standard_library asserts that
# the shared domain module imports nothing outside itself.
#
# The module is a boundary: a model that reached for a driver, for a table or
# for a connection string would break the build of every service of the
# monorepository.
test_the_shared_domain_knows_nothing_but_the_standard_library() {
  local line
  while IFS= read -r line; do
    [ -n "$line" ] || continue

    local imports
    imports=${line#* }

    local imported
    for imported in $imports; do
      is_external "$imported" || continue

      case "$imported" in
        github.com/witsaba/paca/packages/domain*) ;;
        *)
          fail "the shared domain imports nothing but itself: [$imported]"
          ;;
      esac
      ASSERTIONS=$((ASSERTIONS + 1))
    done

    # The standard library is allowed, the technology inside it is not.
    for imported in $imports; do
      local forbidden
      for forbidden in $FORBIDDEN_IN_THE_DOMAIN; do
        assert_not_contains "$imported" "$forbidden" \
          "the shared domain names no technology"
      done
    done
  done < <(imports_of_the_layer packages/domain .)
}

# test_only_the_composition_root_knows_the_adapters asserts that the
# composition root is the only place that knows every layer at once.
#
# An adapter that another layer names is an adapter that the layer decides.
test_only_the_composition_root_knows_the_adapters() {
  local application
  while IFS= read -r application; do
    local module
    module=$(basename "$application")

    local line
    while IFS= read -r line; do
      [ -n "$line" ] || continue

      local package
      package=${line%% *}

      case "$package" in
        */internal/bootstrap|*/internal/bootstrap/*) continue ;;
        */internal/adapters|*/internal/adapters/*) continue ;;
      esac

      local imports
      imports=${line#* }

      local imported
      for imported in $imports; do
        assert_not_contains "$imported" "/internal/adapters/" \
          "only the composition root of $module knows the adapters: [$imported]"
      done
    done < <(imports_of_the_layer "$application" internal)
  done < <(applications_of_the_repository)
}

# test_the_shared_platform_knows_no_model asserts that the shared platform
# module imports nothing of the monorepository but itself.
#
# The platform is the infrastructure: it answers questions about the machine.
# The day it imports a model, the infrastructure decides what the model means.
test_the_shared_platform_knows_no_model() {
  local line
  while IFS= read -r line; do
    [ -n "$line" ] || continue

    local imports
    imports=${line#* }

    local imported
    for imported in $imports; do
      case "$imported" in
        github.com/witsaba/paca/*)
          case "$imported" in
            github.com/witsaba/paca/packages/platform*) ;;
            *)
              fail "the shared platform imports nothing of the monorepository but itself: [$imported]"
              ;;
          esac
          ;;
      esac
      ASSERTIONS=$((ASSERTIONS + 1))
    done
  done < <(imports_of_the_layer packages/platform .)
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

# test_every_image_copies_the_modules_it_imports asserts that the image of
# every service copies every module of the monorepository the service
# imports.
#
# The context of the image is the root of the repository, and a module the
# image does not copy is a module the build cannot resolve: the service
# fails to build inside the container, and the failure is a line of a log
# far from the change that caused it.
test_every_image_copies_the_modules_it_imports() {
  local application
  while IFS= read -r application; do
    local module
    module=$(basename "$application")

    local dockerfile="$REPO_ROOT/$application/Dockerfile"
    [ -f "$dockerfile" ] || continue

    local imported
    imported=$(grep -oE 'github.com/witsaba/paca/packages/[a-z-]+' "$REPO_ROOT/$application/go.mod" |
      sort -u)

    local library
    while IFS= read -r library; do
      [ -n "$library" ] || continue

      local path
      path=${library#github.com/witsaba/paca/}

      # The sources of the module, not only its go.mod: the image builds
      # the service from them.
      assert_contains "$(cat "$dockerfile")" "COPY $path/ ./$path/" \
        "the image of $module copies the sources of $path: the service imports it"
    done <<EOF
$imported
EOF
  done < <(applications_of_the_repository)
}

# ---------------------------------------------------------------------------
# The run
# ---------------------------------------------------------------------------

test_every_module_resolves_its_own_graph
test_every_relative_link_points_at_a_file
test_the_domain_of_every_application_knows_nothing
test_the_application_of_every_application_knows_no_adapter
test_the_adapters_of_every_application_know_no_use_case
test_only_the_composition_root_knows_the_adapters
test_the_shared_domain_knows_nothing_but_the_standard_library
test_the_shared_platform_knows_no_model
test_every_image_copies_the_modules_it_imports

printf '%s assertion(s), %s failure(s)\n' "$ASSERTIONS" "$FAILURES"

if [ "$FAILURES" -ne 0 ]; then
  exit 1
fi

exit 0
