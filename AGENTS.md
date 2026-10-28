# AGENTS.md

The instructions of the agents that work on PACA. They are the canonical entry of the
repository: this file holds the contract of the architecture, and the entry files of the
subprojects add the specifics of their module. The closest entry file to the file being
changed wins; the instructions of the person in the conversation win over every file.

This file is written for a reader that does not know the repository yet, and for a reader
that may write the next service in another language. The rules are therefore stated in the
language of the architecture — hexagonal, DDD, SOLID — and each of them says what enforces
it in every language.

---

## What PACA is

PACA is a monorepository of databases, services and tools for the people that use the
product. It holds:

- the **services** (`apps/`): each one is a process with one job and its own boundaries;
- the **shared libraries** (`packages/`): the code that more than one service needs, so
  it is written once and owned by nobody's business logic;
- the **databases** (`db/`), the **traces** (`observability/`), the pinned **tools**
  (`tools/`), the **scripts** (`scripts/`) and the **feature documents** (`odd/tasks/`).

The domain of PACA today is the *person*: a document ID, a name, a phone, an email, and a
state that moves through a small state machine. The MCP server registers people, moves
their state and corrects their data; the database admin is the single writer of the
schema of the database.

Read `README.md` for the layout and the running of the stacks, and
`packages/domain/README.md` for the model of the person and the rules it carries.

---

## The architecture contract

These rules are the contract. They hold for every module, every package and every
folder of the tree, and they are what a change is measured against.

### 1. The layers and the direction of the dependencies

Every unit of behavior is four layers and one composition root:

| Layer              | Holds                                                        | May depend on          |
|--------------------|--------------------------------------------------------------|------------------------|
| **domain**         | the model, the invariants, the rules, the vocabulary          | nothing but the standard library |
| **application**    | the use cases: the orchestration of the rules                  | the domain and the ports |
| **ports**          | the contracts between the layers                              | the domain             |
| **adapters**       | the inbound protocols and the outbound technologies           | the ports and the domain |
| **composition root**| the wiring: the only place that knows every layer            | everything             |

The dependencies point inward, always. The domain knows no driver, no table, no
protocol and no HTTP. The application knows no concrete adapter. Only the composition
root (the `bootstrap` package and the `main` of the service) knows the adapters, and it
knows nothing else that the layers do not already say.

**What enforces it, by language:** a module boundary (a `go.mod` in Go, a package
boundary in Python, a crate boundary in Rust) or a folder boundary. A rule that only the
compiler of one language can enforce is a rule of that language, and it says so.

**The test of the rule:** if you can name a file that imports another one, the direction
of that import tells you which layer may know which. A file that breaks the direction is
a defect, even when the build is green.

### 2. The rules live in the domain

The domain refuses what the domain forbids. The adapter translates, it does not decide.

- A rule that the domain states is **enforced by the domain**, and the code that calls
  the domain does not re-implement it.
- An adapter that needs a rule asks the domain for it. It never copies the rule.
- A rule that the **table** also holds (a CHECK, a unique index) is deliberate defense in
  depth: the table guards against the writer that does not know the domain. The table
  never becomes the only holder of a rule.
- When a rule couples two values (a state and the reason that belongs to it), the pair is
  a **value object**: the coupling is unrepresentable when it is broken, instead of being
  checked by a procedure that any implementation may forget.

### 3. One owner per model, one owner per rule

- A concept that more than one module needs lives in **one** shared module, and the other
  modules import it. Two copies of the same model are two domains that drift.
- A concept that only one service needs lives in **that service**. It is not promoted to
  a shared library "for symmetry", and it is not copied into the other services.
- A model of a shared concept belongs to the **domain**, not to the persistence layer and
  not to the protocol layer. The row model and the protocol model are its projections,
  mapped in one place each.
- The shared domain module imports nothing but the standard library. It is the boundary
  that makes "the domain knows nothing about the database" a compile error instead of a
  comment.

### 4. The ports are contracts, and they are narrow

- A port is owned by the **consumer** of the contract, and it is written in the language of
  the consumer (the domain speaks in domain types, never in row types).
- A port is **narrow**: one interface, one capability, the methods the use case actually
  calls. A method that no use case calls is not a method of the port yet.
- The port carries the **contract in its documentation**: the error that comes with each
  outcome, and the rule the implementation owes the caller.
- The implementation of a port is proved against the port, not against the internals of
  the adapter.

### 5. The contract of the outside world derives from the domain

When a protocol, a schema, a documentation string or a message is generated from a domain
rule, it is **derived** from the domain — never re-typed by hand.

- The list of the states, the allowed transitions, the shape of a field: they come from
  the domain, and a change of the domain changes them with it.
- Every error of the domain that a caller can see has a message **where the caller is**,
  written in the words of that caller. No error of the domain reaches a user unmapped.
- The description of a tool is a contract with an agent: it promises what the tool does,
  and nothing it does not do.

### 6. SOLID, as five checks of a change

| Principle | The check                                                                  |
|-----------|----------------------------------------------------------------------------|
| **S**ingle responsibility | One unit, one reason to change. A file that changes for two reasons is two files. |
| **O**pen/closed | A new behavior is added by adding a unit, not by editing a `switch` over types in the middle of the layers. |
| **L**iskov | An implementation of a port honors the **whole** contract, including the rules the documentation states. |
| **I**nterface segregation | A port holds the methods of one capability. A caller never depends on a method it does not call. |
| **D**ependency inversion | The middle of the system depends on abstractions it owns; the technology depends on the middle, never the other way around. |

### 7. The rules that are never broken

- **No typed nil inside an interface.** A pointer stored in an interface is never `nil`
  when the pointer is. The absence of a collaborator is expressed by the absence of the
  value itself, or by an explicit "not configured" state that the caller can see.
- **No second writer of the schema.** One service owns the migrations of the database,
  and it applies them at the start. No service migrates on boot.
- **No hand-written generated code.** The code a generator writes is committed, and the
  gate proves it is what the generator writes.
- **No copy of a shared layer.** The platform and the shared domain are shared: they are
  extended, never forked.
- **No dead API.** A function that no production code calls is not a promise, it is
  debris: it is removed, or the caller that needs it is written.
- **No rule that only a comment states.** A rule that the code cannot break is a wish.
  The gate pins the rules that matter.

### 8. The workflow of a change

- The work is tracked in a **feature document** under `odd/tasks/`: the specs (the
  wording of the person, quoted), the tasks (one line each, with the commit that holds
  it), and the log (the request and the decisions, in order).
- The work lands in **work units**: a commit that is reviewable on its own, with the
  tests and the documentation of the behavior it changes, and a Conventional Commit
  message.
- The **gate is green** before a work unit is committed: the format check, the analyzers,
  the linter, the build, the tests, the race detector, the coverage floor and the
  vulnerability scan of every module, and then the container tests.
- A change that changes the observable behavior of a service states it in the feature
  document, and the difference is proved, not asserted.

### 9. The tests are the proof of the contract

- A test exercises the **public interface** of the unit it proves: the port, the use
  case, the protocol, the command. It does not reach into the storage to check a field.
- A test asserts **every observable effect** of the rule it covers: the exit code, the
  exact output, the error, and the state that was left behind.
- A test that a rule names its own cases: the examples, the boundaries, the errors.
- The rules of a table are proved against a real database, in the container tests of the
  library that owns the table. The rules of a domain are proved without one.

---

## The commands of this repository

Every command is copy-pasteable and exists. Run the narrowest one that proves the change.

```sh
make all                # the whole gate of the repository, from nothing
make gate               # fmt, vet, lint, build, test, race, coverage, vuln — every module
make structure-test     # the structure of the repository: the graphs and the links
make packages-integration  # the container tests of the shared libraries
make apps-integration      # the container tests of the services
make stack-up            # the whole monorepository, up and healthy
make stack-schema        # the schema of the database was written by the service that owns it
make stack-down          # stop the containers of the project
make env-init            # write the .env file of every stack
```

The gate of a single module, with its own Makefile and the same targets:

```sh
make -C packages/domain all
make -C packages/database all
make -C apps/mcp-server all
make -C apps/db-admin all
```

The Go specifics of the tree: `go.work` lists every module, the tools are pinned in
`tools/go.mod` and run with `go tool`, and a module resolves its own graph outside the
workspace (`GOWORK=off go list -m all` must succeed).

---

## The pointers

| Where                                            | What it holds                                    |
|--------------------------------------------------|--------------------------------------------------|
| `README.md`                                       | the layout, the stacks, the networks, the gate   |
| `packages/domain/README.md`                       | the model of the person, its rules, its port     |
| `packages/database/README.md`                     | the tables, the queries, the repositories        |
| `apps/*/README.md`                                | the service: its layers, its routes, its tools   |
| `odd/tasks/*.md`                                  | the feature documents: the specs and the evidence |
| `.github/workflows/ci.yml`                        | what the continuous integration runs             |

The entry files of the subprojects (`packages/*/AGENTS.md`, `apps/*/AGENTS.md`) add the
specifics of their module. They never relax the contract of this file.
