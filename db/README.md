# Databases

Every database of the monorepository lives in its own directory under `db/`,
with its own compose file, its own Makefile and its own README. The root of
the repository carries no database-specific target: this folder is the entry
point of the domain.

## The databases

| Directory   | Stack                                | Service         | README |
|-------------|--------------------------------------|-----------------|--------|
| `postgres/` | PostgreSQL 18.6 with pgvector 0.8.7  | `paca-postgres` | [postgres/README.md](postgres/README.md) |

## Commands

The umbrella acts on every database at once:

```sh
make -C db help          # the aggregate targets and the list of databases
make -C db build         # build the image of every database
make -C db up            # start every database and wait until it is healthy
make -C db status        # container status of every database
make -C db smoke         # smoke test of every database
make -C db down          # stop every database (the data volumes are kept)
```

A single database is driven from its own directory:

```sh
make -C db/postgres help
make -C db/postgres up
```

## Adding a database

1. create `db/<name>/` with its own compose file, its own Makefile and its own
   README;
2. add `<name>` to `DATABASES` in `db/Makefile`.

The aggregate targets then cover it, and nothing else in the repository has to
change: the other stacks of the monorepository bring their own tooling.
