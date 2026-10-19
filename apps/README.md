# apps/

Reserved for deployable applications.

Each subdirectory here is one runnable application — a desktop client, a CLI, a local
API server. An app owns its own runtime, its own dependencies, and its own release
cadence, and depends on the shared packages under `packages/`.

Nothing here yet: PACA is database-only for now. The schema that the future
applications will read lives in [`packages/db`](../packages/db/README.md).

Conventions for this directory, once it is populated:

- one directory per application, named after what it is (`desktop/`, `cli/`, `api/`);
- the application holds no schema. Migrations belong to `packages/db/migrations/`;
- a README at the root of each application explaining how to run it against a
  migrated database.
