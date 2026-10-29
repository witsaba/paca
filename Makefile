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
#   scripts/stack-schema.sh    the assertion of the schema of the stack
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
PACKAGES := domain database platform

# The compose project of the monorepository: the root compose file includes
# every stack, so the whole project is driven from here.
COMPOSE := docker compose -f compose.yaml

# The shell scripts of the repository. The shellcheck of the continuous
# integration runs over exactly this list.
SHELL_SCRIPTS := $(shell find . -name '*.sh' -not -path './.git/*' | sort)

.DEFAULT_GOAL := help

.PHONY: help all gate packages-integration apps-integration shellcheck \
        db-up db-smoke db-down \
        observability-up observability-smoke observability-down \
        stack-up stack-schema stack-down \
        env-init env-test structure-test ci-filters-test

help: ## Show this help
	@echo "PACA — the targets of the monorepository"
	@echo ""
	@echo "  make help           show this help"
	@echo "  make all            the full gate of the repository: it proves it from nothing and leaves no container behind"
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
	@echo "The whole monorepository:"
	@echo "  make stack-up        start every stack and wait until the whole project is healthy"
	@echo "  make stack-schema    assert that the schema was written by the service that owns it"
	@echo "  make stack-down      stop every container of the project (keeps the named volumes)"
	@echo ""
	@echo "The installation of the environment:"
	@echo "  make env-init           write the .env file of every stack (asks for the values)"
	@echo "  make env-test           run the tests of the installer of the environment"
	@echo "  make structure-test      run the tests of the structure of the repository"
	@echo "  make ci-filters-test     run the tests of the filters of the continuous integration"
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

# The gate of the repository: the one command that proves it.
#
# It runs from nothing: it writes the environment it needs, it starts what
# it must smoke, and it stops what it started. A step that fails stops the
# gate, and the exit code says so.
#
# The whole recipe is one shell, so a trap can stop the containers the
# gate started when a step fails: the gate never leaves a container
# behind. GNU make 3.81 has no .ONESHELL, so the recipe is one logical
# line with the continuations of the shell.
all: ## The full gate of the repository: it proves it from nothing and leaves no container behind
	@set -e; \
	cleanup() { \
		status=$$?; \
		set +e; \
		if [ "$$status" -ne 0 ]; then \
			echo "==> the gate failed: stopping the containers it started"; \
		fi; \
		$(MAKE) --no-print-directory stack-down; \
		$(MAKE) --no-print-directory db-down; \
		$(MAKE) --no-print-directory observability-down; \
		exit $$status; \
	}; \
	trap cleanup EXIT INT TERM; \
	echo "==> the environment of the stacks"; \
	$(MAKE) --no-print-directory env-init ARGS="--yes"; \
	echo "==> the quality gate"; \
	$(MAKE) --no-print-directory gate; \
	echo "==> the container tests of the packages"; \
	$(MAKE) --no-print-directory packages-integration; \
	echo "==> the container tests of the applications"; \
	$(MAKE) --no-print-directory apps-integration; \
	echo "==> the shellcheck of the scripts"; \
	$(MAKE) --no-print-directory shellcheck; \
	echo "==> the tests of the installer of the environment"; \
	$(MAKE) --no-print-directory env-test; \
	echo "==> the tests of the structure of the repository"; \
	$(MAKE) --no-print-directory structure-test; \
	echo "==> the tests of the filters of the continuous integration"; \
	$(MAKE) --no-print-directory ci-filters-test; \
	echo "==> the database"; \
	$(MAKE) --no-print-directory db-up; \
	$(MAKE) --no-print-directory db-smoke; \
	$(MAKE) --no-print-directory db-down; \
	echo "==> the collector of the traces"; \
	$(MAKE) --no-print-directory observability-up; \
	$(MAKE) --no-print-directory observability-smoke; \
	$(MAKE) --no-print-directory observability-down; \
	echo "==> the whole stack"; \
	$(MAKE) --no-print-directory stack-up; \
	$(MAKE) --no-print-directory stack-schema; \
	$(MAKE) --no-print-directory stack-down; \
	echo "All the steps of the gate passed."

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

# The whole monorepository: the root compose file includes every stack, so
# the project comes up as a whole and the schema is written by the service
# that owns it. The stack is the proof of the rule: from nothing, the
# schema is there when the project is healthy.
stack-up: ## Start every stack and wait until the whole project is healthy
	$(COMPOSE) up -d --build --wait

stack-schema: ## Assert that the schema was written by the service that owns it
	@./scripts/stack-schema.sh

stack-down: ## Stop every container of the project (keeps the named volumes)
	$(COMPOSE) down

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

# The tests of the structure: every module resolves its own graph and its own
# dependencies, every link of the documentation points at a file that exists,
# and the layers of every application respect the direction of the
# dependencies: the domain knows no technology, the application knows no
# adapter, and only the composition root knows the adapters.
structure-test: ## Run the tests of the structure of the repository
	@./scripts/tests/structure-test.sh

# The tests of the filters of the continuous integration: the rules of
# scripts/ci/job-filters.txt say which job of the workflow a change needs,
# and the tests prove it against a matrix of changes. A job runs only when
# the change can affect what it proves.
ci-filters-test: ## Run the tests of the filters of the continuous integration
	@./scripts/tests/ci-filters-test.sh
