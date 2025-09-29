.PHONY: build test lint clean install help

BINARY_NAME=gomini
BINARY_PATH=./bin/$(BINARY_NAME)
GO_FLAGS=-ldflags="-s -w"

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the gomini binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@go build $(GO_FLAGS) -o $(BINARY_PATH) ./cmd/gomini
	@echo "Binary built: $(BINARY_PATH)"

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

lint: ## Run linting tools
	@echo "Running go vet..."
	@go vet ./...
	@echo "Running staticcheck (if available)..."
	@which staticcheck > /dev/null && staticcheck ./... || echo "staticcheck not found, skipping"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

install: ## Install gomini binary
	@echo "Installing $(BINARY_NAME)..."
	@go install ./cmd/gomini

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

check: lint test ## Run all checks (lint and test)

dev: ## Build and run with example
	@$(MAKE) build
	@echo "Running gomini with test arguments..."
	@$(BINARY_PATH) run --bundle ./examples/alpine-bundle --hostname testhost --cpu 10000 --mem 134217728 --pids 64

.DEFAULT_GOAL := help