# AGENTS.md — `apps/db-admin`

The database admin of the monorepository: it applies the pending migrations of the schema
when it starts, and it stays alive with one health endpoint. The contract of the root
applies here and is never relaxed. This file holds the specifics of the service.

## What this service is

- A **hexagonal application**: the rules of the domain live in the middle, the protocol
  lives at the edge, and the composition root is the only place that knows every layer.
- The **single writer of the schema** of the database. No other service migrates, and no
  service migrates on boot: a service that migrates on boot races with the other
  replicas of itself.
- A **client of the shared libraries**: the migrator comes from `packages/database`, and
  the service never writes a migration of its own.

## The layers of this service

```
cmd/paca-db-admin/             the composition root
internal/
  application/                 the use cases: health and migrations
  ports/                      the contracts between the layers
  adapters/
    inbound/http/              the HTTP adapter: the health endpoint
    outbound/database/         the migrator, the pinger and the checker of the database
  platform/config/             the configuration of this service: the only
                               infrastructure the service owns
  bootstrap/                   the wiring of every layer
```

The models live in the shared modules: the state of the schema and the health in
`packages/domain`, the clock, the logs, the telemetry and the server in `packages/platform`.
A service that has a model of its own keeps it in `internal/domain/`; a model that more
than one service needs lives in `packages/domain`, and it is never copied.

## The rules that are specific to this service

- **The schema is written here and nowhere else.** The migrations live in
  `packages/database/migrations/`, the migrator applies them, and this service is the
  only caller of the migrator.
- **The service waits for the database, it does not exit.** A container that exits is
  restarted by the runtime, and the health check of the container declares it unhealthy
  before the restart succeeds.
- **The service opens one connection**, and both the migrations and the health check go
  through it. It does not open a second stack of connections of its own.
- **The state of the schema is one model, in one place.** The model of a migration
  lives in `packages/domain/schema`, and both the service and the library speak it: the
  adapter between them maps nothing. Two copies of the same record drift.
- **The check of the database is the only check of the service.** The service does
  nothing but the database, so the database is the only thing that can be wrong with it.

## The shape of a change here

1. A migration is a new file in `packages/database/migrations/`, numbered after the last
   one.
2. The use case of the migration orchestrates the migrator; it owns no rule of the
   schema.
3. The adapter maps the migrator of the library onto the model of the service, in one
   place.
4. The gate of the service is green: `make -C apps/db-admin all`, and the container
   tests: `make apps-integration`.

## The proof

The container tests prove that the service applies the migrations when it starts, that it
applies nothing on a second run, and that the health endpoint answers.
