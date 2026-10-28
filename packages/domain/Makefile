# The domain of the PACA monorepository.
#
# The module holds the models of the domain and the ports of the storage:
# what a service needs to know about a person, and what it expects from the
# place that keeps the people.
#
# It imports nothing but the standard library, so the dependency rule of the
# hexagonal architecture is enforced by the compiler: a model that reached
# for a driver would break the build of every service.
#
# Every target of the project is here: the dependencies, the build, the
# tests, the coverage gate, the race detector, the benchmarks and the
# vulnerability scan.
#
# The tools are pinned in the repository (tools/go.mod), so `go tool` runs
# the same version on every machine and in every container. A contributor needs
# nothing but the Go toolchain.
#
# Run `make` for the list of targets.

GO      ?= go
COV_DIR ?= $(CURDIR)/.coverage
COV_FILE ?= $(COV_DIR)/coverage.out
PACKAGE ?= ./...

# The coverage the module must reach, in percent. The gate fails below it.
COVERAGE_THRESHOLD ?= 80

# The packages the coverage gate measures: every package of the module.
#
# The module talks to no database, so the whole suite runs without one and
# the gate measures it as a whole.
COVER_PKGS ?= $(shell $(GO) list ./...)

.DEFAULT_GOAL := help

.PHONY: help deps tidy tidy-check fmt fmt-check vet lint build test \
        cover race bench perf vuln all clean

## help: show this list of targets
help:
	@echo "PACA domain — developer targets"
	@echo ""
	@echo "  make help              show this list of targets"
	@echo "  make deps              download the dependencies"
	@echo "  make tidy              tidy the go.mod of the module"
	@echo "  make tidy-check        fail when the go.mod is not tidy"
	@echo "  make fmt               format the sources"
	@echo "  make fmt-check         fail when a source is not gofmt-formatted"
	@echo "  make vet               run the analyzers of the toolchain"
	@echo "  make lint              run golangci-lint"
	@echo "  make build             build every package of the module"
	@echo "  make test              run the test suite"
	@echo "  make cover             run the test suite with coverage and check the threshold"
	@echo "  make race              run the test suite with the race detector"
	@echo "  make bench             run the benchmarks"
	@echo "  make perf              check the allocation budgets and run the benchmarks"
	@echo "  make vuln              scan the dependencies for known vulnerabilities"
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

## cover: run the test suite with coverage and check the threshold
cover:
	@mkdir -p $(COV_DIR)
	$(GO) test -covermode=atomic -coverprofile=$(COV_FILE) $(COVER_PKGS)
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
all: fmt-check vet lint build test race cover vuln

## clean: remove the coverage artifacts
clean:
	@rm -rf $(COV_DIR)
	@echo "cleaned"
