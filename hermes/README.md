# PACA agent: Hermes Agent as a stack of the monorepository

The agent is the unit of PACA that talks to the people who use the product.
It is not a Go module of the repository: it is a container that brings its
own image, like the database and the collector of the traces, so it lives in
a directory of its own with its own compose file.

The stack carries the agent itself. There is no provider, no messaging
platform and no MCP wiring yet: the container starts, the gateway runs and
the health endpoint answers. The feature document of the slice,
[hermes-agent-stack.md](../odd/tasks/hermes-agent-stack.md), holds the
specification and the decisions.

## What you get

- **One container**, `paca-hermes`, that runs the gateway of the agent.
- **Persistent state**: everything the agent owns lives under `/opt/data`,
  on the named volume `paca-hermes-data`, so its identity, its sessions, its
  skills and its memories survive a restart of the stack.
- **The health endpoint on the loopback interface of the container**:
  `http://127.0.0.1:8642/health` answers from inside the container and from
  nowhere else. It says whether the agent is alive, and nothing else.
- **No published port**: the compose file has no `ports:` mapping, so the
  host never reaches the agent.
- **The internal network only**: the agent joins `paca_internal`, the
  service-to-service network of the monorepository, which has no outbound
  access at all. The network of the outbound traffic is not joined yet:
  the slice that configures the provider of the agent decides it, because
  that is the moment the agent starts to talk to the outside world. The
  services of PACA carry personal data, and the agent reaches them.

## Running it

The stack alone:

```sh
make -C hermes up       # start and wait until it is healthy
make -C hermes smoke    # the health endpoint must answer
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

## The ports

| Port   | Service                  | Reachable from                     |
|--------|--------------------------|------------------------------------|
| `8642` | The health of the agent  | The container itself, only.        |

The port is not published and not exposed: the health endpoint is asked from
inside the container, and it is the only route the agent has today. A
container of `paca_internal` reaches the agent on its name, once a slice
gives it a surface to be reached on.

## What is not configured yet

This slice adds the agent itself, and nothing else:

- **No provider**: the agent has no model provider, so it cannot answer
  anyone yet.
- **No messaging platform**: the gateway serves no channel — no Telegram,
  no Discord, no Slack.
- **No MCP wiring**: the agent does not reach the MCP server of the
  monorepository yet.

Those are the decisions of the slices that follow, and each of them brings
its own configuration, its own proof and its own update of this document.

## The layout

```
hermes/
├── Makefile       # the targets of this stack
├── README.md      # this file
└── compose.yaml   # the service, the network, the volume and the healthcheck
```

## Notes and limits

- The image is pinned by the digest of its index, so the same pin serves
  amd64 and arm64, like every image of the repository.
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
