# PACA — MCP server initialization

Feature document for the first slice of the PACA MCP server: a Go service that
speaks the Model Context Protocol over the official Go SDK, exposes a health
endpoint, and registers no tools yet.

- **Project:** `apps/mcp-server` (Go module `github.com/witsaba/paca/apps/mcp-server`).
- **Worktree:** `/Users/braejan/workspace/witsaba/repositories/paca-mcp-server`, branch `feature/mcp-server-init`.
- **Scope:** project initialization only. No MCP tools, no persistence, no TLS, no authentication.

---

## Specs

Each spec is the user's own wording, quoted verbatim. The task entries expand
them into implementation rules and verification.

### S1 — New Go project in the monorepository

> Build a new golang project for adding the mcp-server of this monorepository.

### S2 — Monorepository-aware organization

> Organize this new project keeping in mind this is a monorepostory.

### S3 — Only the MCP server, health only

> The feature consist in adding only the MCP server with the official golang MCP with health only.

### S4 — Hexagonal architecture, DDD, SOLID

> Build this new project inside this monorepository with Hexagonal Architecture, Domain Driving Design as main code desing and keeping the SOLID principles.

### S5 — Makefile, 80%+ coverage threshold, Dockerfile, root compose

> You must to add the Makefile, a testing threshold of 80%+ of testing coverage, adding too the Dockerfile and adding it to the root docker compose.

### S6 — Initialization only, no tools yet

> Focus only in the project initialization with the MCP server properly running even with no tools yet.

### S7 — Research latest and secure versions

> Web research about latest version, secure versions and keeping in mind the security.

### S8 — Makefile validation surface

> The Makefile for this project must to validate too not only build, deps or tests but race and performances validations, vulns, and linters.

### S9 — Go version freedom

> You can use a different golang version in the docker file if the local golang is lower or not secure so far.

### S10 — Testcontainers strategy

> Finally, let use a strategy with testing using test containers for testing the server up and respond the health.

### S11 — The MCP must stay private

> the MCP must to private so far, not allowed to be accesed from out of the container.

### S12 — A unique, internal-only port

> ensure the MCP server use a specific port due to I'll be creating others backends, maybe Rest, gRPC or anothers one in golang or rust or I dont know. So ensure a unique port for the MCP server but must to be only internal for now.

### S13 — Exposure surface (decided)

> Internal containers only (no published port)

The service listens on `0.0.0.0` inside its own container and the compose
service publishes **no** port. It is reachable only from sibling containers
attached to the `paca_internal` network. It is not reachable from the host or
from the internet.

### S14 — Authentication (decided)

> No auth yet, network isolation only

No bearer token and no TLS in this slice. The MCP SDK's own DNS-rebinding
protection and the Go cross-origin protection stay enabled.

### S15 — Health surface (decided)

The health endpoint is served by the same listener as the MCP transport and
therefore follows the same internal-only reachability rules. Container health
checks still work, because they run inside the container.

---

## Tasks

| ID   | Specs   | Route      | Commit | Summary |
|------|---------|------------|--------|---------|
| T1   | S1-S15  | inline     | e90d22d | Go workspace, module skeleton, pinned toolchain (Go 1.27.3, SDK v1.8.0) |
| T2   | S4      | inline     | 53c1e1a | Domain layer: health model, value objects, domain ports, and the health use case |
| T3   | S2, S7   | inline     | 2441991 | Platform: configuration with validation, structured logging, the clock |
| T4   | S6      | inline     | a2a29fd | Platform: the HTTP server with its timeouts, and the health check command |
| T5   | S3, S4   | inline     | 8beacde | Inbound adapter: the health endpoint |
| T6   | S4, S11  | inline     | 95191b0 | Inbound adapter: the router that hosts the health endpoint and the transport |
| T7   | S3, S6, S7, S14 | inline | d14f872 | Inbound adapter: the MCP transport over the official Go SDK |
| T8   | S4, S6   | inline     | cc9ae85 | Outbound adapter: the checker of the MCP transport |
| T9   | S6, S11, S13 | inline | a85f32b | Composition root, the command and the healthcheck command |
| T10  | S5, S8   | inline     | 90b3309 | Project Makefile: deps, tidy, fmt, vet, lint, build, test, race, cover 80% gate, perf, vuln |
| T11  | S5, S9, S11, S12, S13 | inline | 21ce16b | Dockerfile (pinned digests, non-root, distroless), .dockerignore, the stack and the root compose |
| T12  | S10     | inline     | e446c36 | Testcontainers integration suite: server up, health answers, MCP handshake |
| T13  | S2, S5   | inline     | 24d23fd | Documentation: project README and monorepository README |

### The slice boundaries

The delivery is split into work units of roughly 400 authored changed lines or
less, so every commit can be reviewed on its own. The commits are:

1. `e90d22d` — the workspace, the pinned tools and the module of the server.
2. `53c1e1a` — the health model and the use case that reports it.
3. `2441991` — the configuration, the logs and the clock.
4. `a2a29fd` — the HTTP server and the health check of the service.
5. `8beacde` — the health endpoint.
6. `95191b0` — the router that hosts the health endpoint and the transport.
7. `d14f872` — the MCP transport over the official Go SDK.
8. `cc9ae85` — the checker of the MCP transport.
9. `a85f32b` — the composition root.
10. `90b3309` — the quality gate of the project.
11. `21ce16b` — the image of the server and its stack.
12. `e446c36` — the container tests of the server.
13. `24d23fd` — the documentation, which closes the feature.

Every commit builds, and every commit keeps the test suite green.

---

## Log

### L1 — Original request

> I need you work in a new git worktree for adding a new feature. Build a new golang project for adding the mcp-server of this monorepository. Organize this new project keeping in mind this is a monorepostory. The feature consist in adding only the MCP server with the official golang MCP with health only. I need you build this new project inside this monorepository with Hexagonal Architecture, Domain Driving Design as main code desing and keeping the SOLID principles. You must to add the Makefile, a testing threshold of 80%+ of testing coverage, adding too the Dockerfile and adding it to the root docker compose. Focus only in the project initialization with the MCP server properly running even with no tools yet. Web research about latest version, secure versions and keeping in mind the security. The Makefile for this project must to validate too not only build, deps or tests but race and performances validations, vulns, and linters. You can use a different golang version in the docker file if the local golang is lower or not secure so far. Finally, let use a strategy with testing using test containers for testing the server up and respond the health.

### L2 — Privacy correction

> I forgot to mention that the MCP must to private so far, not allowed to be accesed from out of the container. Anything that you think to assume, do not do it, ask me instead.

### L3 — Decisions taken in the questionnaire

- Exposure: internal containers only, no published port (S13).
- Health surface: same private listener as the MCP transport (S15).
- Authentication: none in this slice; network isolation only (S14).
- Location: `apps/mcp-server`, matching the layout this repository used before
  the blank-slate reset (`apps/` for binaries, `packages/` for libraries,
  `go.work` at the root).

### L4 — Research findings (2026-10-27)

- Official MCP Go SDK: `github.com/modelcontextprotocol/go-sdk` **v1.8.0**
  (released 2026-09-14), newest MCP spec `2026-07-28`, requires Go 1.25+.
- Go toolchain: local is **go1.26.3**, below the security release **go1.26.10**
  (crypto/x509 CA mitigation, 2026-10-19). Latest secure is **go1.27.3**
  (same fix, 2026-10-19). The project therefore pins **go1.27.3** in `go.mod`,
  in `go.work` and in the Dockerfile; the local toolchain downloads it
  automatically through `GOTOOLCHAIN`.
- `golangci-lint` **v2.15.0** (2026-10-16), `govulncheck`
  (`golang.org/x/vuln`) **v1.8.0**, `testcontainers-go` **v0.44.0**.
- The SDK ships the protections this slice relies on: DNS-rebinding protection
  (`DisableLocalhostProtection`, off by default), `MaxRequestBodyBytes`
  (4 MiB default), session timeouts, and Go's `http.CrossOriginProtection`.

### L5 — Port allocation decision

The MCP server reserves **port 8081** (container-internal). The repository port
registry: `5432` PostgreSQL, `8081` MCP streamable HTTP, `8080` and `50051`
left for the future REST and gRPC backends. The port is configurable through
the environment but defaults to 8081, and the compose service publishes it to
nobody.

### L6 — What the slice proves

The quality gate of the project (`make all`) is green: the format check, the
analyzers of the toolchain, `golangci-lint` with `gosec`, the build, the test
suite, the race detector, the coverage gate (**85.0%**, threshold 80%) and
`govulncheck` (**no vulnerabilities**).

The container tests build the image from its Dockerfile, start it, and pass:
the server comes up, its health endpoint answers `ok` with the transport
checked, an MCP client of the official SDK connects and lists no tool, and a
cross-origin request is rejected with 403.

### L7 — Defects the tests found, and their fixes

- The aggregate of the statuses trusted an unknown status: `Worst` returned it
  as it was. It now maps an unknown status to `unhealthy`.
- The health service accepted a typed nil clock, so the guard against a
  missing clock never fired.
- The `App` of the bootstrap could not be tested on a free port: the listener
  is now bound apart from the serving.
- The container tests built a URL out of the mapped port of the container
  (`55003/tcp`), which is not a URL. They now use the port alone.

### L8 — What is out of scope of this slice

- No authentication and no TLS: the network is the boundary, and the token
  comes with the first tool that needs it.
- No MCP tool, no persistence, no session state.
- The port is reserved, but the REST and the gRPC backends do not exist yet.
