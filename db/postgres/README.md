# PACA database: PostgreSQL 18.6 + pgvector 0.8.7

Internal-only database for PACA. Nothing is published on the host: the
container lives on an `internal` Docker network, so only other containers on
that network can open a connection.

The domain this database belongs to is described in
[db/README.md](../README.md): the layout of the `db/` folder, the aggregate
commands and how to add another database.

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
compose.yaml            # the root: includes every stack, declares the networks
db/
├── Makefile            # umbrella: aggregates the targets of every database
├── README.md           # the database domain
└── postgres/
    ├── Makefile        # the targets of this stack
    ├── Dockerfile      # postgres 18.6 + pgvector 0.8.7 (multi-stage)
    ├── compose.yaml    # service, network, volume, healthcheck
    ├── env.example     # the manifest of the environment of the stack
    ├── initdb/         # baked into the image, runs on first init
    │   ├── 01-extensions.sql  # vector, uuid-ossp, pgcrypto
    │   └── 02-defaults.sql    # timezone + client_encoding per database/role
    ├── scripts/
    │   └── smoke-test.sh    # asserts encoding, timezone, extensions, isolation
    └── README.md
```

This is a monorepository: every stack owns its own directory and its own
Makefile, and the root of the repository carries no stack-specific targets.
The stack is included by the root [compose.yaml](../../compose.yaml) and
belongs to the same compose project, `paca`, so the container, the volume and
the networks are the same ones whichever way it is started. See
[db/README.md](../README.md) for the domain view.

## Usage

```sh
# The credentials of the stack are written by the installer of the
# environment, which offers the defaults and generates a strong password:
make env-init

make -C db/postgres up       # build, start, wait until healthy
make -C db/postgres shell    # psql inside the container
make -C db/postgres smoke    # assert encoding, timezone, extensions and isolation
make -C db/postgres status   # container state
make -C db/postgres logs     # follow logs
make -C db/postgres down     # stop (the container is removed, the data is kept)
make -C db/postgres reset    # DESTRUCTIVE: stop and delete the data volume
```

The targets drive the root compose project, scoped to this service. The whole
monorepository starts with:

```sh
docker compose up --wait
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

The service is attached to the `paca_internal` bridge network, declared with
`internal: true`: a container on it has no outbound access at all. The
application of the monorepository joins that network through the root
compose:

```yaml
services:
  app:
    networks:
      - internal   # the database, service-to-service
      - egress     # the internet, if the service needs it
    depends_on:
      postgres:
        condition: service_healthy
```

The network is declared in the root compose and in the compose file of every
stack that uses it, with the same attributes and the same fixed name, so the
stack also runs on its own. `paca_egress` is the plain bridge network for the
services that need the internet; the database never joins it.

Or attach a running container ad hoc:

```sh
docker network connect paca_internal <container>
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
| `POSTGRES_PASSWORD` | —       | Required. Generated by the installer of the environment. |

The three variables live in `db/postgres/.env`, which the compose file of the
stack reads with `env_file`. The path is relative to that compose file, so the
stack reads the same file whether it runs alone or included from the root. A
missing `.env` is a hard error, so the password is never silently defaulted.

The file is written by the installer of the environment, from the manifest of
this stack, `env.example`:

```sh
make env-init                # ask for every value
make env-init ARGS="--yes"   # keep what is configured, generate the password
```

The manifest declares the password as a secret that can be generated, so the
installer offers a strong one instead of the placeholder of the example. The
values already configured are kept, and the password is never printed.

The password of a cluster that is already initialized is the one it was
created with: PostgreSQL reads `POSTGRES_PASSWORD` only when the data
directory is empty. Changing the value in the `.env` of an existing cluster
does not change the password of the role, and the cluster then rejects the
new one. To change the password of a running cluster, change it in the
database itself and in the `.env` together.

To write the file by hand instead, copy the example and change the values:

```sh
cp db/postgres/env.example db/postgres/.env
```

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
