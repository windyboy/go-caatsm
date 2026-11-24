# go-caatsm (Civil Aviation Authority Telegram Message Processor)

## Project Overview

`go-caatsm` is a Go application designed to process aviation telegrams (like FPL, ARR, DEP, CNL, DLA) and weather reports (METAR, SPECI, TAF) from NATS JetStream, parse them, persist them to PostgreSQL/TimescaleDB, and republish the parsed results.

> **Architecture Reference:** See [`docs/architecture.md`](docs/architecture.md) for the authoritative 3-layer Go-idiomatic high-cohesion architecture specification.

## Architecture

3-layer Go-idiomatic package layout:

*   **`cmd/`**: Application entry points.
    *   `cmd/receiver`: primary service binary (`bin/receiver`), CLI (`listen`), explicit dependency wiring, and graceful shutdown.
    *   `cmd/seed-telegrams`: utility for generating synthetic aviation telegrams.
*   **`internal/telegram/`**: Pure zero-IO core domain, parsing, and validation layer:
    *   `message.go`, `router.go` (`Router` composite dispatcher), `aftn.go` (`AFTNValidator`).
    *   `aviation/`: ARR, DEP, CNL, DLA, FPL domain models, ICAO validation, and parsers.
    *   `weather/`: METAR, SPECI, TAF domain models, lexer, classifier, and parsers.
*   **`internal/pipeline/`**: Core orchestration layer (`Processor` in `processor.go`), consumer-defined small interfaces (`ports.go`), and permanent/transient error classification (`errors.go`).
*   **`internal/nats/`**: NATS JetStream consumer, publisher, monitor, connection/auth helpers, and DLQ handler (inbound/outbound/DLQ 3-stream topology, `ShouldAutoProvision()` prod guard, OTel TraceContext propagation).
*   **`internal/postgres/`**: PostgreSQL/TimescaleDB repository using `pgx` (`InsertOne`, `InsertBatch`, `InsertRaw`), internal 17-column row mapper, and `telegrams.ddl`.
*   **`internal/config/`, `internal/observability/`, `internal/buildinfo/`**: Koanf TOML/env config (`CAATSM_*`), Zap logger, consolidated OpenTelemetry SDK, Prometheus metrics + HTTP health server, and build metadata.

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

*   `cmd/receiver/main.go`: Application entry point. Sets up config, observability, explicit wiring, and starts the JetStream consumer.
*   `internal/pipeline/processor.go`: `Processor` - The core message orchestration logic.
*   `internal/telegram/`: Pure zero-IO domain models, composite `Router`, `AFTNValidator`, plus `aviation/` and `weather/` parsers.
*   `internal/nats/consumer.go`: JetStream consumer implementation (inbound/outbound/DLQ 3-stream topology, `ShouldAutoProvision()` prod guard).
*   `internal/postgres/telegrams.ddl`: Database schema.
*   `docs/`: Extensive documentation (`architecture.md`, Architecture HA, NATS, Dev Guide, Observability).

## Notes for AI Agent

*   **Architecture Direction:** Follow the 3-layer Go-idiomatic architecture in [`docs/architecture.md`](docs/architecture.md) (pure zero-IO core domain/parsers in `internal/telegram`, consumer-defined small interfaces in `internal/pipeline` and `internal/nats`, explicit wiring in `cmd/receiver`, enforced by `.golangci.yml` `depguard` rules).
*   **Testing:** New features must include tests (Ginkgo BDD for unit tests; standard `testing` with `-tags=integration` in `test/integration/`).
*   **Safety:** Always check `go.mod` before adding imports. Keep `internal/telegram/...` strictly stdlib-only.
