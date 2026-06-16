BIN_DIR := bin
GOLANGCI_VERSION := 2.10.1
GOTESTSUM_VERSION := v1.13.0
GORELEASER_VERSION := v2.16.0

LINTER := $(BIN_DIR)/golangci-lint
TESTSUM := $(BIN_DIR)/gotestsum
GORELEASER := $(BIN_DIR)/goreleaser

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)
GOOS := linux
ifeq ($(UNAME_OS),Darwin)
	GOOS := darwin
endif

GOARCH := amd64
ifeq ($(UNAME_ARCH),aarch64)
	GOARCH := arm64
else ifeq ($(UNAME_ARCH),arm64)
	GOARCH := arm64
endif

GOLANGCI_LINT_URL := https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_VERSION)/golangci-lint-$(GOLANGCI_VERSION)-$(GOOS)-$(GOARCH).tar.gz

help: ## show help message
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[38;2;139;171;73m%-20s\033[0m %s\n", $$1, $$2}'

tools: ## install golangci-lint and gotestsum into bin/
	@mkdir -p $(BIN_DIR)
	@if [ -f $(LINTER) ] && $(LINTER) --version | grep -q "$(GOLANGCI_VERSION)"; then \
		printf "✅ "; \
		$(LINTER) --version; \
	else \
		echo "Installing golangci-lint $(GOLANGCI_VERSION) for $(GOOS)-$(GOARCH)..."; \
		curl -sSfL $(GOLANGCI_LINT_URL) | tar -xz -C $(BIN_DIR) --strip-components=1 golangci-lint-$(GOLANGCI_VERSION)-$(GOOS)-$(GOARCH)/golangci-lint; \
	fi
	@if [ -f $(TESTSUM) ]; then \
		printf "✅ "; \
		$(TESTSUM) --version; \
	else \
		echo "Installing gotestsum $(GOTESTSUM_VERSION)..."; \
		GOBIN=$(PWD)/$(BIN_DIR) go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION); \
	fi

run: ## run the TUI
	@cd src && go run .

build: ## build binary
	@cd src && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../konfi .

release-tools: ## install goreleaser into bin/
	@mkdir -p $(BIN_DIR)
	@if [ -f $(GORELEASER) ] && $(GORELEASER) --version | grep -q "$(GORELEASER_VERSION)"; then \
		printf "✅ "; \
		$(GORELEASER) --version; \
	else \
		echo "Installing goreleaser $(GORELEASER_VERSION)..."; \
		GOBIN=$(PWD)/$(BIN_DIR) go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION); \
	fi

release-check: release-tools ## validate goreleaser config
	@$(GORELEASER) check

release-snapshot: release-tools ## build local release artifacts into dist/
	@$(GORELEASER) release --snapshot --clean

test: ## clean cache and run all tests with gotestsum
	@cd src && go clean -testcache
	@cd tools/schemaverify && go clean -testcache
	@cd tools/upstreamcheck && go clean -testcache
	@cd src && ../$(TESTSUM) --format pkgname --format-hide-empty-pkg --no-summary=skipped -- -race -v -timeout 20s ./...
	@cd tools/schemaverify && ../../$(TESTSUM) --format pkgname --format-hide-empty-pkg --no-summary=skipped -- -race -v -timeout 20s ./...
	@cd tools/upstreamcheck && ../../$(TESTSUM) --format pkgname --format-hide-empty-pkg --no-summary=skipped -- -race -v -timeout 20s ./...

lint: ## run golangci-lint
	@cd src && ../$(LINTER) run ./...
	@cd tools/schemaverify && ../../$(LINTER) run ./...
	@cd tools/upstreamcheck && ../../$(LINTER) run ./...

schema-verify: ## full schema verification (network + introspection)
	@cd tools/schemaverify && go run .

schema-check: ## quick schema check (offline, no exec)
	@cd tools/schemaverify && go run . --offline --no-exec --strict

upstream-check: ## check supported app versions against upstream releases
	@cd tools/upstreamcheck && go run .

e2e: ## run Arch container parser/editing e2e suite
	@e2e/arch-container/run.sh

clean: ## remove build artifacts
	rm -rf dist
	rm -f konfi

.PHONY: help tools run build release-tools release-check release-snapshot test lint clean schema-verify schema-check upstream-check e2e
