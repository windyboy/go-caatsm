# go-caatsm (Civil Aviation Authority Telegram Message Processor)

## Project Overview

`go-caatsm` is a high-performance Go application designed to process aviation telegrams (like FPL, ARR, DEP) and weather reports (METAR, SPECI, TAF) from NATS JetStream, parse them, persist them to PostgreSQL/TimescaleDB, and republish the parsed results. It follows Clean Architecture principles to ensure modularity and testability.

## Architecture

The project is structured using Clean Architecture:

*   **`cmd/`**: Application entry points. `cmd/main` is the primary service, `cmd/seed-telegrams` is a utility for generating test data.
*   **`internal/domain/`**: Core business logic and types (e.g., `aviation.go`, `fpl.go`, `weather/`). Pure Go, no dependencies on outer layers.
*   **`internal/app/`**: Application business rules (use cases). `MessageProcessor` orchestrates the flow between ports.
*   **`internal/adapter/`**: Adapters for external interfaces.
    *   `parser/`: Logic to parse raw telegram text into domain objects.
      *   `aviation/`: Aviation telegram parser (ARR, DEP, CNL, DLA, FPL)
      *   `weather/`: Weather report parser (METAR, SPECI, TAF)
      *   `composite.go`: Composite parser that routes messages to appropriate parser
    *   `validator/`: AFTN protocol validation.
    *   `dto/`: Data Transfer Objects.
*   **`internal/infra/`**: Infrastructure implementations.
    *   `nats/`: NATS JetStream consumer and publisher.
    *   `postgres/`: Database repository using `pgx`.
    *   `config/`, `log/`, `telemetry/`, `monitoring/`: Cross-cutting concerns.
*   **`internal/port/`**: Interfaces defining the contracts for repositories, publishers, and parsers.
*   **`pkg/di/`**: Dependency Injection using Google Wire.

## Tech Stack

*   **Language:** Go 1.24+
*   **Messaging:** NATS JetStream
*   **Database:** PostgreSQL (with TimescaleDB extension for time-series data)
*   **Observability:** OpenTelemetry (OTLP), Prometheus, Jaeger, Grafana, Zap Logger
*   **CLI:** `urfave/cli`
*   **DI:** Google Wire
*   **Testing:** Ginkgo (BDD), Gomega, Testcontainers (integration tests)

## Key Commands (Taskfile)

The project uses `Taskfile.yml` for managing common tasks.

*   **Build:** `task build` (Output: `bin/receiver`)
*   **Run (Dev):** `task run-dev` (Connects to local Docker stack)
*   **Run (Prod):** `task run-prod`
*   **Test (Unit):** `task test`
*   **Test (Integration):** `task test-int` (Requires Docker)
*   **Lint:** `task lint`
*   **Start Infrastructure:** `task up` (Starts Postgres, NATS, Observability stack)
*   **Stop Infrastructure:** `task down`
*   **Seed Data:** `task seed` (Injects sample telegrams into NATS)

## Configuration

Configuration is managed via TOML files in `configs/` and environment variables.
*   `configs/config.dev.toml`: Default for development (`GO_ENV=dev`).
*   `configs/config.prod.toml`: Production settings (`GO_ENV=prod`).
*   Environment Variables: Prefix `CAATSM_` (e.g., `CAATSM_NATS_URL`, `CAATSM_POSTGRES_URL`).

## Development Workflow

1.  **Start Infrastructure:**
    ```bash
    task up
    ```
2.  **Run Service Locally:**
    ```bash
    task run-dev
    ```
3.  **Generate Traffic:**
    ```bash
    task seed
    # OR for continuous traffic
    task seed-slow
    ```
4.  **Observe:**
    *   Grafana: http://localhost:3000 (admin/admin)
    *   Jaeger: http://localhost:16686
    *   Prometheus: http://localhost:9090

## Key Files & Directories

*   `cmd/main/main.go`: Application entry point. Sets up config, DI, and starts the listener.
*   `internal/app/processor.go`: `MessageProcessor` - The core orchestration logic.
*   `internal/adapter/parser/`: Contains parsers for aviation telegrams and weather reports (composite pattern).
*   `internal/infra/nats/consumer.go`: JetStream consumer implementation.
*   `internal/infra/postgres/telegrams.ddl`: Database schema.
*   `docs/`: Extensive documentation (Architecture, NATS, Dev Guide).

## Notes for AI Agent

*   **Conventions:** Follow existing patterns in `internal/`. Use `internal/port` for interfaces.
*   **Testing:** New features must include Ginkgo tests. Integration tests should be added for infrastructure components.
*   **DI:** If adding new components, update `pkg/di/wire.go` and run `task wire` (or `task generate`).
*   **Safety:** Always check `go.mod` before adding imports.
