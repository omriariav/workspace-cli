BINARY := gws
PKG := ./cmd/gws
BUILD_DIR := ./bin

# Version info
VERSION ?= 0.9.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/omriariav/workspace-cli/cmd.Version=$(VERSION) \
	-X github.com/omriariav/workspace-cli/cmd.Commit=$(COMMIT) \
	-X github.com/omriariav/workspace-cli/cmd.BuildDate=$(BUILD_DATE)"

.PHONY: build test lint vet fmt tidy clean run help install

## Build

build: ## Build the binary to ./bin/gws
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(PKG)

install: ## Install gws to $GOPATH/bin
	go install $(LDFLAGS) $(PKG)

run: ## Run gws with arguments (e.g., make run ARGS="gmail list --max 5")
	go run $(PKG) $(ARGS)

## Quality

test: ## Run unit tests
	go test ./...

test-race: ## Run tests with race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

lint: vet ## Run linters (go vet)
	@echo "Lint passed"

fmt: ## Format code
	gofmt -s -w .

## Maintenance

tidy: ## Tidy go modules
	go mod tidy

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
