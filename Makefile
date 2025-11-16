# Build variables
BUILD_DIR ?= bin
BINARY := $(BUILD_DIR)/receiver
CMD := ./cmd/main
GO_ENV ?= dev

# Default target
.PHONY: all
all: build ## Build the application

.PHONY: build
build: ## Build the receiver binary
	@mkdir -p $(BUILD_DIR)
	@echo "Building receiver..."
	@go build -o $(BINARY) $(CMD)

.PHONY: run
run: run-dev ## Alias for run-dev

.PHONY: run-dev
run-dev: build ## Run the receiver in development mode
	@echo "Running receiver in development mode..."
	@GO_ENV=dev $(BINARY) listen

.PHONY: run-prod
run-prod: build ## Run the receiver in production mode
	@echo "Running receiver in production mode..."
	@GO_ENV=prod $(BINARY) listen

.PHONY: run-test
run-test: build ## Run the receiver in test mode
	@echo "Running receiver in test mode..."
	@GO_ENV=test $(BINARY) listen

.PHONY: run-local
run-local: ## Run receiver directly via go run
	@echo "Running receiver via go run..."
	@GO_ENV=$(GO_ENV) go run $(CMD) listen

.PHONY: test
test: ## Run unit tests (Ginkgo, verbose)
	@command -v ginkgo >/dev/null || (echo "Please install ginkgo (go install github.com/onsi/ginkgo/v2/ginkgo@latest)"; exit 1)
	@echo "Running Ginkgo unit test suites (verbose)..."
	@ginkgo -r -v ./cmd ./internal

.PHONY: test-int
test-int: ## Run integration tests (requires Docker)
	@echo "Running integration tests..."
	@GO_ENV=$(GO_ENV) go test -tags=integration ./test/integration/...

.PHONY: test-ginkgo
test-ginkgo: test ## Alias for test (Ginkgo)

.PHONY: test-all
test-all: ## Run unit tests (Ginkgo) and integration tests
	@$(MAKE) test
	@$(MAKE) test-int

.PHONY: coverage
coverage: ## Run coverage and generate report
	@mkdir -p coverage
	@echo "Generating coverage report..."
	@go test ./... -coverprofile=coverage/coverage.out
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

.PHONY: deps
deps: ## Sync go.mod / go.sum
	@echo "Tidying go modules..."
	@go mod tidy

.PHONY: lint
lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null || (echo "Please install golangci-lint (https://golangci-lint.run/)"; exit 1)
	@echo "Linting code..."
	@golangci-lint run ./...

.PHONY: clean
clean: ## Clean build artifacts and coverage files
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) coverage

.PHONY: help
help: ## Show this help
	@printf "Makefile targets:\n"
	@grep -E '^[a-zA-Z0-9_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*##"} {printf "  %-15s %s\n", $$1, $$2}'
