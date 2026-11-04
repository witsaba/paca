# AGENTS.md — `packages/health`

The shared health of the services of the monorepository: the use case that
reports the health of a service and the HTTP endpoint that answers it. The
contract of the root applies here and is never relaxed. This file holds the
specifics of the module.

## What this module is

- A **library**. It publishes no port on the network, it runs no container, it starts no
  process.
- The **single owner** of the health of the services. The model of the health — the
  state, the check and the report — lives in `packages/domain/health`; the use case that
  produces the report and the endpoint that serves it live here.
- The **answer that is the same in every service**. The MCP server and the database
  admin answer the same body, with the same status codes, on the same path.

## The rules of this module

- **The module names the domain and nothing else.** No driver, no SQL, no third-party
  package. The gate and the compiler say so.
- **The three packages are the three layers.** `ports` holds the contracts, `reporter`
  holds the use case, `endpoint` holds the adapter. An adapter that names a use case
  breaks the direction of the dependencies.
- **The port is owned by the consumer of the contract.** The reporter owns the checker
  and the clock; the endpoint owns the querier. A port is narrow: one interface, one
  capability, the methods the caller actually calls.
- **The report is the model of the domain.** The endpoint maps the report to the body of
  the HTTP answer and never decides what the report says.
- **No behavior for the sake of coverage.** A method exists because a caller needs it.
  An unused exported function is debris: it is removed, or its caller is written.

## The shape of a change here

1. The rule changes in the model of the domain (`packages/domain/health`), with its
   test, in that module.
2. The use case changes here, in `reporter`, with its test.
3. The endpoint changes here, in `endpoint`, with its test.
4. The services bind the contracts in their composition root (`internal/bootstrap`), and
   their own copy of the health is deleted.

The gate of the module is `make -C packages/health all`.
