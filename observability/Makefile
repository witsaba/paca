# The observability stack of PACA: Jaeger v2, the OpenTelemetry collector
# and the trace backend of the monorepository.
#
# The stack drives its own compose file, which belongs to the root compose
# project (`name: paca`): the container, the volume and the network are the
# same ones whichever way the stack is started, so this Makefile works
# alone and with the rest of the monorepository:
#
#   make -C observability up
#
# The paths are relative to this file, so the targets work from any working
# directory.

COMPOSE := docker compose -f compose.yaml
SERVICE := jaeger

.DEFAULT_GOAL := help

.PHONY: help up down restart logs status smoke

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

up: ## Start Jaeger and wait until it is healthy
	$(COMPOSE) up -d --wait $(SERVICE)

down: ## Stop Jaeger (keeps the container and the data volume)
	$(COMPOSE) stop $(SERVICE)
	$(COMPOSE) rm -f $(SERVICE)

restart: ## Restart Jaeger
	$(COMPOSE) restart $(SERVICE)

logs: ## Follow the logs of Jaeger
	$(COMPOSE) logs -f $(SERVICE)

status: ## Show the Jaeger container status
	$(COMPOSE) ps $(SERVICE)

smoke: ## Check that the query API of Jaeger answers (it fails when it does not)
	@if $(COMPOSE) exec -T $(SERVICE) /bin/busybox wget -q -O /dev/null http://127.0.0.1:16686/api/v3/services; then \
		echo "jaeger: the query API answers"; \
	else \
		echo "jaeger: the query API does not answer"; \
		exit 1; \
	fi
