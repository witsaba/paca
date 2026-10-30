# The memory of PACA: Engram as a stack of the monorepository

The first slice of the memory: the store of Engram joins the monorepository as
a stack of its own, with its own image, its own state and its own proof. The
agent does not read it yet: the connection is the slice that follows.

## Specs

S1. "before continuing configuring hermes, I wanna give it the main hermes memory with engram:
https://github.com/Gentleman-Programming/engram"

S2. "Just investigate no change anything yet."

S3. "ok for now, just as same exercise with hermes, not connect yet, just add to the infra, what way do
you recommend?"

S4. "Official release tarball, sha256-verified"

S5. "Unix socket in the named volume (recommended)"

S6. "Stack + full infra wiring + gate proof"

## Tasks

- T1 — the stack of the memory: `engram/compose.yaml`, `engram/Dockerfile`,
  `engram/Makefile`, `engram/README.md`, `engram/env.example`. Route: inline.
  Commit: `PENDING`.
- T2 — the wiring of the monorepository: the root `compose.yaml`, the root
  `Makefile`, the root `README.md`, the root `AGENTS.md`,
  `.github/workflows/ci.yml`, `scripts/ci/job-filters.txt` and
  `scripts/tests/ci-filters-test.sh`. Route: delegated writer, diff read by
  the parent. Commit: `PENDING`.
- T3 — the proof: the image is built, the stack comes up healthy, the round
  trip of the memory is asserted and the gate of the repository is green.
  Route: inline. Evidence: L8. Commit: the record of this document.

## Log

L1. "before continuing configuring hermes, I wanna give it the main hermes memory with engram:
https://github.com/Gentleman-Programming/engram Just read about in the setup agent there is not a native
way with hermes, but documentation. https://github.com/Gentleman-Programming/engram/blob/main/docs/AGENT-SETUP.md#any-other-mcp-agent
I wanna you only explore and investigate how could add this as MAIN hermes memory for Paca. This is
needed, due to it'll be the main hermes memory. Just investigate no change anything yet."

L2. The investigation, and the facts it settles.

- Engram has no Hermes integration: the tree of the repository holds no
  mention of Hermes at all, and the catalog of the memory providers of Hermes
  holds RetainDB, ByteRover, Honcho, Hindsight, Mem0, OpenViking,
  Supermemory and Holographic. The "Any other MCP agent" section of the
  documentation of Engram is the pattern `engram mcp` over stdio, which gives
  the agent tools and not the role of the memory.
- The seam that does hold the role of the memory is the contract of the
  memory-provider plugin of Hermes: `MemoryProvider` (`agent/memory_provider.py`),
  discovered in `$HERMES_HOME/plugins/<name>/` and activated by the name in
  `config.yaml` (`memory.provider`). The provider receives the turns
  (`sync_turn`), the end of the session (`on_session_end`), the checkpoint
  before the compression (`on_pre_compress`), the writes of the built-in
  memory (`on_memory_write`) and returns the recall (`prefetch`).
- Engram publishes no image of the local runtime: `.goreleaser.yaml` publishes
  the binaries, and the only image of the repository is `docker/cloud/Dockerfile`,
  of the cloud runtime. The local runtime is one static binary
  (`CGO_ENABLED=0`).
- `engram serve` binds `127.0.0.1:<port>` (`internal/server/server.go`) and
  offers no other address for the local runtime: the only transport that
  leaves the container is the POSIX unix socket of `ENGRAM_SOCKET`, and it
  listens exclusively on it. `ENGRAM_HTTP_TOKEN` guards the whole handler with
  a Bearer token, and the token is optional.

L3. "ok for now, just as same exercise with hermes, not connect yet, just add to the infra, what way do
you recommend?"

L4. The recommendation, and the three decisions of the human that settle it.

- The stack is `engram/`, at the root of the monorepository, next to
  `hermes/` and `observability/`: Engram is an infrastructure service with its
  own state, not an application of Go and not a shared library.
- The image is ours and it carries the official release, verified: the build
  downloads the archive of the release, checks it against the pinned sha256 of
  its architecture and installs the static binary. The base images are pinned
  by digest, like every image of the repository.
- The transport is the unix socket of the named volume of the stack
  (`/data/engram.sock`): the stack has no TCP port at all, it runs on its own,
  and the container of the agent mounts the same volume when the connection
  comes.
- The slice carries the whole wiring of the monorepository and the proof in
  the gate.

L5. The consequences that the decisions of L4 carry, and that the slice has to
respect.

- The stack publishes no port and joins the internal network only, like every
  service of the monorepository that carries data of the people.
- The state is a named volume, never a bind mount: the database is SQLite in
  WAL mode, and WAL is silently corrupted on a mount that crosses the
  boundary of the virtual machine.
- The credentials of the stack live with the stack that owns them
  (`engram/.env`), and the installer of the environment finds the manifest of
  the stack on its own: it looks for every `env.example` of the repository.
- The socket of the volume is created by the process of the stack, and Engram
  keeps it private (`0600`) on purpose: the reachability of the socket is a
  matter of the user of the process, not of its umask. The stack therefore
  runs the store as the user `10000`, which is the user the container of the
  agent runs as, and the consumer that mounts the volume reaches the socket
  as that user. The Bearer token of the stack is the second lock of the door.
- The proof is a round trip and not a heartbeat: the smoke test registers a
  session, writes a memory, searches it back and asserts the bytes. The
  difference is proved, not asserted.

L6. The scope that this slice does not carry, and that the next one does.

- The container of the agent does not mount the volume of the memory yet.
- The provider of the memory is not written yet, and `memory.provider` does
  not name anything.
- The MCP server of the monorepository is not wired to the agent yet: that is
  the slice of the wiring of the agent, and it has its own document.

L7. The slice is on the branch `feat/engram-stack`; `main` is not touched.

L8. The evidence of the slice, observed and not asserted. PENDING.
