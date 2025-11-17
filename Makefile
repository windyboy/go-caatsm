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
run-dev: build ## Run the receiver in development mode (uses core NATS mode by default). Use MONITORING_ADDR=ip:port to set monitoring address
	@echo "Running receiver in development mode (NATS mode: core by default)..."
	@if [ -n "$(MONITORING_ADDR)" ]; then \
		echo "Monitoring address: $(MONITORING_ADDR)"; \
		GO_ENV=dev $(BINARY) listen --monitoring-addr $(MONITORING_ADDR); \
	else \
		GO_ENV=dev $(BINARY) listen; \
	fi

.PHONY: run-prod
run-prod: build ## Run the receiver in production mode (requires config.prod.toml and JetStream mode)
	@echo "Running receiver in production mode..."
	@echo "Note: Production mode requires:"
	@echo "  - configs/config.prod.toml file (or CAATSM_* environment variables)"
	@echo "  - NATS JetStream enabled (nats.mode = jetstream)"
	@echo "  - Stream and Consumer must exist (not auto-created in prod)"
	@echo "  - PostgreSQL connection configured"
	@GO_ENV=prod $(BINARY) listen

.PHONY: run-test
run-test: build ## Run the receiver in test mode
	@echo "Running receiver in test mode..."
	@GO_ENV=test $(BINARY) listen

.PHONY: run-local
run-local: ## Run receiver directly via go run (uses core NATS mode by default in dev). Use MONITORING_ADDR=ip:port to set monitoring address
	@echo "Running receiver via go run (GO_ENV=$(GO_ENV), NATS mode: core by default in dev)..."
	@if [ -n "$(MONITORING_ADDR)" ]; then \
		echo "Monitoring address: $(MONITORING_ADDR)"; \
		GO_ENV=$(GO_ENV) go run $(CMD) listen --monitoring-addr $(MONITORING_ADDR); \
	else \
		GO_ENV=$(GO_ENV) go run $(CMD) listen; \
	fi

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

.PHONY: wire
wire: ## Generate wire dependency injection code
	@command -v wire >/dev/null || (echo "Please install wire (go install github.com/google/wire/cmd/wire@latest)"; exit 1)
	@echo "Generating wire code..."
	@wire ./pkg/di

.PHONY: generate
generate: wire ## Generate all code (wire, etc.)
	@echo "Code generation complete"

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
