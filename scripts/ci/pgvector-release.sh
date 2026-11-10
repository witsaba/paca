#!/usr/bin/env bash
#
# The keeper of the extension: whether the pinned pgvector is behind the
# latest release.
#
# Usage: scripts/ci/pgvector-release.sh <pinned> <latest>
#
# Prints "behind" when the pinned version is older than the latest one, and
# "current" otherwise. The decision is a pure function of the two arguments:
# the network never reaches it, so the test proves it without one.
#
# Exit code 0 means the decision was made; 2 means the arguments are not
# versions.

set -eu

usage() {
  printf 'usage: %s <pinned> <latest>\n' "$0" >&2
  exit 2
}

[ "$#" -eq 2 ] || usage

# The tag of a release carries a `v`; the pin of the Dockerfile does not.
pinned=${1#v}
latest=${2#v}

# A version is a sequence of dot-separated numbers.
is_version() {
  [[ "$1" =~ ^[0-9]+(\.[0-9]+)*$ ]]
}

if ! is_version "$pinned"; then
  printf 'not a version: %s\n' "$1" >&2
  exit 2
fi

if ! is_version "$latest"; then
  printf 'not a version: %s\n' "$2" >&2
  exit 2
fi

# The components are compared as numbers, and the shorter version is padded
# with zeros: 0.8.10 is newer than 0.8.9, and 0.8 is the same as 0.8.0.
IFS=. read -r -a pinned_parts <<< "$pinned"
IFS=. read -r -a latest_parts <<< "$latest"

length=${#pinned_parts[@]}
if [ "${#latest_parts[@]}" -gt "$length" ]; then
  length=${#latest_parts[@]}
fi

for ((index = 0; index < length; index++)); do
  pinned_part=${pinned_parts[index]:-0}
  latest_part=${latest_parts[index]:-0}

  if ((10#$pinned_part < 10#$latest_part)); then
    printf 'behind\n'
    exit 0
  fi

  if ((10#$pinned_part > 10#$latest_part)); then
    printf 'current\n'
    exit 0
  fi
done

printf 'current\n'
