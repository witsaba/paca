#!/bin/sh
# The entrypoint of the image of the memory.
#
# The state of the store lives in the volume of the stack, and Docker creates
# that volume owned by root. The store writes the database of the memories
# into it, and the socket of the API is created there as well: the socket
# belongs to the user of the process, and Engram keeps it private (0600) on
# purpose. The consumer of the socket is the container of the agent, which
# runs as its own user, so the store runs as that same user and the socket
# is reachable by it.
#
# The privileges of the container are dropped before the store runs: the
# process that writes the state is not root.
set -eu

# The state of the store is owned by the user of the process, so the files of
# the database are not written by root.
chown -R 10000:10000 /data

exec su-exec 10000:10000 "${ENGRAM_BIN:-/opt/engram/engram}" "$@"
