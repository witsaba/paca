# Paca

## Databases

- [`db/postgres`](db/postgres/README.md) - PostgreSQL 18.6 with pgvector 0.8.7,
  UUID and UTC support, reachable only from the internal Docker network.

```sh
cp db/postgres/env.example db/postgres/.env   # set the password
make db-up                                     # build, start, wait until healthy
make db-shell                                  # psql inside the container
```
