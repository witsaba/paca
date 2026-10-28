# The entry point of the monorepository.
#
# The root dispatches; it holds no stack-specific logic. Every stack owns its
# own Makefile:
#
#   apps/mcp-server/Makefile   the MCP server
#   apps/db-admin/Makefile     the database admin: the writer of the schema
#   db/Makefile                the databases, one directory and one stack each
#   observability/Makefile     the traces: the collector and the backend
#
# The scripts of the root are the ones of the repository itself:
#
#   scripts/setup-env.sh       the installer of the environment of the stacks
#
# Adding an application means two steps:
#
#   1. create apps/<name>/ with its own Makefile;
#   2. add <name> to APPS below.
#
# A shared library means the same:
#
#   1. create packages/<name>/ with its own Makefile;
#   2. add <name> to PACKAGES below.
#
# Nothing else in the monorepository has to change: the continuous
# integration calls the targets of this file, so the workflow never names a
# stack.
#
# Run `make` for the list of targets.

APPS := mcp-server db-admin

# The shared libraries of the monorepository. A service imports them; they
# publish no port and run no container.
PACKAGES := core

# The shell scripts of the repository. The shellcheck of the continuous
# integration runs over exactly this list.
SHELL_SCRIPTS := $(shell find . -name '*.sh' -not -path './.git/*' | sort)

.DEFAULT_GOAL := help

.PHONY: help all gate packages-integration apps-integration shellcheck \
        db-up db-smoke db-down \
        observability-up observability-smoke observability-down \
        env-init env-test

help: ## Show this help
	@echo "PACA — the targets of the monorepository"
	@echo ""
	@echo "  make help           show this help"
	@echo "  make all            the full gate: the applications, the shellcheck and the databases"
	@echo "  make gate           the quality gate of every application and every package"
	@echo "  make packages-integration  the container tests of every package (Docker)"
	@echo "  make apps-integration     the container tests of every application (Docker)"
	@echo "  make shellcheck     run shellcheck over the shell scripts of the repository"
	@echo "  make db-up          start every database and wait until it is healthy"
	@echo "  make db-smoke       run the smoke test of every database (it must be running)"
	@echo "  make db-down        stop every database (keeps the data volumes)"
	@echo "  make observability-up    start the collector of the traces and wait until it is healthy"
	@echo "  make observability-smoke run the smoke test of the collector (it must be running)"
	@echo "  make observability-down  stop the collector of the traces (keeps the traces)"
	@echo ""
	@echo "The installation of the environment:"
	@echo "  make env-init           write the .env file of every stack (asks for the values)"
	@echo "  make env-test           run the tests of the installer of the environment"
	@echo ""
	@echo "The applications of the monorepository:"
	@for app in $(APPS); do \
		echo "  $$app -> apps/$$app"; \
	done
	@echo ""
	@echo "The shared libraries of the monorepository:"
	@for pkg in $(PACKAGES); do \
		echo "  $$pkg -> packages/$$pkg"; \
	done

all: gate shellcheck db-smoke ## The full gate: the applications, the shellcheck and the databases

gate: ## The quality gate of every application and every package
	@for app in $(APPS); do \
		echo "==> $$app: gate"; \
		$(MAKE) --no-print-directory -C apps/$$app all || exit 1; \
	done
	@for pkg in $(PACKAGES); do \
		echo "==> packages/$$pkg: gate"; \
		$(MAKE) --no-print-directory -C packages/$$pkg all || exit 1; \
	done

packages-integration: ## The container tests of every package
	@for pkg in $(PACKAGES); do \
		echo "==> packages/$$pkg: integration tests"; \
		$(MAKE) --no-print-directory -C packages/$$pkg test-integration || exit 1; \
	done

# The container tests of the applications: they build the image of the
# service, start it against a PostgreSQL and assert the guarantees it
# promises. The rules that live in the composition root — the service
# applies the migrations when it starts and it keeps serving — can only be
# asserted against the real thing, so these tests are part of the gate.
apps-integration: ## The container tests of every application
	@for app in $(APPS); do \
		echo "==> apps/$$app: container tests"; \
		$(MAKE) --no-print-directory -C apps/$$app test-integration || exit 1; \
	done

shellcheck: ## Run shellcheck over the shell scripts of the repository
	@if ! command -v shellcheck >/dev/null 2>&1; then \
		echo "shellcheck is not installed; install it to run this target"; \
		exit 1; \
	fi
	@if [ -z "$(SHELL_SCRIPTS)" ]; then \
		echo "there is no shell script to check"; \
		exit 0; \
	fi
	@echo "==> shellcheck: $(words $(SHELL_SCRIPTS)) script(s)"
	@shellcheck $(SHELL_SCRIPTS)

db-up: ## Start every database and wait until it is healthy
	$(MAKE) --no-print-directory -C db up

db-smoke: ## Run the smoke test of every database (it must be running)
	$(MAKE) --no-print-directory -C db smoke

db-down: ## Stop every database (keeps the data volumes)
	$(MAKE) --no-print-directory -C db down

observability-up: ## Start the collector of the traces and wait until it is healthy
	$(MAKE) --no-print-directory -C observability up

observability-smoke: ## Run the smoke test of the collector (it must be running)
	$(MAKE) --no-print-directory -C observability smoke

observability-down: ## Stop the collector of the traces (keeps the traces)
	$(MAKE) --no-print-directory -C observability down

# The installer of the environment: it writes the .env file of every stack
# that ships an env.example, offering the default of every value and
# generating a strong password for the secrets. The values already
# configured are kept, and can be changed in the questions it asks.
#
# The arguments reach the script, so the non-interactive modes are reachable
# from the Makefile too:
#
#   make env-init                      ask for every value
#   make env-init ARGS="--yes"        keep what is configured, generate the rest
#   make env-init ARGS="--dry-run"     print what it would write, write nothing
env-init: ## Write the .env file of every stack of the monorepository
	@./scripts/setup-env.sh $(ARGS)

# The tests of the installer: they run against fixtures in a temporary
# directory, so the .env of the repository is never touched.
env-test: ## Run the tests of the installer of the environment
	@./scripts/tests/setup-env-test.sh
