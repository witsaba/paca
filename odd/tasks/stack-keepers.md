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
| T1 | The image of the database is one image: the tripwire that refuses the day a place names another | S1, S2, S3, S4, S5 | inline | a180609 |
| T2 | The keeper of the freeze: dependabot for the images and the actions | S3, S4, S5 | inline | 180a5c8 |
| T2b | The keeper of the extension: the check that the pinned pgvector is not behind | S3, S4, S5 | worker | 02e6273 |
| T3 | The scan of the artifact: trivy in the job that owns it | S2, S3, S4, S5 | worker | 634a20f |
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

**L13** — The decision on the declarations of the artifact: the compose file
of the stack declared the base image and the version of pgvector a second
time, and the compose won the build over the Dockerfile. The Dockerfile owns
them now, the compose builds with the declarations of the Dockerfile, and the
rule of the structure refuses the day a second copy appears. Dependabot moves
one declaration instead of two.

**L14** — The decision on the scan, in its exact terms: the job of the
database scans the image it built, with trivy; it fails on the findings of
severity CRITICAL and HIGH that have a fix and that are not in the accepted
baseline; the baseline lives in the repository (`db/postgres/trivy-baseline.json`),
so the debt of the artifact is visible and a new one is refused.

**L15** — The decision on the extension: dependabot moves the pins of the
images, but it cannot move the version of pgvector, because that version is
not an image. The keeper of the extension is a check that runs on a schedule
and says so when the pinned version is behind the latest release. It opens an
issue, because a version of an extension needs the checksum of its tarball
and that decision belongs to a human.

**L16** — The finding about the checksum of the extension, left open on
purpose: the pin of `PGVECTOR_SHA256` is the checksum of the archive that
GitHub generates from the tag, and GitHub does not promise that an archive
stays byte for byte the same. The build refuses a tarball that does not
match, so the failure is loud, but the pin may need to move without a new
release. It is a known limit of the artifact, not of this feature.

**L17** — The proof that the compose changed nothing of the image: the image
built with the build arguments the compose used to pass and the image built
with the declarations of the Dockerfile were compared. The same server
(PostgreSQL 18.6), the same extension (pgvector 0.8.7), the same cluster
flags (`--encoding=UTF8 --locale=C.UTF-8 --data-checksums`), the same
environment, the same labels and the same init scripts, with the same
checksums. The identifiers of the two images differ because two builds carry
their own timestamps in the layers, not because the artifact changed.

**L18** — The proof that the container tests accept the pin: the container
tests of `packages/database` pull the image by digest now, and the suite is
green against it (`ok ... 2.449s`, twice, plus the whole suite). One run
failed while three heavy builds shared the same daemon, and the container of
the suite was terminated in the middle of a test; the same test passes with
the digest and with the tag before it, and the suite is green on every run
that has the daemon to itself. It was the contention of the builds, not the
pin.

**L19** — The delivery of the two delegated units arrived complete, but the
worker died on the quota of its provider before it reported: the four files
were in the worktree and the evidence was not. The evidence was gathered
again by the orchestrator, and the policy of the scan was proved in both
directions: the scan of the image without the accepted findings exits with
1 (30 findings: 4 HIGH in the Debian base and 26 in the Go binary the image
carries), and the scan with the accepted findings derived from the committed
baseline exits with 0.

**L20** — The decision on the shape of the accepted findings: the report of
the scan as it comes out of trivy carries the inventory of the image, 247
kilobytes, and it changes with every rebuild of the base even when no
finding changes. The committed file carries the findings and their detail
(the package, the version that fixes them, the day they were published) and
not the inventory: 10 kilobytes that only move when the findings move. The
README of the stack tells how to regenerate it.

**L21** — The decision on the ignore file: the accepted findings are the
source of truth and the ignore file of trivy is derived from them when the
job runs, so the two cannot drift. The derivation was proved with the very
command of the job.

**L22** — The delivery and the size of the slices: the feature landed as
five work units — the document, the tripwire, the keeper of the freeze, the
scan of the artifact and the keeper of the extension — and the last one
closes it. The slice of the feature is 966 authored lines, and 276 of them
are the accepted findings of the scan, which is a report and not code: the
rest of the feature is 690 lines in five commits, each of them reviewable on
its own. The units that touch the continuous integration are in commits of
their own, so the review can take them alone.

**L23** — The verification of the whole feature: the structure of the
repository (505 assertions, 0 failures), the shell scripts (14 scripts), the
test of the keeper of the extension (26 assertions, 0 failures), the
container tests of the database against the pinned image (green on every run
that had the daemon to itself), the workflows and the compose file (they
parse), and the scan of the artifact in both directions (exit 1 without the
accepted findings, exit 0 with them).

**L24** — The review of the feature could not run, and the reason is outside
the repository. The assessment of the candidate returned the risk "high"
(the escalation of the item 5 landed) and asked for an independent verifier;
the native review opened the lineage `review-ae79303eb80f6c92` (tier high,
four lenses: risk, resilience, readability, reliability) and the relay of the
reviewers failed with "no model is configured for review-risk (routing
source: frozen session profile)". The four reviewer agents have no model in
the routing of the host, and that routing is frozen for this session: it is
the configuration of the shell, outside the repository. The independent
verifier died twice on the quota of its provider.

**L25** — The evidence that stands in the place of the two missing checks:
the gate of the repository is green (505 assertions, 0 failures; 14 shell
scripts clean; 26 assertions of the keeper), the tripwire was broken on
purpose and failed as it must, the image built the old way and the image
built the new way carry the same server, the same extension, the same
cluster flags and the same init scripts, the scan of the artifact fails
without the accepted findings and passes with them, the container tests of
the repository pull the pinned image and are green, and the action of the
scan exists with the six inputs the job uses.

**L26** — The defect that the first run of the keeper found in itself: the
`docker` ecosystem of dependabot reads ONE directory at a time, and the entry
that named the root of the repository found no Dockerfile there (the images
live in `db/postgres`, `apps/mcp-server`, `apps/db-admin`, `hermes` and
`engram`) → the run failed with *"No Dockerfiles nor Kubernetes YAML found in
/"*. The keeper names every directory that holds an image now, and the rule of
the structure of the repository refuses the day an image of the tree has no
entry: it was watched to fail with the old configuration (5 failures) and to
pass with the new one.
