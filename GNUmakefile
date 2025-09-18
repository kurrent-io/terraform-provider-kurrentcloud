default: help

SHELL := bash

# Version detection - tries git describe first, falls back to commit hash, then "dev"
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build
build:  ## Builds the app with version injection
	go build -ldflags "$(LDFLAGS)"

.PHONY: build-simple
build-simple:  ## Builds the app without version injection (for compatibility)
	go build

.PHONY: version
version:  ## Shows the detected version that will be injected
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "LDFLAGS: $(LDFLAGS)"

.PHONY: generate
generate:  ## Generates the docs
	./scripts/generate-docs.sh

.PHONY: fmt
fmt:  ## Formats the codebase. If this doesn't work, run `tools` first
	PATH="$$(go env GOPATH)/bin:$$PATH" goimports -w .
	PATH="$$(go env GOPATH)/bin:$$PATH" gofumpt -w .
	PATH="$$(go env GOPATH)/bin:$$PATH" golines --base-formatter '' -w .

.PHONY: tools
tools:  ## Installs formatting tools
	go install golang.org/x/tools/cmd/goimports@v0.1.11
	go install github.com/segmentio/golines@v0.12.2
	go install mvdan.cc/gofumpt@v0.6.0

.PHONY: ci
ci: ## Performs the same checks as ci
	go build -ldflags "$(LDFLAGS)"
	./scripts/generate-docs.sh
	go fmt
	git diff --exit-code  || (echo 'missing commits - were generated docs checked in?' && exit 1)

.PHONY: install
install: ## Install the binary to the default target path with version injection
	go install -ldflags "$(LDFLAGS)"

.PHONY: help
help: ## Display this information. Default target.
	@echo "Valid targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
