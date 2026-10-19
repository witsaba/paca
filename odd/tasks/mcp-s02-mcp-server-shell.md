# S02 — MCP server shell

Session S02 of the
[PACA MCP server MVP1 roadmap](mcp-server-mvp1.md): the server that will carry
the 25 tools — transport, TLS, bearer auth and `/healthz`.

- **Roadmap entry:** S02 — MCP server shell (Phase 0 — The spine).
- **Branch:** `feat/mcp-s02-mcp-server-shell`, cut from an up-to-date `main`
  (`c23cc5f`), in its own worktree `paca-worktrees/mcp-server-shell`.
- **Done when:** `make all` is green, and the binary serves an empty MCP
  session over `httptest`.

---

## Specs

Each spec is quoted verbatim. `R#` quotes the roadmap or the ADR; `U#` quotes
the user.

- **R1** — Goal: "**Goal:** the server that will carry the tools."
- **R2** — Spec: "**Spec:** C1, C2."
- **R3** — Change: "new module `apps/mcp` with the binary
  `apps/mcp/cmd/paca-mcp`. Flags: `--bind` (default `127.0.0.1`), `--port`
  (default `7333`), `--auth-token`, `--tls-cert`, `--tls-key`, `--db`,
  `--require-auth`. The transport is Streamable HTTP from the official MCP Go
  SDK (`github.com/modelcontextprotocol/go-sdk`; the exact API is confirmed
  against the pinned version at implementation time). A `/healthz` endpoint
  answers liveness and is not part of the MCP surface. The server **refuses to
  start** when `--bind` is not loopback and `--auth-token` is missing, or when
  `--bind` is not loopback and TLS material is missing. A bearer token is
  checked on every MCP request. Startup logs never print the token."
- **R4** — Tests: "**Tests:** starts and serves; refuses to start without a
  token on a non-loopback bind; refuses to start without TLS on a non-loopback
  bind; rejects a request with a wrong or missing token; `/healthz` answers
  200."
- **R5** — Done when: "**Done when:** `make all` is green and the binary
  serves an empty MCP session over `httptest`."
- **R6** — ADR C1: "- Bind address via `--bind` (`127.0.0.1` default). - Port
  via `--port` (`7333` default). - Loopback bind → plain HTTP; auth is
  **optional** unless `--require-auth` is set. - Non-loopback bind → **TLS
  required**; cert and key from `--tls-cert` / `--tls-key`. The server refuses
  to start if either flag is missing while bound to a non-loopback address. -
  Implements `StreamableHTTPServer` from
  [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).
  - Exposes a separate `/healthz` endpoint for liveness (not part of the MCP
  surface)."
- **R7** — ADR C2: "- Bearer token via `Authorization: Bearer <token>`. - Token
  supplied by `--auth-token`. The server **fails to start** when `--bind` is
  non-loopback and no token is set. - Token rotation in MVP1 is \"restart with a
  new flag\" — no graceful revocation. Documented as a limitation. - mTLS is
  deferred to a future ADR."
- **R8** — ADR C8: "`paca-mcp` holds the SQLite file open. Migration procedure
  documented: **stop server → apply migration → start server**. SQLite's write
  lock makes the brief window safe. WAL mode and online migrations are out of
  scope for MVP1."
- **R9** — ADR C10: "- The CLI composes `packages/core/application` services
  **directly**. - The CLI does **not** call the HTTP MCP server. - Both binaries
  are independent composition roots over the same domain code."
- **R10** — ADR C11: "- Every cross-cutting concern (C1–C10) has at least one
  focused test. - Every listed tool has: 1. a service-layer unit test, and 2.
  an integration test using the MCP client SDK over `httptest`."
- **R11** — House rule (coverage): "every new or modified file needs at least
  75% test coverage, measured per file, statement-weighted".
- **R12** — House rule (worktree): "every session works in its own independent
  git worktree, on its own branch, never in the main checkout".
- **R13** — House rule (commits): "The commit is a reviewable slice of roughly
  150–400 authored changed lines, which is the house limit."
- **R14** — Session protocol: "**Write the test first.** One RED test per rule
  the session names: the happy path, every named error, and the boundary
  cases. The happy path goes RED before the implementation exists."

### Decisions of this session (user)

> The user's answers arrived in Spanish and are rendered here in English;
> the decision they carry is unchanged.

- **U1** — Database at boot: "Migrate at startup" — the server opens the file
  given by `--db` and runs the migration runner of S01 at startup. The runner
  is idempotent, so an already-migrated file is untouched and a new file is
  created with the schema.
- **U2** — Bearer token: the token is enforced on every MCP request whenever
  `--auth-token` is set, and `--require-auth` makes the absence of a token a
  startup failure. The server refuses to start when `--require-auth` is set and
  no token is given. The token is never printed in a log line.
- **U3** — Liveness: "make it very simple by responding with plain txt ok" —
  `/healthz` answers `200` with the plain text body `ok`. It touches no
  database and no MCP session, and it is not authenticated. The schema version
  and the uptime belong to the `health` tool of S05, not here.
- **U4** — TLS on the local network: "strict C1: TLS and token mandatory" — a
  non-loopback bind requires both the TLS material and the token, exactly as
  ADR C1 says. No `--allow-insecure-lan` exception and no amendment of the
  ADR. The deployment target is another machine of the same home network,
  reached from `192.168.1.115`, so a self-signed certificate is the material
  that satisfies C1.
- **U5** — Delivery: "a single PR, the list of commits as the map" — the
  branch is opened as one pull request whose body carries the commit list as
  the review map, the same convention as S01.

---

## Delivery strategy

The branch carries **one work-unit commit per task** (T1–T10), the same model
as the S00 and S01 branches: each commit is a behaviour unit with its tests
alongside it.

- **Slice boundary:** the branch is past the ~400-line house limit (1,605
  authored changed lines over five commits). It is a stack of reviewable
  commits, not one oversized commit, and the commit list is the map of the
  review.
- **Pull request:** the branch is delivered as **one pull request**, as the
  user decided (U5). Its body carries the commit list as the review map, the
  same convention as S01.
- **The security slice stands alone:** the bearer token (T4) has a commit of
  its own, so a review can take it apart without the rest of the server.
- **Merge order:** the commits merge in order; the last one closes the
  session. A commit is never half-merged.

---

## Tasks

| ID | Task | Spec | Route | Commit |
|---|---|---|---|---|
| T1 | Branch `feat/mcp-s02-mcp-server-shell` from an up-to-date `main` in its own worktree `paca-worktrees/mcp-server-shell`; this feature document is the first slice | R12, U5 | parent | `352a93f` |
| T2 | New module `apps/mcp` (`github.com/witsaba/paca/apps/mcp`) with the binary `apps/mcp/cmd/paca-mcp`; `go.work` gains the module and the Makefile builds `bin/paca-mcp` next to `bin/paca` | R3, R9 | parent | `7eab10a` |
| T3 | Flag parsing and startup validation in `apps/mcp/internal/config`: `--bind`, `--port`, `--auth-token`, `--tls-cert`, `--tls-key`, `--db`, `--require-auth`; the defaults of R6; the refusal rules of R3 and R6 as typed errors that carry the reason, never the token | R3, R6, R7, U2, U4 | parent | `7eab10a` |
| T4 | Bearer auth middleware in `apps/mcp/internal/auth`: `Authorization: Bearer <token>` checked on every MCP request, `401` on a missing or wrong token, constant-time comparison, the token never logged | R3, R7, U2 | parent (security slice, own commit) | `1e2a962` |
| T5 | Liveness endpoint in `apps/mcp/internal/server`: `/healthz` answers `200` with the plain text `ok`, outside the MCP surface and outside the auth middleware | R3, R6, U3 | parent | `758d6c5` |
| T6 | MCP transport in `apps/mcp/internal/server`: the `StreamableHTTPServer` of the pinned `github.com/modelcontextprotocol/go-sdk`, mounted apart from `/healthz`, with an empty session (the tools arrive in S03 and S05) | R3, R6 | parent | `758d6c5` |
| T7 | The composition root `apps/mcp/cmd/paca-mcp`: open the database of `--db`, migrate it at boot (U1), build the listener (plain HTTP on loopback, TLS elsewhere), serve, and shut down cleanly on a signal; the startup log never prints the token | R3, R8, U1, U2, U4 | parent | `51567ce` |
| T8 | Integration test: the server serves an empty MCP session over `httptest` with the client SDK of the same package | R4, R5, R10 | parent | `51567ce` |
| T9 | Verification: `make all` green over the five modules, the 57-check verifier green, the CLI byte-identical to before, and every new or modified file at or above 75% coverage | R5, R10, R11 | parent | — (no code) |
| T10 | Documentation: the roadmap entry S02 → Done with its commit hash, `## Log` gains the session entry and the decisions U1–U5, `apps/mcp/README.md` written, and the ADR amendments table updated if the session changed a decision of the ADR | R5 | parent | — (the closing slice of the branch) |

---

## Log

### L1 — Origin (verbatim)

> check the lates memories for the S01 job did it and analyze the next one and give
> me a recap of the next one.

> ok, let start with this new feature, use ODD obviously and let do it!

### L2 — The answers to the questionnaire

Rendered in English; the answers arrived in Spanish.

> Migrate at startup.

> Focus it only in accepting local calls, so it is not needed I think, due to
> the main agent is right now in the 192.168.1.115 and I want to deploy this in
> another machine inside my local home, so is it needed?

> Make it very simple by responding with plain txt ok.

> A single PR, the list of commits as the map.

> Strict C1: TLS and token mandatory.

### L3 — Findings of the exploration (2026-10-19)

- The pinned SDK is `github.com/modelcontextprotocol/go-sdk@v1.6.0`, already in
  the local module cache. `mcp.NewStreamableHTTPHandler(getServer func(*http.Request) *Server, opts *StreamableHTTPOptions)`
  (`mcp/streamable.go:194`) is the transport; `mcp.NewServer` (`mcp/server.go:157`)
  builds the server; the client SDK (`mcp.NewClient`, `mcp.NewStreamableHTTPClientTransport`)
  is the integration-test peer of R10.
- `packages/core/adapters/sqlite` exposes `Open(ctx, path)` (`sqlite.go:37`),
  `Migrate(ctx, db, fs.FS)` and `MigrateSchema(ctx, db)` (`migrate.go:43,72`);
  the last one carries the embedded migrations of the `packages/db` module, so
  the server needs no access to the SQL.
- The Makefile discovers the modules with `find`, so a new module is picked up
  by every target; only the binary target is hardcoded to the CLI and gains the
  MCP binary.
- The SDK applies DNS-rebinding protection by default: a request that arrives
  through a loopback address with a non-loopback `Host` header is refused with
  `403`. The default is kept — the deployment of U4 reaches the server by its
  LAN address, not through a loopback address.

### L4 — Findings of the implementation (2026-10-19)

- The pinned SDK builds the transport with
  `mcp.NewStreamableHTTPHandler(getServer, opts)` (`mcp/streamable.go:194`),
  and the client peer of the integration test is
  `&mcp.StreamableClientTransport{Endpoint, HTTPClient}`. The server is built
  once and shared by every request, because the handler owns the sessions.
- The SDK applies a DNS-rebinding protection by default: a request that
  arrives through a loopback address with a non-loopback `Host` header is
  refused with `403`. The default is kept — the deployment of U4 reaches the
  server by its address on the local network, not through a loopback address.
- The composition root held 0% of coverage as it was written, because
  `main()` cannot be called from a test. The body of the process moved into
  `run()` and `execute()`, and the file is now covered through the behaviour
  the binary has: a refusal, a failure and a clean stop.
- The linter refused the first draft of the tests (`bodyclose`) and of the
  composition root (`errcheck`). Both were fixed before the gate, and the
  fixes are in their own commits.

### L5 — The toolchain moved to 1.26.9 (evidence)

- `make all` failed on the vulnerability gate. The server reaches `net/http`
  and `crypto/x509`, so the advisories of the pinned `go1.26.3` became
  reachable: 17 vulnerabilities of the standard library, among them
  GO-2026-5037 (fixed in 1.26.4) and GO-2026-5026 (fixed in 1.26.6). The
  remedy the S01 learnings recorded is a bump of the pinned `go` directive,
  and the workspace and its five modules moved to `go 1.26.9`. After the bump
  `govulncheck` reports no vulnerability in any module.
- The CLI binary is **not** byte-identical after the bump, because the
  toolchain version is embedded in the build. The behaviour is: the output of
  `paca --help` is identical to the one of `main` before the session.

### L6 — The RED evidence (mutation check)

The tests were checked against three mutations of the implementation, and each
mutation turned the suite red:

| Mutation | Red test |
| --- | --- |
| the middleware accepts every token | `TestARefusedRequest` (4 cases) |
| the refusal of the missing TLS material removed | `TestParseRefuses` (2 cases) and `TestRefusalNeverCarriesTheToken` |
| the liveness body changed from `ok` to `nope` | `TestHealthzAnswersOK` |

The implementation was restored after every mutation and the suite was green
again.

### L7 — The verification of the session (2026-10-19)

- `make all` is green over the five modules: fmt-check, vet, lint (0 issues),
  build, test, race and vuln (no vulnerability in any module).
- The verifier of the schema passes 57 of 57 checks.
- The coverage of every new file is above 75%: `main.go` 92.3%, `auth.go`
  100%, `config.go` 100%, `healthz.go` 100%, `run.go` 85.2%, `server.go`
  88.9%.
- The integration test proves the done-when of the session: the binary serves
  an empty MCP session over `httptest` with the client SDK of the same
  package, with and without the token.

### L8 — The verification of the running binary (2026-10-19)

The verification was run against the built binary, through its public
interface, not against the code:

| Check | Observed |
| --- | --- |
| `--bind 0.0.0.0` without a token | `missing bearer token for a non-loopback bind`, exit 2 |
| `--bind 0.0.0.0` with a token and no TLS | `missing TLS material for a non-loopback bind`, exit 2 |
| `--require-auth` without a token | `--require-auth needs --auth-token`, exit 2 |
| no `--db` | `missing database path: --db is required`, exit 2 |
| the token in the output of a refusal or a log | absent |
| `/healthz` | `200`, body `ok`, outside the authentication |
| `/mcp` without a token, with a token configured | `401 unauthorized` |
| `/mcp` with a wrong token | `401 unauthorized` |
| `/mcp` with the right token | `200`, the `initialize` answer of an empty session |
| plain HTTP on loopback | serves |
| TLS with the material | serves, and plain HTTP against it is refused by the listener |
| the database at startup | `PRAGMA user_version = 1`, the five tables of the initial schema |
| `SIGTERM` | the process stops cleanly |

The independent verifier of the harness was not admitted for this change, so
the verification above was run inline, with the focused tests and the suite
of the repository as well.

### L9 — The issue and the pull request of the session

- Issue **#11** (`enhancement`): the acceptance criteria of the session, the
  scope that is out of it and the five decisions.
- Pull request **#12** (`type:feature`): one pull request, with the commit
  list as the map of the review, as U5 decided.
