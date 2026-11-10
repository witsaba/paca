# The keepers of the image of the database

The database of PACA is a custom artifact: the official PostgreSQL image
pinned by digest, plus pgvector compiled from its checksummed source, plus the
cluster flags and the init scripts. The continuous integration proves that the
artifact still delivers its promises; this feature gives that proof the three
keepers it was missing: one image everywhere, a keeper for the freeze, and a
scan of the artifact.

## Specs

The specs quote the maintainer, verbatim. Nothing is added that was not asked
for.

**S1** — The reason of the step of the database in the continuous integration:

> "but what is the reason of the The database step in CI?"

**S2** — The risk of an artifact that is not the official one:

> "is not a risk to hava non official postgres ?"

**S3** — The decision, asked for as a recommendation:

> "what is your recommended one based on good practices and lower risk?"

**S4** — The order to do it:

> "ok, do you recommended one"

**S5** — The recommendation that S4 accepts, in its own words:

> "**Keep the custom pgvector build** (Option A: checksum-verified official
> source, compiled by the toolchain the digest already pins). Do not move to the
> PGDG package, and do not move to the community image."

> "**The three keepers, in the order of value**
>
> 1. **A keeper for the freeze — dependabot for the digest and the extension
>    version.** This is the biggest real exposure in the whole stack today:
>    `.github/` contains only `workflows/`, so upstream PostgreSQL security
>    patches and pgvector releases reach PACA **only when a human remembers**.
> 2. **A scan of the artifact — trivy in the "The database" job.** The gate's
>    `vuln` step is `govulncheck` over the Go dependencies; the container image
>    and the extension are scanned by nothing today.
> 3. **One version, one owner — the pin tripwire.** The stack pins the image by
>    digest while `packages/database/test/integration/container_test.go:44` pins
>    a floating `postgres:18.6-bookworm`; the test that refuses the day the two
>    disagree closes the last hole."

## Tasks

| Task | What | Specs | Route | Commit |
|------|------|-------|-------|--------|
| T1 | The image of the database is one image: the tripwire that refuses the day a place names another | S1, S2, S3, S4, S5 | inline | — |
| T2 | The keeper of the freeze: dependabot for the base image and the actions | S3, S4, S5 | inline | — |
| T3 | The scan of the artifact: trivy in the job that owns it | S2, S3, S4, S5 | inline | — |
| T4 | The documentation of the keepers, in the README of the stack | S1, S5 | inline | — |

## Log

**L1** — The maintainer, on the continuous integration of the database:

> "I wonder know why if this PR introduces a lot of database changes, the CI
> for database doenss't run. https://github.com/witsaba/paca/actions/runs/42845064043/job/129930358291
> investigate and diagnostic"

**L2** — The diagnosis: the continuous integration decides per job with
`scripts/ci/changed-jobs.sh` and `scripts/ci/job-filters.txt`, and the job of
the database proves the guarantees of the bare PostgreSQL stack
(`db/postgres`), not the schema of the product. The bakery pull request
changed none of the paths that job watches. The proof of the migrations was
made by the job of the stack (`make stack-schema`: the applied migrations = 36,
the migrations that are not applied = 0) and by the container tests of the
quality gate.

**L3** — The maintainer, on the reason of the step:

> "but what is the reason of the The database step in CI?"

**L4** — The answer: the database of PACA is a custom artifact, and the job is
the gate that the artifact still delivers what it promises — the pgvector
compilation, the cluster flags, the init scripts, the isolation. The container
tests use the plain upstream image through testcontainers and never build the
custom image, so that job is the only one that would fail if the compilation
of pgvector broke.

**L5** — The maintainer, on the risk of the artifact:

> "is not a risk to hava non official postgres ?"

**L6** — The answer: the server is the official image pinned by digest and the
extension is the official source with its checksum verified; the real exposure
is not distrust but drift — nobody bumps the pins, and nothing scans the
artifact.

**L7** — The maintainer, asking for the recommendation:

> "what is your recommended one based on good practices and lower risk?"

**L8** — The recommendation, accepted by the maintainer:

> "ok, do you recommended one"

**L9** — The decision, with its evidence: the PGDG package was rejected after
checking its metadata (`postgresql-18-pgvector` **Depends: postgresql-18**),
because it would pull a second PostgreSQL server into the official image and
leave the extension pointing at the wrong one; the community image was
rejected because it would cost the digest of the server, the cluster flags and
the init scripts. The build stays as it is, and the three keepers arrive.

**L10** — The finding that became T1: the image of the database is written in
five places — `db/postgres/Dockerfile` (the `ARG`), `db/postgres/compose.yaml`
(the build argument) and the default of the container tests of
`packages/database`, `apps/mcp-server` and `apps/db-admin`. Two of them are
pinned by digest and three name a floating tag, so the migrations are proved
against bytes that the stack does not run.

**L11** — The decision on the branch: the feature starts from `main`, because
the continuous integration and the stack of the database were not touched by
the bakery feature. The assertions of T1 are placed where the bakery did not
touch the structure test, to keep the two merges free of conflicts.

**L12** — The decision on the scan: the job of the database scans the image it
built, and it fails on the findings that are not accepted. The accepted
findings are written in the repository, with their age, so the debt is visible
and a new one is refused.
