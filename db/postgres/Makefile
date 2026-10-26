# The PostgreSQL stack of PACA: image, compose service and smoke test.
#
# Every path here is relative to db/postgres, so the targets work from any
# working directory:
#
#   make -C db/postgres up
#   cd db/postgres && make up
#
# The umbrella for every database of the monorepository is db/Makefile.

COMPOSE := docker compose -f compose.yaml
SCRIPT  := ./scripts/smoke-test.sh

.DEFAULT_GOAL := help

.PHONY: help build up down restart logs status shell psql smoke connection-string reset

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the PostgreSQL + pgvector image
	$(COMPOSE) build --pull

up: build ## Start the database and wait until it is healthy
	$(COMPOSE) up -d --wait

down: ## Stop the database (keeps the data volume)
	$(COMPOSE) down

restart: ## Restart the database
	$(COMPOSE) restart postgres

logs: ## Follow the database logs
	$(COMPOSE) logs -f postgres

status: ## Show the database container status
	$(COMPOSE) ps

shell: ## Open a psql session inside the database container
	$(COMPOSE) exec postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

psql: ## Run a single SQL statement, e.g. make psql SQL="select version()"
	$(COMPOSE) exec -T -e SQL="$(SQL)" postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "$$SQL"'

smoke: ## Run the smoke test (encoding, timezone, extensions, isolation)
	$(SCRIPT)

connection-string: ## Print the internal connection string for the app
	@$(COMPOSE) exec -T postgres sh -c 'echo "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@postgres:5432/$$POSTGRES_DB?sslmode=disable"'

reset: ## DESTRUCTIVE: stop the database and delete its data volume
	$(COMPOSE) down -v
