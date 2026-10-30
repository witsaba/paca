# The platform of the PACA monorepository.

The platform holds the infrastructure every service of the monorepository
needs and none of them should write twice: the clock, the logs, the
telemetry, the HTTP server, the health check the container runs, and the
environment every service reads the same way.

It is a **library**. It publishes no port on the network, it starts no
process, and it reads no environment variable.

## The layout

```
packages/platform/
  clock/         the clock of the service: the time it reports
  env/           the environment of the monorepository: the names every
                 service reads, and the way a value is read
  logging/       the logs of the service: the format, the level, the fields
  telemetry/     the traces of the service: the exporter, the sampler, the
                 propagation between the processes, and the configuration of
                 the tracing
  server/        the HTTP server: the routes, the timeouts, the shutdown,
                 and the middleware every request goes through
  healthcheck/   the health check of the container: it asks the service
                 whether it is alive
```

## The middleware of the HTTP server

The server owns the middleware of the requests, because it owns the server:
every service that serves HTTP needs it the same way, and a copy of it in
two services is two promises that drift apart. The middleware lives in
`server/middleware.go` and is built by `server.NewMiddleware`.

The chain is the same for every service:

- **the tracing** is the outer layer, so the span of a request covers the
  recovery of a panic and the line of the log. Every request leaves a span
  behind, named after the route it took. The paths the caller names as
  untraced leave none: the container runtime asks the health endpoint every
  few seconds, and those spans would fill the traces of the work without
  telling anything about it. A middleware without a tracer provider traces
  nothing: the provider of the no-op implementation stands in for it.
- **the recovery of a panic** turns a panic into a 500, so one broken request
  never takes the server down. The panic is logged with its stack.
- **the line of the log** writes one line per request, with the method, the
  path, the status of the response and the duration of the request. The line
  carries the context of the request, so the logger adds the identifiers of
  the trace the request belongs to.
- **the recorder of the status** remembers the status of the response, which
  the line of the log reports and which defaults to 200.
- **the security headers** keep a browser away from the responses of the
  service: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY` and
  `Referrer-Policy: no-referrer`.

The middleware takes its settings as arguments, like the rest of the
platform: the caller hands it the logger and the tracer provider of its own
service, and it names the paths that leave no span behind. The routes
themselves stay with the service: the platform never holds a route of an
application.

## The rules of the module

- **The platform knows no domain.** It answers questions about the machine:
  what time it is, whether the server answers, which trace belongs to which
  request. It never decides anything about the people or about the data: a
  rule of the domain never comes here.
- **The platform is shared, never forked.** A service that wants a different
  clock, a different log format or a different server extends the platform
  or writes its own — it does not copy the file. A copy of the platform is a
  second platform that drifts.
- **The configuration stays with the service.** The platform takes its
  settings as arguments, and the service that owns the settings reads them
  from its own environment. A library that reads the environment of another
  service is a hidden dependency. The configuration of the tracing is the
  exception: it is shared by every service, so it lives in the platform —
  and the platform still reads no variable of the environment. The service
  hands the platform the reader of its own environment, and the reader is
  the one that knows where the variables live.
- **The module is a boundary.** The packages of the platform are ordinary Go
  packages, but the module is a `go.mod`: a service that imports the
  platform cannot reach into its internals by accident, and the platform
  cannot reach into the domain of a service.

## The commands

```sh
make -C packages/platform all     # the whole quality gate of the module
make -C packages/platform help     # the list of the targets
```
