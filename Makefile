# Variables
APP_NAME = tele-proc
GO_FILES = $(shell find . -name '*.go' -type f)
CONFIG_DIR = configs
BUILD_DIR = build
MAIN_RECEIVER = ./cmd/receiver/main.go

# Default target
.PHONY: all
all: build

# Build the receiver application
.PHONY: build
build: build-receiver

.PHONY: build-receiver
build-receiver:
	@echo "Building receiver..."
	@go build -o $(BUILD_DIR)/receiver $(MAIN_RECEIVER)

# Run the receiver application with different configurations
.PHONY: run
run: run-dev

.PHONY: run-dev
run-dev:
	@echo "Running receiver in development mode..."
	@GO_ENV=development $(BUILD_DIR)/receiver &

.PHONY: run-prod
run-prod:
	@echo "Running receiver in production mode..."
	@GO_ENV=production $(BUILD_DIR)/receiver &

.PHONY: run-test
run-test:
	@echo "Running receiver in test mode..."
	@GO_ENV=test $(BUILD_DIR)/receiver &

# Test the application
.PHONY: test
test:
	@echo "Running tests..."
	@go test ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

# Format the code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod tidy

# Lint the code
.PHONY: lint
lint:
	@echo "Linting code..."
	@golangci-lint run

# Help
.PHONY: help
help:
	@echo "Makefile usage:"
	@echo "  make build          - Build the application"
	@echo "  make run            - Run the receiver in development mode"
	@echo "  make run-dev        - Run the receiver in development mode"
	@echo "  make run-prod       - Run the receiver in production mode"
	@echo "  make run-test       - Run the receiver in test mode"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make fmt            - Format the code"
	@echo "  make deps           - Install dependencies"
	@echo "  make lint           - Lint the code"
	@echo "  make help           - Show this help message"
