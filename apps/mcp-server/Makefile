# The MCP server of the PACA monorepository.
#
# Every target of the project is here: the dependencies, the build, the
# tests, the coverage gate, the race detector, the benchmarks, the
# vulnerability scan and the linters.
#
# The tools are pinned in the repository (tools/go.mod), so `go tool` runs the
# same version on every machine and in every container. A contributor needs
# nothing but the Go toolchain; the container targets need Docker.
#
# Run `make` for the list of targets.

GO      ?= go
BIN_DIR ?= $(CURDIR)/bin
COV_DIR ?= $(CURDIR)/.coverage
COV_FILE ?= $(COV_DIR)/coverage.out
BINARY  ?= $(BIN_DIR)/paca-mcp-server
PACKAGE ?= ./cmd/paca-mcp-server

# The version stamped into the binary. The tags of the repository name it.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# The coverage the module must reach, in percent. The gate fails below it.
COVERAGE_THRESHOLD ?= 80

# The image of the server, and the name of the service in the compose file.
IMAGE  ?= paca/mcp-server:$(VERSION)
SERVICE ?= mcp-server

.DEFAULT_GOAL := help

.PHONY: help deps tidy tidy-check fmt fmt-check vet lint build test \
        test-integration cover race bench perf vuln all docker-build \
        compose-up compose-down compose-logs smoke run clean

## help: show this list of targets
help:
	@echo "PACA MCP server — developer targets"
	@echo ""
	@echo "  make help              show this list of targets"
	@echo "  make deps              download the dependencies"
	@echo "  make tidy              tidy the go.mod of the module"
	@echo "  make tidy-check        fail when the go.mod is not tidy"
	@echo "  make fmt               format the sources"
	@echo "  make fmt-check         fail when a source is not formatted"
	@echo "  make vet               run the analyzers of the toolchain"
	@echo "  make lint              run golangci-lint"
	@echo "  make build             build the binary into bin/"
	@echo "  make test              run the test suite"
	@echo "  make test-integration  run the container tests (Docker)"
	@echo "  make cover             run the test suite with coverage and check the threshold"
	@echo "  make race              run the test suite with the race detector"
	@echo "  make bench             run the benchmarks"
	@echo "  make perf              check the allocation budgets and run the benchmarks"
	@echo "  make vuln              scan the dependencies for known vulnerabilities"
	@echo "  make all               the full quality gate"
	@echo "  make docker-build      build the image of the server"
	@echo "  make compose-up        start the stack and wait until it is healthy"
	@echo "  make compose-down      stop the stack"
	@echo "  make smoke             ask the running server whether it is alive"
	@echo "  make run               run the server"
	@echo "  make clean             remove the build and the coverage artifacts"
	@echo ""
	@echo "The coverage threshold is $(COVERAGE_THRESHOLD)%."

## deps: download the dependencies
deps:
	$(GO) mod download
	$(GO) mod verify

## tidy: tidy the go.mod of the module
tidy:
	$(GO) mod tidy

## tidy-check: fail when the go.mod is not tidy
tidy-check:
	@$(GO) mod tidy
	@if ! git diff --quiet -- go.mod go.sum; then \
		echo "go.mod and go.sum are not tidy, run 'make tidy'"; \
		git diff -- go.mod go.sum; \
		exit 1; \
	fi

## fmt: format the sources
fmt:
	$(GO) fmt ./...

## fmt-check: fail when a source is not formatted
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "these files are not gofmt-formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

## vet: run the analyzers of the toolchain
vet:
	$(GO) vet ./...

## lint: run golangci-lint
lint:
	$(GO) tool golangci-lint run ./...

## build: build the binary into bin/
build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) $(PACKAGE)
	@echo "built $(BINARY) ($(VERSION))"

## test: run the test suite
test:
	$(GO) test ./...

## test-integration: run the container tests
#
# The tests build the image of the server and start it, so they need Docker.
test-integration:
	$(GO) test -tags=integration -timeout=15m -v ./test/...

## cover: run the test suite with coverage and check the threshold
cover:
	@mkdir -p $(COV_DIR)
	$(GO) test -covermode=atomic -coverprofile=$(COV_FILE) ./...
	@$(GO) tool cover -func=$(COV_FILE) | tail -1
	@total=$$($(GO) tool cover -func=$(COV_FILE) | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}'); \
	awk -v total="$$total" -v threshold="$(COVERAGE_THRESHOLD)" 'BEGIN { \
		if (total + 0 < threshold + 0) { \
			printf "the coverage is %s%%, below the %s%% threshold\n", total, threshold; \
			exit 1; \
		} \
		printf "the coverage is %s%%, at or above the %s%% threshold\n", total, threshold; \
	}'

## race: run the test suite with the race detector
race:
	$(GO) test -race ./...

## bench: run the benchmarks
bench:
	$(GO) test -run '^$$' -bench . -benchmem ./...

## perf: check the allocation budgets and run the benchmarks
#
# The budgets live with the code they guard, so a regression fails the test
# suite and not only this target.
perf:
	$(GO) test -run 'Budget' -v ./...
	$(GO) test -run '^$$' -bench . -benchmem ./...

## vuln: scan the dependencies for known vulnerabilities
vuln:
	$(GO) tool govulncheck ./...

## all: the full quality gate
all: fmt-check vet lint build test race cover vuln

## docker-build: build the image of the server
#
# The context of the build is the root of the monorepository, because the
# server imports the shared libraries of the repository.
docker-build:
	docker build --build-arg VERSION=$(VERSION) -f Dockerfile -t $(IMAGE) ../..

## compose-up: start the stack and wait until it is healthy
compose-up:
	docker compose up --build -d --wait $(SERVICE)

## compose-down: stop the stack
compose-down:
	docker compose down

## compose-logs: show the logs of the stack
compose-logs:
	docker compose logs -f $(SERVICE)

## smoke: ask the running server whether it is alive
smoke:
	docker compose exec $(SERVICE) /app/paca-mcp-server healthcheck

## run: run the server
run:
	$(GO) run ./cmd/paca-mcp-server

## clean: remove the build and the coverage artifacts
clean:
	@rm -rf $(BIN_DIR) $(COV_DIR)
	@echo "cleaned"
