# PACA

The monorepository of PACA: the databases, the services and the tools that
support them, in one repository and one compose project.

## The layout

| Directory     | What it holds                                                              |
|---------------|----------------------------------------------------------------------------|
| `apps/`       | The services, one directory and one Go module each.                        |
| `packages/`   | The libraries the services share, one directory and one Go module each.     |
| `db/`         | The databases, one directory and one stack each.                           |
| `observability/` | The traces: the collector and the backend that keeps them.              |
| `hermes/`      | The agent of PACA: Paca, the business assistant of witsaba.com, the unit that talks to the people who use the product through Telegram. |
| `engram/`      | The memory of the monorepository: the store of the memories.             |
| `tools/`      | The developer tools, pinned as Go tool dependencies.                        |
| `scripts/`    | The scripts of the repository: the installer of the environment.            |
| `odd/tasks/`  | The feature documents: the specification of every piece of work.           |

The repository is a Go workspace (`go.work`), so the modules are built and
tested against each other, and every module keeps working on its own.

## The contract of the agents

[`AGENTS.md`](AGENTS.md) holds the contract of the architecture: the layers and the
direction of the dependencies, the ownership of the rules of the domain, the owner of
every model, and the checks of SOLID. It is written for a coding agent, and it is the
canonical entry of the instructions of the repository. The entry file of each module
(`packages/*/AGENTS.md`, `apps/*/AGENTS.md`) adds the specifics of that module.

## Paca, the assistant of PACA

Paca is the agent of PACA: the business assistant of witsaba.com. Her
identity is one file of the repository, [`hermes/soul.md`](hermes/soul.md),
and the runtime reads it in the first slot of its system prompt, so the
identity of Paca replaces the identity the runtime carries by itself. The
file is the source of truth of her personality, and the documentation of the
repository agrees with it.

Her method rests on the seven Hermetic principles, applied as a discipline
of the work and never as a claim about physics:

| Principle | How Paca works |
|-----------|----------------|
| Mentalism | The frame decides the answer: what is really asked is named in one line before the answer, and a wrong frame is corrected early. |
| Correspondence | The small is a mirror of the whole: one number, one message, one error is read as a symptom of the process that produced it. |
| Vibration | Nothing rests: every figure is a snapshot of a moving thing, and every number is read in the cycle it belongs to. |
| Polarity | Every truth has its other pole: both poles of a decision are named, and a recommendation with no downside named is not a recommendation. |
| Rhythm | Everything has its cycle: the recommendation is timed to the cycle of the business, not to the alarm. |
| Cause and Effect | Every effect has a cause: the mechanism is named, the cause is traced before the remedy, and a coincidence is never called a cause. |
| Gender | The masculine and the feminine are in everything: Paca is female-dominant and a duality, she leads and she creates, and she holds both poles. |

Her rules of honesty:

- She is an AI and says so. She does not imitate a human: no performed
  feelings, no analysis of the sentiment of the person she talks to, no
  warmth used as a technique.
- She never flatters. Sycophancy is a defect: she pushes back on a weak
  idea, a wrong number and a bad plan, once and clearly.
- She never makes things up: not a figure, not a citation, not a law, not a
  date, not the name of an API. When she does not know, she says so.
- She researches the web and questions the biases of the sources and of her
  own.
- When she is not sure, she says so and validates with the person she
  serves. She suggests; the last word is theirs.
- She does not narrate herself: she answers who she is in a sentence or two
  and goes to the work, without reciting her own principles, her rules or her
  setup, and without offering an integration that is not connected.

Paca serves one person today: the owner of the business. The scope grows,
and what she learns in one session is available in the next one. Her first
channel is Telegram: one bot, one list of the people she answers to. There is
no Discord, no Slack, no web dashboard and no multi-user support yet.

The runtime of the agent is Hermes Agent, by Nous Research. It keeps a
sentence of its own in a later slot of the system prompt — that it runs on
Hermes Agent — and no configuration removes it. The identity of Paca answers
it from the first slot, and the soul forbids naming the runtime. The terminal
of the agent is branded with her name too: the branding of the runtime is a
skin, and the stack carries the skin of Paca.

## What is in the repository now

| Directory          | What it is                                        | README |
|--------------------|---------------------------------------------------|--------|
| `apps/mcp-server/` | The MCP server: the Model Context Protocol over the official Go SDK, with a health endpoint and three tools for the people who use PACA. | [apps/mcp-server/README.md](apps/mcp-server/README.md) |
| `apps/db-admin/`   | The database admin: the single writer of the schema. It applies the pending migrations when it starts, then stays alive with one health endpoint. | [apps/db-admin/README.md](apps/db-admin/README.md) |
| `packages/database/` | The database library: the pool, the queries, the migrations and the repositories that write the rows. A library, with no port and no container. | [packages/database/README.md](packages/database/README.md) |
| `packages/domain/` | The domain: the models of the people, the port of their storage, the state of the schema and the model of the health. A library that imports nothing but the standard library. | [packages/domain/README.md](packages/domain/README.md) |
| `packages/platform/` | The platform: the clock, the logs, the telemetry, the HTTP server and the health check of the container. The infrastructure every service shares, forked by none. | [packages/platform/README.md](packages/platform/README.md) |
| `db/postgres/`     | PostgreSQL 18.6 with pgvector 0.8.7, internal only. | [db/postgres/README.md](db/postgres/README.md) |
| `observability/`   | Jaeger 2 as the OpenTelemetry collector and the trace backend: the traces of the services, kept for two days. | [observability/README.md](observability/README.md) |
| `hermes/`         | The agent of PACA: Paca, the business assistant of witsaba.com. Her identity is the soul of the repository, `hermes/soul.md`, and her first channel is Telegram: the bot she answers on and the list of the people she answers to. The container and its state, with the provider of the models and the memory of the monorepository on the stdio MCP transport. Its image carries no payload of the store: it mounts the published payload read-only. | [hermes/README.md](hermes/README.md) |
| `engram/`         | The memory of the monorepository: Engram, the store of the memories of the agent and of the work of the repository. A container of its own, with no published port, and the owner of the payload of the store: it builds the official release, verifies it against the pinned sha256 and publishes it on the volume `paca-engram-payload`. | [engram/README.md](engram/README.md) |

## The installation

The stacks need their environment: the credentials of the database, and
whatever a stack declares in its own manifest. Every stack that ships an
`env.example` gets its `.env` written by the installer of the environment:

```sh
make env-init
```

It asks one question per value and offers the default of each one: the value
already configured when the file exists, the value of the manifest otherwise.
The empty answer keeps the default, any other answer replaces it, and `g`
generates a strong password for the secrets.

A value can also come from the environment of the shell: when the
environment holds a key of a manifest and the `.env` of the stack does not
hold it yet, the environment answers the question. The file that is already
on disk wins over the environment, because the file is the configuration of
the stack.

A key marked `# @required` in its manifest is a key the stack cannot run
without: the interactive installation refuses to finish without it, and the
`--yes` mode warns about it and continues. The credential of the provider
of the models of the agent is one of them, and it belongs to the DevOps
engineer who runs the project.

The stack of the agent declares two more, both of the channel of Paca:
`TELEGRAM_BOT_TOKEN`, the token of the bot, issued by @BotFather, and
`TELEGRAM_ALLOWED_USERS`, the numeric ids of the people Paca answers to,
comma-separated. **The agent refuses to come up without them**: without the
token the channel is dead, and without the list the gateway of the runtime
denies every user. They live in `hermes/.env`, the installer of the
environment writes them, and nobody else reads that file.

Running it again keeps what is configured, so a value can be changed at any
time without losing the rest of the file. The secrets are never printed, and
the files that hold them are written for the owner alone.

The installer also works without a terminal, for a scripted installation and
for the continuous integration:

```sh
make env-init ARGS="--yes"       # keep what is configured, generate the rest
make env-init ARGS="--dry-run"    # print what it would write, write nothing
./scripts/setup-env.sh --help     # every option
```

The `.env` files are never committed. The manual way, with an editor, keeps
working: copy the example of the stack and change the values.

## Running the stacks

Every stack keeps its own compose file; the root one includes them all, and
declares the networks they share.

```sh
docker compose up --wait                    # every stack of the monorepository
docker compose up -d --wait mcp-server      # one service of it
docker compose down                         # stop everything
```

The root `Makefile` drives the same project, and the targets of the whole
stack are the ones the continuous integration calls:

```sh
make stack-up        # start every stack and wait until the whole project is healthy
make stack-schema    # assert that the schema was written by the service that owns it
make stack-down      # stop every container of the project, keeping the named volumes
```

The payload of the memory belongs to the stack of the memory: it is
published there, on the volume `paca-engram-payload`, and the stack of the
agent mounts it read-only. `make -C engram payload` publishes it by hand.

A single stack also runs on its own, from its own directory:

```sh
make -C apps/mcp-server compose-up
make -C apps/mcp-server smoke
```

## The networks

| Network         | Attribute        | Purpose                                                                 |
|-----------------|------------------|-------------------------------------------------------------------------|
| `paca_internal` | `internal: true` | Service-to-service traffic. A container on it has **no outbound access at all**. |
| `paca_egress`   | plain bridge     | Outbound traffic for the services that need the internet.                |

A service that only talks to the other services joins `paca_internal` alone.
A service that needs the internet joins `paca_egress` as well. The agent of
PACA joins both: it reaches the services of the monorepository and the
provider of the models, and nothing else of PACA leaves the machine.

## The ports

The ports of the monorepository are reserved, so a new service never takes
the port of another one.

| Port  | Service       | Reachable from                                     |
|-------|---------------|----------------------------------------------------|
| `5432`| PostgreSQL    | The containers of `paca_internal`, only.            |
| `8081`| MCP server    | The containers of `paca_internal`, only.            |
| `8082`| Database admin| The containers of `paca_internal`, only.            |
| `4317`| Jaeger (OTLP) | The containers of `paca_internal`, only.            |
| `4318`| Jaeger (OTLP) | The containers of `paca_internal`, only.            |
| `16686`| Jaeger UI    | The loopback interface of the host, only.           |
| `8642` | Hermes agent | The container itself, only.                       |
| `8080`| *reserved*    | For the REST backend.                               |
| `50051`| *reserved*   | For the gRPC backend.                               |

The MCP server publishes no port at all. It binds the interface of its own
container and sits on `paca_internal`, so the other containers of the stack
reach it and nothing else does. The health endpoint shares that surface.

The database admin publishes no port either. Its only route is the health
endpoint, and it serves the same purpose: the stack asks the container
whether it is alive, from inside the container.

The interface of Jaeger is the only exception: it is published on the
loopback interface of the host, and on nothing else. It is a surface for the
developer who reads the traces, not a surface of the network.

## The observability

The services trace themselves with [OpenTelemetry](https://opentelemetry.io)
and send the traces to the collector of the `observability/` stack. The
collector is Jaeger 2, which keeps the traces for two days in an embedded
database and serves the interface that reads them.

```sh
make observability-up      # start the collector and wait until it is healthy
make observability-smoke   # ask the collector whether it answers
make observability-down    # stop it
```

The interface of the traces is at <http://127.0.0.1:16686>.

The logs of the services stay on the standard output, where the runtime of
the containers collects them. Every log record carries the identifiers of
the trace it belongs to, so a line of the log opens the whole request it
comes from.

## The agent

The agent is the unit of PACA that talks to the people who use the
product. It is Paca, the business assistant of witsaba.com, and it runs on
the Hermes Agent runtime, in the `hermes/` stack: one container, the gateway
of the agent, with its state on a named volume.

```sh
make hermes-up       # start the agent and wait until it is healthy
make hermes-smoke    # ask the agent whether it answers and whether it reaches the memory
make hermes-down     # stop it
```

The agent reads the memory of the monorepository: the stack of the memory
owns the payload of the store — it builds the official release, verifies it
against the pinned sha256 and publishes it on the volume
`paca-engram-payload` — and the agent mounts that volume read-only at
`/opt/engram`. The configuration of the agent names the store as an MCP
server over the stdio transport, at `/opt/engram/engram mcp --tools=agent`:
the MCP server of the store is a process of the agent, and it reads and
writes the state of the store on the volume the two stacks already share.
The agent waits, with a bound, for the payload to be published, and it
refuses the stack with a message that names `make -C engram payload` when it
is not there. A memory that the agent writes belongs to the project `paca`,
and it is the same store the `engram/` stack runs.

The agent needs a provider: the key of OpenRouter, written by the installer
of the environment, and the model it calls, `minimax/minimax-m3.1-flash` by
default. **The stack refuses to come up without the key**: the agent cannot
answer anyone without it. The key belongs to the DevOps engineer who runs
the project, and nobody else reads it.

Telegram is the first channel of Paca. The gateway of the runtime serves
the bot of `TELEGRAM_BOT_TOKEN` and answers only the users of
`TELEGRAM_ALLOWED_USERS`, so the bot stays closed to the people of the
business. The list holds at least one id, and the first one is the owner of
the business: the stack writes the chat of the owner into the configuration
of the agent as its home channel, so the runtime never asks the user to set
one. The identity of the agent is the soul of the repository,
`hermes/soul.md`: the seeding of the stack puts it in the state of the
agent, and the runtime reads it in the first slot of its system prompt.

The agent joins `paca_internal` and `paca_egress`: it reaches the services
of the monorepository, and it reaches the provider of the models, which is
the moment it starts to talk to the outside world. Nothing else of PACA
leaves the machine.

## The schema of the database

The schema has **one writer**, and it is the `db-admin` service. It applies
the pending migrations of `packages/database/migrations/` when it starts, and it
says so in its log. A service never migrates on boot: a service that
migrates on boot races with the other replicas of itself.

The service runs in the container, on the internal network, and it needs the
credentials of the PostgreSQL stack to reach the database. The credentials
live with the stack that owns them (`db/postgres/.env`); the services that
talk to the database load that file and assemble the connection string from
it.

## The quality gate

Every Go module brings its own Makefile, with the same targets: the format
check, the analyzers, the linter, the build, the tests, the race detector,
the coverage gate, the benchmarks and the vulnerability scan.

```sh
make -C apps/mcp-server all
make -C packages/domain all
make -C packages/database all
make -C packages/platform all
```

The tools are pinned in `tools/go.mod` and run with `go tool`, so every
machine and every container uses the same versions.

## The gate of the repository

`make all` is the one command that proves the repository: it runs every
check the continuous integration runs, and it runs them **from nothing** —
no image, no container, no volume and no `.env` are needed.

```sh
make all
```

It writes the environment it needs, it runs the quality gate of every
application and every package, the container tests of the packages, the
container tests of the applications, the shellcheck of the scripts, the
tests of the installer of the environment, the tests of the structure of the
repository, the tests of the filters of the continuous integration, the
smoke test of every database, the smoke test of the collector of the traces,
the smoke test of the agent, the smoke test of the memory, and finally the
whole stack: it comes up, the schema is asserted, and it goes down.

A step that fails stops the gate, and the exit code says so. The gate
leaves no container behind: the containers it started are stopped when it
ends, whether it passed or failed.

The gate proves the architecture as well. `make structure-test` asserts that
every module resolves its own graph and its own dependencies without the
workspace of the repository, that every link of the documentation points at a
file that exists, and that the layers of every application respect the
direction of the dependencies: the domain names no technology, the
application knows no adapter, the adapters know no use case, and only the
composition root knows the adapters. The rules of the architecture are not a
comment that a reviewer has to remember: they are assertions that fail.

The last step of the gate is the proof of the stack. `make stack-up`
brings the whole monorepository up and waits until it is healthy, and
`make stack-schema` asserts, against the running project, that every
container is running and healthy, that the writer of the schema was not
restarted (it waits for its database instead of exiting and being
restarted), that every migration of `packages/database/migrations` is applied,
and that the table `users` exists. The script behind it is
[`scripts/stack-schema.sh`](scripts/stack-schema.sh).

## The continuous integration

The root `Makefile` dispatches to the stacks; it holds no stack-specific
logic. It is the entry point of the continuous integration, so the workflow
never names an application or a database.

```sh
make gate               # the quality gate of every application and every package
make packages-integration  # the container tests of every package
make shellcheck         # the shell scripts of the repository
make db-up              # every database, up and healthy
make db-smoke           # the smoke test of every database
make db-down            # stop every database, keeping the data volumes
make stack-up           # the whole monorepository, up and healthy
make stack-schema       # the schema of the database was written by the service that owns it
make stack-down         # stop every container of the project, keeping the named volumes
make all                # the full gate of the repository, from nothing
```

The workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on
every push to `main` and on every pull request, with six checks:

| Job            | What it verifies                                                    |
|----------------|---------------------------------------------------------------------|
| `gate`         | fmt-check, vet, lint, build, test, race, the coverage floor and the vulnerability scan of every application and every package, and then the container tests of the shared libraries and of the applications: the rules of the tables, asserted against a real PostgreSQL. |
| `database`     | the image of the stack builds, the cluster starts, and the smoke test asserts UTF-8, UTC, UUID, vector search, checksums and isolation. |
| `observability`| the collector of the traces starts, and its query API answers.       |
| `memory`       | the store of the memories starts, and its smoke test asserts that a memory is kept and given back. |
| `stack`        | the whole monorepository comes up from nothing, and the schema of the database is there: it was written by the service that owns it. |
| `shellcheck`   | the shell scripts of the repository pass shellcheck.                  |

**A check runs only when the change can affect what it proves.** GitHub
Actions filters the files of a run only at the level of the workflow, so
the job `changes` computes the files of the run and the rules of
[`scripts/ci/job-filters.txt`](scripts/ci/job-filters.txt) say which check
each one needs. A check that is not needed is skipped, and the skip is a
visible statement: the workflow always reaches a conclusion, and a pull
request is green when every job is green.

The rules live in the repository, in one file, so a rule is a change of the
code and never a comment of the workflow. The tests of the filters prove
them against a matrix of changes, and they run in the workflow itself:

```sh
make ci-filters-test   # the tests of the filters of the continuous integration
```

The tests also prove that the decision reaches the jobs: the workflow reads
what the job `changes` produces through the id of its step, and a step
without an id produces outputs that no job can read. The tests fail if the
workflow reads the outputs of a step that carries no id, or if the job does
not declare an output for every check the rules name. A run that skips every
check because a decision never arrived is green, and it proves nothing.

A change of the workflow, of the scripts of the continuous integration or
of the root `Makefile` changes what every check proves, so every check runs
when one of them changes. A manual run of the workflow (`workflow_dispatch`)
runs every check too: it carries no change to compare.
