# The entry point of the monorepository.
#
# The root dispatches; it holds no stack-specific logic. Every stack owns its
# own Makefile:
#
#   apps/mcp-server/Makefile   the MCP server
#   db/Makefile                the databases, one directory and one stack each
#
# Adding an application means two steps:
#
#   1. create apps/<name>/ with its own Makefile;
#   2. add <name> to APPS below.
#
# Nothing else in the monorepository has to change: the continuous
# integration calls the targets of this file, so the workflow never names a
# stack.
#
# Run `make` for the list of targets.

APPS := mcp-server

# The shell scripts of the repository. The shellcheck of the continuous
# integration runs over exactly this list.
SHELL_SCRIPTS := $(shell find . -name '*.sh' -not -path './.git/*' | sort)

.DEFAULT_GOAL := help

.PHONY: help all gate shellcheck db-up db-smoke db-down

help: ## Show this help
	@echo "PACA — the targets of the monorepository"
	@echo ""
	@echo "  make help           show this help"
	@echo "  make all            the full gate: the applications, the shellcheck and the databases"
	@echo "  make gate           the quality gate of every application"
	@echo "  make shellcheck     run shellcheck over the shell scripts of the repository"
	@echo "  make db-up          start every database and wait until it is healthy"
	@echo "  make db-smoke       run the smoke test of every database (it must be running)"
	@echo "  make db-down        stop every database (keeps the data volumes)"
	@echo ""
	@echo "The applications of the monorepository:"
	@for app in $(APPS); do \
		echo "  $$app -> apps/$$app"; \
	done

all: gate shellcheck db-smoke ## The full gate: the applications, the shellcheck and the databases

gate: ## The quality gate of every application
	@for app in $(APPS); do \
		echo "==> $$app: gate"; \
		$(MAKE) --no-print-directory -C apps/$$app all || exit 1; \
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
