# PACA agent: Hermes Agent as a stack of the monorepository

The agent is the unit of PACA that talks to the people who use the product.
It is not a Go module of the repository: it is a container that brings its own
image, like the database and the collector of the traces, so it lives in a
directory of its own with its own compose file.

The stack carries the agent itself, the provider of the models it calls, and
the memory of the monorepository: the agent reads and writes the memories of
PACA over the stdio MCP transport. The feature document of the slice,
[hermes-engram-memory.md](../odd/tasks/hermes-engram-memory.md), holds the
specification and the decisions, and the separation of the memory from the
agent has its own document,
[hermes-engram-separation.md](../odd/tasks/hermes-engram-separation.md).

## What you get

- **One container**, `paca-hermes`, that runs the gateway of the agent.
- **Persistent state**: everything the agent owns lives under `/opt/data`,
  on the named volume `paca-hermes-data`, so its identity, its sessions, its
  skills and its memories survive a restart of the stack.
- **The memory of the monorepository**: the image of the agent carries no
  payload of the store. The stack of the memory builds the official release
  of Engram, verifies it against the sha256 that is pinned for its
  architecture, and publishes it on the named volume `paca-engram-payload`;
  the agent mounts that volume read-only at `/opt/engram`, and the
  configuration of the agent names the store as an MCP server over the stdio
  transport, at `/opt/engram/engram mcp --tools=agent`. The agent and the
  store share the volume `paca-engram-data`, and both run as the same user,
  so the memories the agent writes are the memories the `engram/` stack
  keeps.
- **The provider of the models**: OpenRouter, with the key of the operator
  and the model of the project.
- **The health endpoint on the loopback interface of the container**:
  `http://127.0.0.1:8642/health` answers from inside the container and from
  nowhere else. It says whether the agent is alive, and nothing else.
- **No published port**: the compose file has no `ports:` mapping, so the
  host never reaches the agent.

## The credentials

The stack owns its credentials, in `hermes/.env`, and the installer of the
environment writes them:

```sh
make env-init --stack hermes
```

| Key                   | What it is                                                    |
|-----------------------|---------------------------------------------------------------|
| `OPENROUTER_API_KEY`  | The key of the provider of the models. **Required.**           |
| `AGENT_MAIN_MODEL`    | The model of the agent. `minimax/minimax-m3.1-flash` by default. |

The key is a secret: the installer never prints it, and the file that holds
it is written for its owner alone. It belongs to the DevOps engineer who
runs the project.

**The stack refuses to come up without the key.** The seeding of the agent
reads it, and it fails with the command that writes it. An agent that is
there and cannot answer anyone is worse than an agent that is not there at
all.

The value of the environment of the shell answers the question of the
installer when the file does not hold the key yet, and it reaches the
container when it is set:

```sh
OPENROUTER_API_KEY=... make env-init --stack hermes
```

## Running it

The stack alone:

```sh
make -C hermes up       # start and wait until it is healthy
make -C hermes smoke    # the agent is alive and it reaches the memory
make -C hermes status   # container state
make -C hermes logs     # follow the logs
make -C hermes down     # stop (the container is removed, the state is kept)
```

Or with the compose file directly:

```sh
docker compose -f hermes/compose.yaml up --wait
```

With the monorepository, the root compose includes the stack, so one command
starts everything:

```sh
docker compose up --wait
```

Both ways drive the same compose project, `paca`: the container, the volume
and the network are the same ones whichever way the stack is started.

The agent needs the payload of the memory: the stack of the memory publishes
it, and `make -C engram payload` publishes it by hand. The stack waits for
it, and it refuses to come up without it.

## The services of the stack

| Service       | What it does                                                        |
|---------------|---------------------------------------------------------------------|
| `hermes-init` | The one-shot service: it checks the credential of the provider, it waits for the payload of the memory to be published, it gives the state of the memory to the user of the agent, and it seeds the configuration of the agent into the volume of its state. It ends when it is done. |
| `hermes`      | The gateway of the agent. It starts when the seeding has ended.     |

The configuration of the agent cannot be a read-only mount: Hermes rewrites
`config.yaml` at boot (it migrates it and it keeps timestamped backups of
it), so the file is seeded into the volume and it is left to Hermes
afterwards. The seeding writes it with the writer of Hermes itself
(`hermes config set`), so the file keeps the schema of the version of the
image.

## The memory

The agent reaches the memory over the **stdio** MCP transport: it spawns
the MCP server of the store as a process of its own, and the server writes
the state of the store directly, because the local runtime of Engram is
local first. There is no HTTP MCP transport in Engram, and there is none
needed: the stdio transport is the only one, and it works.

The executable of that server is the payload of the store: the `engram/`
stack builds it, verifies it against the pinned sha256 and publishes it on
the volume `paca-engram-payload`, and the agent mounts that volume read-only
at `/opt/engram`. The agent executes it and it never writes it, and the
volume of the payload is not the volume of the state: the state of the
store travels on `paca-engram-data`, the one the two stacks already share.

The HTTP API of the store is a REST API and not an MCP endpoint: the store
keeps answering the REST clients on the socket of the volume of its state,
and it does not become an MCP endpoint.

The configuration that names it:

```yaml
mcp_servers:
  engram:
    command: /opt/engram/engram
    args:
      - mcp
      - --tools=agent
    env:
      ENGRAM_DATA_DIR: /opt/engram-data
      ENGRAM_PROJECT: paca
```

The environment of the MCP server is declared there and not taken from the
container: Hermes hands a stdio child a safe baseline of the environment
(PATH, HOME, TERM, XDG_\*) and the `env:` block of the server, so the state
of the memory and the project of the monorepository are declared in one
place.

The memories belong to the project `paca`, the same project the work of the
repository is remembered in.

## The networks

The agent joins `paca_internal`, the service-to-service network of the
monorepository, and `paca_egress`, the outbound network.

It reaches the provider of the models, and nothing else of PACA leaves the
machine. This is the decision the stack of the agent deferred, and the
credential of the provider makes it unavoidable: without the outbound
access the agent cannot call a model at all.

## The ports

| Port   | Service                  | Reachable from                     |
|--------|--------------------------|------------------------------------|
| `8642` | The health of the agent  | The container itself, only.        |

The port is not published and not exposed: the health endpoint is asked from
inside the container, and it is the only route the agent has today. A
container of `paca_internal` reaches the agent on its name, once a slice
gives it a surface to be reached on.

## The proof

The smoke test of the stack is a round trip and not a heartbeat:

```sh
make -C hermes smoke
```

It asserts that the agent is alive, and it asserts that the memory works:
it spawns the MCP server of the store the way the agent spawns it, it
writes a memory through it, it reads that memory back through it, and it
removes it afterwards, so the proof leaves no memory behind.

## What is not configured yet

- **No messaging platform**: the gateway serves no channel — no Telegram,
  no Discord, no Slack.
- **No MCP server of PACA**: the agent does not reach the MCP server of the
  monorepository yet. Only the memory is wired in this slice.

## The layout

```
hermes/
├── Dockerfile        # the image of the agent
├── Makefile          # the targets of this stack
├── README.md         # this file
├── compose.yaml      # the services, the networks, the volumes and the healthcheck
├── env.example       # the manifest of the credentials of the stack
├── init.sh           # the seeding of the state of the agent
├── mcp-roundtrip.py  # the round trip of the proof, over the stdio MCP
└── smoke.sh          # the proof of the stack
```

## Notes and limits

- The image of the agent is pinned by the digest of its index, so the same
  pin serves amd64 and arm64, like every image of the repository. The image
  carries no payload of the store: the payload is built, verified and
  published by the `engram/` stack, and the agent mounts it read-only.
- The seeding waits for the payload, with a bound, and it refuses the stack
  when the payload is not there: the refusal names `make -C engram payload`,
  the command that publishes it. The read-only volume is never written by
  the agent.
- The container keeps the capabilities of the image and does not run with
  `no-new-privileges`: the image drops its own privileges at boot and needs
  `CAP_SETUID` and `CAP_SETGID` to do it. The state of the agent lives on
  the named volume, and the container writes nothing else.
- The entrypoint of the image is left alone: it starts the supervision tree
  that restarts the gateway when it crashes, and it reaps the orphans the
  tools of the agent leave behind. A container that overrides it loses both.
- The state is a named volume and never a bind mount: the sessions of the
  agent live in a SQLite database in WAL mode, and WAL is not safe on a
  mount that crosses the boundary of the virtual machine — the pages of the
  database would be silently corrupted.
- The container is bounded: four gigabytes of memory and two cores, the
  limits the documentation of the image recommends, and a gigabyte of shared
  memory for the browser of the agent.
- The healthcheck asks the health endpoint of the API server with the
  `curl` of the image, and the endpoint needs no key.
- The first answer takes a while: the image unpacks its bundled skills and
  starts its supervision tree before the gateway answers.
