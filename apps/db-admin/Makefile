# The database admin service of the PACA monorepository.
#
# The service is the writer of the schema: it applies the migrations of the
# core when it starts, and it serves a single health endpoint.
#
# Every target of the project is here: the dependencies, the build, the
# tests, the coverage gate, the race detector, the benchmarks, the
# vulnerability scan and the linters.
#
# The tools are pinned in the repository (tools/go.mod), so `go tool` runs
# the same version on every machine and in every container. A contributor
# needs nothing but the Go toolchain; the container targets need Docker.
#
# Run `make` for the list of targets.

GO      ?= go
BIN_DIR ?= $(CURDIR)/bin
COV_DIR ?= $(CURDIR)/.coverage
COV_FILE ?= $(COV_DIR)/coverage.out
BINARY  ?= $(BIN_DIR)/paca-db-admin
PACKAGE ?= ./cmd/paca-db-admin

# The version stamped into the binary. The tags of the repository name it.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# The coverage the module must reach, in percent. The gate fails below it.
COVERAGE_THRESHOLD ?= 80

# The packages the coverage gate measures: the ones whose rules can be
# asserted without a database.
#
# The composition root and the binary are not one of them: they open the
# database, apply the migrations and serve, and the container tests are what
# prove them.
COVER_PKGS ?= $(shell $(GO) list ./... | grep -v '/internal/bootstrap' | grep -v '/cmd/paca-db-admin')

# The files of the module that talk to the database and can only be measured
# against a real one.
#
# The coverage gate of a database-bound module is split in two: the unit gate
# measures what can be asserted without a database, and the container tests
# measure the rest. The file below belongs to the second half: it applies the
# migrations of the schema, and the container tests are what prove it.
COVER_EXCLUDE_FILES ?= internal/adapters/outbound/database/migrator\.go

# The image of the service, and the name of the service in the compose file.
IMAGE  ?= paca/db-admin:$(VERSION)
SERVICE ?= db-admin

.DEFAULT_GOAL := help

.PHONY: help deps tidy tidy-check fmt fmt-check vet lint build test \
        test-integration cover race bench perf vuln all docker-build \
        compose-up compose-down compose-logs smoke run clean

## help: show this list of targets
help:
	@echo "PACA database admin — developer targets"
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
	@echo "  make docker-build      build the image of the service"
	@echo "  make compose-up        start the stack and wait until it is healthy"
	@echo "  make compose-down      stop the stack"
	@echo "  make compose-logs      show the logs of the stack"
	@echo "  make smoke             ask the running service whether it is alive"
	@echo "  make run               run the service"
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
# The tests build the image of the service, start it against a PostgreSQL,
# and assert that the migrations were applied and that the health endpoint
# answers, so they need Docker.
test-integration:
	$(GO) test -tags=integration -timeout=15m -v ./test/...

## cover: run the test suite with coverage and check the threshold
#
# The gate measures the packages whose rules can be asserted without a
# database, and it leaves out the file that applies the migrations: the
# container tests are what prove it.
cover:
	@mkdir -p $(COV_DIR)
	$(GO) test -covermode=atomic -coverprofile=$(COV_FILE) $(COVER_PKGS)
	@grep -v -E '$(COVER_EXCLUDE_FILES)' $(COV_FILE) > $(COV_FILE).unit || true
	@$(GO) tool cover -func=$(COV_FILE).unit | tail -1
	@total=$$($(GO) tool cover -func=$(COV_FILE).unit | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}'); \
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

## docker-build: build the image of the service
#
# The context of the build is the root of the monorepository, because the
# service imports the shared libraries of the repository.
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

## smoke: ask the running service whether it is alive
smoke:
	docker compose exec $(SERVICE) /app/paca-db-admin healthcheck

## run: run the service
#
# The service needs a database: it is the writer of the schema.
run:
	$(GO) run ./cmd/paca-db-admin

## clean: remove the build and the coverage artifacts
clean:
	@rm -rf $(BIN_DIR) $(COV_DIR)
	@echo "cleaned"
