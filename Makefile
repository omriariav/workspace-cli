BINARY := gws
PKG := ./cmd/gws
BUILD_DIR := ./bin

.PHONY: build test lint vet fmt tidy clean run help

## Build

build: ## Build the binary to ./bin/gws
	go build -o $(BUILD_DIR)/$(BINARY) $(PKG)

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
