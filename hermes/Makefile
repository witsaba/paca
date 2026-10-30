# The stack of the agent of PACA: Hermes Agent, the unit of the
# monorepository that talks to the people who use PACA.
#
# The stack drives its own compose file, which belongs to the root compose
# project (`name: paca`): the container, the volume and the network are the
# same ones whichever way the stack is started, so this Makefile works
# alone and with the rest of the monorepository:
#
#   make -C hermes up
#
# The paths are relative to this file, so the targets work from any working
# directory.

COMPOSE := docker compose -f compose.yaml
SERVICE := hermes

.DEFAULT_GOAL := help

.PHONY: help up down restart logs status smoke

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

up: ## Start the agent and wait until it is healthy
	$(COMPOSE) up -d --wait $(SERVICE)

down: ## Stop the agent (keeps the container and the data volume)
	$(COMPOSE) stop $(SERVICE)
	$(COMPOSE) rm -f $(SERVICE)

restart: ## Restart the agent
	$(COMPOSE) restart $(SERVICE)

logs: ## Follow the logs of the agent
	$(COMPOSE) logs -f $(SERVICE)

status: ## Show the agent container status
	$(COMPOSE) ps $(SERVICE)

smoke: ## Check that the health endpoint of the agent answers (it fails when it does not)
	@if $(COMPOSE) exec -T $(SERVICE) /usr/bin/curl -fsS -m 5 -o /dev/null http://127.0.0.1:8642/health; then \
		echo "hermes: the health endpoint answers"; \
	else \
		echo "hermes: the health endpoint does not answer"; \
		exit 1; \
	fi
