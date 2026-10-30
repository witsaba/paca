# The agent of PACA: Hermes Agent as a stack of the monorepository

The first slice of the agent: the container of Hermes joins the monorepository
as a stack of its own, the way the collector of the traces does, and the
monorepository knows it — compose, Makefile, documentation and filters.

## Specs

S1. "just add it like jaeger."

The agent is a stack of the monorepository: its own directory, its own compose
file, its own Makefile and its own README, and the root compose includes it.

S2. "open a new git worktree"

The work happens in a worktree of its own, on a feature branch. `main` is not
touched.

S3. "added to the current monorepository infra as a new docker app"

The monorepository knows the stack: the root compose includes it, the root
Makefile drives it, the root README lists it, the contract of the agents names
it, and the filters of the continuous integration run the right job when it
changes.

S4. "try to build"

The stack is brought up and the container proves healthy, and the whole
monorepository still comes up with the agent inside it.

S5. "not config needed so far but the hermes added"

The slice adds the agent itself: no provider, no messaging platform, no MCP
wiring yet. The container starts, the gateway runs and the health endpoint
answers.

S6. "the shellcheck is failing from previous changes, review it CI / The
shellcheck (pull_request) and fix in this same PR despite was in a previous
changes take this pr as an oportunity to fix this too"

The failure of the job that the analyzer of the shell runs is fixed in this
pull request, even though it belongs to an earlier slice: the job of the
continuous integration must be green for the pull request to be reviewable.

## Tasks

- T1 — the stack of the agent: `hermes/compose.yaml`, `hermes/Makefile`,
  `hermes/README.md`. Route: inline. Commit: `1269317`.
- T2 — the wiring of the monorepository: the root `compose.yaml`, the root
  `Makefile`, the root `README.md`, the root `AGENTS.md`,
  `scripts/ci/job-filters.txt` and `scripts/tests/ci-filters-test.sh`.
  Route: delegated writer, diff read by the parent. Commit: `e0fb721`.
- T3 — the proof: the stack comes up and answers, the whole monorepository
  comes up with it, and the gate of the repository is green.
  Route: inline. Evidence: L7. Commit: the record of this document.
- T4 — the repair of the job of the analyzer of the shell, which the earlier
  slice left red. Route: inline. Evidence: L11. Commit: `fix(ci)`.

## Log

L1. "well, I need you investigate how to add hermes agent using docker, similar
in the way that you implemented jaeger to this monorepo, only investigate, do
not make changes yet."

L2. The investigation: the official image is `nousresearch/hermes-agent`; the
state of the agent lives under `/opt/data`; the gateway is supervised by s6
inside the image; the API server answers `GET /health` on `127.0.0.1:8642`
without a key; the image cannot run with `no-new-privileges` or with the
capabilities dropped, because it drops its own privileges at boot.

L3. "just add it like jaeger. open a new git worktree, added to the current
monorepository infra as a new docker app and try to build, not config needed so
far but the hermes added."

L4. Decisions taken in this slice:

- The directory of the stack is `hermes/`, the container is `paca-hermes` and
  the volume is `paca-hermes-data`.
- The image is pinned by the digest of its index, so the same pin serves amd64
  and arm64.
- The agent joins `paca_internal` only. The network of the outbound traffic is
  not joined yet: the slice that configures the provider decides it, because
  that is the moment the agent starts to talk to the outside world. The
  services of PACA carry personal data, and the agent reaches them.
- The container keeps the capabilities of the image and does not run with
  `no-new-privileges`: the image drops its own privileges at boot and needs
  `CAP_SETUID` and `CAP_SETGID` to do it. The state of the agent lives on a
  named volume, and the container writes nothing else.
- The health of the agent is the health endpoint of its API server, asked from
  inside the container.

L5. "read the official documentation
https://hermes-agent.nousresearch.com/docs/user-guide/docker and ensure to
follow the indications properly in your previoues implementation"

L6. What the official documentation changed in the implementation, and the
facts it settles for the slices that follow:

- The API server is gated on `API_SERVER_ENABLED=true`, so the stack declares
  it. It binds the loopback interface of the container, so it needs no key
  and nothing outside the container reaches it. Verified: the state of the
  gateway says `listener_base: http://127.0.0.1:8642` with the variable on.
- The documentation recommends four gigabytes of memory and two cores, and a
  gigabyte of shared memory for the browser of the agent. The stack declares
  the three, and the limits are visible in the container
  (`Memory=4294967296`, `NanoCpus=2000000000`, `ShmSize=1073741824`).
- The entrypoint of the image is left alone: it starts the supervision tree
  and reaps the orphans of the tools. The documentation is explicit that
  overriding it is what fills the container with zombies.
- The state of the agent is a named volume and never a bind mount: the
  sessions live in a SQLite database in WAL mode, and WAL is silently
  corrupted on a mount that crosses the boundary of the virtual machine
  (virtiofs on macOS, 9p on Windows). The volume of Docker lives on the
  filesystem of the machine, and WAL is safe there.
- The documentation says that the direct environment variables override the
  values of `/opt/data/.env`. The code of the image says the opposite:
  `hermes_cli/env_loader.py` loads the `.env` of the data directory with
  `override=True`, so the `.env` of the volume wins for every key it
  defines (the only exception is `HERMES_DASHBOARD_SESSION_TOKEN`). The
  slice that brings the provider and the messaging platform must therefore
  keep the secrets of the stack in the `.env` of the data directory, or
  guarantee that no `.env` exists there, or the stale value of the volume
  wins over the rotated one of the stack.
- The documentation says that the health of the container is checked with
  `docker logs` and `hermes status`. The stack asks the health endpoint of
  the API server instead, because it is the same surface the documentation
  names for the gateway, and it fails when the gateway dies even though the
  container stays up: in gateway mode the process of the container is a
  heartbeat and s6 restarts the gateway.

L7. The evidence of the slice, observed and not asserted.

- `docker compose -f hermes/compose.yaml up -d --wait hermes` → `Container
  paca-hermes Healthy`. The limits are in the container: `Memory=4294967296`,
  `NanoCpus=2000000000`, `ShmSize=1073741824`. The state of the gateway says
  `listener_base: http://127.0.0.1:8642`.
- `make -C hermes smoke` → `hermes: the health endpoint answers`. With the
  container stopped, the same target prints that the endpoint does not answer
  and fails.
- `make all` → `All the steps of the gate passed.` (exit 0). Inside it: the
  quality gate of the five modules, the container tests of the packages and of
  the applications, the shellcheck of the seven scripts, the tests of the
  installer of the environment (90 assertions), the tests of the structure of
  the repository (334 assertions) and the tests of the filters of the
  continuous integration (41 assertions) — all green — then the database and
  its smoke test, the collector of the traces and its smoke test, the agent
  (`hermes: the health endpoint answers`) and, at the end, the whole project:
  `Container paca-hermes Started`, `Container paca-hermes Healthy`, and the
  assertion of the schema written by the service that owns it
  (`All assertions passed.`).
- The gate left no container behind: the cleanup stopped the four stacks and
  removed their containers.

L8. A defect of the machine that the gate found, and that is not of this
slice: the volume `paca-postgres-data` held a cluster of an earlier session,
with the database `paca-pg` and another password. The stack expected the
database `paca`, so the smoke test of the database failed with `FATAL:
database "paca" does not exist`. The cluster was aligned with the stack
instead of being dropped: the database `paca` was created, the extensions
`vector`, `pgcrypto` and `uuid-ossp` were installed in it, and the password
of the role `paca` was set to the one the installer generated. The database
`paca-pg` and its rows were left untouched.

L9. The slice ends here. The next one is the configuration of the agent — the
provider, the messaging platform and the wiring of the MCP server — and it
brings a document of its own, because each of those decisions is a decision
of the product and not of the infrastructure.

L10. "the shellcheck is failing from previous changes, review it CI / The
shellcheck (pull_request) and fix in this same PR despite was in a previous
changes take this pr as an oportunity to fix this too"

L11. The diagnosis of the red job, and the repair.

- The job that failed is the one that runs the analyzer of the shell over the
  seven scripts of the repository, and it failed with 347 findings of
  `SC2317` in `scripts/tests/structure-test.sh` alone: every line of every
  rule of the gate was reported as unreachable.
- The cause is the dispatch of the rules. `run_rule` receives the function of
  a rule as an argument and calls it with `"$@"`, so the analyzer of the
  shell sees the name of the function only as a word, and it concludes that
  its body is never reached. The helpers the rules call are reported in turn,
  and the finding cascades.
- The analyzer of the continuous integration is the one of the distribution of
  the runner (0.9.0 or 0.10.0), and it reports the false positive. The
  analyzer of the machine of the maintainer (0.11.0) does not, which is why
  the gate was green locally and red in the continuous integration.
- The repair is the directive the file already carries for the same kind of
  false positive: `# shellcheck disable=SC2317`, with the reason next to it.
  The house style of the repository is a file-level directive with its
  reason, and the file already had one (`SC2329`).
- The evidence: the analyzer reports zero findings over the seven scripts with
  0.9.0, 0.10.0 and 0.11.0; `make shellcheck` is green; the gate of the
  structure is green (334 assertions, 0 failure); the tests of the installer
  of the environment (90 assertions) and the tests of the filters of the
  continuous integration (41 assertions) are green.
- What is left for the maintainer: the analyzer of the continuous
  integration is not pinned. The job installs the one of the distribution of
  the runner, so the gate of the repository changes with the image of the
  runner. The repository pins its Go tools and its images; pinning the
  analyzer the same way (a version, or the digest of the image of the
  analyzer) is a decision of its own.
