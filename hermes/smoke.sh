#!/bin/sh
#
# The proof of the stack of the agent: the agent is alive and it reaches
# the memory of the monorepository over the stdio MCP transport.
#
# It is a round trip and not a heartbeat: a server that answers does not
# prove that the memory works. The script runs inside the container of the
# agent, so it reads the state of the memory and the project of the
# monorepository from the environment of the container, and it writes no
# credential anywhere.
#
# The record of the proof is removed at the end: the proof leaves no
# memory behind.

set -eu

fail() {
  printf 'hermes: %s\n' "$1" >&2
  exit 1
}

# The agent is alive: the API server of the agent answers on the loopback
# interface of the container.
curl -fsS -m 5 -o /dev/null http://127.0.0.1:8642/health ||
  fail "the health endpoint of the agent does not answer"

# The memory works: the MCP server of the store is spawned the way the
# agent spawns it, a memory is written through it and read back through it.
python3 /usr/local/bin/paca-hermes-mcp-roundtrip.py ||
  fail "the agent does not reach the memory over the stdio MCP"

printf 'hermes: the agent is alive and it reaches the memory\n'
