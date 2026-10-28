# AGENTS.md — `packages/domain`

The shared domain of the monorepository: the models, the rules and the ports of the
storage. The contract of the root applies here and is never relaxed. This file holds the
specifics of the module.

## What this module is

- A **library**. It publishes no port on the network, it runs no container, it starts no
  process.
- The **single owner** of the concepts that more than one service needs. The person, its
  states, its rules, and the port of its storage live here and nowhere else.
- The **purest** module of the tree: it imports nothing but the standard library. That is
  the boundary that makes "the domain knows nothing about the database" a compile error.

## The rules of this module

- **The module imports nothing but the standard library.** No driver, no SQL, no HTTP, no
  protocol SDK, no third-party package. The gate and the compiler say so.
- **The model owns its invariants.** A value that is not valid cannot be built. A rule
  that couples two values is a value object, not a procedure (`Status` and the observation
  that belongs to it are one concept, not two arguments).
- **The port is narrow and is documented as a contract.** Each method states the error it
  returns and the rule the implementation owes the caller. The vocabulary of the errors
  is the vocabulary of the domain: an error that names a table or a driver does not
  belong here.
- **No persistence vocabulary.** `NULL`, a column, a query, a row: none of them is a
  concept of the domain. The rule "empty means not set" is a domain rule, and the
  adapter maps it to the storage.
- **No behavior for the sake of coverage.** A method exists because a caller needs it.
  An unused exported function is debris: it is removed, or its caller is written.

## The shape of a change here

1. The rule changes in the model, with its test, in this module.
2. The adapter that implements the port is updated to **call** the rule, never to
   re-implement it (`packages/database/postgres/repository.go` is the reference).
3. The projection of the model (a row, a protocol payload) is updated in one place, the
   mapper.
4. The gate of the module is green: `make -C packages/domain all`.

## The proof

The rules of the domain are proven here, without a database: the unit tests live with
the code they cover. The rules of the table are proven against a real PostgreSQL by the
container tests of `packages/database`.
