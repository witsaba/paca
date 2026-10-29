#!/usr/bin/env bash
#
# The detection of the files a run carries, and the jobs of the continuous
# integration that need them.
#
# GitHub Actions filters the files of a run only at the level of the
# workflow: the built-in `paths` filter of the trigger decides whether the
# whole workflow runs, never whether a single job runs. To skip the jobs
# that a change cannot affect, the workflow computes the changed files with
# this script, and the rules of scripts/ci/job-filters.txt say which job
# each file needs.
#
# The rules live in the repository, in one file, so the script and the tests
# of the filters read the same truth: a rule that changes is a change of
# the code, never a comment of the workflow.
#
# Usage:
#
#   scripts/ci/changed-jobs.sh --filters <file> --base <ref> --head <ref> \
#                              [--event <name>]
#   scripts/ci/changed-jobs.sh --filters <file> --files <file>
#
# The standard output carries one `job=true|false` line per job: those are
# the outputs of the step of the workflow. The standard error carries the
# report of the run: the files it saw, and the job each one needs.
#
# When the base of the comparison cannot be read, every job is needed: a
# missing proof is worse than a slow run.

set -u

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P)
REPO_ROOT=$(dirname "$(dirname "$SCRIPT_DIR")")

FILTERS=""
BASE=""
HEAD=""
EVENT=""
FILES=""

usage() {
  printf 'usage: %s --filters <file> (--base <ref> --head <ref> | --files <file>)\n' "$0" >&2
}

while [ $# -gt 0 ]; do
  case "$1" in
    --filters) FILTERS="${2:-}"; shift 2 ;;
    --base) BASE="${2:-}"; shift 2 ;;
    --head) HEAD="${2:-}"; shift 2 ;;
    --event) EVENT="${2:-}"; shift 2 ;;
    --files) FILES="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; usage; exit 2 ;;
  esac
done

if [ -z "$FILTERS" ] || [ ! -f "$FILTERS" ]; then
  printf 'the rules of the filters are needed: %s\n' "${FILTERS:-<none>}" >&2
  exit 2
fi

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT
RULES_FILE="$WORK_DIR/rules"
CHANGED_FILE="$WORK_DIR/changed"

# ---------------------------------------------------------------------------
# The rules
# ---------------------------------------------------------------------------

# The jobs of the workflow, in the order the rules declare them.
JOBS=""

while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    ''|'#'*) continue ;;
  esac

  job=${line%% *}
  pattern=${line#* }

  if [ "$pattern" = "$line" ]; then
    printf 'a rule without a pattern: %s\n' "$line" >&2
    exit 2
  fi

  if [ "$job" != "all" ]; then
    case " $JOBS " in
      *" $job "*) ;;
      *) JOBS="$JOBS $job" ;;
    esac
  fi

  printf '%s\t%s\n' "$job" "$pattern" >> "$RULES_FILE"
done < "$FILTERS"

if [ -z "$JOBS" ]; then
  printf 'the rules of the filters name no job: %s\n' "$FILTERS" >&2
  exit 2
fi

# ---------------------------------------------------------------------------
# The files of the change
# ---------------------------------------------------------------------------

EVERYTHING=0

if [ -n "$FILES" ]; then
  if [ ! -f "$FILES" ]; then
    printf 'the list of the files is not here: %s\n' "$FILES" >&2
    exit 2
  fi
  cat "$FILES" > "$CHANGED_FILE"
elif [ "$EVENT" = "workflow_dispatch" ]; then
  # A manual run carries no change to compare: every job runs.
  printf 'a manual run carries no change: every job runs.\n' >&2
  EVERYTHING=1
  : > "$CHANGED_FILE"
else
  if [ -z "$BASE" ] || [ -z "$HEAD" ]; then
    printf 'the change to compare is not given: every job runs.\n' >&2
    EVERYTHING=1
  elif ! git -C "$REPO_ROOT" cat-file -e "${BASE}^{commit}" 2>/dev/null; then
    printf 'the base of the comparison is not here: %s\n' "$BASE" >&2
    EVERYTHING=1
  elif ! git -C "$REPO_ROOT" diff --name-only "$BASE" "$HEAD" > "$CHANGED_FILE" 2>/dev/null; then
    printf 'the change cannot be read: every job runs.\n' >&2
    EVERYTHING=1
  fi

  if [ "$EVERYTHING" = "1" ]; then
    : > "$CHANGED_FILE"
  fi
fi

# ---------------------------------------------------------------------------
# The patterns
# ---------------------------------------------------------------------------

# The translation of a pattern of the filters into the regular expression
# that matches a path of the repository.
pattern_to_regex() {
  local pattern="$1"
  local out=""
  local index=0
  local length=${#pattern}
  local char
  local BACKSLASH="\\"

  while [ "$index" -lt "$length" ]; do
    char=${pattern:$index:1}
    case "$char" in
      '*')
        if [ "${pattern:$((index + 1)):1}" = "*" ]; then
          if [ "${pattern:$((index + 2)):1}" = "/" ]; then
            # "**/" matches any number of directories, none included.
            out="${out}(.*/)?"
            index=$((index + 3))
          else
            # "**" matches every character, the slash included.
            out="${out}.*"
            index=$((index + 2))
          fi
        else
          # "*" matches every character, the slash except.
          out="${out}[^/]*"
          index=$((index + 1))
        fi
        ;;
      '?')
        # "?" matches one character, the slash except.
        out="${out}[^/]"
        index=$((index + 1))
        ;;
      '.')
        # The characters that are regular expressions on their own are
        # escaped, so a path is matched literally.
        out="${out}${BACKSLASH}${char}"
        index=$((index + 1))
        ;;
      *)
        case "$char" in
          [a-zA-Z0-9/_-]) out="${out}${char}" ;;
          *) out="${out}${BACKSLASH}${char}" ;;
        esac
        index=$((index + 1))
        ;;
    esac
  done

  printf '%s' "$out"
}

# The patterns of a job, one per line.
patterns_of() {
  local job="$1"
  local rule
  local pattern
  while IFS=$'\t' read -r rule pattern; do
    if [ "$rule" = "$job" ]; then
      printf '%s\n' "$pattern"
    fi
  done < "$RULES_FILE"
}

# Whether a path is matched by a pattern of the filters.
matches() {
  local file="$1"
  local pattern="$2"
  local regex
  regex=$(pattern_to_regex "$pattern")
  if [[ "$file" =~ ^${regex}$ ]]; then
    return 0
  fi
  return 1
}

# ---------------------------------------------------------------------------
# The decision
# ---------------------------------------------------------------------------

if [ "$EVERYTHING" = "1" ]; then
  for job in $JOBS; do
    printf '%s=true\n' "$job"
  done
  exit 0
fi

printf 'the change carries %s file(s):\n' "$(grep -c . "$CHANGED_FILE" || true)" >&2
if [ -s "$CHANGED_FILE" ]; then
  sed 's/^/  /' "$CHANGED_FILE" >&2
fi

NEEDED=""
for job in $JOBS; do
  needed=0
  why=""

  while IFS= read -r pattern; do
    [ -n "$pattern" ] || continue
    while IFS= read -r file; do
      [ -n "$file" ] || continue
      if matches "$file" "$pattern"; then
        needed=1
        why="${why}
    ${file} matches ${pattern}"
        break
      fi
    done < "$CHANGED_FILE"
    [ "$needed" = "1" ] && break
  done < <(patterns_of "$job")

  if [ "$needed" = "0" ]; then
    # The machinery of the filters is the rule of every job.
    while IFS= read -r pattern; do
      [ -n "$pattern" ] || continue
      while IFS= read -r file; do
        [ -n "$file" ] || continue
        if matches "$file" "$pattern"; then
          needed=1
          why="${why}
    ${file} matches ${pattern} (the machinery of the filters)"
          break
        fi
      done < "$CHANGED_FILE"
      [ "$needed" = "1" ] && break
    done < <(patterns_of "all")
  fi

  if [ "$needed" = "1" ]; then
    printf '%s=true\n' "$job"
    NEEDED="${NEEDED} ${job}"
    printf 'the job %s runs:%s\n' "$job" "$why" >&2
  else
    printf '%s=false\n' "$job"
    printf 'the job %s is skipped: no file of the change can affect what it proves.\n' "$job" >&2
  fi
done

printf 'the jobs of this run:%s\n' "$NEEDED" >&2
