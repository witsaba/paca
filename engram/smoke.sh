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

fail() {
  printf 'engram: %s\n' "$1" >&2
  exit 1
}

# request METHOD PATH [BODY]
#
# The token of the stack guards the writes that destroy, so it travels with
# every request when the stack has one.
request() {
  method="$1"
  path="$2"
  body="${3:-}"
  set -- --unix-socket "$socket" -X "$method" -H "Accept: application/json"
  if [ -n "${ENGRAM_HTTP_TOKEN:-}" ]; then
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

# The record of the proof is removed: the proof leaves no memory behind. The
# deletion is a hard one, because a session that still holds observations
# cannot be removed.
request DELETE "/observations/${observation}?expected_project=${project}&hard=true" >/dev/null ||
  fail "the store does not remove the memory of the proof"
request DELETE "/sessions/${session}" >/dev/null ||
  fail "the store does not remove the session of the proof"

printf 'engram: the memory is kept and given back\n'
