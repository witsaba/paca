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
  `engram/Makefile`, `engram/README.md`, `engram/env.example`.
  Route: inline. Commit: `f7f0902`.
- T2 — the wiring of the monorepository: the root `compose.yaml`, the root
  `Makefile`, the root `README.md`, the root `AGENTS.md`,
  `.github/workflows/ci.yml`, `scripts/ci/job-filters.txt` and
  `scripts/tests/ci-filters-test.sh`. Route: delegated writer, diff read by
  the parent. Commit: `5578352`.
- T3 — the proof: the image is built, the stack comes up healthy, the round
  trip of the memory is asserted and the gate of the repository is green.
  Route: inline. Evidence: L8. Commit: the record of this document.
- T4 — the repair of the review: the proof of the stack asserts the guard
  of the token, and the packages of the image are pinned like every other
  dependency of the repository. Route: inline. Evidence: L9. Commit:
  `1ef1eb8`.

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
- The token of the stack guards the writes that destroy, and the proof says
  so: it fails when the token is missing or is still the placeholder of the
  manifest, and it refuses that the store accepted a destructive write
  without the token.

L6. The scope that this slice does not carry, and that the next one does.

- The container of the agent does not mount the volume of the memory yet.
- The provider of the memory is not written yet, and `memory.provider` does
  not name anything.
- The MCP server of the monorepository is not wired to the agent yet: that is
  the slice of the wiring of the agent, and it has its own document.

L7. The slice is on the branch `feat/engram-stack`; `main` is not touched.

L8. The evidence of the slice, observed and not asserted.

- The image of the stack builds from the release of Engram `3.6.0`: the
  archive of the architecture is checked against the sha256 that is pinned
  (`945fb471…` for amd64, `00a2d764…` for arm64) and the build fails when
  the bytes are not the ones that were pinned. The base image is
  `alpine:3.22` pinned by digest.
- `docker compose -f engram/compose.yaml up -d --wait engram` → `Container
  paca-engram Healthy`. The socket of the volume is `srw-------` and it
  belongs to the user `10000`: the store runs as that user and the consumer
  that mounts the volume reaches it as that user.
- `make -C engram smoke` → `engram: the memory is kept and given back`. The
  proof runs as the user `10000`, the same user the container of the agent
  runs as, and it registers a session, keeps a memory, searches it back,
  removes it and removes the session: the proof leaves no memory behind.
- With the container of the store stopped, the same target fails
  (`service "engram" is not running`), so the proof proves something.
- `make all` → `All the steps of the gate passed.` (exit 0). Inside it: the
  quality gate of the five modules, the container tests of the packages and
  of the applications, the shellcheck of the nine scripts, the tests of the
  installer of the environment, the tests of the structure of the repository
  (335 assertions) and the tests of the filters of the continuous
  integration (44 assertions) — all green — then the database and its smoke
  test, the collector of the traces and its smoke test, the agent
  (`hermes: the health endpoint answers`), the memory (`engram: the memory is
  kept and given back`) and, at the end, the whole project: `Container
  paca-postgres Started`, `Container paca-mcp-server Started`, `Container
  paca-db-admin Started`, `Container paca-hermes Started`, `Container
  paca-jaeger Started`, `Container paca-engram Started`, all of them
  `Healthy`, and the assertion of the schema written by the service that
  owns it (`All assertions passed.`).
- The gate left no container behind: the cleanup stopped the five stacks and
  removed their containers.
- Two defects of the machine, and not of this slice, that the gate found.
  Both are the credential drift of the volume of PostgreSQL, the same class
  of defect that L8 of the slice of the agent recorded: the `.env` of the
  stack of the database named a database that the cluster did not hold
  (`paca-pg` against `paca`), and the password of the role did not match
  the one the stack declares. The cluster was aligned with the stack, and
  not the other way around: the database name of the `.env` was corrected,
  and the password of the role `paca` was set to the one the installer
  declares. The rows of the database were not touched, and the cluster was
  not dropped. The slice of the agent had done the same repair before.
- The socket of the volume is the only door of the store, and it is bound to
  the user of the process. The container of the agent runs as the user
  `10000`, and the store runs as that same user, so the consumer reaches the
  socket when it mounts the volume. If the agent ever runs as another user,
  the transport has to change and not only the configuration: the local
  runtime of Engram binds the loopback of its own container and refuses any
  other address.


L9. The review of the candidate, and the repair it asked for.

- The candidate was reviewed by the risk lens of the native review, over the
  frozen view of the fifteen changed paths. The review was **approved**: no
  finding opened a correction, and the receipt of the review stands. The
  three findings it carries are informational, and the native says so:
  they are separate work and never a reason to review the candidate again.
- The findings, and what this slice did with them.
  - `R1-001` (WARNING): the manifest of the credentials ships a literal
    token value, and the manual copy of the manifest leaves that placeholder
    in place, so a manual installation starts the store with a known token.
    The repair: the proof of the stack now refuses the placeholder, so the
    stack does not come up with a known token and say nothing about it.
  - `R1-002` (WARNING): the proof added the token only when it was there,
    so it never asserted that the guard exists. The repair: the proof fails
    when the token is missing or is the placeholder, and it asserts that the
    store refuses a destructive write that travels without the token. The
    control that the documentation claims is now a control that the gate
    proves.
  - `R1-003` (SUGGESTION): the packages of the base image float, while the
    base image and the payload of the store are pinned. The repair: the
    packages are pinned by version, like every other dependency of the
    repository, and the bump of the pin is a deliberate act.
- The evidence of the repair, observed and not asserted.
  - `make -C engram smoke` → `engram: the memory is kept and given back`.
  - With the token of the stack removed from the environment of the
    container, the same proof fails: `engram: the token of the stack is
    missing or is still the placeholder of the manifest` (exit 1).
  - With the placeholder of the manifest as the token, the same proof fails
    the same way (exit 1).
  - The proof itself asserts that the store refuses the destructive write
    that travels without the token: the observation of the proof survives
    that attempt and is removed afterwards with the token.
  - `shellcheck` over the scripts of the stack is clean.
  - The whole gate of the repository is green with the repair:
    `make all` → `All the steps of the gate passed.` (exit 0), with the
    memory beside the agent, the database and the traces, and the schema of
    the database asserted.
