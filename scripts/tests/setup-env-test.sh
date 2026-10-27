#!/usr/bin/env bash
#
# The tests of the installer of the environment.
#
# The tests run the script against fixtures in a temporary directory, so the
# `.env` of the repository is never touched. Every assertion goes through
# the public interface of the script: the files it writes, their content,
# their mode and what it prints.
#
# The script has no dependency besides a POSIX shell and the usual system
# tools: there is no test framework to install.
#
# Usage: scripts/tests/setup-env-test.sh

set -u

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
REPO_ROOT=$(dirname "$(dirname "$SCRIPT_DIR")")
SETUP_ENV="$REPO_ROOT/scripts/setup-env.sh"

# The scratch of the tests lives in the temporary directory of the system,
# never in the repository.
TMP_ROOT=$(mktemp -d) || exit 1
trap 'rm -rf "$TMP_ROOT"' EXIT INT TERM

# ---------------------------------------------------------------------------
# The state of the run
# ---------------------------------------------------------------------------

ASSERTIONS=0
FAILURES=0
STATUS=0
OUTPUT=''

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

assert_not_equals() {
  local unexpected="$1"
  local actual="$2"
  local message="$3"
  ASSERTIONS=$((ASSERTIONS + 1))
  if [ "$unexpected" = "$actual" ]; then
    fail "$message: the value [$actual] was not expected"
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
  if [ ! -f "$path" ]; then
    fail "$message: the file [$path] does not exist"
  fi
}

assert_file_missing() {
  local path="$1"
  local message="$2"
  ASSERTIONS=$((ASSERTIONS + 1))
  if [ -f "$path" ]; then
    fail "$message: the file [$path] exists"
  fi
}

assert_file_contains() {
  local path="$1"
  local needle="$2"
  local message="$3"
  ASSERTIONS=$((ASSERTIONS + 1))
  if [ ! -f "$path" ]; then
    fail "$message: the file [$path] does not exist"
    return
  fi
  case "$(cat "$path")" in
    *"$needle"*) ;;
    *) fail "$message: [$needle] was not found in [$path]" ;;
  esac
}

assert_file_mode() {
  local path="$1"
  local expected="$2"
  local message="$3"
  local actual
  ASSERTIONS=$((ASSERTIONS + 1))
  if [ ! -f "$path" ]; then
    fail "$message: the file [$path] does not exist"
    return
  fi
  actual=$(file_mode "$path")
  if [ "$actual" != "$expected" ]; then
    fail "$message: the mode of [$path] is [$actual], expected [$expected]"
  fi
}

assert_file_value() {
  local path="$1"
  local key="$2"
  local expected="$3"
  local message="$4"
  local actual
  ASSERTIONS=$((ASSERTIONS + 1))
  if [ ! -f "$path" ]; then
    fail "$message: the file [$path] does not exist"
    return
  fi
  actual=$(sed -n "s/^${key}=//p" "$path" | head -n 1)
  if [ "$actual" != "$expected" ]; then
    fail "$message: the value of [$key] is [$actual], expected [$expected]"
  fi
}

assert_no_file_matching() {
  local directory="$1"
  local pattern="$2"
  local message="$3"
  local found
  ASSERTIONS=$((ASSERTIONS + 1))
  found=$(find "$directory" -maxdepth 1 -name "$pattern" -print | head -n 1)
  if [ -n "$found" ]; then
    fail "$message: the file [$found] was left behind"
  fi
}

# ---------------------------------------------------------------------------
# The helpers
# ---------------------------------------------------------------------------

file_mode() {
  local path="$1"
  if stat -f '%Lp' "$path" >/dev/null 2>&1; then
    stat -f '%Lp' "$path"
  else
    stat -c '%a' "$path"
  fi
}

file_mtime() {
  local path="$1"
  if stat -f '%m' "$path" >/dev/null 2>&1; then
    stat -f '%m' "$path"
  else
    stat -c '%Y' "$path"
  fi
}

value_of() {
  local path="$1"
  local key="$2"
  sed -n "s/^${key}=//p" "$path" | head -n 1
}

line_of() {
  local path="$1"
  local key="$2"
  grep -n "^${key}=" "$path" | head -n 1 | cut -d: -f1
}

checksum_of() {
  cksum < "$1"
}

# make_fixture DIRECTORY
#
# Builds a root with two stacks: the root itself and the PostgreSQL one. The
# `.git` directory holds an `env.example` too: the walk must ignore it.
make_fixture() {
  local root="$1"
  mkdir -p "$root/db/postgres" "$root/.git"

  cat > "$root/env.example" <<'EOF'
# The version of the images of the monorepository.

# The tag of the images of the monorepository.
PACA_VERSION=dev
EOF

  cat > "$root/db/postgres/env.example" <<'EOF'
# The credentials of the PostgreSQL stack.

# The owner role and the superuser of the application database.
POSTGRES_USER=paca

# The application database of the stack.
POSTGRES_DB=paca

# The password of the role. It is a secret.
# @secret
# @generate
POSTGRES_PASSWORD=change-me-generate-a-strong-password
EOF

  cat > "$root/.git/env.example" <<'EOF'
# This file lives in the metadata of the repository: the walk must ignore it.

# It must never reach a `.env`.
GIT_METADATA=paca
EOF
}

# run_setup_env ROOT [ARGUMENTS...]
#
# Runs the installer and keeps its exit code and its output.
run_setup_env() {
  local root="$1"
  shift
  OUTPUT=$("$SETUP_ENV" --root "$root" "$@" 2>&1)
  STATUS=$?
}

# run_setup_env_with_input ROOT INPUT [ARGUMENTS...]
#
# Runs the installer with the standard input fed with the answers.
run_setup_env_with_input() {
  local root="$1"
  local input="$2"
  shift 2
  OUTPUT=$(printf '%s' "$input" | "$SETUP_ENV" --root "$root" "$@" 2>&1)
  STATUS=$?
}

# ---------------------------------------------------------------------------
# The tests
# ---------------------------------------------------------------------------

test_the_help_is_readable() {
  OUTPUT=$("$SETUP_ENV" --help 2>&1)
  STATUS=$?
  assert_equals "0" "$STATUS" "the help exits with 0"
  assert_contains "$OUTPUT" "Usage" "the help shows the usage"
  assert_contains "$OUTPUT" "--dry-run" "the help lists the options"
}

test_an_unknown_option_is_a_usage_error() {
  OUTPUT=$("$SETUP_ENV" --nonsense 2>&1)
  STATUS=$?
  assert_equals "1" "$STATUS" "an unknown option exits with 1"
}

test_an_unknown_password_mode_is_a_usage_error() {
  OUTPUT=$("$SETUP_ENV" --password nonsense 2>&1)
  STATUS=$?
  assert_equals "1" "$STATUS" "an unknown password mode exits with 1"
}

test_the_keys_are_written_with_their_defaults() {
  local root="$TMP_ROOT/defaults"
  make_fixture "$root"
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_exists "$root/.env" "the root .env is written"
  assert_file_exists "$root/db/postgres/.env" "the .env of the stack is written"
  assert_file_contains "$root/.env" "PACA_VERSION=dev" "the root keeps its default"
  assert_file_contains "$root/db/postgres/.env" "POSTGRES_USER=paca" "the user keeps its default"
  assert_file_contains "$root/db/postgres/.env" "POSTGRES_DB=paca" "the database keeps its default"
  assert_file_missing "$root/.git/.env" "the metadata of the repository is ignored"
}

test_the_secret_gets_a_generated_strong_password() {
  local root="$TMP_ROOT/secret"
  local password
  make_fixture "$root"
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  password=$(value_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  assert_not_equals "change-me-generate-a-strong-password" "$password" "the placeholder is never written"
  assert_equals "32" "${#password}" "the password is 32 characters long"
  case "$password" in
    *[!A-Za-z0-9_-]*) fail "the password holds a character outside of A-Za-z0-9-_" ;;
    *) ;;
  esac
}

test_running_twice_keeps_the_password() {
  local root="$TMP_ROOT/twice"
  local first
  local second
  make_fixture "$root"
  run_setup_env "$root" --yes
  first=$(value_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  run_setup_env "$root" --yes
  second=$(value_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  assert_equals "$first" "$second" "the password survives a second run"
}

test_a_value_modified_by_hand_is_kept_and_offered() {
  local root="$TMP_ROOT/kept"
  make_fixture "$root"
  cat > "$root/db/postgres/.env" <<'EOF'
POSTGRES_USER=custom-user
EOF
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_contains "$root/db/postgres/.env" "POSTGRES_USER=custom-user" "the value configured by hand is kept"

  run_setup_env_with_input "$root" "
"
  assert_contains "$OUTPUT" "POSTGRES_USER [custom-user]" "the value configured by hand is offered as the default"
}

test_a_key_added_by_hand_is_preserved() {
  local root="$TMP_ROOT/extra"
  local user_line
  local password_line
  local extra_line
  make_fixture "$root"
  cat > "$root/db/postgres/.env" <<'EOF'
POSTGRES_USER=paca
EXTRA_KEY=extra-value
EOF
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_contains "$root/db/postgres/.env" "EXTRA_KEY=extra-value" "the key added by hand is preserved"
  user_line=$(line_of "$root/db/postgres/.env" POSTGRES_USER)
  password_line=$(line_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  extra_line=$(line_of "$root/db/postgres/.env" EXTRA_KEY)
  ASSERTIONS=$((ASSERTIONS + 1))
  if [ "$user_line" -ge "$password_line" ] || [ "$password_line" -ge "$extra_line" ]; then
    fail "the keys of the manifest come first, in their order, and the hand written keys come last"
  fi
}

test_the_yes_mode_asks_nothing_and_works_with_a_closed_input() {
  local root="$TMP_ROOT/yes"
  make_fixture "$root"
  OUTPUT=$("$SETUP_ENV" --root "$root" --yes 0<&- 2>&1)
  STATUS=$?
  assert_equals "0" "$STATUS" "the run succeeds with the standard input closed"
  assert_file_exists "$root/db/postgres/.env" "the .env of the stack is written"
  assert_not_contains "$OUTPUT" "POSTGRES_USER [" "no question is asked"
  assert_contains "$OUTPUT" "POSTGRES_USER=paca" "the values are reported"
}

test_an_empty_configured_value_is_no_value() {
  local root="$TMP_ROOT/empty"
  local password
  make_fixture "$root"
  cat > "$root/db/postgres/.env" <<'EOF'
POSTGRES_USER=custom-user
POSTGRES_PASSWORD=
EOF
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_contains "$OUTPUT" "empty" "the script warns about the empty value"
  password=$(value_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  assert_not_equals "" "$password" "the empty password is not kept"
  assert_equals "32" "${#password}" "the password is 32 characters long"
  assert_file_value "$root/db/postgres/.env" "POSTGRES_USER" "custom-user" "the other values are kept"
}

test_a_file_with_a_hand_written_key_is_private() {
  local root="$TMP_ROOT/private"
  make_fixture "$root"
  # The stack of the root declares no secret: only the key that was added
  # by hand can make the file a secret.
  cat > "$root/.env" <<'EOF'
EXTRA_KEY=extra-value
EOF
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_mode "$root/.env" "600" "a key that the manifest does not declare may be a secret"
}

test_a_quoted_empty_value_is_no_value() {
  local root="$TMP_ROOT/quoted-empty"
  local password
  make_fixture "$root"
  cat > "$root/db/postgres/.env" <<'EOF'
POSTGRES_PASSWORD=""
EOF
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_contains "$OUTPUT" "empty" "the script warns about the empty value"
  password=$(value_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  assert_equals "32" "${#password}" "the password is 32 characters long"
  assert_not_contains "$(cat "$root/db/postgres/.env")" 'POSTGRES_PASSWORD=""' "the empty value is replaced"
}

test_the_header_names_the_manifest() {
  local root="$TMP_ROOT/header"
  make_fixture "$root"
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_contains "$root/db/postgres/.env" "Generated by scripts/setup-env.sh" "the file says that it is generated"
  assert_file_contains "$root/db/postgres/.env" "Source: db/postgres/env.example" "the file names its manifest"
  assert_file_contains "$root/db/postgres/.env" "Never commit" "the file says that it must never be committed"
}

test_the_description_is_shown_for_every_key() {
  local root="$TMP_ROOT/description"
  make_fixture "$root"
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_contains "$OUTPUT" "The owner role and the superuser" "the description of a key is reported"
}

test_the_dry_run_hides_the_secret() {
  local root="$TMP_ROOT/dry-secret"
  local password
  make_fixture "$root"
  run_setup_env "$root" --yes
  password=$(value_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  run_setup_env "$root" --yes --dry-run
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_not_contains "$OUTPUT" "$password" "the dry run hides the secret"
}

test_no_temporary_file_is_left_behind() {
  local root="$TMP_ROOT/atomic"
  make_fixture "$root"
  run_setup_env "$root" --yes
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_no_file_matching "$root/db/postgres" ".env.tmp.*" "no temporary file is left behind"
  assert_no_file_matching "$root" ".env.tmp.*" "no temporary file is left behind"
}

test_the_temporary_file_of_the_write_is_ignored_by_git() {
  assert_file_contains "$REPO_ROOT/.gitignore" ".env.tmp.*" "the temporary file of the write is ignored by git"
}

test_a_link_is_not_followed_outside_of_the_root() {
  local root="$TMP_ROOT/link/root"
  local neighbour="$TMP_ROOT/link/neighbour"
  mkdir -p "$neighbour"
  make_fixture "$root"
  # The target of the link holds exactly what the installer writes, so the
  # run takes the path that leaves the content alone.
  run_setup_env "$root" --yes
  cp "$root/db/postgres/.env" "$neighbour/secret.env"
  chmod 644 "$neighbour/secret.env"
  rm -f "$root/db/postgres/.env"
  ln -s "$neighbour/secret.env" "$root/db/postgres/.env"
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_mode "$neighbour/secret.env" "644" "the mode of the file outside of the root is not changed"
  assert_contains "$OUTPUT" "link outside of the root" "the script says that the link points outside"
}

test_a_link_inside_of_the_root_gets_its_mode_fixed() {
  local root="$TMP_ROOT/link-inside/root"
  mkdir -p "$root/shared"
  make_fixture "$root"
  run_setup_env "$root" --yes
  cp "$root/db/postgres/.env" "$root/shared/.env"
  chmod 644 "$root/shared/.env"
  rm -f "$root/db/postgres/.env"
  ln -s "$root/shared/.env" "$root/db/postgres/.env"
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_mode "$root/shared/.env" "600" "the mode of a file inside of the root is corrected"
}

test_a_stack_that_is_a_link_outside_of_the_root_is_refused() {
  local root="$TMP_ROOT/link-stack/root"
  local neighbour="$TMP_ROOT/link-stack/neighbour"
  mkdir -p "$neighbour"
  make_fixture "$root"
  printf '# A stack outside of the root.\n\n# The value.\nVALUE=paca\n' > "$neighbour/env.example"
  ln -s "$neighbour" "$root/evil"
  run_setup_env "$root" --yes --stack evil
  assert_equals "1" "$STATUS" "the run fails"
  assert_file_missing "$neighbour/.env" "nothing is written outside of the root"
}

test_the_dry_run_writes_nothing() {
  local root="$TMP_ROOT/dry-run"
  make_fixture "$root"
  run_setup_env "$root" --yes --dry-run
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_missing "$root/.env" "the root .env is not written"
  assert_file_missing "$root/db/postgres/.env" "the .env of the stack is not written"
  assert_contains "$OUTPUT" "POSTGRES_USER=paca" "the dry run prints the values"
}

test_the_password_never_appears_in_the_output() {
  local root="$TMP_ROOT/quiet"
  local password
  make_fixture "$root"
  run_setup_env "$root" --yes
  password=$(value_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  assert_not_contains "$OUTPUT" "$password" "the password is not printed"
  assert_not_contains "$OUTPUT" "change-me-generate-a-strong-password" "the placeholder is not printed"

  cat > "$root/db/postgres/.env" <<'EOF'
POSTGRES_PASSWORD=a-secret-configured-by-hand
EOF
  run_setup_env_with_input "$root" "
"
  assert_not_contains "$OUTPUT" "a-secret-configured-by-hand" "a configured secret is not printed"
}

test_the_password_keep_keeps_the_placeholder_and_warns() {
  local root="$TMP_ROOT/keep"
  make_fixture "$root"
  run_setup_env "$root" --yes --password keep
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_contains "$root/db/postgres/.env" "POSTGRES_PASSWORD=change-me-generate-a-strong-password" "the placeholder is kept"
  assert_contains "$OUTPUT" "placeholder" "the script warns about the weak placeholder"
}

test_the_file_that_holds_a_secret_has_the_mode_600() {
  local root="$TMP_ROOT/mode"
  make_fixture "$root"
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_mode "$root/db/postgres/.env" "600" "the file that holds a secret is private"
  assert_file_mode "$root/.env" "644" "the file that holds no secret is readable"
}

test_the_stacks_are_all_generated_and_the_stack_option_limits_them() {
  local root="$TMP_ROOT/stacks"
  local other
  make_fixture "$root"
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_exists "$root/.env" "the root is generated"
  assert_file_exists "$root/db/postgres/.env" "the stack is generated"

  other="$TMP_ROOT/stacks-one"
  make_fixture "$other"
  run_setup_env "$other" --yes --stack db/postgres
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_exists "$other/db/postgres/.env" "the stack of the option is generated"
  assert_file_missing "$other/.env" "the other stacks are left alone"
}

test_the_manifest_is_never_modified() {
  local root="$TMP_ROOT/manifest"
  local before
  local after
  make_fixture "$root"
  before=$(checksum_of "$root/db/postgres/env.example")
  run_setup_env "$root" --yes
  after=$(checksum_of "$root/db/postgres/env.example")
  assert_equals "$before" "$after" "the manifest is not modified"
}

test_the_script_refuses_to_write_outside_the_root() {
  local root="$TMP_ROOT/outside/root"
  local neighbour="$TMP_ROOT/outside/neighbour"
  mkdir -p "$neighbour"
  make_fixture "$root"
  cat > "$neighbour/env.example" <<'EOF'
# A stack that lives outside of the root.

# The value of the neighbour.
NEIGHBOUR=paca
EOF
  run_setup_env "$root" --yes --stack ../neighbour
  assert_equals "1" "$STATUS" "the run fails"
  assert_file_missing "$neighbour/.env" "nothing is written outside of the root"
  assert_contains "$OUTPUT" "outside" "the script says the stack is outside of the root"
}

test_the_placeholder_of_a_copied_manifest_is_replaced() {
  local root="$TMP_ROOT/copied"
  local password
  make_fixture "$root"
  cp "$root/db/postgres/env.example" "$root/db/postgres/.env"
  run_setup_env "$root" --yes
  assert_equals "0" "$STATUS" "the run succeeds"
  password=$(value_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  assert_not_equals "change-me-generate-a-strong-password" "$password" "the placeholder of the copy is replaced"
  assert_equals "32" "${#password}" "the password is 32 characters long"
  assert_file_mode "$root/db/postgres/.env" "600" "the file that holds a secret is private"
}

test_the_answers_replace_the_values() {
  local root="$TMP_ROOT/answers"
  make_fixture "$root"
  run_setup_env_with_input "$root" "another-user
another-db

"
  assert_equals "0" "$STATUS" "the run succeeds"
  assert_file_contains "$root/db/postgres/.env" "POSTGRES_USER=another-user" "the answer replaces the value"
  assert_file_contains "$root/db/postgres/.env" "POSTGRES_DB=another-db" "the answer replaces the value"
}

test_the_answer_g_generates_a_strong_password() {
  local root="$TMP_ROOT/generate"
  local password
  make_fixture "$root"
  run_setup_env_with_input "$root" "


g
"
  assert_equals "0" "$STATUS" "the run succeeds"
  password=$(value_of "$root/db/postgres/.env" POSTGRES_PASSWORD)
  assert_not_equals "change-me-generate-a-strong-password" "$password" "the placeholder is never written"
  assert_equals "32" "${#password}" "the password is 32 characters long"
}

test_a_file_that_does_not_change_is_not_rewritten() {
  local root="$TMP_ROOT/untouched"
  local before
  local after
  make_fixture "$root"
  run_setup_env "$root" --yes
  before=$(file_mtime "$root/db/postgres/.env")
  sleep 1
  run_setup_env "$root" --yes
  after=$(file_mtime "$root/db/postgres/.env")
  assert_equals "$before" "$after" "the file is left untouched"
}

# ---------------------------------------------------------------------------
# The runner
# ---------------------------------------------------------------------------

if [ ! -x "$SETUP_ENV" ]; then
  printf 'the installer [%s] is missing\n' "$SETUP_ENV" >&2
  exit 1
fi

test_the_help_is_readable
test_an_unknown_option_is_a_usage_error
test_an_unknown_password_mode_is_a_usage_error
test_the_keys_are_written_with_their_defaults
test_the_secret_gets_a_generated_strong_password
test_running_twice_keeps_the_password
test_a_value_modified_by_hand_is_kept_and_offered
test_a_key_added_by_hand_is_preserved
test_the_yes_mode_asks_nothing_and_works_with_a_closed_input
test_an_empty_configured_value_is_no_value
test_a_quoted_empty_value_is_no_value
test_a_file_with_a_hand_written_key_is_private
test_the_header_names_the_manifest
test_the_description_is_shown_for_every_key
test_the_dry_run_hides_the_secret
test_no_temporary_file_is_left_behind
test_the_temporary_file_of_the_write_is_ignored_by_git
test_a_link_is_not_followed_outside_of_the_root
test_a_link_inside_of_the_root_gets_its_mode_fixed
test_a_stack_that_is_a_link_outside_of_the_root_is_refused
test_the_dry_run_writes_nothing
test_the_password_never_appears_in_the_output
test_the_password_keep_keeps_the_placeholder_and_warns
test_the_file_that_holds_a_secret_has_the_mode_600
test_the_stacks_are_all_generated_and_the_stack_option_limits_them
test_the_manifest_is_never_modified
test_the_script_refuses_to_write_outside_the_root
test_the_placeholder_of_a_copied_manifest_is_replaced
test_the_answers_replace_the_values
test_the_answer_g_generates_a_strong_password
test_a_file_that_does_not_change_is_not_rewritten

printf '%s assertion(s), %s failure(s)\n' "$ASSERTIONS" "$FAILURES"

if [ "$FAILURES" -ne 0 ]; then
  exit 1
fi

exit 0
