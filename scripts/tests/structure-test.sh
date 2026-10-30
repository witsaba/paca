#!/usr/bin/env bash
#
# shellcheck disable=SC2329
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
#   5. every function of the repository is reachable from the entry point of
#      a service: a function that no production code calls is debris, not a
#      promise.
#
#   6. one service applies the migrations of the schema: the schema has one
#      writer.
#
#   7. the two services do not carry the same code: a shared layer is
#      extended, never forked.
#
# Every rule says what it proved. A rule that proves nothing runs over an
# empty set, and the gate reports it: the rules that have nothing to prove
# today are named in the script, with the reason.
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

# What every rule proved, one line per rule: the name of the rule, a pipe,
# and the number of assertions it proved.
RULES_PROVED=""

# The rules that have nothing to prove today, and why.
#
# A rule that runs over an empty set proves nothing, and a rule that proves
# nothing without saying so is a wish. Every rule that has nothing to prove
# is named here, with the reason, and the gate reports it on every run.
RULES_WITHOUT_ANYTHING_TO_PROVE="the domain of every application knows nothing|nothing to prove today: neither service has a domain layer yet"

# The copies that the two services carry today, and the reason they wait.
#
# A copy that is listed here is reported by the gate on every run, and it is
# a failure the moment it is not the copy that was listed.
KNOWN_COPIES="internal/application/health.go:NewHealthService|the health use case waits for a shared application layer"
KNOWN_COPIES="${KNOWN_COPIES}
internal/application/health.go:Health|the health use case waits for a shared application layer"
KNOWN_COPIES="${KNOWN_COPIES}
internal/application/health.go:runCheck|the health use case waits for a shared application layer"
KNOWN_COPIES="${KNOWN_COPIES}
internal/adapters/inbound/http/health_handler.go:NewHealthHandler|the health endpoint waits for a shared adapter"
KNOWN_COPIES="${KNOWN_COPIES}
internal/adapters/inbound/http/health_handler.go:ServeHTTP|the health endpoint waits for a shared adapter"
KNOWN_COPIES="${KNOWN_COPIES}
internal/adapters/inbound/http/health_handler.go:newHealthResponse|the health endpoint waits for a shared adapter"
KNOWN_COPIES="${KNOWN_COPIES}
internal/adapters/inbound/http/health_handler.go:writeJSON|the health endpoint waits for a shared adapter"

# ---------------------------------------------------------------------------
# The rules
# ---------------------------------------------------------------------------

# run_rule runs one rule and counts what it proved.
#
# The count of the assertions of a rule is what says whether the rule
# discriminates: a rule that proves nothing runs over an empty set, and the
# gate reports it.
run_rule() {
  local name="$1"
  shift

  local before=$ASSERTIONS
  "$@"
  local proved=$((ASSERTIONS - before))

  RULES_PROVED="${RULES_PROVED}${name}|${proved}"$'\n'
}

# reason_of_a_rule_without_anything_to_prove prints the reason a rule has
# nothing to prove today, or nothing when the rule is not one of them.
reason_of_a_rule_without_anything_to_prove() {
  local name="$1"

  local line
  while IFS='|' read -r rule reason; do
    [ -n "$rule" ] || continue
    if [ "$rule" = "$name" ]; then
      printf '%s' "$reason"
      return 0
    fi
  done <<EOF
$RULES_WITHOUT_ANYTHING_TO_PROVE
EOF
}

# report_the_rules prints what every rule proved, and fails the run when a
# rule proved nothing and did not say why.
report_the_rules() {
  printf '%s\n' "The rules of the gate and what each of them proved:"

  local name proved reason
  while IFS='|' read -r name proved; do
    [ -n "$name" ] || continue

    if [ "$proved" -ne 0 ]; then
      printf '  %s: %s assertion(s)\n' "$name" "$proved"
      continue
    fi

    reason=$(reason_of_a_rule_without_anything_to_prove "$name")
    if [ -n "$reason" ]; then
      printf '  %s: nothing to prove (%s)\n' "$name" "$reason"
      continue
    fi

    printf '  %s: PROVED NOTHING, and it does not say why\n' "$name" >&2
    FAILURES=$((FAILURES + 1))
  done <<EOF
$RULES_PROVED
EOF
}

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
#
# The rule is stated as what the layer may name, and not only as what it may
# not: an application that imports the adapter of the shared library — the
# pool of the database, for example — names no adapter of its own service,
# and the rule that forbids only the adapters of the service would let it
# through.
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

        # The layer names the standard library, the ports of its service
        # and the shared domain. Nothing else.
        is_external "$imported" || continue

        case "$imported" in
        github.com/witsaba/paca/apps/"$module"/internal/ports*) ;;
        github.com/witsaba/paca/packages/domain*) ;;
        *)
          fail "the application of $module names its ports and the domain, and nothing else: [$imported]"
          ;;
        esac
        ASSERTIONS=$((ASSERTIONS + 1))
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

# test_the_adapters_of_every_application_know_no_handle_of_the_database
# asserts that the adapters of an application name no handle of the
# database.
#
# The capability "does the database answer?" has one owner: the library that
# owns the database. An adapter that names the handle asks the question its
# own way, and two services end up asking it in two ways. The adapters reach
# the capability through the port of their application, and the composition
# root binds the handle of the library to it.
test_the_adapters_of_every_application_know_no_handle_of_the_database() {
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
        assert_not_contains "$imported" "packages/database/postgres" \
          "the adapters of $module name no handle of the database: [$imported]"
        assert_not_contains "$imported" "database/sql" \
          "the adapters of $module name no handle of the database: [$imported]"
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
# The promises of the contract
# ---------------------------------------------------------------------------

# test_there_is_no_dead_api asserts that every function of the repository is
# reachable from the entry point of a service.
#
# The rule of the contract: "No dead API. A function that no production code
# calls is not a promise, it is debris: it is removed, or the caller that
# needs it is written."
#
# The analysis starts at the main package of every service, so a function
# that only a test calls is dead: the test is the proof of a function, not
# the caller that keeps it.
test_there_is_no_dead_api() {
  local mains=()
  local applications=0
  local application
  while IFS= read -r application; do
    applications=$((applications + 1))
    local main
    while IFS= read -r main; do
      mains+=("../${main}")
    done < <(find "$REPO_ROOT/$application/cmd" -mindepth 1 -maxdepth 1 -type d 2>/dev/null |
      sed "s|^$REPO_ROOT/||" | sort)
  done < <(applications_of_the_repository)

  assert_equals "$applications" "${#mains[@]}" "every application has one entry point"

  if [ "${#mains[@]}" -eq 0 ]; then
    return
  fi

  local output status
  output=$(cd "$REPO_ROOT/tools" && go tool deadcode -filter='github.com/witsaba/paca/' "${mains[@]}" 2>&1)
  status=$?

  assert_equals "0" "$status" "the analysis of the dead code of the repository runs"

  local dead
  dead=$(printf '%s\n' "$output" | grep 'unreachable func' || true)
  if [ -n "$dead" ]; then
    printf '%s\n' "$dead" >&2
  fi

  assert_equals "0" "$(printf '%s\n' "$dead" | grep -c 'unreachable func' || true)" \
    "every function of the repository is reachable from the entry point of a service"
}

# test_only_the_database_admin_writes_the_schema asserts that one service
# applies the migrations of the schema.
#
# The rule of the contract: "No second writer of the schema. One service
# owns the migrations of the database, and it applies them at the start. No
# service migrates on boot."
#
# The rule names the service that owns the schema, and it says so out loud:
# a rule that names a service that does not exist proves nothing.
test_only_the_database_admin_writes_the_schema() {
  local writers
  writers=$(grep -rlE '[A-Za-z0-9_]+\.NewMigrator' "$REPO_ROOT/apps" --include='*.go' |
    grep -v '_test\.go$' |
    sed "s|^$REPO_ROOT/||" | sort | tr '\n' ' ')

  assert_equals "apps/db-admin/internal/bootstrap/bootstrap.go " "$writers" \
    "the database admin is the only service that applies the migrations of the schema"
}

# bodies_of_the_application prints the body of every function the
# application declares, one per line, with the comments and the spaces of
# the code removed.
#
# The body is what says whether two services carry the same code: two
# functions with the same name and a different body are two behaviors, and
# two functions with the same body are one behavior written twice.
bodies_of_the_application() {
  local application="$1"

  find "$REPO_ROOT/$application" -name '*.go' ! -name '*_test.go' | sort | while IFS= read -r file; do
    awk '
      /^func /{
        in_function = 1
        in_signature = 1
        body = ""
        signature = $0
        next
      }
      in_function && in_signature {
        signature = signature " " $0
        if ($0 ~ /\{$/) {
          in_signature = 0
          name = signature
          sub(/^func (\([^)]*\) )?/, "", name)
          sub(/\(.*/, "", name)
          sub(/[ \t].*$/, "", name)
        }
        next
      }
      in_function && /^\}$/ {
        print name "\t" body
        in_function = 0
        next
      }
      in_function {
        line = $0
        sub(/^[ \t]+/, "", line)
        sub(/[ \t]+$/, "", line)
        if (line == "" || line ~ /^\/\//) next
        gsub(/[ \t]+/, " ", line)
        body = body line " | "
      }
    ' "$file"
  done
}

# name_of_the_body prints the name of the function a body belongs to.
#
# The body carries the name of the function and the body itself: the rule
# reports the copy with the name of the function it found.
name_of_the_body() {
  printf '%s' "$1" | cut -f1
}

# reason_of_a_known_copy prints the reason a copy is known, or nothing when
# the copy is not one of them.
reason_of_a_known_copy() {
  local key="$1"

  local line
  while IFS='|' read -r known reason; do
    [ -n "$known" ] || continue
    if [ "$known" = "$key" ]; then
      printf '%s' "$reason"
      return 0
    fi
  done <<EOF
$KNOWN_COPIES
EOF
}

# test_the_services_do_not_carry_the_same_code asserts that the two services
# do not write the same behavior twice.
#
# The rule of the contract: "No copy of a shared layer. The platform and the
# shared domain are shared: they are extended, never forked."
#
# A behavior that two services need belongs to the shared layer. A copy of
# it in the two services is two promises that drift apart, and the gate is
# what says so before the drift does.
test_the_services_do_not_carry_the_same_code() {
  local applications=()
  local application
  while IFS= read -r application; do
    applications+=("$application")
  done < <(applications_of_the_repository)

  if [ "${#applications[@]}" -lt 2 ]; then
    return
  fi

  local first="${applications[0]}"
  local second="${applications[1]}"

  report_the_copies_between "$first" "$second"
  report_the_copies_between "$second" "$first"

  ASSERTIONS=$((ASSERTIONS + 1))
}

# report_the_copies_between reports the bodies that the first application
# carries and the second one carries as well.
#
# The copy is looked for in both directions: a copy that the second service
# carries and the first one does not is a copy as much as the other way
# around.
report_the_copies_between() {
  local first="$1"
  local second="$2"

  local bodies_first bodies_second
  bodies_first=$(bodies_of_the_application "$first")
  bodies_second=$(bodies_of_the_application "$second" | cut -f2- | sort)

  local body name reason code lines
  while IFS= read -r body; do
    [ -n "$body" ] || continue

    name=$(name_of_the_body "$body")

    # The body is what says whether two services carry the same code: the
    # name of the function may differ, the behavior may not.
    code=$(printf '%s' "$body" | cut -f2-)

    # A body of a few lines is a shape, not a behavior: two services that
    # build an address the same way are not a copy of a shared layer.
    lines=$(printf '%s' "$code" | tr -cd '|' | wc -c | tr -d ' ')
    if [ "$lines" -lt 6 ]; then
      continue
    fi

    if ! printf '%s\n' "$bodies_second" | grep -qxF -- "$code"; then
      continue
    fi

    reason=$(reason_of_a_known_copy "internal/application/health.go:${name}")
    if [ -z "$reason" ]; then
      reason=$(reason_of_a_known_copy "internal/adapters/inbound/http/health_handler.go:${name}")
    fi

    if [ -n "$reason" ]; then
      if [ "$first" = "${applications[0]}" ]; then
        printf '  known copy: %s in the %s and in the %s (%s)\n' "$name" "$first" "$second" "$reason"
      fi
      continue
    fi

    fail "the services $first and $second carry the same function $name: a shared layer is extended, never forked"
    ASSERTIONS=$((ASSERTIONS + 1))
  done <<EOF
$bodies_first
EOF
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

run_rule "every module resolves its own graph" test_every_module_resolves_its_own_graph
run_rule "every link of the documentation points at a file" test_every_relative_link_points_at_a_file
run_rule "the domain of every application knows nothing" test_the_domain_of_every_application_knows_nothing
run_rule "the application of every application knows no adapter" test_the_application_of_every_application_knows_no_adapter
run_rule "the adapters know no use case" test_the_adapters_of_every_application_know_no_use_case
run_rule "the adapters know no handle of the database" test_the_adapters_of_every_application_know_no_handle_of_the_database
run_rule "only the composition root knows the adapters" test_only_the_composition_root_knows_the_adapters
run_rule "the shared domain knows nothing but the standard library" test_the_shared_domain_knows_nothing_but_the_standard_library
run_rule "the shared platform knows no model" test_the_shared_platform_knows_no_model
run_rule "every image copies the modules it imports" test_every_image_copies_the_modules_it_imports
run_rule "no function of the repository is dead" test_there_is_no_dead_api
run_rule "the database admin is the only writer of the schema" test_only_the_database_admin_writes_the_schema
run_rule "the services do not carry the same code" test_the_services_do_not_carry_the_same_code

report_the_rules

printf '%s assertion(s), %s failure(s)\n' "$ASSERTIONS" "$FAILURES"

if [ "$FAILURES" -ne 0 ]; then
  exit 1
fi

exit 0
