# The stack of the memory of PACA: Engram.
#
# The store of the memory of the monorepository. It is a container of its
# own, with its own image and its own state, and it publishes no port: the
# agent will reach it through the socket of its volume, when the slice of
# the wiring of the agent comes.
#
# The stack drives its own compose file, which belongs to the root compose
# project (`name: paca`): the container, the volume and the network are the
# same ones whichever way the stack is started, so this Makefile works
# alone and with the rest of the monorepository:
#
#   make -C engram up
#
# The paths are relative to this file, so the targets work from any working
# directory.

COMPOSE := docker compose -f compose.yaml
SERVICE := engram

.DEFAULT_GOAL := help

.PHONY: help up down restart logs status smoke payload

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

up: ## Start the memory and wait until it is healthy
	$(COMPOSE) up -d --wait $(SERVICE)

down: ## Stop the memory (keeps the container and the data volume)
	$(COMPOSE) stop $(SERVICE)
	$(COMPOSE) rm -f $(SERVICE)

restart: ## Restart the memory
	$(COMPOSE) restart $(SERVICE)

logs: ## Follow the logs of the memory
	$(COMPOSE) logs -f $(SERVICE)

status: ## Show the memory container status
	$(COMPOSE) ps $(SERVICE)

smoke: ## Assert that the store keeps a memory and gives it back (it fails when it does not)
	@$(COMPOSE) exec -T --user 10000:10000 $(SERVICE) sh /usr/local/bin/paca-engram-smoke

# The publication of the payload of the store: the verified executable of
# the store is written on the volume `paca-engram-payload`, which the
# stack of the agent mounts read-only. The service is a one-shot one, so
# the target waits for it and returns its exit code: the payload is
# published or it is not.
#
# It is safe to run it twice: the service writes the payload again and the
# volume ends up with the same bytes in it.
payload: ## Publish the verified payload of the store on the volume the agent mounts
	$(COMPOSE) up --build --exit-code-from engram-payload engram-payload
