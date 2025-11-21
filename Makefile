# Build variables
BUILD_DIR ?= bin
BINARY := $(BUILD_DIR)/receiver
CMD := ./cmd/main
GO_ENV ?= dev
VERSION ?= dev

# Build info variables
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME := $(shell printf 'package main\nimport ("fmt"\n"time")\nfunc main() { fmt.Print(time.Now().UTC().Format(time.RFC3339)) }' | go run -)
LDFLAGS := -X 'caatsm/internal/infra/buildinfo.Version=$(VERSION)' \
           -X 'caatsm/internal/infra/buildinfo.Commit=$(GIT_COMMIT)' \
           -X 'caatsm/internal/infra/buildinfo.BuiltAt=$(BUILD_TIME)'

# Default target
.PHONY: all
all: build ## Build the application

.PHONY: build
build: ## Build the receiver binary
	@mkdir -p $(BUILD_DIR)
	@echo "Building receiver..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(GIT_COMMIT)"
	@echo "  Built: $(BUILD_TIME)"
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

.PHONY: run
run: run-dev ## Alias for run-dev

.PHONY: run-dev
run-dev: build ## Run the receiver in development mode (uses JetStream mode). Use MONITORING_ADDR=ip:port to set monitoring address
	@echo "Running receiver in development mode (NATS mode: JetStream)..."
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
run-local: ## Run receiver directly via go run (uses JetStream mode). Use MONITORING_ADDR=ip:port to set monitoring address
	@echo "Running receiver via go run (GO_ENV=$(GO_ENV), NATS mode: JetStream)..."
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

.PHONY: seed
seed: ## Generate sample telegrams (publishes to JetStream)
	@GO_ENV=dev \
	NATS_URL=$${CAATSM_NATS_URL:-nats://localhost:4222} \
	SUBJECT=$${CAATSM_NATS_SUBJECT:-telegram.serial} \
	COUNT=$${COUNT:-10} \
	CATEGORY=$${CATEGORY:-mixed} \
	STATUS=$${STATUS:-random} \
	bash -c ' \
		set -euo pipefail; \
		cmd=(go run ./cmd/seed-telegrams); \
		if [ -n "$$NATS_URL" ]; then \
			cmd+=("--nats-url" "$$NATS_URL"); \
		fi; \
		cmd+=("--subject" "$$SUBJECT" "--count" "$$COUNT" "--category" "$$CATEGORY" "--status" "$$STATUS"); \
		exec "$${cmd[@]}" \
	'

.PHONY: seed-slow
seed-slow: ## Continuously send telegrams slowly (until Ctrl-C). Uses JetStream by default. Configurable interval.
	@echo "Starting slow continuous telegram seeding..."
	@echo "  Mode: $${MODE:-interval}"
	@echo "  Interval: $${INTERVAL_MIN:-2s} - $${INTERVAL_MAX:-5s}"
	@echo "  Category: $${CATEGORY:-mixed}"
	@echo "  Status: $${STATUS:-random}"
	@echo "  Press Ctrl-C to stop"
	@echo ""
	@GO_ENV=dev \
	NATS_URL=$${CAATSM_NATS_URL:-nats://localhost:4222} \
	SUBJECT=$${CAATSM_NATS_SUBJECT:-telegram.serial} \
	CATEGORY=$${CATEGORY:-mixed} \
	STATUS=$${STATUS:-random} \
	MODE=$${MODE:-interval} \
	INTERVAL_MIN=$${INTERVAL_MIN:-2s} \
	INTERVAL_MAX=$${INTERVAL_MAX:-5s} \
	USE_JS=$${USE_JS:-true} \
	JS_STREAM=$${JS_STREAM:-TELEGRAM} \
	JS_SUBJECT=$${JS_SUBJECT:-} \
	bash -c ' \
		set -euo pipefail; \
		cmd=(go run ./cmd/seed-telegrams); \
		if [ -n "$$NATS_URL" ]; then \
			cmd+=("--nats-url" "$$NATS_URL"); \
		fi; \
		if [ "$$USE_JS" = "true" ]; then \
			cmd+=("--jetstream"); \
			if [ -n "$$JS_STREAM" ]; then \
				cmd+=("--stream" "$$JS_STREAM"); \
			fi; \
			if [ -n "$$JS_SUBJECT" ]; then \
				cmd+=("--js-subject" "$$JS_SUBJECT"); \
			fi; \
		fi; \
		cmd+=("--subject" "$$SUBJECT" "--count" "0" "--category" "$$CATEGORY" "--status" "$$STATUS" "--mode" "$$MODE" "--interval-min" "$$INTERVAL_MIN" "--interval-max" "$$INTERVAL_MAX"); \
		exec "$${cmd[@]}" \
	'

.PHONY: help
help: ## Show this help
	@printf "Makefile targets:\n"
	@grep -E '^[a-zA-Z0-9_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*##"} {printf "  %-15s %s\n", $$1, $$2}'
