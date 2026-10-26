# Paca

## Databases

- [`db/postgres`](db/postgres/README.md) - PostgreSQL 18.6 with pgvector 0.8.7,
  UUID and UTC support, reachable only from the internal Docker network.

This is a monorepository: every stack owns its own directory and its own
Makefile, and the root of the repository carries no stack-specific targets.
`db/Makefile` is the entry point of the database domain and forwards to the
stack of each database.

```sh
cp db/postgres/env.example db/postgres/.env   # set the password
make -C db/postgres up                         # build, start, wait until healthy
make -C db/postgres shell                      # psql inside the container
make -C db/postgres smoke                      # assert the guarantees
```

`make -C db help` lists the aggregate targets that apply to every database.
