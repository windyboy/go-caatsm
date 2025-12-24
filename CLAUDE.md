# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**go-caatsm** is a Civil Aviation Authority Telegram Message Processor - a high-performance, production-ready message processing system for aviation telegrams. It uses Clean Architecture, NATS JetStream for reliable message streaming, and PostgreSQL/TimescaleDB for persistence.

The system processes ICAO-format aviation telegrams (ARR, DEP, CNL, DLA, FPL), parsing raw messages from NATS JetStream, storing them in PostgreSQL, and republishing structured JSON to downstream consumers.

## Essential Commands

### Build & Run
```bash
# Build binary (outputs to bin/receiver)
make build                # or: task build
VERSION=v1.0.0 make build # with custom version

# Development (JetStream mode, auto-creates stream/consumer)
make run-dev              # or: task run-dev
make run-local            # or: task run-local (go run, honors GO_ENV)

# Production (requires config.prod.toml, manual stream/consumer setup)
make run-prod             # or: task run-prod
```

### Testing
```bash
# Unit tests (Ginkgo, verbose)
make test                 # or: task test
ginkgo -r -v ./cmd ./internal

# Run single test by description
ginkgo -r -v --focus "Test Description" ./path/to/package

# Integration tests (requires Docker)
make test-int             # or: task test-int

# All tests
make test-all             # or: task test-all

# Coverage report (target >80%)
make coverage             # or: task coverage
# Opens coverage/coverage.html
```

### Code Quality
```bash
# Lint (golangci-lint)
make lint                 # or: task lint

# Format code
make fmt                  # or: task fmt
go fmt ./...

# Regenerate dependency injection (after modifying pkg/di/wire.go)
make wire                 # or: task wire
wire ./pkg/di
```

### Development Tools
```bash
# Install Ginkgo tooling
task install-test
go install github.com/onsi/ginkgo/v2/ginkgo@latest

# Generate sample telegrams (burst mode)
make seed COUNT=100 CATEGORY=mixed STATUS=random  # or: task seed

# Continuous slow seeding (useful for monitoring)
make seed-slow INTERVAL_MIN=2s INTERVAL_MAX=5s    # or: task seed-slow
# Press Ctrl-C to stop

# Start dev infrastructure (Docker Compose)
task up                   # Postgres, NATS, observability stack
task dev-run              # Start infra + run receiver locally
task down                 # Stop and remove containers
```

## Architecture Overview

### Clean Architecture Layers

```
internal/
├── domain/           # Pure business entities (no dependencies)
│   └── aviation.go   # ARR, DEP, CNL, DLA, FPL domain models
├── port/             # Interface contracts (Repository, Publisher)
│   ├── repository.go
│   └── publisher.go
├── app/              # Application logic (orchestration)
│   └── processor.go  # MessageProcessor - main processing pipeline
├── adapter/          # Interface implementations & data transformations
│   ├── parser/       # Aviation telegram parsers (regex-based)
│   ├── mapper/       # Domain ↔ DTO transformations
│   └── dto/          # Data Transfer Objects (ParsedTelegram, MessageStatus)
└── infra/            # Infrastructure concerns
    ├── nats/         # NATS JetStream consumer & publisher
    ├── postgres/     # PostgreSQL repository (pgx, batch inserts)
    ├── config/       # Koanf configuration loading
    ├── log/          # Zap structured logging
    ├── telemetry/    # OpenTelemetry traces/metrics
    ├── metrics/      # Prometheus metrics
    ├── monitoring/   # Health/metrics HTTP server
    └── buildinfo/    # Build metadata
```

**Dependency Flow**: `infra` → `adapter` → `app` → `domain` ← `port`
The domain layer has zero external dependencies. All layers depend on interfaces (ports), not concrete implementations.

### Key Components

1. **NATS Consumer** (`internal/infra/nats/consumer.go`)
   - Pulls messages from JetStream in batches (default: 50)
   - ACK/NAK handling with automatic retries
   - Routes to DLQ after max delivery attempts (default: 3)
   - Monitors consumer lag and pending messages

2. **MessageProcessor** (`internal/app/processor.go`)
   - Orchestrates: Parse → Persist → Publish
   - Error categorization: `parsed`, `header_error`, `body_error`, `publish_error`, `repository_error`
   - Parser failures are permanent (ACK'd, no retry)
   - Repository failures are transient (NAK'd, retry with backoff)

3. **Parser** (`internal/adapter/parser/`)
   - Regex-based parsing for ICAO telegram formats
   - Supports ARR, DEP, CNL, DLA, FPL message types
   - Returns structured domain models or error status

4. **Repository** (`internal/infra/postgres/repository.go`)
   - PostgreSQL persistence via pgx connection pool
   - Batch insert support via `COPY FROM`
   - Stores parsed telegrams in `aviation.telegrams` table

5. **Monitoring Server** (`internal/infra/monitoring/server.go`)
   - `GET /livez` - Liveness (process info, no dependencies checked)
   - `GET /readyz` - Readiness (pings Postgres & NATS, 503 on failure)
   - `GET /metrics` - Prometheus metrics endpoint
   - Default address: `:2112`

### Configuration

Uses Koanf for config loading from TOML files + environment variables:
- Config file: `configs/config.{GO_ENV}.toml` (GO_ENV defaults to `dev`)
- Env overrides: `CAATSM_` prefix (e.g., `CAATSM_NATS_URL`, `CAATSM_POSTGRES_URL`)
- Key settings:
  - `nats.mode`: Must be `"jetstream"` (only supported mode)
  - `nats.stream`: JetStream stream name (default: `TELEGRAM`)
  - `nats.consumer`: Durable consumer name (default: `telegram-consumer`)
  - `nats.consumer_rules.max_deliver`: Max retry attempts (default: 3)
  - `dlq.enabled`: Enable Dead-Letter Queue for poison messages
  - `app.batch_size`: JetStream pull batch size (default: 50)
  - `monitoring.addr`: Metrics/health server address (default: `:2112`)

### Dependency Injection

Uses Google Wire for compile-time DI:
- Wire spec: `pkg/di/wire.go`
- Generated code: `pkg/di/wire_gen.go` (NEVER edit manually)
- To add a new dependency:
  1. Create a `ProvideXxx` function in the appropriate package
  2. Add it to the `runtimeSet` in `pkg/di/wire.go`
  3. Run `make wire` or `task wire`

### Message Processing Flow

```
NATS JetStream → Consumer.Fetch(batch) → MessageProcessor.Handle(msg) →
  ├─ Parser.Parse(raw) → domain model or error status
  ├─ Repository.Insert(parsed) → Postgres aviation.telegrams
  ├─ Publisher.Publish(json) → NATS output topic
  └─ ACK (success) | NAK (transient failure) | ACK (permanent failure)
```

**Error Handling**:
- **Parser failures** (invalid format): Store raw + status in DB, ACK message (permanent)
- **Repository failures** (DB down): NAK message, JetStream redelivers (transient)
- **Publisher failures** (downstream topic): Store raw + error status, ACK message (permanent, can replay from DB)
- After `max_deliver` attempts: Route to DLQ if enabled, else discard

### Observability

Three observability surfaces:

1. **Prometheus Metrics** (`/metrics`)
   - `caatsm_messages_total{stream,consumer,result}` - Throughput & results
   - `caatsm_handle_latency_seconds_bucket{stream,consumer}` - Processing latency
   - `caatsm_retries_total{stream,consumer,reason}` - JetStream retries
   - `caatsm_db_queries_total{operation,result}` - DB activity
   - `caatsm_nats_consumer_pending_messages{stream,consumer}` - Consumer lag
   - `caatsm_dlq_messages_total{stream,consumer}` - DLQ routing

2. **OpenTelemetry** (traces + metrics)
   - Environment-based sampling: Production (1%), Staging (10%), Dev/Test (100%)
   - Semantic attributes: `messaging.system`, `db.system`, `caatsm.message.category`
   - Configure via `[telemetry]` block (endpoint, enabled, insecure)
   - Export to OTLP/HTTP collector (default: `localhost:4318`)

3. **Health Endpoints**
   - `GET /livez` - Process liveness (no dependency checks)
   - `GET /readyz` - Readiness (checks Postgres & NATS, timeout: 2s)
   - Both return JSON with build metadata + dependency status

### Testing Philosophy

- **Unit tests**: Ginkgo BDD style, table-driven, mock interfaces
  - Location: Alongside code (`*_test.go`)
  - Run with `make test` or `ginkgo -r -v ./cmd ./internal`
  - Target: >80% coverage

- **Integration tests**: Docker-based (testcontainers), full E2E flow
  - Location: `test/integration/`
  - Run with `make test-int` (requires Docker)
  - Spins up real Postgres + NATS JetStream

- **Benchmarks**: Performance-critical paths (parser, mapper, processor)
  - Run with `go test -bench=BenchmarkXxx -benchmem ./path/to/pkg`

## Important Development Notes

### Code Style (from .cursor/rules/do.mdc)
- **Clean Architecture**: Strict layer separation, dependency inversion
- **Interface-driven**: All public functions interact with interfaces, not concrete types
- **Error handling**: Always wrap errors with context (`fmt.Errorf("context: %w", err)`)
- **Context propagation**: Pass `context.Context` everywhere (deadlines, cancellations, tracing)
- **No global state**: Use constructor functions with DI
- **Resource cleanup**: Use `defer` for closing resources
- **Security**: Input validation, secure defaults, retries with exponential backoff
- **Observability**: Trace all service boundaries (HTTP, NATS, DB), structured logs (JSON)

### Never Edit Generated Files
- `pkg/di/wire_gen.go` - Wire-generated DI code
- Any `*_gen.go` files - Code generation output

### Common Gotchas

1. **JetStream-only mode**: The system only supports JetStream mode (`nats.mode = "jetstream"`). Do not attempt to use Core NATS mode.

2. **Production setup**: In production (`GO_ENV=prod`):
   - Stream and Consumer must be created manually (not auto-created)
   - Requires `configs/config.prod.toml` or `CAATSM_*` env vars
   - Enable TLS for NATS/Postgres connections

3. **Parsing failures are permanent**: Parser errors (invalid format) are ACK'd and stored with error status. They do NOT trigger JetStream retries.

4. **Repository failures are transient**: DB connection failures trigger NAK, causing JetStream to redeliver the message.

5. **Batch processing**: Default batch size is 50 messages. Tune `app.batch_size` for your workload. Larger batches improve throughput but increase latency.

6. **Consumer lag**: Monitor `caatsm_nats_consumer_pending_messages` metric. High values indicate slow processing or insufficient consumer instances.

7. **DLQ routing**: Enable DLQ (`dlq.enabled = true`) to capture poison messages after max delivery attempts. Inspect DLQ subject (`caatsm.dlq`) for failed messages.

## Additional Documentation

- `README.md` - Comprehensive user guide (setup, configuration, deployment)
- `AGENTS.md` - Quick reference for AI assistants (commands, style guide)
- `docs/prod-guide.md` - Production deployment guide
- `docs/nats.md` - NATS/JetStream configuration details
- `docs/observability.md` - Observability setup & metrics
- `docs/migrations.md` - Database migration strategy
- `docs/performance.md` - Performance tuning guidelines
