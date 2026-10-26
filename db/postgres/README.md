# PACA database: PostgreSQL 18.6 + pgvector 0.8.7

Internal-only database for PACA. Nothing is published on the host: the
container lives on an `internal` Docker network, so only other containers on
that network can open a connection.

## Versions (checked 2026-10-26)

| Component  | Version  | Notes                                                                 |
|------------|----------|-----------------------------------------------------------------------|
| PostgreSQL | 18.6     | Latest stable minor release (18.5 was skipped after a post-wrap regression; PostgreSQL 19 is still beta). |
| pgvector   | 0.8.7    | Latest pgvector release (2026-10-01), compiled from source in the image. |
| Base image | `postgres:18.6-bookworm` | Pinned by digest in `Dockerfile` and `compose.yaml`. |

## What you get

- **UTF-8 everywhere**: the cluster is created with `--encoding=UTF8 --locale=C.UTF-8`,
  and `client_encoding` is set to `UTF8` per database.
- **UTC everywhere**: `TZ`/`PGTZ` are `UTC` in the container, `timezone` is set to
  `UTC` per database, per role and in `template1`.
- **UUID support**: `uuid-ossp` and `pgcrypto` are enabled in the application
  database. The core `gen_random_uuid()` (PostgreSQL 13+) also works.
- **Vector search**: the `vector` extension is enabled in the application database.
- **Ready to use**: the image, the cluster, the database, the role, the
  extensions and the timezone defaults are created on first start.
- **Data checksums** on the cluster (`--data-checksums`).

## Layout

```
db/
├── Makefile            # umbrella: aggregates the targets of every database
└── postgres/
    ├── Makefile        # the targets of this stack
    ├── Dockerfile      # postgres 18.6 + pgvector 0.8.7 (multi-stage)
    ├── compose.yaml    # service, internal network, volume, healthcheck
    ├── env.example     # copy to .env and set the password
    ├── initdb/         # baked into the image, runs on first init
    │   ├── 01-extensions.sql  # vector, uuid-ossp, pgcrypto
    │   └── 02-defaults.sql    # timezone + client_encoding per database/role
    ├── scripts/
    │   └── smoke-test.sh    # asserts encoding, timezone, extensions, isolation
    └── README.md
```

This is a monorepository: every stack owns its own directory and its own
Makefile, and the root of the repository carries no stack-specific targets.
`db/Makefile` is the entry point of the database domain — it knows the list of
databases and forwards to them.

## Usage

```sh
cp db/postgres/env.example db/postgres/.env
# set POSTGRES_PASSWORD in db/postgres/.env

make -C db/postgres up       # build, start, wait until healthy
make -C db/postgres shell    # psql inside the container
make -C db/postgres smoke    # assert encoding, timezone, extensions and isolation
make -C db/postgres status   # container state
make -C db/postgres logs     # follow logs
make -C db/postgres down     # stop (data is kept)
make -C db/postgres reset    # DESTRUCTIVE: stop and delete the data volume
```

`make -C db/postgres help` lists every target of the stack, and
`make -C db help` lists the aggregate ones. From inside the stack,
`cd db/postgres && make up` works the same way.

The umbrella applies a target to every database at once:

```sh
make -C db up       # start every database
make -C db down     # stop every database
make -C db status   # status of every database
make -C db smoke    # smoke test of every database
```

Adding a database later is two steps: create `db/<name>/` with its own
Makefile and compose file, then add its name to `DATABASES` in `db/Makefile`.

## Connecting from the host

The database publishes no port, so there is nothing to connect to from the
host. Use the container:

```sh
make -C db/postgres shell
# or
docker compose -f db/postgres/compose.yaml exec postgres \
  sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

## Connecting from another container (the app)

The service is attached to the `paca-postgres_internal` bridge network. An
application compose file in the same project can join it:

```yaml
services:
  app:
    networks:
      - app
      - paca-postgres_internal
    depends_on:
      postgres:
        condition: service_healthy

networks:
  app:
  paca-postgres_internal:
    external: true
```

Keep the app on its own `app` network as well: the `paca-postgres_internal`
network has **no external connectivity** (`internal: true`), so a container
that only joins it loses internet access.

Or attach a running container ad hoc:

```sh
docker network connect paca-postgres_internal <container>
```

Connection string from inside the network:

```
postgres://paca:<password>@postgres:5432/paca?sslmode=disable
```

`make -C db/postgres connection-string` prints it from the running container.

## Configuration

| Variable            | Default | Meaning                                        |
|---------------------|---------|------------------------------------------------|
| `POSTGRES_USER`     | `paca`  | Owner role and superuser of the app database.  |
| `POSTGRES_DB`       | `paca`  | Application database.                          |
| `POSTGRES_PASSWORD` | —       | Required. Set it in `db/postgres/.env`.        |

The password is stored in the cluster using `scram-sha-256` (the PostgreSQL 18
default) and the container runs with `no-new-privileges`.

## Verifying a running instance

`make -C db/postgres smoke` runs `db/postgres/scripts/smoke-test.sh`, which
asserts every guarantee listed above and exits non-zero on the first
violation. Manual equivalents:

```sh
make -C db/postgres psql SQL="select version()"
make -C db/postgres psql SQL="show server_encoding"
make -C db/postgres psql SQL="show timezone"
make -C db/postgres psql SQL="select extname from pg_extension order by extname"
make -C db/postgres psql SQL="select gen_random_uuid(), vector_dims('[1,2,3]'::vector)"
```

## Upgrading

1. Bump `POSTGRES_IMAGE` (tag **and** digest) in `Dockerfile` and `compose.yaml`,
   and `PGVECTOR_VERSION` + `PGVECTOR_SHA256` in `Dockerfile`.
2. `make -C db/postgres build && make -C db/postgres up`.
3. For a major version change, dump with the old image and restore with the new
   one (`pg_dumpall` / `psql`), because the data directory format changes.

## Notes and limits

- The init scripts are **baked into the image** and run only on the **first**
  start of an empty data volume. After changing `initdb/`, rebuild the image
  and recreate the cluster with `make -C db/postgres reset` (this deletes all
  data) or apply the changes manually.
- `internal: true` means the container has no outbound network access either.
  That is intentional for a database, but it blocks image pulls from inside
  the container and any extension that downloads at runtime.
- Connection authentication uses `scram-sha-256` for network clients (the
  PostgreSQL 18 default). Local socket connections inside the container use
  `trust`, which is the official image default.
