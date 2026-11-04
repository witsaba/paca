# packages/health

The health of the services of the monorepository: the use case that reports it and the
HTTP endpoint that answers it.

The module is a library. It publishes no port on the network and it runs no container.

## The layout

| Package    | What it holds                                                       |
|------------|---------------------------------------------------------------------|
| `ports`    | The contracts: the querier of the endpoint, the checker and the clock of the reporter |
| `reporter` | The use case that reports the health of a service                    |
| `endpoint` | The HTTP endpoint that answers the health                            |

## The model

The model of the health lives in `packages/domain/health`: the state of the service, the
outcome of a single check and the report that aggregates them. This module depends on
that module and on nothing else.

**One failing component makes the whole service unhealthy.** The report is the
aggregate of the checks, and the sickest check wins. A degraded component still serves:
the service answers 200 and the body says that something is wrong.

**A report is returned even when a component fails.** The failure is part of the report,
not an error of the call. The error is reserved for the calls that cannot produce a
report at all, such as a cancelled context.

**Every check is bounded.** A check that hangs may not hang the report: the reporter
gives every check a timeout, and the timeout belongs to the service that names it.

## The endpoint

The path is `/healthz`, and the body is the same in every service:

```json
{
  "status": "ok",
  "service": {"name": "paca-mcp-server", "version": "0.1.0"},
  "uptime_seconds": 60,
  "checked_at": "2026-10-27T10:00:00Z",
  "checks": [{"name": "mcp-transport", "status": "ok", "latency_ms": 1}]
}
```

The endpoint answers `200` while the service can serve, and `503` when it cannot.
