# The installer of the environment: the `.env` files of the monorepository

## Specs

The user's request, verbatim:

> open a new git worktree to work in a script that allows to generate the necesary .env files for the user when want to install, MUST to even offer the default value and when being password, bring a option to generate a strong password. The script MUST to allow to keep the previous configured values but modify if the user wants to.

The answer to the question about the scope of the installer, verbatim:

> Manifest-driven, all stacks

**S1 — A worktree of its own.** The work happens in the worktree `/Users/braejan/workspace/witsaba/repositories/paca-env-init`, on the branch `feat/env-init`, cut from `main` at `71945b3`. `main` is not touched.

**S2 — One script that generates the necessary `.env` files.** A single script generates the `.env` files a person needs to install and run the monorepository. It is the only entry point; the person who installs does not copy files by hand and does not edit a compose file.

**S3 — The discovery is manifest-driven, and every stack joins automatically.** The script walks the repository and finds every file named `env.example`. Each one is the manifest of a stack, and the file it generates is the `.env` next to it. A new stack that ships an `env.example` is installed by the same command, with no change to the script and no list to update. The walk ignores `.git`.

**S4 — The manifest stays a valid Compose file.** `env.example` keeps the format Docker Compose reads: only `KEY=value` lines and comments. `cp db/postgres/env.example db/postgres/.env` keeps working. The metadata of the manifest lives in comments, never in a second file.

**S5 — The metadata of the manifest.** Three directives, each on its own comment line, apply to the key line that follows them:
- `# @secret` — the value is a secret. It is never printed to the terminal, and the file that holds it is written with the mode `600`.
- `# @generate` — a strong value can be generated for it.
- Any other comment line directly above a key is its description: the text shown to the person who installs.

**S6 — Every value is offered with its default.** The script never installs a value the person did not see. For every key of the manifest, the script shows the description, the name of the key and the default it offers.

**S7 — The default of a value is the value already configured.** When the `.env` exists and holds a value for a key, that value is the default offered. When the key has no value yet, the default is the value of the manifest. Nothing that was configured before is lost, and the person who installs can change any of them.

**S8 — A password can be generated, strong.** A secret marked `# @generate` gets a strong password generated for it: 32 characters taken from `A-Za-z0-9-_`, read from `openssl rand` and, when `openssl` is missing, from `/dev/urandom`. The characters are safe in a `.env` file, in a Compose file and in a connection string: no `$`, no `#`, no quote, no space.

**S9 — A password is never offered as a weak placeholder.** The placeholder of the manifest (`change-me-generate-a-strong-password`) is never written silently into a `.env`. A secret marked `# @generate` with no value configured yet is offered a freshly generated strong password. A secret that is not marked `# @generate` is offered the value of the manifest, and the script warns that the value is a secret the person must change. An empty value counts as no value configured: the default of the manifest applies again, and a secret that can be generated gets a strong password.

**S10 — The password of the manifest is never printed.** A secret is never written to the terminal, never printed by the dry run, and never written to a log. The files that hold a secret are written with the mode `600`.

**S11 — The previous values are kept, and can be modified.** Running the script again over a `.env` that already exists keeps every value that is still declared by the manifest, and keeps the values the person added by hand, even when the manifest does not declare them. The person who installs can change any value: the offered default is the current one, and any other answer replaces it.

**S12 — The answers are asked one by one.** In the interactive mode, every key is asked in the order of the manifest, and the answers are read from the standard input. The answer that keeps the default is the empty answer. The answer `g` generates a strong value for a secret marked `# @generate`. Any other answer replaces the value.

**S13 — The script also works without a person.** `--yes` asks nothing: it keeps every configured value, generates a strong password for a secret marked `# @generate` that has no value yet, and uses the value of the manifest for the rest. The script must not block on the standard input in that mode, and it must work when the standard input is closed.

**S14 — The dry run writes nothing.** `--dry-run` prints what it would write — the keys, the values, the secrets hidden — and writes no file at all.

**S15 — The written file is complete and ordered.** The generated file holds the keys of the manifest, in the order of the manifest, then the keys that were configured by hand and are not declared by the manifest. The header of the file says that it is generated, names the manifest it comes from, and says that it must never be committed.

**S16 — The write is atomic.** The file is written to a temporary file in the same directory and renamed over the target, so an interrupted run never leaves a half-written `.env` behind.

**S17 — A file that does not change is not rewritten.** When the generated content is identical to the content already on disk, the file is not written again: its modification time stays untouched.

**S18 — The exit codes are meaningful.** `0` when the run succeeds, `1` on a usage or a write error, `2` when the person interrupts the run.

**S19 — The manifest is never modified.** The script only writes `.env` files. `env.example` is read, never touched.

**S20 — The installation has one command.** The root `Makefile` exposes the installer (`make env-init`), so the continuous integration and the person who installs use the same entry point. The root of the repository keeps no stack-specific logic.

**S21 — The tests of the installer are real.** The tests run the script against fixtures in a temporary directory, never against the `.env` of the repository, and they assert the outcome through the public interface of the script: the files it writes, their content, their mode, and what it prints.

**S22 — The documentation tells the person how to install.** The root README and the README of the database stack explain the installation with the installer, and the manual `cp env.example .env` stays documented as the fallback.

**S23 — The secrets of the installation stay out of the repository.** The generated `.env` files are never committed: the root `.env` is added to `.gitignore`, next to `db/postgres/.env`.

## Tasks

- **T1** (S3, S4, S5) — The manifest format: the directives of `db/postgres/env.example`, the descriptions, and the root `env.example` with the version of the images. Commit: `feat(env): the manifest of the environment of every stack`.
- **T2** (S2, S6, S7, S8, S9, S10, S11, S12, S13, S14, S15, S16, S17, S18, S19) — `scripts/setup-env.sh`: the discovery, the parsing of the manifest, the resolution of the defaults, the interactive mode, `--yes`, `--dry-run`, `--password`, the generation of the strong password, the atomic write. Commit: `feat(scripts): the installer of the environment of the monorepository`.
- **T3** (S21) — `scripts/tests/setup-env-test.sh`: the fixtures and the assertions. Commit: `test(scripts): the tests of the installer of the environment`.
- **T4** (S20, S22, S23) — The wiring: the root `Makefile` (`env-init`, `env-test`), the continuous integration, the `.gitignore`, the root README and the README of the database stack. Commit: `chore(env): the installer in the Makefile, the CI and the documentation`.

## Log

- **L1** — The user's request, verbatim:

  > open a new git worktree to work in a script that allows to generate the necesary .env files for the user when want to install, MUST to even offer the default value and when being password, bring a option to generate a strong password. The script MUST to allow to keep the previous configured values but modify if the user wants to.

- **L2** — The scope question was asked before any code, with the recommendation of the database stack alone. The answer, verbatim: "Manifest-driven, all stacks". The discovery is therefore driven by the `env.example` files of the repository, and every stack joins automatically.
- **L3** — The worktree `/Users/braejan/workspace/witsaba/repositories/paca-env-init` was created from `main` at `71945b3`, on the branch `feat/env-init`.
- **L4** — The decision on the shape of the manifest: the metadata lives in the comments of `env.example`, so the file stays a valid Compose file and there is no second file to keep in sync.
- **L5** — The decision on the place of the script: `scripts/setup-env.sh` at the root, because the installer spans the stacks; the stacks keep their own tooling, and the root dispatches.
- **L6** — The decision on the secrets: a strong password is 32 characters of `A-Za-z0-9-_`, because those characters are safe in a `.env` file, in a Compose file and in a connection string, and `openssl rand -base64` produces `+` and `/`, which are not.
- **L7** — The decision on the non-interactive mode: `--yes` keeps what is configured and generates the strong password of a secret that has no value yet, so the continuous integration and a scripted installation never depend on a terminal.
- **L8** — The decision on the tests: the fixtures live in a temporary directory and the script is pointed at them with `--root`, so the `.env` of the repository is never touched by the tests.
- **L9** — The work units and their commits, on the branch `feat/env-init`:

  | Task | Commit | What it holds |
  |------|--------|---------------|
  | T1 | `72c0874` | The manifest format: `db/postgres/env.example` and the root `env.example`. |
  | T2 | `f722672` | `scripts/setup-env.sh`: the installer of the environment. |
  | T3 | `0954509` | `scripts/tests/setup-env-test.sh`: 19 cases, 59 assertions. |
  | T4 | `0aa9a47` | The wiring: the root `Makefile`, the continuous integration, the `.gitignore` and the documentation. |

- **L10** — The size of the slice: 1563 authored changed lines in four commits, over the 400 lines of the generic rule of the harness. The practice of this repository wins (one pull request per feature slice, the commits are the review path), so the slice is not split on its own: the maintainer decides. The deviation is stated in the body of the pull request, with the offer to split it.
- **L11** — The verification of the slice, in the worktree: `make shellcheck` green over the three scripts of the repository, `make env-test` green (59 assertions, 0 failures), `make env-init ARGS="--dry-run"` writes nothing, and the real generation over the credentials of the running stack keeps the configured password, writes the file with the mode `600`, prints the secret masked, leaves the modification time untouched on a second run, and passes `make db-smoke` against the running PostgreSQL.
- **L12** — The independent verification of the slice (`gentle-ai-verify`, over the committed range `71945b3..895c4c2`) found one defect and three advisories, and all four are fixed in the commit `fix(scripts)`: (1) **D1** — an empty configured value counted as a configured value, so a `.env` with `POSTGRES_PASSWORD=` kept the empty value and the PostgreSQL image refused to start; an empty value is now no value, and the default applies again. (2) **D2** — the mode of a `.env` that is a link was changed through the link, so a file outside of the root was touched; a link is now left alone. (3) **A1** — the temporary file of the atomic write was not ignored by git; it is now. (4) **A2** — a key that the manifest does not declare may be a secret, so a file that holds one is now private as well.
- **L13** — The verification also found that S4 asked for the continuous integration to keep copying the example. The clause was rewritten: the continuous integration now creates its credentials with the installer, so the weak placeholder never reaches a running cluster. The change is in the commit `chore(env)`, and the manual copy stays documented as the fallback.
- **L14** — The verification found that the description of a key was shown only when the value was asked. It is now shown for every key that nobody was asked about, so the report of the non-interactive modes carries the description of every value it installs.
