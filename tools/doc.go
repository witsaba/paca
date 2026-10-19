// Package tools is the home of the repository's developer tooling.
//
// The tooling itself is not vendored here: since Go 1.24 a tool can be
// declared as a dependency of a module with the `tool` directive, and the Go
// toolchain then builds and runs it on demand with `go tool <name>`. This
// module declares the tools the repository pins:
//
//   - github.com/golangci/golangci-lint/v2/cmd/golangci-lint — the linter;
//   - golang.org/x/vuln/cmd/govulncheck — the vulnerability scanner.
//
// The exact versions live in tools/go.mod and tools/go.sum, so a contributor
// and a CI run use the same tools without installing anything globally.
//
// This package intentionally has no code: it exists so that the module is a
// real Go module, and so that the dependency rule of the tools is documented
// in one place.
package tools
