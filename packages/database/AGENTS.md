# AGENTS.md — `packages/database`

The persistence library of the monorepository: the pool, the queries, the migrations and
the repositories. The contract of the root applies here and is never relaxed. This file
holds the specifics of the module.

## What this module is

- A **library**. It publishes no port on the network, it runs no container, it starts no
  process. A service maps its own environment onto it, opens the pool and uses the
  repositories.
- The **implementation** of the ports that `packages/domain` owns. It depends on the
  domain, never the other way around.
- The **single writer of the schema**: the migrations live here and are applied by
  `db-admin`, which is the only service that migrates.

## The rules of this module

- **The module depends on `packages/domain`, and never on an application.** The direction
  is enforced by `go.mod`.
- **The library reads no environment variable.** It receives a configuration; the
  service resolves its own environment. The only shared piece is the pure assembly of
  the connection string (`packages/database/config`).
- **The rules of the domain are called, not copied.** The repository asks the domain
  whether a move is allowed, whether an observation fits a state, whether a value is
  valid. It never re-implements a rule of the model.
- **The SQL is generated, not written.** The queries live in `queries/*.sql`, the schema
  in `migrations/*.sql`, and `internal/db` is the output of the generator. The gate
  proves that the committed code is what the generator writes.
- **The generated code stays internal.** It is `internal/db` for that reason: the public
  surface of the library is curated by hand, and it speaks in the types of the domain.
- **One connection stack.** The pool of the module is the door to PostgreSQL; a service
  does not open a connection of its own with another driver.

## The shape of a change here

1. A query changes in `queries/*.sql`, the generator runs, `internal/db` changes with it.
2. A migration is a new file in `migrations/`, numbered after the last one, with its
   `Up` and its `Down`. The schema of a table that exists is never rewritten.
3. The repository maps the row onto the model of the domain in one place
   (`postgres/user.go`) and translates the errors of the database into the errors of the
   port in one place (`postgres/repository.go`).
4. The gate of the module is green: `make -C packages/database all`, and the container
   tests of the rules of the tables: `make packages-integration`.

## The proof

The container tests of this module assert the rules of the tables against a real
PostgreSQL: the checks, the unique indexes, the NULL convention, the migrations applied
once and only once.
