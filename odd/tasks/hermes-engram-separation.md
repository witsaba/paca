# The memory of PACA is separated from the agent of PACA

The agent of PACA does not carry the memory anymore. The payload of the store
is built, verified and published by the stack of the memory, and the agent
mounts it and points its MCP configuration at it, exactly as the documentation
of the store prescribes for any other MCP agent. The image of the agent knows
nothing about the store.

## Specs

S1. "Ok, I want that the engram be separated totally. But using the allowed
MCP server in engram, must to be the entry point for hermes or any other
agent."

S2. "I need you refactor this solution in order to hermes use engram using the
MCP engram server, you must to ensure the engram MCP server run properly."

S3. "Read this first to understand the official documentation due to hermes
doens't have rn the agent support directly so is needed to down with the `Any
other MCP agent`
https://github.com/Gentleman-Programming/engram/blob/main/docs/AGENT-SETUP.md#any-other-mcp-agent."

S4. "This must to be internally, engram is not exposed to the host and hermes
must to be allowed to connect to engram througth the MCP regarding the official
documentation."

## The facts the refactor stands on

- The MCP server of the store is **stdio only**. The binary of the release
  3.6.0 says so: `engram mcp [--tools=PROFILE] [--project NAME] — Start MCP
  server (stdio transport, for any AI agent)`. The documentation says the same
  in its section "Any other MCP agent": "The pattern is always the same — point
  your agent's MCP config to `engram mcp` via stdio transport." There is no
  MCP over HTTP upstream (the proposal is issue #112), so the MCP server of the
  store is a process of its client, and the client needs the executable of the
  store at hand.
- The executable of the store is **static** (`not a dynamic executable`), so it
  runs on the base of the agent and it can live on a volume.
- The `engram serve` HTTP API is **not** an MCP server. The documentation says
  it about the Pi native package: "The URL is the HTTP REST API base; it is
  not an MCP endpoint." The stack of the memory serves it on the socket of its
  volume, for the clients that speak REST, and it stays that way.

## The contract of the separation

These names are the contract between the two stacks. They do not change.

| What | Name |
|------|------|
| The volume that carries the payload of the store | `paca-engram-payload` |
| The path of the payload in the agent | `/opt/engram/engram` |
| The command of the MCP server of the store | `/opt/engram/engram mcp --tools=agent` |
| The service that publishes the payload | `engram-payload` (the stack of the memory) |

## Tasks

- T1 — the stack of the memory publishes the payload. `engram/compose.yaml`
  gains the one-shot service `engram-payload`, built from the image of the
  store, that writes the verified executable into the volume
  `paca-engram-payload`; `engram/Makefile` gains the target that runs it.
  Route: delegated writer. Commit: `d2d5851`.
- T2 — the image of the agent carries no payload. `hermes/Dockerfile` loses the
  download and the verification of the store, and `hermes/compose.yaml` loses
  the pins of the store and mounts the published payload read-only at
  `/opt/engram`. Route: delegated writer. Commit: `d2d5851`.
- T3 — the seeding of the agent waits for the published payload and refuses
  the stack without it. `hermes/init.sh` waits, with a bound, for
  `/opt/engram/engram` and fails with a message that names the command that
  publishes it; it never touches the read-only volume. Route: delegated
  writer. Commit: `d2d5851`.
- T4 — the repository proves the separation. The rule of the structure that
  pins the payload of the memory against the drift becomes the rule that the
  stack of the agent carries no payload of its own. Route: delegated writer.
  Commit: `d2d5851`.
- T5 — the order of the start. The root `Makefile` publishes the payload
  before it starts the agent, so the agent never waits for nothing.
  Route: delegated writer. Commit: `d2d5851`.
- T6 — the documentation: the root `README.md`, `hermes/README.md` and
  `engram/README.md` say who publishes the payload and who mounts it.
  Route: delegated writer. Commit: `ccf5bbe`.
- T7 — the proof: the round trip over the stdio MCP from inside the container
  of the agent, the refusal of the stack without the payload published, and
  the gate of the repository green. Route: inline. Evidence: L4.
  Commit: this one.

## Log

L1. "Ok, I want that the engram be separated totally. But using the allowed
MCP server in engram, must to be the entry point for hermes or any other
agent. I need you refactor this solution in order to hermes use engram using
the MCP engram server, you must to ensure the engram MCP server run properly.
Read this first to understand the official documentation due to hermes doens't
have rn the agent support directly so is needed to down with the `Any other MCP
agent`
https://github.com/Gentleman-Programming/engram/blob/main/docs/AGENT-SETUP.md#any-other-mcp-agent.
This must to be internally, engram is not exposed to the host and hermes must
to be allowed to connect to engram througth the MCP regarding the official
documentation."

L2. The reading of the official documentation, and what it settles.

- The store is the entry point for any MCP agent, and the transport is the
  stdio one. The agent of PACA therefore spawns the MCP server of the store as
  its own process, and it needs the executable of the store at hand.
- The separation is not about the process: it is about **who owns the
  executable**. Today the image of the agent downloads the release of the
  store and verifies it, so the payload of the store is built in two places
  and pinned in five declarations, and the rule of the structure of the
  repository exists only to keep the two copies from drifting.
- The refactor gives the payload one owner: the stack of the memory builds it,
  verifies it and publishes it on a volume; the stack of the agent mounts that
  volume read-only and points its MCP configuration at the executable, exactly
  as the documentation prescribes. Any other MCP agent does the same: it mounts
  the volume and names the command.
- The memory stays internal: no port is published, the payload travels on an
  internal volume, and the state of the store keeps travelling on the volume
  it already travels on.

L3. The decisions of the refactor.

- **The payload is published by a one-shot service of the stack of the
  memory**, not by the store itself: the publication is then a step of the
  lifecycle of the stack of the memory, with its own exit code and its own log,
  and the store keeps doing only its job.
- **The volume is mounted read-only in the agent**: the agent executes the
  payload of the store, and it never writes it.
- **The seeding waits, with a bound, for the payload**: the two stacks are
  separate, so the agent cannot depend on a service of the other one; it waits
  for the artifact it needs and refuses the stack with a message that names
  the command that publishes it.
- **The path of the payload does not change** (`/opt/engram/engram`): the
  configuration of the agent, and therefore the MCP server of the store as the
  entry point of the agent, stays the same.

L4. The evidence of the separation, observed from zero.

- The rule of the structure of the repository proves the separation, and it
  discriminates: against the code before the change it fails with five
  failures (the stack of the agent declares the build argument `ENGRAM_VERSION`
  with the value `3.6.0`, the same for the two checksums, and the image of the
  agent downloads a release and verifies a checksum), and against the code
  after it it passes: `the stack of the agent carries no payload of its own:
  5 assertion(s)`, `341 assertion(s), 0 failure(s)`.
- The round trip over the stdio MCP, from inside the container of the agent,
  with the payload published by the other stack:
  `make -C hermes smoke` → `hermes: the memory is written and given back over
  the stdio MCP` (exit 0).
- The memory of the monorepository keeps its own proof:
  `make -C engram smoke` → `engram: the memory is kept and given back` (exit 0).
- The refusal of the agent without the published payload, observed with the
  payload emptied from the volume:
  ```
  hermes: the payload of the memory is not published: /opt/engram/engram is missing or is not executable.
  hermes: the stack of the memory publishes it with:
  hermes:   make -C engram payload
  ```
  The seeding ends with the exit code 1, and the gateway of the agent never
  comes up, because it waits for the seeding to succeed.
- The image of the agent carries no payload: `test -e /opt/engram` inside the
  image is false, the mount of the container is read-only (`ro` in
  `/proc/self/mounts`), and a write inside it answers `Read-only file system`.
- The gate of the repository from zero (`docker compose down -v` and then
  `make all`): **All the steps of the gate passed** (exit 0) — the quality
  gate of the five modules, the container tests of the packages and of the
  applications, the shellcheck of the eleven scripts, the tests of the installer
  of the environment (114 assertions), the tests of the structure of the
  repository (341 assertions) and the tests of the filters of the continuous
  integration — all green — then the database and its smoke test, the collector
  of the traces, the agent, the memory and, at the end, the whole project with
  the assertion of the schema written by the service that owns it.
- The publication is idempotent: `make -C engram payload` twice in a row ends
  with the exit code 0 both times, and the volume holds the executable with
  the mode `755` and the identity `root:root`.
