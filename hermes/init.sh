#!/bin/sh
#
# The seeding of the state of the agent.
#
# This script is the one-shot service of the stack of the agent: it runs
# once, before the gateway comes up, and it prepares the state that the
# agent finds when it starts. Five things happen here, in this order.
#
# 1. The credential of the provider of the models is checked. The agent
#    cannot call a model without it, so the stack refuses to come up when
#    it is missing: an agent that is there and cannot answer anyone is
#    worse than an agent that is not there at all. The refusal names the
#    command that writes the credential, because the credential belongs to
#    the DevOps engineer who runs the project.
# 2. The payload of the memory is waited for. The executable of the store
#    is published by the stack of the memory on the volume that is mounted
#    here read-only, and the two stacks are separate, so the agent cannot
#    depend on a service of the other one: it waits, with a bound, for the
#    artifact it needs, and it refuses the stack with a message that names
#    the command that publishes it.
# 3. The state of the agent and the state of the memory are given to the
#    user of the agent. The MCP server of the store is a process of the
#    agent, and it writes the state of the store, which lives on the
#    volume of the memory.
# 4. The configuration of the agent is written into the volume of its
#    state. The file cannot be a read-only mount: Hermes rewrites it at
#    boot (it migrates it and it keeps timestamped backups of it), so it is
#    seeded here, once, and it is left to Hermes afterwards.
#
# The keys are written with the writer of Hermes itself, so the file keeps
# the schema of the version of the image, and Hermes migrates it at boot.
#
# The service runs as the root of the container, because the ownership of
# the volumes is the one thing a process without capabilities cannot do.
# Every command of Hermes drops to the user of the agent on its own, so
# the files it writes belong to that user.
#
# The mount of the payload of the memory is read-only: this script never
# writes inside it and never changes the ownership of it.

set -eu

home="${HERMES_HOME:-/opt/data}"
config="${home}/config.yaml"
memory_dir="${ENGRAM_DATA_DIR:-/opt/engram-data}"
payload_dir="${ENGRAM_PAYLOAD_DIR:-/opt/engram}"
payload="${payload_dir}/engram"
project="${ENGRAM_PROJECT:-paca}"
model="${AGENT_MAIN_MODEL:-minimax/minimax-m3.1-flash}"

# The bound of the wait for the payload of the memory, and the pause
# between two attempts.
payload_wait_seconds="${ENGRAM_PAYLOAD_WAIT_SECONDS:-60}"
payload_wait_pause=2

# 1. The credential of the provider of the models. It comes from the `.env`
# of the stack, which the installer of the environment writes.
if [ -z "${OPENROUTER_API_KEY:-}" ]; then
  printf 'hermes: OPENROUTER_API_KEY is not configured.\n' >&2
  printf 'hermes: the agent cannot call a model without it. Write it with:\n' >&2
  printf 'hermes:   make env-init --stack hermes\n' >&2
  exit 1
fi

# 2. The payload of the memory. The executable of the store is published
# by the stack of the memory, and the agent mounts it read-only: the agent
# waits for it, and it refuses the stack when it does not arrive.
waited=0
while [ ! -x "${payload}" ]; do
  if [ "${waited}" -ge "${payload_wait_seconds}" ]; then
    printf 'hermes: the payload of the memory is not published: %s is missing or is not executable.\n' "${payload}" >&2
    printf 'hermes: the stack of the memory publishes it with:\n' >&2
    printf 'hermes:   make -C engram payload\n' >&2
    exit 1
  fi
  sleep "${payload_wait_pause}"
  waited=$((waited + payload_wait_pause))
done

# 3. The state of the agent and the state of the memory belong to the user
# of the agent: the store writes it as that user, and so does the MCP
# server of the store that the agent spawns.
chown 10000:10000 "${home}" || true
if [ -d "${memory_dir}" ]; then
  chown -R 10000:10000 "${memory_dir}" || true
fi

# 4. The configuration of the agent. The file is created the way the
# installers of Hermes create it: from the template of the image.
if [ ! -f "${config}" ]; then
  cp /opt/hermes/cli-config.yaml.example "${config}"
fi

hermes config set model.default "${model}"
hermes config set model.provider openrouter
hermes config set mcp_servers.engram.command /opt/engram/engram
hermes config set mcp_servers.engram.args '["mcp","--tools=agent"]'
hermes config set mcp_servers.engram.env "{\"ENGRAM_DATA_DIR\":\"${memory_dir}\",\"ENGRAM_PROJECT\":\"${project}\"}"
hermes config set mcp_servers.engram.enabled true

printf 'hermes: the configuration of the agent is seeded (model %s, memory %s)\n' "${model}" "${project}"
