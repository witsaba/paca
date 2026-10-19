# apps/

Deployable applications.

Each subdirectory here is one runnable application — a desktop client, a CLI, a local
API server. An app owns its own runtime, its own dependencies, and its own release
cadence, and depends on the shared packages under `packages/`.

Today:

- [`cli/`](cli/README.md) — the `paca` command, the command line application of PACA.
  It is a Go module of the workspace (`github.com/witsaba/paca/apps/cli`) and currently
  ships the command header only: `--help` is the only supported command. It composes
  the shared core in [`packages/core`](../packages/core/README.md).

Conventions for this directory:

- one directory per application, named after what it is (`desktop/`, `cli/`, `api/`);
- one Go module per application, tied together by the workspace at the repository root;
- the application holds no schema. Migrations belong to `packages/db/migrations/`;
- a README at the root of each application explaining how to run it against a
  migrated database.
