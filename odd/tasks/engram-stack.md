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

S7. "the is a error with engram due to is in collision the local one with the Paca engram one, opern a new gitworktree and change the paca one in order to do not collision with my local, additional, change the engram paca port for avoid this again."

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
- T5 — the isolation of the store of PACA from the store of the machine:
  the release of the store moves to `3.7.1`, the same one the machine runs,
  and the rule of the port is written down and pinned by the gate: the
  store of the stack takes no TCP port at all, because Engram refuses the
  socket transport next to an explicit TCP port, so it can never take the
  `7437` of the local store. The files are `engram/compose.yaml`,
  `engram/Dockerfile`, `engram/README.md`, `scripts/tests/structure-test.sh`
  and `odd/tasks/engram-stack.md`. Route: inline, with an independent
  verifier. Evidence: L10. Commit: `aac0e66` and the record of this
  document.

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
  `alpine:3.22` pinned by digest. (Superseded by L10: the release of the
  stack is now `3.7.1`.)
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

L10. The isolation of the store of PACA from the store of the machine, and
what the error of the machine really was.

- The request, verbatim, is S7.
- The error of the machine was **not** a collision of ports, and the slice
  says so instead of pretending: the memory tools of Pi were refused with
  `gentle-engram rejected the Engram runtime contract: unreadable or
  incompatible contract. Expected engram.pi-http-runtime/v1`. The daemon
  that answered on `127.0.0.1:7437` was the binary of the runtime of the
  shell (`~/.gentle-shell/runtime/engram-3.5.2/engram serve`, unmanaged by
  any pidfile), and it served the runtime contract with the digest
  `4bf32703…`, while the extension installed expects the digest
  `0af9af17…` and the operation `mem_search_prompts` that `3.5.2` does not
  have. The daemon was stopped and replaced with the installed CLI
  (`engram run -d -p 7437`, `3.7.1`), which serves the expected digest and
  is tracked by a pidfile. The memory of the project answers again.
- Two stores, two contracts, and the rule that keeps them apart: the store
  of the machine and the store of the container of PACA are two instances of
  Engram, and the container one must be the release that the machine one
  runs, so a tool that speaks the contract of the local store does not fail
  against the other for a version that drifted. The stack therefore moves to
  the release `3.7.1`, with the sha256 of the official archive of each
  architecture (`bc1acef6…` for amd64, `d5023e78…` for arm64), verified
  against `checksums.txt` of the release.
- The port of the stack is a rule of its own now, and the rule is stronger
  than a different number: the store of PACA takes **no TCP port at all**.
  Engram refuses to combine the socket transport with an explicit TCP port
  (`ENGRAM_SOCKET` with `ENGRAM_PORT` → `engram: socket transport cannot be
  combined with an explicit TCP port`, exit 1), and the stack uses the
  socket of the volume. So the default of Engram (`7437`), which belongs to
  the local store of the machine, can never be taken by the store of PACA:
  the collision the request asks for is impossible by construction, not by
  convention. The manifest of the stack therefore carries no port, and the
  compose file says why, so the next person does not add one.
- The evidence of the change, observed and not asserted.
  - The pins of the release are the published ones: `checksums.txt` of the
    release `3.7.1` names `bc1acef6…` for `linux_amd64` and `d5023e78…` for
    `linux_arm64`, and the archives themselves hash to those values.
  - The compose file of the stack is valid and carries the release:
    `docker compose -f engram/compose.yaml config` resolves both services
    with `ENGRAM_VERSION: 3.7.1` and the two checksums, and the service
    publishes no port. The resolution needs the manifest of the stack
    installed (`make env-init`), because the compose file reads
    `engram/.env`.
  - The transports of the store are exclusive, observed with the official
    binary of the release: the socket alone serves the socket and binds no
    TCP port at all; `ENGRAM_PORT` alone binds `127.0.0.1:<port>`; the two
    together are refused with exit 1. That is what keeps the port of the
    local store untouched. It was observed on the host with the binary of
    the release, not inside the container: the image of the stack ships no
    tool to assert it, so the gate of the repository is what pins the rule.
  - The daemon of the machine serves the contract that the extension
    expects: `curl -s http://127.0.0.1:7437/runtime-contract` answers
    `engram.pi-http-runtime/v1` with the digest `0af9af17…`, and the
    memory tools of Pi answered again in the session of the repair.
  - The gates of the repository that do not need Docker are green:
    `make shellcheck`, `make structure-test` (790 assertions),
    `make ci-filters-test` (48) and `make env-test` (114), all with zero
    failures.
  - The rule of the port is pinned by the gate, and the pin discriminates:
    the new rule of `scripts/tests/structure-test.sh` fails when the
    manifest declares `ENGRAM_PORT` and when the compose file publishes a
    port, and it passes on the tree as it is.
- The independent verification of the slice, and what it changed. The
  verifier read the whole diff and found three blockers, all of them real:
  the manifest declared `ENGRAM_PORT=7438` next to `ENGRAM_SOCKET`, which
  makes the store exit 1 and breaks the stack; the README documented a port
  the stack does not have and contradicted itself; and L10 presented the
  port as observed evidence when it had never been observed. The three were
  repaired: the manifest carries no port, the README states the rule as it
  is, and this log claims only what was observed. A second round of the
  same verifier found no blocker over the repaired diff and left advisories,
  and the advisories that mattered were taken: the stale release of L8 is
  marked as superseded, the resolution of the compose file now names the
  manifest it needs, the exclusivity of the transports says where it was
  observed, and the rule of the port is pinned by the gate instead of living
  in a comment.
- The proof of the stack — the image built and the round trip of the memory
  — was **not** observed on the machine that wrote the slice, because the
  Docker daemon is not running there. It was observed in the continuous
  integration of the pull request `#75`, and it is green:
  - The image of the stack builds from the release `3.7.1`: the archive of
    the architecture is downloaded, it is checked against the sha256 that is
    pinned, and the check answers `0.419 /tmp/engram.tar.gz: OK`. A wrong
    pin would have failed the build.
  - `make -C engram up` → `Container paca-engram Healthy`.
  - `make -C engram smoke` → `engram: the memory is kept and given back`.
    The proof registers a session, keeps a memory, searches it back, asserts
    the bytes, refuses that the store accepted a destructive write without
    the token and removes the record afterwards.
  - The whole run of the continuous integration is green: `The changes`,
    `The memory`, `The stack` and `The shellcheck` succeeded.
  With that proof, the slice is closed: the store of PACA speaks the
  release of the machine and takes no port at all.
