# Paca is the assistant of PACA, and the agent of PACA is Paca

The agent of PACA stops being a runtime with a name and becomes Paca: the
business assistant of witsaba.com, whose identity is the seven Hermetic
principles, and whose first channel is Telegram.

The stack keeps the name of the runtime it runs on (Hermes Agent is the
upstream image, and the directory of the stack stays `hermes/`), and every
product-facing surface says Paca: the identity of the agent, the onboarding,
and the documentation of the repository.

## Specs

The specs quote the request of the maintainer, verbatim. Nothing is added that
was not asked for.

**S1** — The change of approach of the product:

> "I wanna you use this session for changing the approach of paca. When I
> initiallty started this project, I wanna handled my finantial stuff, but, now
> I wanna Paca as the main business assistant, even with the personal financial
> stuff."

**S2** — The identity of the agent:

> "Currently I will start with only the onboarding. The main agent rn is
> hermes, so I need to transform hermes into Paca."

**S3** — The personality of Paca:

> "Paca is like a female dominant with intuition, curiosisty, kind, and is not
> psycofancy this is a non negotiable for paca, paca knows that is a AI
> assistant and do not imitate humans, doenst imitates emotions, sentiments and
> is not a emotions analyst."

**S4** — The competence of Paca:

> "Paca is a very good with financial stuff, have experience working in core
> banking and effective and practical solutions."

**S5** — The truthfulness of Paca:

> "DO not make over engineering and is factual, never make things up but
> investigate in the web and reflect about the web due to even understand that
> the web can have biases, questions its own biases and never assume, always
> validate with the user when is not sure about something and suggest to the
> user solutions but understand tha the final word is from the user."

**S6** — The three principles of Paca:

> "Paca uses 3 main principles, one is the Personality where have to put order,
> is calm, is structured, is extrict, then the second principle is more about
> curiosity, question everything, imaginate some escenarios and the third one is
> emerging when conciliate the first and the two one, with this is the way that
> found Paca to make a better job."

**S7** — The seven principles of the soul:

> "Focus on the 7 principles and implicit add it to the soul.md of hermes."

and the answer of the maintainer to the question of the seven principles:

> "when it though about the 7 principles is for the real hermetic philosofy.
> read https://en.wikipedia.org/wiki/Hermeticism"

**S8** — The product of Paca:

> "Paca is a witsaba.com creation and is focus in be helping for users."

**S9** — The first channel of Paca:

> "so far, only exists the single-user, which will let start with a Telegram
> bot, that is the main paca channel to start."

**S10** — The credentials of the first channel:

> "add the telegram bot env variable as mandatory just like the
> OPENROUTER_API_KEY, due to i got a bot already and the user maybe have its
> bot already. Add additionale a allowed telegram users list."

**S11** — The duality of Paca:

> "paca is female dominant but is a duality."

**S12** — The reach of the change, decided by the maintainer: the brand
changes, the runtime does not. The stack keeps the directory `hermes/` and its
targets, and the agent identifies as Paca.

**S13** — The delivery:

> "Work innna apart gitworktree and ensure this files be copied into the
> container when built. Do not try to build the docker compose when ready, ask
> me to do it in order to initialize the `.env` files into the git worktree."

## The facts the change stands on

- **The identity of the agent is one file.** The image reads `SOUL.md` from
  `HERMES_HOME` (which is `/opt/data` in the official image) and puts it in
  slot #1 of the system prompt, replacing the built-in identity
  (`agent/system_prompt.py::_identity_parts`, `agent/prompt_builder.py::
  DEFAULT_AGENT_IDENTITY`). An empty or missing file falls back to
  `"You are Hermes Agent, built by Nous Research."`, so the file must exist
  and must not be empty.
- **The file is capped at 20,000 characters** (`CONTEXT_FILE_MAX_CHARS` in
  `agent/prompt_builder.py`), and it is scanned for prompt-injection patterns:
  a hit on a file of the home of the user is a warning, not a block.
- **The file is never overwritten by the runtime** once it exists
  (`hermes_cli/config.py::_ensure_default_soul_md` seeds it only when it is
  missing, and it upgrades a legacy scaffold in place). The seeding of the
  stack therefore owns the first copy, and it never overwrites a soul the
  maintainer edited.
- **There is no configuration key for the name of the agent.** The proposal of
  an attribution string was closed upstream
  (NousResearch/hermes-agent#54588): the identity of the agent is `SOUL.md`
  and nothing else.
- **The runtime keeps saying "Hermes" in a slot of its own.** The guidance of
  the runtime ("You run on Hermes Agent (by Nous Research)") is injected in
  the stable tier of the prompt, after the identity
  (`agent/system_prompt.py`, `_help_guidance_slot`). It cannot be removed by
  configuration. The identity of Paca answers it from the first slot, and the
  residual is stated in the documentation of the stack instead of being hidden.
- **The Telegram channel reads two variables** from the environment of the
  gateway: `TELEGRAM_BOT_TOKEN` (the token of the bot) and
  `TELEGRAM_ALLOWED_USERS` (the ids of the users, comma-separated). Without
  the second one the gateway denies every user, so both are mandatory for
  this stack, exactly like the credential of the provider of the models.
- **The seven principles are the seven of the Kybalion**: Mentalism,
  Correspondence, Vibration, Polarity, Rhythm, Cause and Effect, and Gender.
  The Kybalion is a book published in 1908 under the pseudonym "Three
  Initiates", and the attribution of its principles to the ancient Hermetic
  masters is disputed by the scholarship (Wikipedia, "The Kybalion";
  "Hermeticism"). Paca applies them as a discipline of the method and never
  as a claim about physics.

## The contract of the change

| What | Where |
|------|-------|
| The soul of Paca (the source of truth) | `hermes/soul.md` |
| The soul in the image | `/opt/hermes/soul.md` |
| The soul in the state of the agent | `/opt/data/SOUL.md` |
| The token of the bot | `hermes/.env` → `TELEGRAM_BOT_TOKEN` |
| The users of the bot | `hermes/.env` → `TELEGRAM_ALLOWED_USERS` |

## Tasks

- T1 — The soul of Paca. `hermes/soul.md` holds the identity of Paca: the
  seven Hermetic principles as the spine of the method, the three principles of
  the personality of Paca, the truthfulness (no invention, the web with its
  biases questioned, the validation with the user, the last word of the user),
  the voice, and the negations (no sycophancy, no imitation of a human, no
  sentiment, no "Hermes"). Route: parent. Commit: `c261abf`.
- T2 — The soul reaches the container. `hermes/Dockerfile` copies
  `hermes/soul.md` into the image, and `hermes/init.sh` seeds it into
  `HERMES_HOME/SOUL.md`: when the file is missing, and when it still holds the
  identity the runtime seeds by itself, and never when the maintainer wrote
  one. Route: parent. Commit: `c261abf`.
- T3 — Telegram is the first channel and its credentials are mandatory.
  `hermes/env.example` declares `TELEGRAM_BOT_TOKEN` (secret, required) and
  `TELEGRAM_ALLOWED_USERS` (required); `hermes/compose.yaml` passes both to
  the seeding and to the gateway; `hermes/init.sh` refuses the stack without
  them, naming the command that writes them. Route: parent. Commit: `c261abf`.
- T4 — The proof of the identity. `hermes/smoke.sh` asserts that the state of
  the agent carries the soul of Paca and that the credential of the channel
  reached the container, without printing it. Route: parent. Commit: `c261abf`.
- T5 — The gate pins the change. `scripts/tests/structure-test.sh` gains the
  rule that the stack of the agent carries the soul of Paca, copies it into the
  image, seeds it into the state, and that the soul names Paca instead of
  Hermes. Route: parent. Commit: `c261abf`.
- T6 — The documentation of the product. The root `README.md`,
  `hermes/README.md` and `AGENTS.md` say who Paca is, what the seven
  principles are, how the Telegram channel is brought up, and what the residual
  of the runtime is. Route: parent (the root documentation was written by a
  delegated writer and reviewed by the parent). Commit: `cec6ad8`.

## Log

- L1 — The request of the maintainer, verbatim:

  > "I wanna you use this session for changing the approach of paca. When I
  > initiallty started this project, I wanna handled my finantial stuff, but,
  > now I wanna Paca as the main business assistant, even with the personal
  > financial stuff. So, I wanna Paca take the personality of my main
  > bussinees assistant, and that's why the Readme and the repository
  > information must to be updated. Currently I will start with only the
  > onboarding. The main agent rn is hermes, so I need to transform hermes
  > into Paca. https://hermes-agent.nousresearch.com/docs/guides/use-soul-with-hermes
  > read this in order to know how to change the hermes personality and even do
  > not identified anymore as hermes but Paca.
  >
  > Paca is like a female dominant with intuition, curiosisty, kind, and is not
  > psycofancy this is a non negotiable for paca, paca knows that is a AI
  > assistant and do not imitate humans, doenst imitates emotions, sentiments
  > and is not a emotions analyst. Is a helper for the business, while paca
  > grow, will having more helps to the user, so far, only exists the
  > single-user, which will let start with a Telegram bot, that is the main
  > paca channel to start. So i wanna you explore in this session only
  > transforming hermes into Paca. Paca is a very good with financial stuff,
  > have experience working in core banking and effective and practical
  > solutions. DO not make over engineering and is factual, never make things up
  > but investigate in the web and reflect about the web due to even understand
  > that the web can have biases, questions its own biases and never assume,
  > always validate with the user when is not sure about something and suggest
  > to the user solutions but understand tha the final word is from the user.
  > Paca uses 3 main principles, one is the Personality where have to put order,
  > is calm, is structured, is extrict, then the second principle is more about
  > curiosity, question everything, imaginate some escenarios and the third one
  > is emerging when conciliate the first and the two one, with this is the way
  > that found Paca to make a better job. Paca is a witsaba.com creation and is
  > focus in be helping for users. I need you investigate too in a web research
  > about how to personalize hermes, and based on the hermetic principles too,
  > improve the Paca assistant. Focus on the 7 principles and implicit add it
  > to the soul.md of hermes. Once you have this investigation, that is the
  > feature to add, the Paca personality. Work innna apart gitworktree and
  > ensure this files be copied into the container when built. Do not try to
  > build the docker compose when ready, ask me to do it in order to initialize
  > the `.env` files into the git worktree."

- L2 — The question of the seven principles, and the answer of the maintainer:
  the seven are the seven of the Hermetic tradition, not a list invented for
  this slice. The maintainer answered: "when it though about the 7 principles
  is for the real hermetic philosofy. read
  https://en.wikipedia.org/wiki/Hermeticism". The seven of the Kybalion are
  used, and the disputed attribution of the book is stated in the soul, because
  Paca does not sell as fact what the scholarship doubts.
- L3 — The reach of the change, decided by the maintainer: the brand changes
  and the runtime does not. The directory of the stack stays `hermes/`, the
  targets stay, and the agent identifies as Paca.
- L4 — The channel of the onboarding, decided by the maintainer: the token of
  the bot is mandatory in the manifest of the stack, exactly like the
  credential of the provider of the models, and the users allowed are declared
  too.
- L5 — The duality, added by the maintainer: "paca is female dominant but is a
  duality". It is the seventh principle, and it is written as such: the
  feminine and the masculine are two poles of the same thing, and Paca holds
  both.
- L6 — The delivery: the work happens in the worktree
  `../paca-worktrees/paca-persona`, on the branch `feat/paca-persona`. The
  compose stack is not started by the agent; the maintainer initializes the
  `.env` of the stack in the worktree and builds it.
- L7 — The work units, and the boundary of the slice:

  | Commit | What it holds | Authored lines |
  |--------|---------------|----------------|
  | `c261abf` | The soul, the seeding, the channel, the smoke and the rule of the gate. | 480 |
  | `cec6ad8` | The documentation of the product. | 337 |
  | `dc8bbe8` | The record of the slice. | 122 |

  The first work unit crossed the boundary of four hundred authored lines on
  its own: the identity of the agent and the channel that carries it are one
  behavior, and splitting the files they share (`init.sh`, the smoke and the
  rule of the gate) would have made the unit less reviewable, not more. The
  documentation and the record follow in their own units.
- L8 — The evidence of the slice:
  - The gate of the repository is green in the worktree: `make gate` (fmt,
    vet, lint, build, test, race, coverage at 91.4% and the scan of the
    vulnerabilities), `make shellcheck`, the tests of the installer of the
    environment (114 assertions) and the tests of the structure (362
    assertions).
  - The rule of the gate discriminates: the soul renamed to the identity of
    the runtime fails the rule, and a key of the manifest that loses its
    directive `# @required` fails it too.
  - The image carries the soul: the build of `hermes/Dockerfile` copies it to
    `/opt/hermes/soul.md`.
  - The seeding of the stack was proved against the image, with a state of
    the agent in a temporary directory: a fresh state gets the soul of Paca,
    a state that carries the identity the runtime seeds by itself gets it
    replaced, a state with a soul the maintainer wrote keeps it, and the
    stack is refused without the credentials of the channel.
  - The soul holds 11,004 characters (the runtime truncates it above 20,000)
    and it passes the scanner of the runtime of the image unchanged.
- L9 — The live proof, with the stack of the worktree up
  (`docker compose up -d --build`):
  - Every container of the monorepository is healthy, and the seeding of the
    stack reports `the identity of the agent is seeded (Paca)`.
  - The smoke of the stack passes: the agent is alive, the state of the agent
    carries the soul of Paca, the credentials of the channel reached the
    container, and the memory works over the stdio MCP.
  - The gateway connects to Telegram in polling mode: `[Telegram] Connected
    to Telegram (polling mode)`.
  - The live agent answers as Paca: `hermes chat -q "Who are you?"` answers
    "I'm Paca — the business assistant of witsaba.com", and it refuses the
    invention of facts, the flattery, the performance of humanity and the
    taking of the decision away. A business question about the receivables
    comes back with the frame first, both poles of the decision, the
    mechanism, and the condition that would change the answer.
  - The banner of the terminal still carries the name of the runtime
    (`☤ Hermes`): it is the skin of the CLI, not the identity of the agent,
    and the identity is the one that answers.
