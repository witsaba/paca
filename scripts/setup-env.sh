#!/usr/bin/env bash
#
# The installer of the environment of the monorepository.
#
# It walks the repository, finds every `env.example` — the manifest of a
# stack — and writes the `.env` next to it. Every value is offered with its
# default, the values that are already configured are kept, and the secrets
# get a strong password that is never printed.
#
# The usage:
#
#   scripts/setup-env.sh [options]
#
# The exit codes: 0 when the run succeeds, 1 on a usage or a write error,
# 2 when the person interrupts the run.

set -euo pipefail

# ---------------------------------------------------------------------------
# The constants
# ---------------------------------------------------------------------------

readonly EXIT_OK=0
readonly EXIT_ERROR=1
readonly EXIT_INTERRUPTED=2

# A strong password: 32 characters of A-Za-z0-9-_. Those characters are safe
# in a `.env` file, in a Compose file and in a connection string.
readonly PASSWORD_LENGTH=32

# The value that hides a secret in the output.
readonly MASK='********'

# ---------------------------------------------------------------------------
# The state of the run
# ---------------------------------------------------------------------------

OPT_ROOT=''
OPT_YES=0
OPT_DRY_RUN=0
OPT_PASSWORD='ask'
OPT_STACKS=()

# The manifest that is being read, and the file that is written.
MANIFEST=''
MANIFEST_REL=''
TARGET=''
TARGET_REL=''

# The manifest, parsed: the keys in the order of the file, their value, their
# description and the two directives.
MF_KEYS=()
MF_VALUES=()
MF_DESCRIPTIONS=()
MF_SECRET=()
MF_GENERATE=()

# Whether the value of a key was asked to the person. A key that was not
# asked is reported with its description, because nobody saw it.
MF_ASKED=()

# The `.env` that is already on disk.
ENV_KEYS=()
ENV_VALUES=()

# The values of the run: the keys of the manifest, then the keys that were
# configured by hand.
VALUES=()
EXTRA_KEYS=()
EXTRA_VALUES=()

# The file that is being written.
CONTENT=''
HAS_SECRET=0

# The answer of the last question, and the state of the standard input.
ANSWER=''
NO_INPUT=0

# The temporary file of the atomic write.
TMP_FILE=''

# The number of the files that were written.
WRITTEN=0

# ---------------------------------------------------------------------------
# The messages
# ---------------------------------------------------------------------------

print_help() {
  cat <<'EOF'
Usage: scripts/setup-env.sh [options]

The installer of the environment of the monorepository. It walks the
repository, finds every `env.example` — the manifest of a stack — and writes
the `.env` next to it. Every value is offered with its default, the values
that are already configured are kept, and the secrets get a strong password
that is never printed.

Options:
  --root DIR        the root of the repository. The default is the parent
                    directory of this script.
  --stack PATH      work on one stack only. The path is relative to the
                    root. The option can be repeated.
  --password MODE   how the secrets are handled:
                      ask       ask for the value of every secret (default)
                      generate  generate a strong password, never ask
                      keep      keep the value as it is, never ask
  --yes             ask nothing: keep the values that are configured and
                    generate the passwords that are not configured yet
  --dry-run         print what it would write and write no file at all
  --help            show this help

The answers of the interactive mode are read from the standard input:
  the empty answer keeps the default
  the answer `g` generates a strong value for a secret marked `# @generate`
  any other answer replaces the value

Exit codes:
  0  the run succeeded
  1  a usage error or a write error
  2  the run was interrupted
EOF
}

warn() {
  printf 'setup-env: %s\n' "$1" >&2
}

die() {
  printf 'setup-env: %s\n' "$1" >&2
  exit "$EXIT_ERROR"
}

die_usage() {
  printf 'setup-env: %s\n' "$1" >&2
  printf 'Try scripts/setup-env.sh --help for the usage.\n' >&2
  exit "$EXIT_ERROR"
}

# ---------------------------------------------------------------------------
# The traps
# ---------------------------------------------------------------------------

cleanup() {
  if [ -n "$TMP_FILE" ] && [ -f "$TMP_FILE" ]; then
    rm -f "$TMP_FILE"
  fi
}

on_interrupt() {
  printf '\nsetup-env: interrupted\n' >&2
  exit "$EXIT_INTERRUPTED"
}

trap cleanup EXIT
trap on_interrupt INT

# ---------------------------------------------------------------------------
# The helpers
# ---------------------------------------------------------------------------

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

file_mode_of() {
  local path="$1"
  if stat -f '%Lp' "$path" >/dev/null 2>&1; then
    stat -f '%Lp' "$path"
  else
    stat -c '%a' "$path"
  fi
}

is_within_root() {
  local path="$1"
  case "$path" in
    "$OPT_ROOT") return 0 ;;
    "$OPT_ROOT"/*) return 0 ;;
    *) return 1 ;;
  esac
}

# bare_value VALUE
#
# Prints the value as Docker Compose reads it in an `env_file`: without the
# spaces around it and without the quotes that wrap it. The value itself is
# never changed: a quoted value stays quoted in the file that is written.
bare_value() {
  local value
  value=$(trim "$1")
  case "$value" in
    '""') printf '' ;;
    "''") printf '' ;;
    '"'*'"')
      value="${value#\"}"
      value="${value%\"}"
      printf '%s' "$value"
      ;;
    "'"*"'")
      value="${value#\'}"
      value="${value%\'}"
      printf '%s' "$value"
      ;;
    *) printf '%s' "$value" ;;
  esac
}

# resolve_link PATH
#
# Prints the path a link points at, resolved. A path that is not a link is
# printed as it is.
resolve_link() {
  local path="$1"
  local target
  local guard=0
  while [ -L "$path" ] && [ "$guard" -lt 10 ]; do
    guard=$((guard + 1))
    target=$(readlink "$path")
    case "$target" in
      /*) path="$target" ;;
      *) path="$(dirname "$path")/$target" ;;
    esac
  done
  printf '%s\n' "$path"
}

# random_material
#
# Prints the raw material of a strong password: characters of A-Za-z0-9-_.
# The source is `openssl rand`, and `/dev/urandom` when openssl is missing.
random_material() {
  local raw
  if command -v openssl >/dev/null 2>&1; then
    raw=$(openssl rand -base64 48 2>/dev/null | tr -d '\n=' | tr '+/' '-_' || true)
    if [ -n "$raw" ]; then
      printf '%s\n' "$raw"
      return 0
    fi
  fi
  dd if=/dev/urandom bs=256 count=1 2>/dev/null | LC_ALL=C tr -dc 'A-Za-z0-9_-' || true
}

# generate_password
#
# Prints a strong password: 32 characters of A-Za-z0-9-_.
generate_password() {
  local password
  local attempt=0
  while [ "$attempt" -lt 5 ]; do
    attempt=$((attempt + 1))
    password=$(random_material | cut -c "1-$PASSWORD_LENGTH")
    if [ "${#password}" -ge "$PASSWORD_LENGTH" ]; then
      printf '%s\n' "$password"
      return 0
    fi
  done
  die "cannot generate a strong password"
}

# ---------------------------------------------------------------------------
# The options
# ---------------------------------------------------------------------------

default_root() {
  local script_dir
  script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
  printf '%s\n' "$(dirname "$script_dir")"
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --root)
        if [ "$#" -lt 2 ]; then
          die_usage "--root needs a directory"
        fi
        OPT_ROOT="$2"
        shift 2
        ;;
      --root=*)
        OPT_ROOT="${1#*=}"
        shift
        ;;
      --stack)
        if [ "$#" -lt 2 ]; then
          die_usage "--stack needs a path"
        fi
        OPT_STACKS+=("$2")
        shift 2
        ;;
      --stack=*)
        OPT_STACKS+=("${1#*=}")
        shift
        ;;
      --password)
        if [ "$#" -lt 2 ]; then
          die_usage "--password needs a mode"
        fi
        OPT_PASSWORD="$2"
        shift 2
        ;;
      --password=*)
        OPT_PASSWORD="${1#*=}"
        shift
        ;;
      --yes)
        OPT_YES=1
        shift
        ;;
      --dry-run)
        OPT_DRY_RUN=1
        shift
        ;;
      --help|-h)
        print_help
        exit "$EXIT_OK"
        ;;
      *)
        die_usage "unknown option: $1"
        ;;
    esac
  done

  case "$OPT_PASSWORD" in
    ask|generate|keep) ;;
    *) die_usage "unknown password mode: $OPT_PASSWORD (ask, generate or keep)" ;;
  esac
}

resolve_root() {
  if [ ! -d "$OPT_ROOT" ]; then
    die "the root [$OPT_ROOT] is not a directory"
  fi
  OPT_ROOT=$(CDPATH='' cd -- "$OPT_ROOT" && pwd -P)
}

# ---------------------------------------------------------------------------
# The discovery
# ---------------------------------------------------------------------------

# find_manifests
#
# Prints the manifest of every stack of the repository: every file named
# `env.example`. The metadata of the repository is ignored.
find_manifests() {
  find "$OPT_ROOT" -name '.git' -prune -o -name 'env.example' -type f -print | LC_ALL=C sort
}

# resolve_stack_manifest STACK
#
# Prints the manifest of one stack. The stack is a directory of the root, or
# the manifest itself. A stack outside of the root is refused.
resolve_stack_manifest() {
  local stack="$1"
  local candidate
  local dir

  case "$stack" in
    /*) candidate="$stack" ;;
    *) candidate="$OPT_ROOT/$stack" ;;
  esac

  if [ -d "$candidate" ]; then
    candidate="$candidate/env.example"
  fi
  if [ ! -f "$candidate" ]; then
    die "there is no manifest for the stack [$stack]"
  fi

  dir=$(CDPATH='' cd -- "$(dirname -- "$candidate")" && pwd -P)
  if ! is_within_root "$dir"; then
    die "the stack [$stack] is outside of the root [$OPT_ROOT]"
  fi

  printf '%s\n' "$dir/env.example"
}

# ---------------------------------------------------------------------------
# The parsing
# ---------------------------------------------------------------------------

# parse_manifest MANIFEST
#
# Reads the manifest and fills the MF_* arrays: the keys in the order of the
# file, their value, their description and the two directives.
#
# A directive applies to the key that follows it, and the comments that are
# not directives are the description of that key. A blank line ends the
# block of comments.
parse_manifest() {
  local manifest="$1"
  local line
  local text
  local key
  local value
  local description=()
  local secret=0
  local generate=0

  MF_KEYS=()
  MF_VALUES=()
  MF_DESCRIPTIONS=()
  MF_SECRET=()
  MF_GENERATE=()
  MF_ASKED=()

  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"
    case "$line" in
      '#'*)
        text=$(trim "${line#\#}")
        case "$text" in
          '@secret') secret=1 ;;
          '@generate') generate=1 ;;
          *) description+=("$text") ;;
        esac
        continue
        ;;
    esac
    case "$line" in
      '')
        description=()
        secret=0
        generate=0
        continue
        ;;
      *=*)
        key=$(trim "${line%%=*}")
        value="${line#*=}"
        MF_KEYS+=("$key")
        MF_VALUES+=("$value")
        MF_DESCRIPTIONS+=("${description[*]}")
        MF_SECRET+=("$secret")
        MF_GENERATE+=("$generate")
        description=()
        secret=0
        generate=0
        continue
        ;;
    esac
  done < "$manifest"
}

# read_env_file FILE
#
# Reads the `.env` that is already on disk: the keys in the order of the
# file, with their value. The comments and the blank lines are ignored.
read_env_file() {
  local file="$1"
  local line
  local key
  local value

  ENV_KEYS=()
  ENV_VALUES=()

  if [ ! -f "$file" ]; then
    return 0
  fi

  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"
    case "$line" in
      '#'*|'') continue ;;
      *=*)
        key=$(trim "${line%%=*}")
        value="${line#*=}"
        ENV_KEYS+=("$key")
        ENV_VALUES+=("$value")
        continue ;;
    esac
  done < "$file"
}

# env_lookup KEY
#
# Prints the value of a key of the `.env` that is already on disk. It fails
# when the file does not hold the key.
env_lookup() {
  local wanted="$1"
  local index
  for index in "${!ENV_KEYS[@]}"; do
    if [ "${ENV_KEYS[$index]}" = "$wanted" ]; then
      printf '%s' "${ENV_VALUES[$index]}"
      return 0
    fi
  done
  return 1
}

# manifest_declares KEY
#
# Succeeds when the manifest declares the key.
manifest_declares() {
  local wanted="$1"
  local index
  for index in "${!MF_KEYS[@]}"; do
    if [ "${MF_KEYS[$index]}" = "$wanted" ]; then
      return 0
    fi
  done
  return 1
}

# ---------------------------------------------------------------------------
# The questions
# ---------------------------------------------------------------------------

# should_ask SECRET
#
# Succeeds when the value of the key is asked to the person.
should_ask() {
  local secret="$1"
  if [ "$OPT_YES" -eq 1 ]; then
    return 1
  fi
  if [ "$NO_INPUT" -eq 1 ]; then
    return 1
  fi
  if [ "$secret" -eq 1 ] && [ "$OPT_PASSWORD" != 'ask' ]; then
    return 1
  fi
  return 0
}

# ask_value KEY DESCRIPTION DEFAULT SECRET GENERATE SOURCE
#
# Asks the value of one key and keeps the answer in ANSWER. It fails when
# the standard input is over.
ask_value() {
  local key="$1"
  local explanation="$2"
  local default_value="$3"
  local secret="$4"
  local generate="$5"
  local source="$6"
  local answer=''
  local offered

  if [ -n "$explanation" ]; then
    printf '%s\n' "$explanation"
  fi

  if [ "$secret" -eq 1 ]; then
    case "$source" in
      configured) offered='the current value is kept' ;;
      generated) offered='a strong password is offered' ;;
      *) offered='the value of the manifest is offered' ;;
    esac
    if [ "$generate" -eq 1 ]; then
      offered="$offered; answer g to generate another one"
    fi
    printf '%s' "$key [secret: $offered]: "
    if ! IFS= read -r -s answer; then
      printf '\n'
      return 1
    fi
    printf '\n'
  else
    printf '%s' "$key [$default_value]: "
    if ! IFS= read -r answer; then
      printf '\n'
      return 1
    fi
  fi

  ANSWER="$answer"
  return 0
}

# ---------------------------------------------------------------------------
# The resolution of the values
# ---------------------------------------------------------------------------

# resolve_values
#
# Fills VALUES with the value of every key of the manifest, in the order of
# the manifest: the value already configured, else the value of the
# manifest, else a strong password for a secret that can be generated.
resolve_values() {
  local index
  local key
  local manifest_value
  local existing
  local configured
  local secret
  local generate
  local value
  local source
  local answer
  local asked
  local existing_bare

  VALUES=()
  MF_ASKED=()

  for index in "${!MF_KEYS[@]}"; do
    key="${MF_KEYS[$index]}"
    manifest_value="${MF_VALUES[$index]}"
    secret="${MF_SECRET[$index]}"
    generate="${MF_GENERATE[$index]}"

    configured=0
    existing=''
    if existing=$(env_lookup "$key"); then
      configured=1
    fi

    # The bare value is the value as Compose reads it: without the spaces
    # and without the quotes that wrap it.
    existing_bare=$(bare_value "$existing")

    # An empty value is no value: the file was left half written, or the key
    # was emptied by hand. The default of the manifest applies again, and a
    # secret gets a strong password instead of nothing.
    if [ "$configured" -eq 1 ] && [ -z "$existing_bare" ]; then
      warn "$key is empty in the environment of the stack: the default is used instead"
      configured=0
      existing=''
      existing_bare=''
    fi

    # The value of the manifest is not a configured value: the placeholder
    # of a secret is never kept silently, not even when the file that holds
    # it was copied from the manifest.
    if [ "$secret" -eq 1 ] && [ "$configured" -eq 1 ] && [ "$existing_bare" = "$manifest_value" ]; then
      configured=0
      existing=''
      existing_bare=''
    fi

    if [ "$configured" -eq 1 ]; then
      value="$existing"
      source='configured'
    else
      value="$manifest_value"
      source='manifest'
    fi

    if [ "$secret" -eq 1 ]; then
      if [ "$generate" -eq 1 ]; then
        if [ "$configured" -eq 0 ]; then
          if [ "$OPT_PASSWORD" = 'keep' ]; then
            warn "the password of $key is the weak placeholder of the manifest: change it before you use the stack"
          else
            value="$(generate_password)"
            source='generated'
          fi
        fi
      elif [ "$configured" -eq 0 ]; then
        warn "$key is a secret: the value of the manifest is a placeholder, change it before you use the stack"
      fi
    fi

    asked=0
    if should_ask "$secret"; then
      asked=1
      if ask_value "$key" "${MF_DESCRIPTIONS[$index]}" "$value" "$secret" "$generate" "$source"; then
        answer="$ANSWER"
        if [ -n "$answer" ]; then
          if [ "$answer" = 'g' ] && [ "$generate" -eq 1 ]; then
            value="$(generate_password)"
            source='generated'
          else
            value="$answer"
            source='configured'
          fi
        fi
      else
        NO_INPUT=1
        asked=0
      fi
    fi

    MF_ASKED+=("$asked")
    VALUES+=("$value")
  done
}

# collect_extra_keys
#
# Fills EXTRA_KEYS with the keys of the `.env` that the manifest does not
# declare: the ones that were configured by hand. They are kept.
collect_extra_keys() {
  local index
  local key

  EXTRA_KEYS=()
  EXTRA_VALUES=()

  for index in "${!ENV_KEYS[@]}"; do
    key="${ENV_KEYS[$index]}"
    if ! manifest_declares "$key"; then
      EXTRA_KEYS+=("$key")
      EXTRA_VALUES+=("${ENV_VALUES[$index]}")
    fi
  done
}

# ---------------------------------------------------------------------------
# The file
# ---------------------------------------------------------------------------

# build_content
#
# Builds the content of the file: the header, the keys of the manifest in
# their order, then the keys that were configured by hand.
build_content() {
  local index
  local lines=()

  lines+=("# Generated by scripts/setup-env.sh. Do not edit it by hand.")
  lines+=("# Source: $MANIFEST_REL")
  lines+=("# Never commit this file: it holds the secrets of the stack.")

  for index in "${!MF_KEYS[@]}"; do
    lines+=("${MF_KEYS[$index]}=${VALUES[$index]}")
  done
  for index in "${!EXTRA_KEYS[@]}"; do
    lines+=("${EXTRA_KEYS[$index]}=${EXTRA_VALUES[$index]}")
  done

  CONTENT=$(printf '%s\n' "${lines[@]}")
}

# report_values
#
# Prints what the file holds. The secrets are hidden.
report_values() {
  local index
  printf '  %s\n' "$TARGET_REL"
  for index in "${!MF_KEYS[@]}"; do
    # A key that was asked carries its description in the question itself.
    # A key that nobody was asked about is reported with it.
    if [ "${MF_ASKED[$index]}" -eq 0 ] && [ -n "${MF_DESCRIPTIONS[$index]}" ]; then
      printf '    %s\n' "${MF_DESCRIPTIONS[$index]}"
    fi
    if [ "${MF_SECRET[$index]}" -eq 1 ]; then
      printf '    %s=%s\n' "${MF_KEYS[$index]}" "$MASK"
    else
      printf '    %s=%s\n' "${MF_KEYS[$index]}" "${VALUES[$index]}"
    fi
  done
  for index in "${!EXTRA_KEYS[@]}"; do
    printf '    %s (kept as it is)\n' "${EXTRA_KEYS[$index]}"
  done
}

# write_target
#
# Writes the file: a temporary file in the same directory, renamed over the
# target. A file whose content did not change is not written again.
write_target() {
  local dir
  local tmp
  local expected_mode
  local current_mode
  local resolved
  local resolved_dir

  dir=$(dirname "$TARGET")

  if [ "$OPT_DRY_RUN" -eq 1 ]; then
    printf '  would write %s\n' "$TARGET_REL"
    return 0
  fi

  expected_mode='644'
  if [ "$HAS_SECRET" -eq 1 ]; then
    expected_mode='600'
  fi

  tmp=$(mktemp "$dir/.env.tmp.XXXXXX") || die "cannot create a temporary file in [$dir]"
  TMP_FILE="$tmp"

  printf '%s\n' "$CONTENT" > "$tmp" || die "cannot write the temporary file [$tmp]"
  chmod "$expected_mode" "$tmp" || die "cannot set the mode of [$tmp]"

  if [ -f "$TARGET" ] && cmp -s "$tmp" "$TARGET"; then
    rm -f "$tmp"
    TMP_FILE=''
    # A link is resolved: the file it points at can live outside of the
    # root, and the installer writes nothing outside of it.
    resolved=$(resolve_link "$TARGET")
    resolved_dir=$(CDPATH='' cd -- "$(dirname "$resolved")" && pwd -P)
    resolved="$resolved_dir/$(basename "$resolved")"
    if ! is_within_root "$resolved"; then
      printf '  unchanged %s (a link outside of the root)\n' "$TARGET_REL"
      return 0
    fi
    current_mode=$(file_mode_of "$resolved")
    if [ "$current_mode" != "$expected_mode" ]; then
      chmod "$expected_mode" "$resolved" || die "cannot set the mode of [$resolved]"
      printf '  fixed the mode of %s\n' "$TARGET_REL"
    else
      printf '  unchanged %s\n' "$TARGET_REL"
    fi
    return 0
  fi

  mv -f "$tmp" "$TARGET" || die "cannot write [$TARGET]"
  TMP_FILE=''
  WRITTEN=$((WRITTEN + 1))
  printf '  wrote %s\n' "$TARGET_REL"
}

# ---------------------------------------------------------------------------
# The stack
# ---------------------------------------------------------------------------

# process_manifest MANIFEST
#
# Installs one stack: the manifest is read, the values are resolved and the
# `.env` is written next to it.
process_manifest() {
  MANIFEST="$1"
  MANIFEST_REL="${MANIFEST#"$OPT_ROOT"/}"
  TARGET="$(dirname "$MANIFEST")/.env"
  TARGET_REL="${TARGET#"$OPT_ROOT"/}"

  parse_manifest "$MANIFEST"
  read_env_file "$TARGET"

  printf '==> %s\n' "$MANIFEST_REL"

  HAS_SECRET=0
  local index
  for index in "${!MF_KEYS[@]}"; do
    if [ "${MF_SECRET[$index]}" -eq 1 ]; then
      HAS_SECRET=1
    fi
  done

  resolve_values
  collect_extra_keys

  # A key that the manifest does not declare may be a secret: the file that
  # holds it is private either way.
  if [ "${#EXTRA_KEYS[@]}" -gt 0 ]; then
    HAS_SECRET=1
  fi

  build_content

  report_values
  write_target
}

# ---------------------------------------------------------------------------
# The run
# ---------------------------------------------------------------------------

main() {
  local manifest
  local stack
  local manifests=()

  parse_args "$@"

  if [ -z "$OPT_ROOT" ]; then
    OPT_ROOT=$(default_root)
  fi
  resolve_root "$OPT_ROOT"

  # The manifests are collected first: the questions of the interactive mode
  # read the standard input, and the standard input must never be the list
  # of the manifests.
  if [ "${#OPT_STACKS[@]}" -gt 0 ]; then
    for stack in "${OPT_STACKS[@]}"; do
      manifest=$(resolve_stack_manifest "$stack")
      manifests+=("$manifest")
    done
  else
    while IFS= read -r manifest; do
      if [ -n "$manifest" ]; then
        manifests+=("$manifest")
      fi
    done < <(find_manifests)
  fi

  for manifest in ${manifests[@]+"${manifests[@]}"}; do
    process_manifest "$manifest"
  done

  printf '%s file(s) written.\n' "$WRITTEN"
  return "$EXIT_OK"
}

main "$@"
