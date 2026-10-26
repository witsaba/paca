# The database umbrella of the monorepository.
#
# Every database lives in its own directory and owns its own Makefile:
#
#   db/postgres/Makefile   the PostgreSQL + pgvector stack
#
# This Makefile only dispatches. Adding a database means two steps:
#
#   1. create db/<name>/ with its own Makefile and compose file;
#   2. add <name> to DATABASES below.
#
# Nothing else in the monorepository has to change: the root of the
# repository stays free of stack-specific targets, so the Go backend, the
# MCP server and the web client can each bring their own.

DATABASES := postgres

.DEFAULT_GOAL := help

.PHONY: help build up down status smoke

help: ## Show this help
	@echo "Databases of the monorepository:"
	@for db in $(DATABASES); do \
		echo "  $$db -> db/$$db"; \
	done
	@echo
	@echo "Aggregate targets (every database):"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'
	@echo
	@echo "Per-database targets: make -C db/<name> help"

build: ## Build the image of every database
	@for db in $(DATABASES); do \
		echo "==> $$db: build"; \
		$(MAKE) --no-print-directory -C $$db build || exit 1; \
	done

up: ## Start every database and wait until it is healthy
	@for db in $(DATABASES); do \
		echo "==> $$db: up"; \
		$(MAKE) --no-print-directory -C $$db up || exit 1; \
	done

down: ## Stop every database (keeps the data volumes)
	@for db in $(DATABASES); do \
		echo "==> $$db: down"; \
		$(MAKE) --no-print-directory -C $$db down || exit 1; \
	done

status: ## Show the container status of every database
	@for db in $(DATABASES); do \
		echo "==> $$db: status"; \
		$(MAKE) --no-print-directory -C $$db status || exit 1; \
	done

smoke: ## Run the smoke test of every database
	@for db in $(DATABASES); do \
		echo "==> $$db: smoke"; \
		$(MAKE) --no-print-directory -C $$db smoke || exit 1; \
	done
