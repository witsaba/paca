# The PostgreSQL stack of PACA: image, compose service and smoke test.
#
# The stack is part of the root compose project, so the commands here drive
# the root compose file and scope themselves to this service:
#
#   make -C db/postgres up
#
# The paths are relative to this file, so the targets work from any working
# directory. The umbrella for every database of the monorepository is
# db/Makefile.

COMPOSE := docker compose -f ../../compose.yaml
SERVICE := postgres

.DEFAULT_GOAL := help

.PHONY: help build up down restart logs status shell psql smoke connection-string reset

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the PostgreSQL + pgvector image
	$(COMPOSE) build --pull $(SERVICE)

up: build ## Start the database and wait until it is healthy
	$(COMPOSE) up -d --wait $(SERVICE)

down: ## Stop the database (keeps the container and the data volume)
	$(COMPOSE) stop $(SERVICE)
	$(COMPOSE) rm -f $(SERVICE)

restart: ## Restart the database
	$(COMPOSE) restart $(SERVICE)

logs: ## Follow the database logs
	$(COMPOSE) logs -f $(SERVICE)

status: ## Show the database container status
	$(COMPOSE) ps $(SERVICE)

shell: ## Open a psql session inside the database container
	$(COMPOSE) exec $(SERVICE) sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

psql: ## Run a single SQL statement, e.g. make psql SQL="select version()"
	$(COMPOSE) exec -T -e SQL="$(SQL)" $(SERVICE) sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "$$SQL"'

smoke: ## Run the smoke test (encoding, timezone, extensions, isolation)
	./scripts/smoke-test.sh

connection-string: ## Print the internal connection string for the app
	@$(COMPOSE) exec -T $(SERVICE) sh -c 'echo "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@postgres:5432/$$POSTGRES_DB?sslmode=disable"'

reset: ## DESTRUCTIVE: remove the database container and its data volume
	$(COMPOSE) stop $(SERVICE)
	$(COMPOSE) rm -f $(SERVICE)
	docker volume rm -f paca-postgres-data
