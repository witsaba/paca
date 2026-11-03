# The image of the agent of PACA: Hermes Agent, without the payload of the
# memory.
#
# The agent reads the memory of the monorepository over the stdio MCP
# transport, and the MCP server is a process that the agent spawns: the
# binary of the store has to be at hand of the agent, exactly as the
# documentation of the store prescribes for any other MCP agent.
#
# The binary is not in this image anymore. The stack of the memory builds
# it, verifies it and publishes it on the volume `paca-engram-payload`,
# and the stack of the agent mounts that volume read-only at
# `/opt/engram`: the image of the agent knows nothing about the store.
#
# The base image is pinned by digest, like every image of the repository.
#
# The entrypoint of the base image is left alone on purpose: it starts the
# supervision tree of the agent and it reaps the orphans that the tools
# leave behind.

ARG HERMES_IMAGE=nousresearch/hermes-agent:stable@sha256:9c5561579f3ffdf333401797094ac3a743738d6c25d40f4331c7bf77b694238d

FROM ${HERMES_IMAGE}

# The seeding of the configuration of the agent: the one-shot service of
# the stack runs it before the gateway comes up.
COPY hermes/init.sh /opt/hermes/init.sh
RUN chmod 0755 /opt/hermes/init.sh

# The soul of Paca: the identity of the agent, in the file the runtime
# reads at the start of every session.
#
# The runtime reads it from `$HERMES_HOME/SOUL.md` and puts it in the
# first slot of the system prompt, so it is the identity of the agent and
# not one layer of it. The file cannot be a read-only mount, because the
# state of the agent lives in a named volume and the maintainer is allowed
# to edit the soul in that volume: the image carries the source, and the
# seeding of the stack copies it into the volume once.
COPY hermes/soul.md /opt/hermes/soul.md

# The skin of Paca: the branding of the terminal of the agent. The runtime
# brands itself with its own name, and the name is what the person sees
# every time the agent answers in the terminal.
COPY hermes/skin.yaml /opt/hermes/skin.yaml
