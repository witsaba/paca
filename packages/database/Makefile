# The database library of the PACA monorepository.
#
# The library holds the implementation of the persistence: the pool of
# connections, the queries, the migrations of the schema and the
# repositories that write the rows. It is a library: it publishes no port,
# it runs no container, and the services import it.
#
# The models of the domain live in another module, packages/domain, and this
# one depends on it. The direction of the dependency is the toolchain: the
# domain imports nothing but the standard library, so it cannot reach the
# queries of this module.
#
# Every target of the project is here: the dependencies, the generation of
# the code, the build, the tests, the coverage gate, the race detector, the
# benchmarks and the vulnerability scan.
#
# The tools are pinned in the repository (tools/go.mod), so `go tool` runs the
# same version on every machine and in every container. A contributor needs
# nothing but the Go toolchain; the integration tests need Docker.
#
# Run `make` for the list of targets.

GO      ?= go
COV_DIR ?= $(CURDIR)/.coverage
COV_FILE ?= $(COV_DIR)/coverage.out
PACKAGE ?= ./...

# The coverage the module must reach, in percent. The gate fails below it.
COVERAGE_THRESHOLD ?= 80

# The coverage of the container tests. It is measured over the whole module,
# because the container tests are what cover the packages that talk to the
# database.
COV_FILE_INTEGRATION ?= $(COV_DIR)/coverage-integration.out

# The packages the coverage gate measures: the ones whose rules can be
# asserted without a database.
#
# The generated code is not one of them: it is written by the generator,
# never by hand, and it is reviewed as the schema it comes from. Neither are
# the packages that talk to the database: the container tests cover them,
# and they report their own coverage.
COVER_PKGS ?= $(shell $(GO) list ./... | grep -v '/internal/db' | grep -v '/database$$')

# The files of the module that talk to the database and can only be measured
# against a real one.
#
# The coverage gate of a database-bound module is split in two: the unit gate
# measures what can be asserted without a database, and the container tests
# measure the rest. The files below belong to the second half: their queries
# run against PostgreSQL, and the container tests are what prove them.
COVER_EXCLUDE_FILES ?= packages/database/postgres/repository\.go

.DEFAULT_GOAL := help

.PHONY: help deps tidy tidy-check fmt fmt-check vet lint build test \
        test-integration cover race bench perf vuln all generate \
        generate-check clean

## help: show this list of targets
help:
	@echo "PACA database — developer targets"
	@echo ""
	@echo "  make help              show this list of targets"
	@echo "  make deps              download the dependencies"
	@echo "  make tidy              tidy the go.mod of the module"
	@echo "  make tidy-check        fail when the go.mod is not tidy"
	@echo "  make fmt               format the sources"
	@echo "  make fmt-check         fail when a source is not formatted"
	@echo "  make vet               run the analyzers of the toolchain"
	@echo "  make lint              run golangci-lint"
	@echo "  make build             build every package of the module"
	@echo "  make test              run the test suite"
	@echo "  make test-integration  run the container tests (Docker)"
	@echo "  make cover             run the test suite with coverage and check the threshold"
	@echo "  make race              run the test suite with the race detector"
	@echo "  make bench             run the benchmarks"
	@echo "  make perf              check the allocation budgets and run the benchmarks"
	@echo "  make vuln              scan the dependencies for known vulnerabilities"
	@echo "  make generate          generate the code from the schema (sqlc)"
	@echo "  make generate-check    fail when the generated code is not the one the schema describes"
	@echo "  make all               the full quality gate"
	@echo "  make clean             remove the coverage artifacts"
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

## build: build every package of the module
build:
	$(GO) build $(PACKAGE)
	@echo "built $(PACKAGE)"

## test: run the test suite
test:
	$(GO) test $(PACKAGE)

## test-integration: run the container tests
#
# The tests start a PostgreSQL with the schema of the migrations and assert
# the rules of the tables against it, so they need Docker. They cover the
# packages that talk to the database, and they report their coverage.
#
# The coverage is measured with -coverpkg, so the coverage of the adapter and
# of the domain comes from the container tests and not from the unit tests:
# the two numbers together are the coverage of the module.
test-integration:
	@mkdir -p $(COV_DIR)
	$(GO) test -tags=integration -timeout=15m -v -covermode=atomic \
		-coverpkg=./postgres/... \
		-coverprofile=$(COV_FILE_INTEGRATION) $(PACKAGE)
	@$(GO) tool cover -func=$(COV_FILE_INTEGRATION) | tail -1

## cover: run the test suite with coverage and check the threshold
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
	$(GO) test -race $(PACKAGE)

## bench: run the benchmarks
bench:
	$(GO) test -run '^$$' -bench . -benchmem $(PACKAGE)

## perf: check the allocation budgets and run the benchmarks
perf:
	$(GO) test -run 'Budget' -v $(PACKAGE)
	$(GO) test -run '^$$' -bench . -benchmem $(PACKAGE)

## vuln: scan the dependencies for known vulnerabilities
vuln:
	$(GO) tool govulncheck $(PACKAGE)

## all: the full quality gate
all: fmt-check generate-check vet lint build test race cover vuln

## generate: generate the code from the schema
#
# The schema of the tables is the source of truth, and the generated code is
# never edited by hand. The generator is pinned in tools/go.mod.
generate:
	$(GO) tool sqlc generate

## generate-check: fail when the generated code is stale
#
# The gate regenerates the code and fails when it differs from the one that
# is committed, so a change of the schema never reaches the repository
# without the code that goes with it.
generate-check:
	@$(GO) tool sqlc generate
	@if ! git diff --quiet -- internal/db; then \
		echo "the generated code is not the one the schema describes, run 'make generate'"; \
		git diff -- internal/db; \
		exit 1; \
	fi

## clean: remove the coverage artifacts
clean:
	@rm -rf $(COV_DIR)
	@echo "cleaned"
