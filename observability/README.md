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

The query API is the v3 one: Jaeger v2.22 removed the v1 endpoint
`/api/services`, which answers 404. The endpoints are under `/api/v3`:

```sh
curl http://127.0.0.1:16686/api/v3/services
curl http://127.0.0.1:16686/api/v3/traces/<trace-id>
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
