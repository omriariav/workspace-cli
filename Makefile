BINARY := gws
PKG := ./cmd/gws
BUILD_DIR := ./bin

# Version info
VERSION ?= 1.28.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/omriariav/workspace-cli/cmd.Version=$(VERSION) \
	-X github.com/omriariav/workspace-cli/cmd.Commit=$(COMMIT) \
	-X github.com/omriariav/workspace-cli/cmd.BuildDate=$(BUILD_DATE)"

.PHONY: build test lint vet fmt tidy clean run help install release-check release

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

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format code
	gofmt -s -w .

## Release

RELEASE_DIR := $(BUILD_DIR)/release
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

release-check: fmt vet test ## Pre-release quality gate
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "ERROR: working tree is dirty — commit or stash changes first"; exit 1; \
	fi
	@echo "All checks passed."

release: release-check ## Full release: build, tag, upload, verify (VERSION required)
	@if [ "$(VERSION)" = "" ]; then echo "ERROR: VERSION is required (make release VERSION=x.y.z)"; exit 1; fi
	@echo "==> Releasing v$(VERSION)..."
	git tag v$(VERSION)
	git push origin v$(VERSION)
	gh release create v$(VERSION) --title "v$(VERSION)" --notes "Release v$(VERSION)" --draft
	@echo "==> Cross-compiling..."
	@mkdir -p $(RELEASE_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		echo "  Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(RELEASE_DIR)/$(BINARY)-$$os-$$arch $(PKG); \
	done
	@echo "==> Uploading binaries..."
	gh release upload v$(VERSION) $(RELEASE_DIR)/$(BINARY)-* --clobber
	@echo "==> Verifying..."
	@if [ "$$(uname -s)-$$(uname -m)" = "Darwin-arm64" ]; then \
		$(RELEASE_DIR)/$(BINARY)-darwin-arm64 version; \
	elif [ "$$(uname -s)-$$(uname -m)" = "Linux-x86_64" ]; then \
		$(RELEASE_DIR)/$(BINARY)-linux-amd64 version; \
	fi
	@echo "==> Release v$(VERSION) complete (draft). Edit notes and publish:"
	@echo "    gh release edit v$(VERSION) --draft=false --notes \"...\""

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
