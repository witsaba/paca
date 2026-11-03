# The agent of PACA reads the memory of PACA: Engram over the stdio MCP

The slice that connects the agent to the memory: the container of the agent
carries the binary of the store, the configuration of the agent names the
store as an MCP server over the stdio transport, and the credentials of the
agent (the key of the provider and the model) are asked by the installer of
the environment. The MCP server of PACA is not touched by this slice.

## Specs

S1. "Ok let use this session to ensure a proper connection between hermes and engram only DO NOT FOCUS YET in connecting the paca mcp server yet. Only ensuring engram memory to hermes."

S2. "Is needed that for building the project, when installing hermes, ask for the OPENROUTER_API_KEY mandatory, it must to Be in the .env (dont read or modify this file, NEVER) this must to be in charge of the devops enginner that run the project."

S3. "Ensure that this option be when running the script for customizing the env variables."

S4. "Add to a AGENT_MAIN_MODEL env variable which will have for default the `minimax/minimax-m3.1-flash` model."

S5. "Ensure too that the hermes agent properly connect engram with the stdio MCP It means is needed that the engram can see the binary (let me know if this is not posible or not trivial) investigate too in a web research if needed."

S6. "I'll connect to the hermes tui when the docker compose is up and test is using and saving in engram. If not ease, is time to up the MCP server http of engram and connect it to hermes."

S7. "START ALWAYS reading the engram project memories and data before reading the projects files, save this as a user preference in the AGENTS.md too"

S8. "I dont want again you working in branches, ALWAYS open a new gitworktree for it."

S9. "let proceed with the plan, let me know when ready to run the script for setting the openrouter api key before building, even, run the docker compose up and ensure that fails due to the api key is not setted yet."

S10. "Yes: join paca_egress now" (the answer of the maintainer on the outbound
access of the agent).

S11. "Mandatory interactively, warning in --yes" (the answer of the maintainer
on the strictness of the installer).

## Tasks

- T1 — the rules of the session: `AGENTS.md`. Route: inline. Commit: `16f8e22`.
- T2 — the credentials of the agent: `hermes/env.example`, the directive
  `# @required` and the environment override of `scripts/setup-env.sh`, the
  tests of `scripts/tests/setup-env-test.sh`, the help of the installer and
  the documentation that describes it. Route: delegated writer, diff read by
  the parent. Commit: `ac5146b`. Evidence: L7.
- T3 — the image of the agent carries the binary of the memory:
  `hermes/Dockerfile`, the build, the volumes, the networks and the init
  service of `hermes/compose.yaml`, and the rule of
  `scripts/tests/structure-test.sh` that pins the payload of the memory
  against the drift. Route: inline. Commit: `897b9a2`. Evidence: L8.
- T4 — the agent reaches the memory over the stdio MCP: `hermes/init.sh`,
  `hermes/smoke.sh`, the targets of `hermes/Makefile`. Route: inline.
  Commits: `4f9888c`, `80aa3d4`. Evidence: L8.
- T5 — the wiring of the continuous integration and the documentation:
  `.github/workflows/ci.yml`, `hermes/README.md`, the root `README.md`, the
  root `AGENTS.md`. Route: inline. Commits: `4d9ebbf`, `640557d`.
- T6 — the proof: the stack refuses to come up without the key of the
  provider, the round trip of the memory is asserted from inside the
  container of the agent, and the gate of the repository is green.
  Route: inline. Evidence: L8. Commit: this one.

## Log

L1. "Ok let use this session to ensure a proper connection between hermes and engram only DO NOT FOCUS YET in connecting the paca mcp server yet. Only ensuring engram memory to hermes. Is needed that for building the project, when installing hermes, ask for the OPENROUTER_API_KEY mandatory, it must to Be in the .env (dont read or modify this file, NEVER) this must to be in charge of the devops enginner that run the project. Ensure that this option be when running the script for customizing the env variables. Add to a AGENT_MAIN_MODEL env variable which will have for default the `minimax/minimax-m3.1-flash` model. Ensure too that the hermes agent properly connect engram with the stdio MCP It means is needed that the engram can see the binary (let me know if this is not posible or not trivial) investigate too in a web research if needed. I'll connect to the hermes tui when the docker compose is up and test is using and saving in engram. If not ease, is time to up the MCP server http of engram and connect it to hermes. This session is only for this. Let start exploring it and tell me the technical viability before starting. I dont want again you working in branches, ALWAYS open a new gitworktree for it. START ALWAYS reading the engram project memories and data before reading the projects files, save this as a user preference in the AGENTS.md too"

L2. The exploration, and the viability it settles.

- The stdio path is PROVEN: the previous session ran a live round trip inside
  the real image of the agent (Engram #4671). The binary of the store runs in
  the image of the agent as uid 10000, `engram mcp --tools=agent` completes
  the handshake, exposes 23 tools, and a `mem_save` lands in the project
  `paca` and reads back.
- Engram has NO MCP-over-HTTP: `engram serve` is a REST API on the loopback
  (or on the unix socket) and `engram mcp` is stdio only. The HTTP fallback
  of S6 does not exist upstream (issue #112 is an open proposal). The stdio
  transport is the only path, and it works: the fallback is not needed.
- Hermes reads MCP servers ONLY from `config.yaml` -> `mcp_servers`. A stdio
  entry is `{command: string, args: [string], env: {}}`
  (`tools/mcp_tool_config.py`, `cli-config.yaml.example:1609`).
- The environment of the stdio child is a SAFE BASELINE (PATH, HOME, TERM,
  XDG_*) plus the `env:` block of the server itself
  (`mcp_tool_config.py:199`): `ENGRAM_DATA_DIR` and `ENGRAM_PROJECT` must be
  declared inside that block, never expected from the container environment.
- `config.yaml` lives at `$HERMES_HOME/config.yaml` and Hermes REWRITES it at
  boot, so it can never be a read-only mount: it must be seeded into the
  volume by a one-shot init service. The image's dispatcher runs the command
  of the container as the main program of `/init`, drops to uid 10000 and
  gives the container the exit code of that command (proved: exit 7 in,
  exit 7 out).
- The seeding uses the writer of Hermes itself: `hermes config set` is not
  interactive, creates the file when it is missing, and parses the YAML and
  JSON literals (so `args '["mcp","--tools=agent"]'` becomes a real list).
  `hermes mcp add` is not usable in an init service: it probes the server and
  asks which tools to enable.
- The model of the agent is `model.default` with `model.provider: openrouter`,
  and the key is the environment variable `OPENROUTER_API_KEY`. The ids of
  OpenRouter use the plain `<author>/<slug>` form, so `minimax/minimax-m3.1-flash`
  is already the right shape.
- The key must reach the container through the `env_file` of the stack and
  NEVER through `environment:`: in the root compose project the `${VAR}` is
  interpolated from the root `.env`, so an `environment:` entry would replace
  the real key with an empty string.

L3. The decisions the maintainer took on the two open questions.

- The outbound access of the agent is taken in this slice: the agent joins
  `paca_egress` and reaches the provider of the models. This is the decision
  the stack of the agent deferred (Engram #4655), and the key of the provider
  makes it unavoidable.
- The installer is mandatory in the interactive mode and a warning in the
  `--yes` mode, so the continuous integration stays green.

L4. "let proceed with the plan, let me know when ready to run the script for setting the openrouter api key before building, even, run the docker compose up and ensure that fails due to the api key is not setted yet."

L5. The plan that was agreed, and the proof it asks for.

- The stack of the agent REFUSES to come up when the key of the provider is
  not configured: the init service reads it from the environment of the
  stack and fails with a message that names the command that writes it.
- The proof of the memory: a round trip from inside the container of the
  agent, over the stdio MCP, into the project `paca` of the store.

L6. The rules of the session, in `AGENTS.md` (commit `16f8e22`): the memory
of the project is read before the files of the project, the `.env` files are
never read and never written by the agent, and every change happens in a
worktree of its own.

L7. The installer of the environment, and the delegated writer that wrote it
(commit `ac5146b`).

- The manifest of the stack of the agent declares the key of the provider as
  `# @required`: the interactive run refuses it with the exit code 1, and it
  writes the file of the stack empty anyway, because the refusal of the
  stack has to reach the operator and not a missing file of Compose.
- The `--yes` run only warns: nobody can answer a question there.
- The environment of the process is a source of values: it fills a key that
  the stack does not configure, and it never overrides a value the stack does
  configure.
- The proof of the unit: 114 assertions of `scripts/tests/setup-env-test.sh`,
  and the suite discriminates: with the exit code of the refusal turned into
  a success, the suite fails on `the run fails: expected [1], got [0]`.

L8. The proof of the slice.

- The image and the wiring (commit `897b9a2`): the image of the agent
  carries the binary of the store, verified against the checksum of the
  release, and the rule of `scripts/tests/structure-test.sh` proves that the
  payload of the memory cannot drift between the stack of the agent and the
  stack of the memory. The suite holds 347 assertions, and the rule
  discriminates: a deliberate drift of the pin fails the suite.
- The round trip (commit `4f9888c`, repaired in `80aa3d4`): from inside the
  container of the agent, over the stdio MCP, a memory is written into the
  project `paca` of the store and given back:
  `hermes: the memory is written and given back over the stdio MCP`.
- The refusal without the key: `docker compose -f hermes/compose.yaml up
  --wait` ends with the exit code 1 and the operator reads
  `hermes: OPENROUTER_API_KEY is not configured.`,
  `hermes: the agent cannot call a model without it. Write it with:` and
  `hermes:   make env-init --stack hermes`.
- The rebuild from zero of the maintainer: the installer wrote the three
  files of the environment, `docker compose up -d --build` built the five
  images and brought up the six services with the init service exited with
  the code 0, and every service is healthy.
- The gate of the repository from zero: `make all` ends with
  `All the steps of the gate passed.` The steps that carry the slice are
  green one by one: the installer (114 assertions), the structure of the
  repository (347 assertions), the linter of the shell (11 scripts), the
  smoke of the database, the smoke of the memory (`engram: the memory is
  kept and given back`) and the smoke of the agent (`hermes: the memory is
  written and given back over the stdio MCP`).

L9. The memory of the project holds the discoveries of this slice
(Engram #4672 and #4673): the stdio path of the store, the shape of its
answers, and the shape of the configuration of the agent that names it.
