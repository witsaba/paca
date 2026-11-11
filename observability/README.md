# PACA observability: tracing with Jaeger v2

The telemetry stack of the monorepository: Jaeger v2 as the OpenTelemetry
collector and as the trace backend. There is no second container — Jaeger v2
*is* a distribution of the collector, so it receives the OTLP traffic of the
services and stores the traces in the same process.

The collector configuration belongs to the repository, in
[`jaeger/config.yaml`](jaeger/config.yaml), in the standard OpenTelemetry
Collector format.

## What you get

- **One container**, `paca-jaeger`, that receives the spans over OTLP and
  serves the UI.
- **Persistent storage**: the traces live in Badger, on the named volume
  `paca-jaeger-data`, so they survive a restart of the stack.
- **A retention of 48 hours**: the traces are deleted two days after they
  were written. The storage does it, not a cron job.
- **The UI on the loopback interface only**: `http://127.0.0.1:16686` is
  reachable from the machine of the developer and from nowhere else.
- **The OTLP receivers on the internal network only**: the services send
  their spans to `jaeger:4317` (gRPC) or `jaeger:4318` (HTTP), and the host
  never sees them.

## Running it

The stack alone:

```sh
make -C observability up       # start and wait until it is healthy
make -C observability smoke    # the query API must answer
make -C observability status   # container state
make -C observability logs     # follow the logs
make -C observability down     # stop (the container is removed, the data is kept)
```

Or with the compose file directly:

```sh
docker compose -f observability/compose.yaml up --wait
```

With the monorepository, the root compose includes the stack, so one command
starts everything:

```sh
docker compose up --wait
```

Both ways drive the same compose project, `paca`: the container, the volume
and the networks are the same ones whichever way the stack is started.

## The ports

| Port   | Service                     | Reachable from                                        |
|--------|-----------------------------|-------------------------------------------------------|
| `4317` | OTLP over gRPC              | The containers of `paca_internal`, only.               |
| `4318` | OTLP over HTTP              | The containers of `paca_internal`, only.               |
| `16686`| Jaeger UI and query API     | The host, on `127.0.0.1` only.                        |

The OTLP receivers are not published: the compose file has no `ports:`
mapping for them. The applications of the monorepository point their OTLP
exporter at `jaeger:4317` and reach it over `paca_internal`, the network
with no outbound access at all.

The query API, the interface and the receivers are not authenticated. Every
container of `paca_internal` can read the traces and write spans. That is
the same trust domain as the database, which keeps its credentials in the
stack that owns it: a container of the internal network is inside the
boundary of the monorepository.

## The networks

The service joins two networks:

- `paca_internal`, the service-to-service network of the monorepository, with
  `internal: true`. The applications send their spans there.
- `paca_observability_ui`, a plain bridge private to this stack. It exists
  because Docker does not program the published ports of a container whose
  only networks are internal ones: without it, the published port of the UI
  is silently dropped by the daemon and the UI is unreachable from the host.
  No other service of the monorepository joins this network.

The price of that second network is the outbound access of the collector: on
a plain bridge it can reach the internet. The collector needs nothing from
the internet, but if the collector has to stay sealed, drop the `ui` network
from `compose.yaml` and reach the UI from a container of `paca_internal`
instead of from the browser of the host.

## The retention

The traces live two days and are deleted afterwards. The policy is in the
collector configuration:

```yaml
extensions:
  jaeger_storage:
    backends:
      traces:
        badger:
          ephemeral: false
          ttl:
            spans: 48h
```

To change it, edit `jaeger/config.yaml`, change `ttl.spans` to the duration
you want (`24h`, `72h`, `168h`) and restart the stack:

```sh
make -C observability restart
```

The storage is on the volume `paca-jaeger-data`, and `ephemeral: false` is
what makes Badger write there instead of a temporary filesystem. Without it,
the traces would be lost at every restart.

## The UI

Open <http://127.0.0.1:16686> in a browser. The UI is published on the
loopback interface only, so it is not reachable from another machine of the
network; to share it, use an SSH tunnel:

```sh
ssh -L 16686:127.0.0.1:16686 <host>
```

## Reading a trace

1. Open the UI and pick a service in the *Service* list.
2. Pick a trace and open it: the spans are ordered in time, and the tree on
   the left is the call hierarchy.
3. Open a span to see its attributes, its duration and the events it
   carries.

The traces arrive over OTLP, so the services need an OTLP exporter pointed
at `jaeger:4317`. The MCP server of the monorepository sends its HTTP spans
there; the trace ID of a span is the same one the server logs.

### The services of the monorepository

Only the services of the monorepository are in the store. The collector
traces its own API — every search of the interface writes a span of its own
— and a filter drops those spans before they reach the storage:

```yaml
filter/own:
  error_mode: ignore
  traces:
    span:
      - 'resource.attributes["service.name"] == "jaeger"'
```

Without that filter, the store fills with the noise of the reading of the
traces: two hundred spans of the service `jaeger` in three hours were
measured on the stack of the monorepository, most of them the polling of the
interface.

The filter governs the spans that **arrive over OTLP**. The spans of the
query API itself are written straight to the storage and never travel
through the pipeline, so no processor can filter them: the switch of the
SDK of the collector is what stops them.

```yaml
environment:
  - OTEL_SDK_DISABLED=true
```

It turns off the instrumentation of the collector and nothing else: its
logs stay, and the spans of the services of the monorepository keep
arriving over OTLP.

The spans of the services are the ones the code creates:

| Service            | Span                    | What it is                                          |
|--------------------|-------------------------|-----------------------------------------------------|
| `paca-mcp-server`  | `POST /mcp`             | The request, from the instrumentation of the HTTP transport. |
| `paca-mcp-server`  | the name of the tool    | The call of a tool: `open_business`, `read_balances`, `record_commitment`. |
| `paca-mcp-server`  | `pg.query`              | A statement of the database.                          |
| `paca-db-admin`    | `db-admin.migrate`      | The boot of the writer of the schema.                |
| `paca-db-admin`    | `schema.migrate`        | The application of the pending migrations.            |
| `paca-db-admin`    | `schema.lock`           | The wait for the lock of the migrations.              |

### The query API

The query API is the v3 one: Jaeger v2.22 removed the v1 endpoint
`/api/services`, which answers 404. The endpoints are under `/api/v3`:

```sh
curl http://127.0.0.1:16686/api/v3/services
curl http://127.0.0.1:16686/api/v3/traces/<trace-id>
```

The parameters of the search are prefixed with `query.`, and the times are
RFC 3339:

```sh
now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
from=$(date -u -v-2H +%Y-%m-%dT%H:%M:%SZ)

curl "http://127.0.0.1:16686/api/v3/traces\
?query.serviceName=paca-mcp-server\
&query.startTimeMin=${from}\
&query.startTimeMax=${now}\
&query.searchDepth=200"
```

Two things surprise everyone once:

- **The name of the service is required.** A search without one answers
  `501`: *"this storage backend requires a service name to search"*. The
  Badger backend of this stack cannot search every service at once.
- **The answer is `{"result": {"resourceSpans": [...]}}`**, not
  `{"traces": [...]}`. The v3 API answers the OpenTelemetry shape, and the
  spans are inside `resourceSpans`.

A single trace is read with the older shape, which answers the trace with
its spans:

```sh
curl "http://127.0.0.1:16686/api/v3/traces/<trace-id>?prettyPrint=true"
```

## The logs

The logs of the services stay on the standard output: Jaeger stores traces
and metrics, it has no log backend, and a second destination for the logs
would be a second container and a second retention to maintain.

The correlation with the traces is made by the `trace_id`, which is written
in every log record of the services. A log line and the trace it belongs to
share that identifier, so the trace of a request can be found from a log
line, and the log lines of a request can be found from the trace.

## Layout

```
observability/
├── Makefile             # the targets of this stack
├── README.md            # this file
├── compose.yaml         # service, networks, volume, healthcheck
└── jaeger/
    └── config.yaml      # the collector configuration of the repository
```

## Notes and limits

- Badger is a **single node** storage. It is the right choice for a
  development stack; it does not scale to several collectors, and it does
  not replicate.
- The container runs as root, because the daemon creates the named volume
  owned by `root:root` and the image runs as uid `10001`, which cannot write
  there. It keeps no capability (`cap_drop: ALL`) and cannot gain more
  (`no-new-privileges`).
- The healthcheck asks the query API whether it answers, with the busybox
  `wget` of the image: the image has no curl and no shell of its own.
