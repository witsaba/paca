# tools/

Repository-wide developer tooling: things that operate across the monorepo rather than
inside a single package.

## The pinned Go tools

This directory is a Go module of the workspace (`github.com/witsaba/paca/tools`) whose
purpose is to pin the developer tools the repository uses, as Go *tool dependencies*:

| Tool | Purpose |
| --- | --- |
| `github.com/golangci/golangci-lint/v2/cmd/golangci-lint` | the linter, configured in the root `.golangci.yml` |
| `golang.org/x/vuln/cmd/govulncheck` | the vulnerability scanner |

They are declared with the `tool` directive in [`go.mod`](go.mod) and run with
`go tool <name>`, so a contributor needs only the Go toolchain installed and every run
uses the pinned version:

```sh
go tool golangci-lint run ./...
go tool govulncheck ./...
```

Through the Makefile, from the repository root: `make lint` and `make vuln`.

The module also holds the package documentation of the pinning itself
([`doc.go`](doc.go)). It has no other code.

## The schema verification script

The one tool PACA had before the Go work is package-scoped rather than
repository-wide, so it lives with the package it verifies:
[`packages/db/tools/verify_schema.sh`](../packages/db/tools/verify_schema.sh).

## Conventions for this directory

- `scripts/` — one-off maintenance and release helpers;
- `checks/` — repository-wide quality gates that run across every package.
