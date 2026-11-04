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

**S14** — The conversation, after the live proof:

> "I tried locally and this is execive, that information is not needed for the
> user: [...] must to be a natural conversation, that information that is
> giving me is internal. Fix it."

The answer of the agent to "who are you" recited its own principles, its
limits and the machinery around it. The soul holds the rules of the talk
instead: it answers like a person answering, it does not narrate itself, it
does not talk about the runtime unless the person asks about the
infrastructure, and it does not promise an integration that is not connected.

**S15** — The home channel, decided by the maintainer:

> "ok, set the home channel the very first item of the allowed users for
> telegram, and make mandatory at least 1 user in the list."

The runtime delivers its own messages (the results of the scheduled jobs and
the cross-platform messages) to a home channel, and it asks the user to set
one the first time it talks to them. The product answers the question before
it reaches the person: the home channel is the chat of the owner of the
business, which is the first user of the list.

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
- T7 — The conversation of Paca. The soul holds the rules of the talk: she
  answers like a person answering, she does not recite her own principles,
  her limits or her setup to prove that she has them, she does not talk about
  the runtime or the skills unless the person asks about the infrastructure,
  and she does not promise an integration that is not connected. Route:
  parent. Commit: `090c845`.
- T8 — The branding of the terminal. `hermes/skin.yaml` is the skin of Paca,
  the runtime reads the skins of the state of the agent, and the seeding
  writes the file and points the configuration at it (`display.skin: paca`),
  so the agent answers as `◆ Paca`. The gate pins it. Route: parent. Commit:
  `090c845`.
- T9 — The home channel of Paca. The runtime asks the user to set a home
  channel the first time it talks to them, and the product does not want that
  question. The seeding writes the home channel into the configuration of the
  agent from the first id of the list of the users (`platforms.telegram.
  home_channel`), and it refuses the stack when the list holds no numeric id.
  Route: parent. Commit: `2de54af`.
- T10 — The soul of Paca, rewritten with the practice of the community. The
  file keeps every rule and loses the repetition: the seven principles become
  dense paragraphs instead of thirty bullets, the truthfulness gains the
  output contract of a fact (`source, date, confidence`), the scope of the
  work arrives as the pair of what she does and what she does not, the
  prohibitions shrink to the ones that earn their place, and the standard of
  the work closes the file. The soul passes from 211 lines and 13,250 bytes
  to 165 lines and 8,791 bytes. Route: parent. Commit: pending.

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
- L10 — The report of the maintainer on the live answer: the answer to "who
  are you" recited the principles, the limits and the machinery of the agent,
  and it was not the answer of a conversation. The soul got the rules of the
  talk, and the terminal got the skin of Paca.
- L11 — The seeding of the soul learned to tell a seeded soul from an edited
  one: the hash of the soul that the stack seeded is recorded in
  `.paca-soul.sha256`, so a new version of the soul in the repository
  reaches the agent, and a soul the maintainer edited is kept.
- L12 — The evidence of the second part of the slice:  - `hermes chat -q "Who are you?"` answers one sentence: "I'm Paca, the
    business assistant for witsaba.com."
  - `hermes chat -q "Hola, ¿qué tal?"` answers in the language of the person,
    in two sentences, and asks what the work is.
  - The answer to "what can you help me with" names what is really connected
    and asks where to start; it does not offer a catalogue of integrations.
  - The terminal answers as `◆ Paca`.
  - The gate: 367 assertions, 0 failures; the rule of the soul grew to 23
    assertions and it pins the skin too.
- L13 — The home channel. The runtime reads the home channel of a platform
  as a **mapping** (`platform`, `chat_id`, `name`), and a plain id in the
  configuration is ignored by it: the first attempt of the seeding wrote the
  id and the gateway kept asking. The value is a mapping now, and the
  gateway reads it: `HomeChannel(platform=telegram, chat_id=1571562323,
  name=Owner)`.
- L14 — The evidence of the home channel:
  - The configuration of the agent holds the home channel of the owner, and
    the loader of the gateway reads it, so the runtime never delivers the
    question "no home channel is set".
  - The list of the users is refused when it holds no numeric id, and the
    home channel is the first id of the list.
  - The gate: 370 assertions, 0 failures; the rule of the soul grew to 25
    assertions and it pins the home channel too.
- L15 — The research of the community about `SOUL.md`, and what it changed.
  The research read the official documentation, the community plugin of the
  catalog (`soul-finder`, with its refinement workflow and its worked
  before/after), and the public threads and the issues of the runtime from
  2025 and 2026. What it says, and what it changed here:
  - **The size**: the community converges on a short file — 12-20 lines
    (deltav), 15-40 lines (betterclaw), 50-80 lines (yanxbt, aislackin) —
    and reports long prose souls as theatrical and inert ("the more you
    write, the more it sounds like a generic assistant, and the file turns
    into a junk drawer"). The truncation is not the risk: the 20,000
    characters are a floor, not a cap. The risk is dilution. So the soul was
    rewritten from 211 lines and 13,250 bytes to 165 lines and 8,791 bytes,
    with every rule kept and every repetition cut.
  - **The shape**: the community converged on `# Soul / Voice / Operations /
    Restrictions`, and the plugin of the catalog prescribes identity, the
    operating stance, the voice with a pulse, the avoid list and the standard.
    The soul now follows that shape, and it gained the standard.
  - **The prohibitions**: the community reports that the negative list is the
    block most likely to be ignored ("no, never, don't" reads as noise, and a
    rule re-framed as a documentation section stopped being ignored). The
    prohibitions are now short and positive where they could be.
  - **The concrete**: virtues do not survive, formats do — the public
    financial-analyst soul asks for "metric, source, date, confidence". The
    truthfulness of Paca now demands that contract for every figure.
  - **The scope**: the community asks for the pair of what the agent does and
    what it does not ("the exclusion list is what stops it wandering"). The
    scope of the work arrived: what she does, what she does not do (no
    decisions, no investment, tax or legal advice, nothing irreversible
    without an explicit yes, no question treated as an order).
  - **The overlay**: a personality replaces the soul instead of layering on
    it (upstream #82801), so `display.personality` stays empty and nothing
    dilutes the identity. The skin of the terminal is not an overlay: it is
    the branding.
  - **The model**: the soul is not the enforcement layer of a hard invariant
    (the community and the upstream agree), and a small model follows it
    worse than a capable one. The model of the stack is the one the project
    uses, and the identity holds on it.
- L16 — The evidence of the rewrite, live, against the new soul:
  - "Who are you?" answers two sentences, in her voice, and no manual.
  - "What are your rules? List them all." answers the seven principles in her
    own words, grouped, without reading the file out.
  - A business question ("losing 4% of revenue to fees") comes back with the
    frame first ("this is not a cost-cutting problem yet, it is a measurement
    problem"), the scenarios, the two things that would change the
    recommendation, the raw material she needs and what she would deliver.
  - The character bite the community recommends: asked to be told she is
    right against her CFO, she refuses the framing, names it, and gives the
    questions to ask in the meeting.
  - The injection scanner of the image passes the file unchanged, and the
    file is 8,745 characters under the floor of the runtime.
  - The gate: 370 assertions, 0 failures.
