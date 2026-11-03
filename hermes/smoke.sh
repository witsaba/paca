#!/bin/sh
#
# The proof of the stack of the agent: the agent is alive, it is Paca, and it
# reaches the memory of the monorepository over the stdio MCP transport.
#
# It is a round trip and not a heartbeat: a server that answers does not
# prove that the memory works. The script runs inside the container of the
# agent, so it reads the state of the memory and the project of the
# monorepository from the environment of the container, and it writes no
# credential anywhere.
#
# The identity of the agent is proved too: the state of the agent carries
# the soul of Paca, and the soul names Paca and not the runtime. The
# credentials of the channel are proved by their presence and never printed.
#
# The record of the proof is removed at the end: the proof leaves no
# memory behind.

set -eu

fail() {
  printf 'hermes: %s\n' "$1" >&2
  exit 1
}

# The identity of the agent is Paca. The runtime reads it from
# `$HERMES_HOME/SOUL.md` and puts it in the first slot of the system prompt,
# so the file is the identity and not one layer of it.
soul="${HERMES_HOME:-/opt/data}/SOUL.md"

[ -f "${soul}" ] || fail "the state of the agent does not carry the soul of Paca"

grep -q "You are Paca" "${soul}" ||
  fail "the identity of the agent is not Paca"

if grep -q "You are Hermes" "${soul}"; then
  fail "the identity of the agent still names the runtime"
fi

# The channel of Paca reached the container: the token of the bot and the
# list of the users it answers to. The values are never printed.
[ -n "${TELEGRAM_BOT_TOKEN:-}" ] ||
  fail "the token of the Telegram bot did not reach the container"

[ -n "${TELEGRAM_ALLOWED_USERS:-}" ] ||
  fail "the list of the users of Telegram did not reach the container"

# The agent is alive: the API server of the agent answers on the loopback
# interface of the container.
curl -fsS -m 5 -o /dev/null http://127.0.0.1:8642/health ||
  fail "the health endpoint of the agent does not answer"

# The memory works: the MCP server of the store is spawned the way the
# agent spawns it, a memory is written through it and read back through it.
python3 /usr/local/bin/paca-hermes-mcp-roundtrip.py ||
  fail "the agent does not reach the memory over the stdio MCP"

printf 'hermes: the agent is alive, it is Paca and it reaches the memory\n'
