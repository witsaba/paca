# PACA — repository-wide developer workflow.
#
# The repository is a Go workspace (see go.work). Every module of the tree is
# built, tested, and scanned by the targets below, so a new application or a
# new shared package is picked up without editing this file.
#
# The toolchain is pinned in the repository: the Go version in go.work, and
# golangci-lint and govulncheck as tool dependencies in tools/go.mod. A
# contributor needs nothing but the Go toolchain installed — `go tool` runs
# the pinned linter and scanner.
#
# Run `make` for the list of targets.

GO       ?= go
GOFMT    ?= $(shell $(GO) env GOROOT)/bin/gofmt
BIN_DIR  ?= $(CURDIR)/bin
COV_DIR  ?= $(CURDIR)/.coverage
COV_FILE ?= $(CURDIR)/coverage.out
# The coverage floor of a file of the repository, in percent. The house rule
# of the repository is 75% per file, measured statement-weighted.
COVERAGE_FLOOR ?= 75
CLI_DIR  ?= apps/cli
CLI_BIN  ?= $(BIN_DIR)/paca
MCP_DIR  ?= apps/mcp
MCP_BIN  ?= $(BIN_DIR)/paca-mcp

# Every module of the workspace, discovered from the go.mod files.
MODULES := $(shell find . -maxdepth 3 -name go.mod \
	-not -path './.git/*' \
	-not -path '*/node_modules/*' \
	-exec dirname {} \; | sort)

# Every Go file of the repository, ignoring vendored and generated trees.
GO_FILES = $(shell find . -name '*.go' \
	-not -path './.git/*' \
	-not -path '*/vendor/*' \
	-not -path '*/node_modules/*')

.DEFAULT_GOAL := help

# Run a go command inside every module of the workspace. $(1) is the command.
define in_modules
	@set -e; for module in $(MODULES); do \
		echo "==> $$module: $(1)"; \
		(cd $$module && $(1)) || exit 1; \
	done
endef

.PHONY: help deps tidy build test cover cover-check race vuln lint vet fmt fmt-check all run clean

## help: show this list of targets
help:
	@echo "PACA — developer targets"
	@echo ""
	@echo "  make deps       download the dependencies of every module"
	@echo "  make tidy       run go mod tidy on every module"
	@echo "  make build      compile every module and write the binaries to bin/"
	@echo "  make test       run the test suite"
	@echo "  make cover      run the test suite with coverage and report it"
	@echo "  make cover-check run the test suite and fail when a file is below the floor"
	@echo "  make race       run the test suite with the race detector"
	@echo "  make vuln       scan the dependencies for known vulnerabilities"
	@echo "  make vet        run the Go vet analyzers over every module"
	@echo "  make lint       run golangci-lint over every module"
	@echo "  make fmt        format the Go sources"
	@echo "  make fmt-check  fail when a Go source is not formatted"
	@echo "  make all        the full quality gate: fmt-check, vet, lint, build, test, race, vuln"
	@echo "  make run        run the paca CLI (make run ARGS=\"--help\")"
	@echo "  make clean      remove the build and coverage artifacts"
	@echo ""
	@echo "Modules: $(MODULES)"

## deps: download the dependencies of every module
deps:
	$(call in_modules,$(GO) mod download)

## tidy: run go mod tidy on every module
tidy:
	$(call in_modules,$(GO) mod tidy)

## build: compile every module and write the binaries to bin/
build:
	$(call in_modules,$(GO) build ./...)
	@mkdir -p $(BIN_DIR)
	@echo "==> $(CLI_DIR): building paca into $(CLI_BIN)"
	@cd $(CLI_DIR) && $(GO) build -o $(CLI_BIN) ./cmd/paca
	@echo "==> $(MCP_DIR): building paca-mcp into $(MCP_BIN)"
	@cd $(MCP_DIR) && $(GO) build -o $(MCP_BIN) ./cmd/paca-mcp

## test: run the test suite
test:
	$(call in_modules,$(GO) test ./...)

## cover: run the test suite with coverage and report it
cover:
	@mkdir -p $(COV_DIR)
	@set -e; for module in $(MODULES); do \
		name=$$(echo $$module | sed -e 's|^\./||' -e 's|/|_|g'); \
		echo "==> $$module: coverage"; \
		(cd $$module && $(GO) test -covermode=atomic -coverprofile=$(COV_DIR)/$$name.out ./...) || exit 1; \
	done
	@ls $(COV_DIR)/*.out >/dev/null 2>&1 && \
		cat $(COV_DIR)/*.out | awk 'NR == 1 || !/^mode:/' > $(COV_FILE) || true
	@$(GO) tool cover -func=$(COV_FILE)

## cover-check: fail when a file of the repository is below the coverage floor
cover-check: cover
	@awk -v floor=$(COVERAGE_FLOOR) 'NR > 1 { \
		file = $$1; \
		sub(/:[0-9].*$$/, "", file); \
		total[file] += $$2; \
		if ($$3 > 0) { covered[file] += $$2 } \
	} \
	END { \
		failed = 0; \
		for (file in total) { \
			percent = 100 * covered[file] / total[file]; \
			if (percent + 0 < floor + 0) { \
				printf "  %s: %.1f%% (the floor is %s%%)\n", file, percent, floor; \
				failed = 1; \
			} \
		} \
		if (failed) { \
			print "a file of the repository is below the coverage floor"; \
			exit 1; \
		} \
		printf "every file of the repository is at or above %s%% of coverage\n", floor; \
	}' $(COV_FILE)

## race: run the test suite with the race detector
race:
	$(call in_modules,$(GO) test -race ./...)

## vuln: scan the dependencies for known vulnerabilities
vuln:
	$(call in_modules,$(GO) tool govulncheck ./...)

## vet: run the Go vet analyzers over every module
#
# `go vet` is the toolchain's own analyzer. The linter also runs the govet
# checks, so this target is not a second opinion on the same rules: it is the
# check that needs nothing but the Go toolchain, and the fastest one in the
# gate. It runs before the linter for that reason.
vet:
	$(call in_modules,$(GO) vet ./...)

## lint: run golangci-lint over every module
lint:
	$(call in_modules,$(GO) tool golangci-lint run ./...)

## fmt: format the Go sources
fmt:
	@$(GOFMT) -w $(GO_FILES)

## fmt-check: fail when a Go source is not formatted
fmt-check:
	@unformatted=$$($(GOFMT) -l $(GO_FILES)); \
	if [ -n "$$unformatted" ]; then \
		echo "these files are not gofmt-formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

## all: the full quality gate
all: fmt-check vet lint build test race vuln

## run: run the paca CLI
run:
	@cd $(CLI_DIR) && $(GO) run ./cmd/paca $(ARGS)

## clean: remove the build and coverage artifacts
clean:
	@rm -rf $(BIN_DIR) $(COV_DIR) $(COV_FILE)
	@echo "cleaned"
