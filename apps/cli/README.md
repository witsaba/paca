# `apps/cli` — the paca command

The command line application of PACA, a personal finance manager for the Colombian
market. This build is the **command header**: the application exists, the command tree
does not. `--help` is the only supported command.

## Run it

```sh
make run                       # from the repository root
make run ARGS="--help"
```

or, from this directory:

```sh
go run ./cmd/paca --help
```

The exit code is `0` for `--help` and for an invocation without arguments (the help is
shown), and non-zero for anything else.

## Build it

```sh
make build                     # writes bin/paca
./bin/paca --help
```

## Test it

```sh
make test                      # the whole workspace
make cover                     # with coverage
make race                      # with the race detector
make vet                       # the Go vet analyzers, toolchain only
```

The command behaviour is tested in
[`internal/adapters/cli/root_test.go`](internal/adapters/cli/root_test.go), through the
public interface of the adapter: the command is built, the arguments are set, and the
output is asserted.

## Layout

```
apps/cli/
├── cmd/paca/                     the composition root: build the command, run it, exit
└── internal/
    └── adapters/cli/             the driving adapter: the cobra command tree
```

The inner layers live in the shared core,
[`packages/core`](../../packages/core/README.md), which this module requires:

```
packages/core/
├── domain/                       entities, value objects, invariants
├── ports/                        the interfaces of both boundaries
└── application/                  the use cases
```

The dependency rule is inward only: `cmd` → `adapters` → `application` →
`ports` → `domain`. The domain imports nothing from the project and touches no
I/O, so the rules of PACA behave the same in every delivery mechanism. Each
package documents its own rule in its `doc.go`.

The reasoning behind the layout, the tool pinning, and the quality gate is recorded in
[`docs/architecture/backend-scaffold.md`](../../docs/architecture/backend-scaffold.md).

## What comes next

The next features attach as subcommands of the root command — `accounts`, `movements`,
`loans`, `reports` — each in its own file under `internal/adapters/cli`, each calling
a use case in `packages/core/application` through a port. The database arrives as a
driven adapter implementing the repository ports, reading the schema in
[`packages/db`](../../packages/db/README.md).
