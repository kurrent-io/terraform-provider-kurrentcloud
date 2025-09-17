default: help

SHELL := bash

.PHONY: build
build:  ## Builds the app
	go build

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
	go build
	./scripts/generate-docs.sh
	go fmt
	git diff --exit-code  || (echo 'missing commits - were generated docs checked in?' && exit 1)

.PHONY: install
install: ## Install the binary to the default target path
	go install

.PHONY: help
help: ## Display this information. Default target.
	@echo "Valid targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
