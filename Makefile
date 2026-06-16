BIN_DIR := bin
GOLANGCI_VERSION := 2.10.1
GOTESTSUM_VERSION := v1.13.0
GORELEASER_VERSION := v2.16.0

LINTER := $(BIN_DIR)/golangci-lint
TESTSUM := $(BIN_DIR)/gotestsum
GORELEASER := $(BIN_DIR)/goreleaser
KONFI_VERSION_PKG := github.com/getkonfi/konfi/setup/cst
KONFI_TAG_VERSION := $(shell git tag --sort=-version:refname 2>/dev/null | sed -n '1{s/^v//;p;}')
KONFI_BASE_VERSION := $(if $(KONFI_TAG_VERSION),$(KONFI_TAG_VERSION),0.0.0)
KONFI_DEV_SUFFIX := $(shell \
	if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then \
		if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null || [ -n "$$(git ls-files --others --exclude-standard)" ]; then \
			printf '%s' '-dev'; \
		fi; \
	fi)
KONFI_VERSION := $(KONFI_BASE_VERSION)$(KONFI_DEV_SUFFIX)
KONFI_LDFLAGS := -w -s -X $(KONFI_VERSION_PKG).AppVersion=$(KONFI_VERSION)

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
	@cd src && go run -ldflags="$(KONFI_LDFLAGS)" .

build: ## build binary
	@cd src && CGO_ENABLED=0 go build -ldflags="$(KONFI_LDFLAGS)" -o ../konfi .

goreleaser-tools: ## install goreleaser into bin/
	@mkdir -p $(BIN_DIR)
	@if [ -f $(GORELEASER) ] && $(GORELEASER) --version | grep -q "$(GORELEASER_VERSION)"; then \
		printf "✅ "; \
		$(GORELEASER) --version; \
	else \
		echo "Installing goreleaser $(GORELEASER_VERSION)..."; \
		GOBIN=$(PWD)/$(BIN_DIR) go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION); \
	fi

goreleaser-check: goreleaser-tools ## validate goreleaser config
	@$(GORELEASER) check

release-snapshot: goreleaser-tools ## build local release artifacts into dist/
	@$(GORELEASER) release --snapshot --clean

test: ## clean cache and run all tests with gotestsum
	@cd src && go clean -testcache
	@cd tools/schema_verify && go clean -testcache
	@cd tools/release_check && go clean -testcache
	@cd src && ../$(TESTSUM) --format pkgname --format-hide-empty-pkg --no-summary=skipped -- -race -v -timeout 20s ./...
	@cd tools/schema_verify && ../../$(TESTSUM) --format pkgname --format-hide-empty-pkg --no-summary=skipped -- -race -v -timeout 20s ./...
	@cd tools/release_check && ../../$(TESTSUM) --format pkgname --format-hide-empty-pkg --no-summary=skipped -- -race -v -timeout 20s ./...

lint: ## run golangci-lint
	@cd src && ../$(LINTER) run ./...
	@cd tools/schema_verify && ../../$(LINTER) run ./...
	@cd tools/release_check && ../../$(LINTER) run ./...

schema-verify: ## full schema verification (network + introspection)
	@cd tools/schema_verify && go run .

schema-check: ## quick schema check (offline, no exec)
	@cd tools/schema_verify && go run . --offline --no-exec --strict

release-check: ## check schema support against latest app releases
	@cd tools/release_check && go run .

release-field-check: ## check whether newer app releases add config fields
	@cd tools/release_check && go run . -fields

e2e: ## run Arch container parser/editing e2e suite
	@e2e/arch-container/run.sh

clean: ## remove build artifacts
	rm -rf dist
	rm -f konfi

.PHONY: help tools run build goreleaser-tools goreleaser-check release-snapshot test lint clean schema-verify schema-check release-check release-field-check e2e
