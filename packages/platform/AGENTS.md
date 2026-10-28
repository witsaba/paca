# The contract of the agents — the platform

This module is the **infrastructure** of the services of the monorepository.
Read the root `AGENTS.md` for the contract of the whole repository; this file
holds what is specific to this module.

## What this module is

- A **library** of the infrastructure: the clock, the logs, the telemetry, the
  HTTP server, and the health check of the container.
- The **single owner** of that infrastructure. Every service imports it; none
  of them forks it.

## What this module is not

- It is **not a domain**. It holds no model, no invariant and no rule about
  the people or about the data. A rule that belongs to a domain never comes
  here.
- It is **not a configuration**. The module takes its settings as arguments;
  the service that owns the settings reads them from its own environment.
- It is **not a port**. The contracts between the layers belong to the
  consumer that owns them.

## The rules that hold here

- **The settings come from the caller.** A package of the platform never
  reads an environment variable, never opens a file and never dials a
  service by itself. It receives what it needs.
- **The shared code is extended, never copied.** A service that needs
  something else adds it here, with its tests, or keeps it in the service —
  it does not fork the file.
- **A public promise has a test.** Every exported behavior of a package of
  the platform is proved by a test of that package, through its public
  interface.

## The commands

```sh
make -C packages/platform all     # fmt, vet, lint, build, test, race, coverage, vuln
```
