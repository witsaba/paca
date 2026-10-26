# PACA - developer commands
#
# Database targets live in db/postgres (see db/postgres/README.md).

COMPOSE := docker compose -f db/postgres/compose.yaml

.DEFAULT_GOAL := help

.PHONY: help db-build db-up db-down db-restart db-logs db-status db-shell db-psql db-smoke db-reset db-connection-string

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

db-build: ## Build the PostgreSQL + pgvector image
	$(COMPOSE) build --pull

db-up: db-build ## Start the database and wait until it is healthy
	$(COMPOSE) up -d --wait

db-down: ## Stop the database (keeps the data volume)
	$(COMPOSE) down

db-restart: ## Restart the database
	$(COMPOSE) restart postgres

db-logs: ## Follow the database logs
	$(COMPOSE) logs -f postgres

db-status: ## Show the database container status
	$(COMPOSE) ps

db-shell: ## Open a psql session inside the database container
	$(COMPOSE) exec postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

db-psql: ## Run a single SQL statement, e.g. make db-psql SQL="select version()"
	$(COMPOSE) exec -T -e SQL="$(SQL)" postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB" -c "$$SQL"'

db-smoke: ## Run the database smoke test (encoding, timezone, extensions, isolation)
	./db/postgres/scripts/smoke-test.sh

db-connection-string: ## Print the internal connection string for the app
	@$(COMPOSE) exec -T postgres sh -c 'echo "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@postgres:5432/$$POSTGRES_DB?sslmode=disable"'

db-reset: ## DESTRUCTIVE: stop the database and delete its data volume
	$(COMPOSE) down -v
