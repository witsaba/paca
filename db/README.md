# Databases

Every database of the monorepository lives in its own directory under `db/`,
with its own compose file, its own Makefile and its own README. The root of
the repository carries no database-specific target: this folder is the entry
point of the domain.

## The databases

| Directory   | Stack                                | Service         | README |
|-------------|--------------------------------------|-----------------|--------|
| `postgres/` | PostgreSQL 18.6 with pgvector 0.8.7  | `paca-postgres` | [postgres/README.md](postgres/README.md) |

## One entry point, one project

The root [compose.yaml](../compose.yaml) includes the compose file of every
stack and declares the networks they share. Everything runs under the same
compose project, `paca`, so the container, the volume and the networks are
the same ones whichever way the stack is started.

```sh
docker compose up --wait          # every stack of the monorepository
docker compose up -d --wait postgres   # one service of it
docker compose down               # stop everything
```

A single database is driven from its own directory, with the same commands
scoped to its service:

```sh
make -C db/postgres help
make -C db/postgres up
make -C db/postgres smoke
```

The umbrella applies a target to every database at once:

```sh
make -C db help          # the aggregate targets and the list of databases
make -C db build         # build the image of every database
make -C db up            # start every database and wait until it is healthy
make -C db status        # container status of every database
make -C db smoke         # smoke test of every database
make -C db down          # stop every database (the data volumes are kept)
```

## The networks

The root compose declares two networks, and a stack attaches to the ones it
needs. Their definitions are repeated, identically, in the compose file of
each stack, so a stack also runs on its own.

| Network       | Attribute  | Purpose                                                                 |
|---------------|------------|-------------------------------------------------------------------------|
| `paca_internal` | `internal: true` | Service-to-service traffic: the database and the services that talk to it. A container on it has **no outbound access at all**. |
| `paca_egress`  | plain bridge     | Outbound traffic for the services that need the internet.                |

A service that needs the database **and** the internet joins both:

```yaml
services:
  api:
    networks:
      - internal   # the database
      - egress     # the internet
```

A service that only talks to the database joins `internal` alone. The names
are fixed, so a service finds the database on `paca_internal` and reaches it
at `postgres:5432` (the compose service name).

## The credentials

Each stack owns its secrets: the variables live in the `.env` file of the
stack, and the compose file of the stack reads it with `env_file`. The path is
relative to the compose file that declares it, so the stack reads the same
file whether it runs alone or included from the root. The root of the
repository has no `.env` and needs none.

## Adding a database

1. create `db/<name>/` with its own compose file, its own Makefile, its own
   README and its own `.env` (from a `env.example`);
2. declare the networks it attaches to, with the same attributes as the root
   compose;
3. add the path to the `include` list of the root [compose.yaml](../compose.yaml);
4. add its name to `DATABASES` in `db/Makefile`.

The aggregate targets then cover it, and nothing else in the repository has to
change: the other stacks of the monorepository bring their own tooling.
