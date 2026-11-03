#!/bin/sh
#
# The seeding of the state of the agent.
#
# This script is the one-shot service of the stack of the agent: it runs
# once, before the gateway comes up, and it prepares the state that the
# agent finds when it starts. Five things happen here, in this order.
#
# 1. The credentials of the stack are checked. The agent cannot call a model
#    without the key of the provider, and it cannot answer anyone without the
#    channel: Telegram, the first channel of Paca. The stack refuses to come
#    up when one of them is missing: an agent that is there and cannot answer
#    anyone is worse than an agent that is not there at all. The refusal
#    names the command that writes the credentials, because the credentials
#    belong to the DevOps engineer who runs the project.
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
# 5. The soul of Paca is written into the volume of the state of the agent.
#    The runtime reads it from `$HERMES_HOME/SOUL.md` and puts it in the
#    first slot of the system prompt, so it is the identity of the agent and
#    not one layer of it.
#
#    The runtime never overwrites a soul the maintainer wrote, and neither
#    does this seeding: the soul of the volume is replaced while it holds a
#    soul this stack seeded — the identity of the runtime, the soul of the
#    image, or a soul of a previous version of the image, which the record
#    of the hash tells apart — and it is kept the moment it is edited.
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

# The channel of Paca. The token is the bot of the owner of the business,
# and the users are the people the bot answers to: without the list, the
# gateway of the runtime denies every user, so both are required.
if [ -z "${TELEGRAM_BOT_TOKEN:-}" ]; then
  printf 'hermes: TELEGRAM_BOT_TOKEN is not configured.\n' >&2
  printf 'hermes: Paca cannot reach Telegram without the token of the bot. Write it with:\n' >&2
  printf 'hermes:   make env-init --stack hermes\n' >&2
  exit 1
fi

if [ -z "${TELEGRAM_ALLOWED_USERS:-}" ]; then
  printf 'hermes: TELEGRAM_ALLOWED_USERS is not configured.\n' >&2
  printf 'hermes: Paca answers nobody without the list of the users. Write it with:\n' >&2
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

# The skin of Paca. The runtime brands the terminal of the agent with its
# own name, and the file is seeded into the state of the agent before the
# configuration points at it.
skin_source="/opt/hermes/skin.yaml"
skin_dir="${home}/skins"
skin_file="${skin_dir}/paca.yaml"

if [ ! -f "${skin_source}" ]; then
  printf 'hermes: the skin of Paca is missing from the image: %s\n' "${skin_source}" >&2
  exit 1
fi

mkdir -p "${skin_dir}"
cp "${skin_source}" "${skin_file}"
chown -R 10000:10000 "${skin_dir}" || true
chmod 0644 "${skin_file}" || true

hermes config set display.skin paca
hermes config set mcp_servers.engram.command /opt/engram/engram
hermes config set mcp_servers.engram.args '["mcp","--tools=agent"]'
hermes config set mcp_servers.engram.env "{\"ENGRAM_DATA_DIR\":\"${memory_dir}\",\"ENGRAM_PROJECT\":\"${project}\"}"
hermes config set mcp_servers.engram.enabled true

printf 'hermes: the configuration of the agent is seeded (model %s, memory %s)\n' "${model}" "${project}"

# soul_is_a_seeded_identity answers whether the soul of the state of the
# agent is one that this stack seeded, and nobody edited.
#
# Three cases are ours: the identity the runtime seeds by itself, the soul of
# the image, and the soul that a previous start of the stack seeded — the
# record of the hash tells the last one apart from a soul the maintainer
# wrote.
#
# Anything else is the work of the maintainer, and it is kept.
soul_is_a_seeded_identity() {
  SOUL_PATH="$1" SOUL_SOURCE="$2" SOUL_RECORD="$3" python3 -c '
import hashlib
import os
import sys

from hermes_cli import default_soul


def normalize(text):
    return text.replace("\r\n", "\n").replace("\r", "\n").lstrip("\ufeff").strip()


def digest(text):
    return hashlib.sha256(normalize(text).encode("utf-8")).hexdigest()


def read(path):
    try:
        with open(path, encoding="utf-8") as handle:
            return normalize(handle.read())
    except OSError:
        return None


current = read(os.environ["SOUL_PATH"])
if current is None:
    sys.exit(1)

templates = [default_soul.DEFAULT_SOUL_MD]
templates += list(getattr(default_soul, "_LEGACY_TEMPLATE_SOULS", ()))
if any(current == normalize(t) for t in templates):
    sys.exit(0)

source = read(os.environ["SOUL_SOURCE"])
if source is not None and current == source:
    sys.exit(0)

recorded = read(os.environ["SOUL_RECORD"])
if recorded is not None and recorded.strip() == digest(current):
    sys.exit(0)

sys.exit(1)
'
}

# soul_digest prints the hash of a soul, the record that tells a soul this
# stack seeded apart from a soul the maintainer wrote.
soul_digest() {
  SOUL_PATH="$1" python3 -c '
import hashlib
import os
import sys

with open(os.environ["SOUL_PATH"], encoding="utf-8") as handle:
    text = handle.read()

text = text.replace("\r\n", "\n").replace("\r", "\n").lstrip("\ufeff").strip()
sys.stdout.write(hashlib.sha256(text.encode("utf-8")).hexdigest())
'
}

# 5. The soul of Paca. The file of the image is the source; the file of the
#    volume is the identity the agent reads.
soul_source="/opt/hermes/soul.md"
soul="${home}/SOUL.md"
soul_record="${home}/.paca-soul.sha256"

if [ ! -f "${soul_source}" ]; then
  printf 'hermes: the soul of Paca is missing from the image: %s\n' "${soul_source}" >&2
  exit 1
fi

if [ -f "${soul}" ] && ! soul_is_a_seeded_identity "${soul}" "${soul_source}" "${soul_record}"; then
  printf 'hermes: the identity of the agent is kept: %s was edited\n' "${soul}"
else
  cp "${soul_source}" "${soul}"
  soul_digest "${soul}" >"${soul_record}"
  chown 10000:10000 "${soul}" "${soul_record}" || true
  chmod 0644 "${soul}" || true
  chmod 0600 "${soul_record}" || true
  printf 'hermes: the identity of the agent is seeded (Paca)\n'
fi
