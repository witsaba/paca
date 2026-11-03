#!/bin/sh
#
# The proof of the stack of the memory: the store of Engram keeps a memory
# and gives it back.
#
# It is a round trip and not a heartbeat: a process that is alive does not
# prove that the memory works. The script runs inside the container, so it
# reads the socket of the volume and the token of the stack from the
# environment of the container, and it writes no credential anywhere.
#
# The record of the proof is removed at the end: the proof leaves no memory
# behind.

set -eu

socket="${ENGRAM_SOCKET:-/data/engram.sock}"
project="${ENGRAM_PROJECT:-paca}"
base="http://localhost"
# The token of the stack guards the writes that destroy. The proof asserts
# that the guard is there: a token that is missing, or the placeholder of the
# manifest, would leave the store open, and the proof must fail in that case.
token="${ENGRAM_HTTP_TOKEN:-}"
placeholder="change-me-generate-a-strong-token"
use_token=1

fail() {
  printf 'engram: %s\n' "$1" >&2
  exit 1
}

if [ -z "$token" ] || [ "$token" = "$placeholder" ]; then
  fail "the token of the stack is missing or is still the placeholder of the manifest"
fi

# request METHOD PATH [BODY]
#
# The token of the stack guards the writes that destroy, so it travels with
# every request when the stack has one.
request() {
  method="$1"
  path="$2"
  body="${3:-}"
  set -- --unix-socket "$socket" -X "$method" -H "Accept: application/json"
  if [ "$use_token" -eq 1 ] && [ -n "${ENGRAM_HTTP_TOKEN:-}" ]; then
    set -- "$@" -H "Authorization: Bearer ${ENGRAM_HTTP_TOKEN}"
  fi
  if [ -n "$body" ]; then
    set -- "$@" -H "Content-Type: application/json" --data "$body"
  fi
  curl -fsS "$@" "${base}${path}"
}

health="$(request GET /health)" ||
  fail "the health endpoint of the store does not answer"
case "$health" in
  *'"status":"ok"'*) ;;
  *) fail "the health endpoint of the store does not answer" ;;
esac

session="paca-smoke-$(date +%s)-$$"

created="$(request POST /sessions "{\"id\":\"${session}\",\"project\":\"${project}\"}")" ||
  fail "the store does not accept the session of the proof"
case "$created" in
  *'"status":"created"'*) ;;
  *) fail "the store does not accept the session of the proof" ;;
esac

saved="$(request POST /observations "{\"session_id\":\"${session}\",\"type\":\"config\",\"title\":\"paca stack smoke\",\"content\":\"the stack of the memory of PACA keeps this record and gives it back\"}")" ||
  fail "the store does not keep the memory of the proof"

observation="$(printf '%s' "$saved" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')"
if [ -z "$observation" ]; then
  fail "the store does not keep the memory of the proof"
fi

found="$(request GET "/search?q=paca+stack+smoke&project=${project}&response_format=compact")" ||
  fail "the store does not give the memory of the proof back"
case "$found" in
  *'paca stack smoke'*) ;;
  *) fail "the store does not give the memory of the proof back" ;;
esac

# The guard of the store is enforced by the store and not only by a
# convention: a destructive write without the token is refused.
use_token=0
if request DELETE "/observations/${observation}?expected_project=${project}" >/dev/null 2>&1; then
  fail "the store accepted a destructive write without the token"
fi
use_token=1

# The record of the proof is removed: the proof leaves no memory behind. The
# deletion is a hard one, because a session that still holds observations
# cannot be removed.
request DELETE "/observations/${observation}?expected_project=${project}&hard=true" >/dev/null ||
  fail "the store does not remove the memory of the proof"
request DELETE "/sessions/${session}" >/dev/null ||
  fail "the store does not remove the session of the proof"

printf 'engram: the memory is kept and given back\n'
