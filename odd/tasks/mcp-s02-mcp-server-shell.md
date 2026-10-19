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

## Tasks

| ID | Task | Spec | Route | Commit |
|---|---|---|---|---|
| T1 | Branch `feat/mcp-s02-mcp-server-shell` from an up-to-date `main` in its own worktree `paca-worktrees/mcp-server-shell`; this feature document is the first slice | R12, U5 | parent | — |
| T2 | New module `apps/mcp` (`github.com/witsaba/paca/apps/mcp`) with the binary `apps/mcp/cmd/paca-mcp`; `go.work` gains the module and the Makefile builds `bin/paca-mcp` next to `bin/paca` | R3, R9 | parent | — |
| T3 | Flag parsing and startup validation in `apps/mcp/internal/config`: `--bind`, `--port`, `--auth-token`, `--tls-cert`, `--tls-key`, `--db`, `--require-auth`; the defaults of R6; the refusal rules of R3 and R6 as typed errors that carry the reason, never the token | R3, R6, R7, U2, U4 | parent | — |
| T4 | Bearer auth middleware in `apps/mcp/internal/auth`: `Authorization: Bearer <token>` checked on every MCP request, `401` on a missing or wrong token, constant-time comparison, the token never logged | R3, R7, U2 | parent (security slice, own commit) | — |
| T5 | Liveness endpoint in `apps/mcp/internal/server`: `/healthz` answers `200` with the plain text `ok`, outside the MCP surface and outside the auth middleware | R3, R6, U3 | parent | — |
| T6 | MCP transport in `apps/mcp/internal/server`: the `StreamableHTTPServer` of the pinned `github.com/modelcontextprotocol/go-sdk`, mounted apart from `/healthz`, with an empty session (the tools arrive in S03 and S05) | R3, R6 | parent | — |
| T7 | The composition root `apps/mcp/cmd/paca-mcp`: open the database of `--db`, migrate it at boot (U1), build the listener (plain HTTP on loopback, TLS elsewhere), serve, and shut down cleanly on a signal; the startup log never prints the token | R3, R8, U1, U2, U4 | parent | — |
| T8 | Integration test: the server serves an empty MCP session over `httptest` with the client SDK of the same package | R4, R5, R10 | parent | — |
| T9 | Verification: `make all` green over the five modules, the 57-check verifier green, the CLI byte-identical to before, and every new or modified file at or above 75% coverage | R5, R10, R11 | parent | — |
| T10 | Documentation: the roadmap entry S02 → Done with its commit hash, `## Log` gains the session entry and the decisions U1–U5, `apps/mcp/README.md` written, and the ADR amendments table updated if the session changed a decision of the ADR | R5 | parent | — |

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
