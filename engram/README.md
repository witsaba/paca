# The memory of PACA: Engram as a stack of the monorepository

The memory of the monorepository is a container that brings its own image,
like the database and the collector of the traces, so it lives in a directory
of its own with its own compose file. It is [Engram](https://github.com/Gentleman-Programming/engram),
the store of the memories of the agent and of the work of the repository.

This slice adds the store and nothing else: it keeps the memories, it
answers, and it proves it. The agent does not read it yet — the connection
of the agent to its memory is the slice that follows, and it brings its own
document.

## What you get

- **One container**, `paca-engram`, that runs the store of the memories.
- **An image built by the repository**, with the official release of Engram
  inside it, verified against the sha256 that is pinned for its
  architecture. The build fails when the bytes are not the ones that were
  pinned.
- **Persistent state**: the database of the memories lives under `/data`, on
  the named volume `paca-engram-data`, so the memories survive a restart of
  the stack.
- **No published port**: the store publishes no port at all. It answers on a
  unix socket inside its own volume, and the container that mounts that
  volume reaches it there.
- **The internal network only**: the store joins `paca_internal`, the
  service-to-service network of the monorepository, which has no outbound
  access at all.
- **A token that guards the destructive writes**: the token of the stack
  lives in `engram/.env` and the store refuses the writes that destroy
  without it. The proof of the stack asserts that guard: it fails when the
  token is missing or is still the placeholder of the manifest.

## Running it

The stack alone:

```sh
make -C engram up       # start and wait until it is healthy
make -C engram smoke    # the memory must be kept and given back
make -C engram status   # container state
make -C engram logs     # follow the logs
make -C engram down     # stop (the container is removed, the state is kept)
```

Or with the compose file directly:

```sh
docker compose -f engram/compose.yaml up --wait
```

With the monorepository, the root compose includes the stack, so one command
starts everything:

```sh
docker compose up --wait
```

Both ways drive the same compose project, `paca`: the container, the volume
and the network are the same ones whichever way the stack is started.

The stack needs its credentials before it starts: the token of the store
lives in `engram/.env`, and the installer of the environment writes it:

```sh
make env-init
```

The manual fallback of the manifest leaves the placeholder of the token in
place, and the proof of the stack fails while it is there: the token of a
stack is a secret of its own, and the installer generates it.

## The ports

There are none. The store answers on a unix socket inside the volume of the
stack, `/data/engram.sock`, and on nothing else. The local runtime of Engram
binds the loopback of its own container and refuses any other address, so
the socket is the only address that leaves the container. The container that
mounts the volume of the stack reaches the store through it, as the user
`10000` — the same user the container of the agent runs as — and the token
of the stack guards the writes that destroy.

## What is not configured yet

This slice adds the store of the memory, and nothing else:

- **The agent does not read it**: the container of the agent does not mount
  the volume of the memory, and no provider of the memory is active in the
  agent.
- **The memory is not shared**: the store is local to this deployment. The
  Git sync of Engram and the cloud of Engram are not configured.
- **The MCP server of the monorepository is not wired to the agent**: that
  is the slice of the wiring of the agent.

Those are the decisions of the slices that follow, and each of them brings
its own configuration, its own proof and its own update of this document.

## The layout

```
engram/
├── compose.yaml   # the service, the network, the volume and the healthcheck
├── Dockerfile     # the image: the official release of Engram, verified
├── entrypoint.sh  # the entrypoint of the image: the umask of the socket
├── Makefile       # the targets of this stack
├── README.md      # this file
├── env.example    # the manifest of the credentials of the stack
└── smoke.sh       # the proof of the stack: a memory kept and given back
```

## Notes and limits

- The image is built from the official release of Engram, and the checksum
  of the archive is pinned for both architectures of the repository, so the
  same image serves amd64 and arm64.
- The packages of the base image are pinned by version too: the base image
  is pinned by digest, and the packages are the ones of its release. When
  the release gets a security update, the pin is bumped with it, on
  purpose, and the build fails until it is.
- The store runs as the user `10000` inside its container, not as root: the
  entrypoint of the image gives the state of the volume to that user and
  drops the privileges of the process before the store runs. It keeps only
  the three capabilities that the drop needs, and it cannot gain more.
- The file system of the container is read-only: the database of the
  memories is the only thing the process writes, and it goes to the volume.
- The state is a named volume and never a bind mount: the database is SQLite
  in WAL mode, and WAL is not safe on a mount that crosses the boundary of
  the virtual machine — the pages of the database would be silently
  corrupted.
- The socket of the volume is private to the user of the store: Engram keeps
  it at `0600` on purpose. The store therefore runs as the same user as the
  container of the agent (`10000`), so the agent reaches the socket when it
  mounts the volume. The token of the stack is the second lock of the door.
- The proof of the stack is a round trip and not a heartbeat: it registers a
  session, keeps a memory, searches it back, asserts the bytes, refuses
  that the store accepted a destructive write without the token, and removes
  the record afterwards.
