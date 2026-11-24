# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**go-caatsm** is a Civil Aviation Authority Telegram Message Processor - a message processing system for aviation telegrams and weather reports using NATS JetStream for reliable message streaming and PostgreSQL/TimescaleDB for persistence.

> **Architecture Blueprint:** [`docs/architecture.md`](docs/architecture.md) is the authoritative reference documenting the historical baseline and the implemented **3-layer Go-idiomatic architecture**.

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
# Lint (golangci-lint, enforces .golangci.yml depguard rules)
make lint                 # or: task lint

# Format code
make fmt                  # or: task fmt
go fmt ./...
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

### Package Layout

```
cmd/
├── receiver/         # Primary entrypoint (main.go), CLI (listen), explicit wiring & shutdown
└── seed-telegrams/   # Synthetic aviation telegram generator
internal/
├── telegram/         # Pure zero-IO core domain, routing & validation (stdlib only)
│   ├── message.go    # Canonical Message envelope, MessageStatus & Protocol constants
│   ├── router.go     # Composite Router dispatching weather vs aviation
│   ├── aftn.go       # AFTN protocol validator (4/8-char addr, skips weather reports)
│   ├── aviation/     # ARR, DEP, CNL, DLA, FPL models, ICAO Validate() & parsers
│   └── weather/      # METAR, SPECI, TAF models, lexer, classifier & parsers
├── pipeline/         # Core orchestration pipeline
│   ├── processor.go  # Processor - Parse -> AFTN Validate -> Persist -> Publish
│   ├── ports.go      # Consumer-defined small interfaces (Parser, Validator, Store, Publisher, Recorder)
│   └── errors.go     # Permanent / Transient error helpers
├── nats/             # NATS JetStream consumer, publisher, monitor, conn & DLQ handler
├── postgres/         # PostgreSQL/TimescaleDB repository (pgx), internal 17-col mapper & telegrams.ddl
├── config/           # Koanf TOML + CAATSM_* env configuration & validation
├── observability/    # Zap logging, consolidated OTel SDK, Prometheus metrics & HTTP health server
└── buildinfo/        # Build version, commit, and timestamp metadata
```

**Dependency Rules**: Enforced by `.golangci.yml` (`depguard`): `internal/telegram/...` depends solely on the Go standard library (and internal sibling subpackages via `router.go`); `internal/pipeline` depends only on `internal/telegram`, context/otel/zap (zero infra imports); `internal/{nats,postgres,config,observability}` implement infrastructure without importing `internal/pipeline` concrete types beyond `pipeline.Permanent`/`pipeline.IsPermanent`.

### Key Components

1. **NATS Consumer** (`internal/nats/consumer.go`)
   - Pulls messages from JetStream in batches (default: 50) with context cancellation (`nats.Context(ctx)`) and processes them sequentially via `MessageHandler.Handle`
   - Separates inbound (`nats.stream`, `WorkQueuePolicy`), outbound (`nats.publish_stream`, `LimitsPolicy`), and DLQ (`nats.dlq_stream`, `LimitsPolicy`) streams; auto-provisions in non-prod and verifies read-only in `prod` (`ShouldAutoProvision()`)
   - Routes permanent errors (`pipeline.IsPermanent`) to DLQ when enabled; falls back to NAK+backoff retry if DLQ publishing fails
   - Monitors consumer lag and pending messages (`ConsumerMonitor`)

2. **Processor** (`internal/pipeline/processor.go`)
   - Orchestrates: Parse → (optional AFTN Validate) → Persist → Publish
   - Telegram parse/validation status (`telegram.MessageStatus`): `unknown`, `parsed`, `header_error`, `body_error`, `aftn_error`
   - Parser and AFTN validation failures are permanent (stored in `aviation.telegrams_raw`, ACK'd, no retry)
   - Repository failures are transient (NAK'd, retry with backoff); publisher transient vs permanent errors are classified by the underlying publisher (`pipeline.Permanent`)

3. **Telegram Router & Parsers** (`internal/telegram/`)
   - `Router` (`internal/telegram/router.go`) combines `weather.Parser` and `aviation.Parser` (surfacing weather parse errors directly without falling back to `aviation.Parser`)
   - **Aviation Parser** (`internal/telegram/aviation/`) - ICAO telegram formats (ARR, DEP, CNL, DLA, FPL) with co-located `Validate()`
   - **Weather Parser** (`internal/telegram/weather/`) - Weather reports (METAR, SPECI, TAF)
   - Returns `*telegram.Message` (with structured domain data in `BodyData`)

4. **Repository** (`internal/postgres/repository.go`)
   - PostgreSQL persistence via `pgxpool.Pool`
   - `InsertOne` (with 7-day time-bounded deduplication on `(message_id, date_time)`), `InsertBatch` (`COPY FROM`), and `InsertRaw` (for failed/unparsed audit storage)
   - Internal 17-column row mapping in `internal/postgres/mapper.go`

5. **Monitoring Server** (`internal/observability/server.go`)
   - `GET /livez` - Liveness (process info, no dependency checks)
   - `GET /readyz` - Readiness (pings Postgres & NATS, 503 on failure)
   - `GET /metrics` - Prometheus metrics endpoint
   - Default address: `:2112`

### Configuration

Uses Koanf for config loading from TOML files + environment variables:
- Config file: `configs/config.{GO_ENV}.toml` (when present; falls back cleanly to `CAATSM_*` env vars if absent; `GO_ENV` defaults to `dev`)
- Env overrides: `CAATSM_` prefix with `__` nesting (e.g., `CAATSM_APP__BATCH_SIZE=100`, `CAATSM_MONITORING__ENABLE_METRICS=true`; top-level known prefixes like `CAATSM_NATS_URL` and `CAATSM_POSTGRES_URL` also map automatically)
- Key settings:
  - `nats.mode`: Must be `"jetstream"` (only supported mode)
  - `nats.stream` / `publish_stream` / `dlq_stream`: Separate streams (defaults: `TELEGRAM`, `TELEGRAM_OUT`, `TELEGRAM_DLQ`)
  - `nats.auto_provision`: Overrides non-prod/prod default provisioning (`prod` defaults to read-only verification)
  - `nats.consumer`: Durable consumer name (default: `telegram-consumer`)
  - `nats.consumer_rules.max_deliver`: Max retry attempts (default: 3)
  - `dlq.enabled`: Enable Dead-Letter Queue for poison messages
  - `app.batch_size`: JetStream pull batch size (default: 50)
  - `monitoring.addr`: Metrics/health server address (default: `:2112`)

### Parser Architecture

1. **Composite Router** (`internal/telegram/router.go`)
   - Inspects payload headers via `weather.Classifier` (`METAR`/`SPECI`/`TAF`) and delegates to `weather.Parser` or `aviation.Parser`
   - Surfaces parser and validation errors directly without cross-parser fallback

2. **Aviation Parser** (`internal/telegram/aviation/`)
   - Direct RE2 matching and parsing for ARR, DEP, CNL, DLA, FPL message types plus ICAO `Validate()`
   - Extracts header metadata, flight details, aircraft info, timestamps, and airports

3. **Weather Parser** (`internal/telegram/weather/`)
   - Lexer/token-based parsing for METAR, SPECI, and TAF reports
   - Classifies format and extracts weather elements (wind, visibility, clouds, phenomena, temperature, altimeter)

### Explicit Wiring & Lifecycle

Dependencies are wired explicitly in `cmd/receiver/main.go` (no code-generated DI), with reverse-order `defer` cleanup closing `pgxpool.Pool`, NATS connection, logger, OTel providers, HTTP monitor server, and `Consumer.Shutdown(ctx)`.

### Message Processing Flow

```
NATS JetStream → Consumer.Fetch(ctx, batch) → Processor.Handle(ctx, raw, msgID) →
  ├─ Router.Parse(raw) → *telegram.Message or error status
  ├─ AFTNValidator.Validate(msg) (optional, AFTN protocol only)
  ├─ Store.InsertOne(ctx, msg) → Postgres aviation.telegrams
  ├─ Publisher.Publish(ctx, msg) → NATS output stream (with OTel TraceContext)
  └─ ACK (ok / DLQ-routed permanent fail) | NAK with backoff (transient / DLQ fail)
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

### Code Style (aligned with `docs/architecture.md`, `.golangci.yml` & `.cursor/rules/do.mdc`)
- **Modular 3-Layer Architecture**: Strictly keep `internal/telegram/...` pure stdlib Go with zero external IO/logging, define minimal interfaces at consumers (`internal/pipeline/ports.go`, `internal/nats/consumer.go`), and wire explicitly in `cmd/receiver`.
- **Interface-driven**: All core pipeline collaborators interact via small consumer interfaces, not concrete types
- **Error handling**: Always wrap errors with context (`fmt.Errorf("context: %w", err)`) and use `errors.Is` / `errors.As`
- **Context propagation**: Pass `context.Context` everywhere (deadlines, cancellations, OTel tracing)
- **No global state**: Construct components explicitly in `cmd/receiver`
- **Resource cleanup**: Use `defer` for closing resources in reverse initialization order
- **Security**: Input validation, secure defaults, retries with exponential backoff, sanitized NATS URL logging
- **Observability**: Trace all service boundaries (NATS Consume -> Pipeline -> DB -> NATS Publish), structured logs (JSON)

### Never Edit Generated Files
- Any `*_gen.go` files - Code generation output

### Common Gotchas

1. **JetStream-only mode**: The system only supports JetStream mode (`nats.mode = "jetstream"`). Do not attempt to use Core NATS mode.

2. **Production setup**: In production (`GO_ENV=prod`):
   - `ShouldAutoProvision()` defaults to `false`: startup runs read-only verification of inbound/outbound/DLQ streams and the durable consumer instead of auto-creating them
   - Requires `configs/config.prod.toml` or `CAATSM_*` env vars
   - Enable TLS for NATS/Postgres connections

3. **Parsing failures are permanent**: Parser errors (invalid format) are ACK'd and stored with error status. They do NOT trigger JetStream retries.

4. **Repository failures are transient**: DB connection failures trigger NAK, causing JetStream to redeliver the message.

5. **Batch processing**: Default batch size is 50 messages. Tune `app.batch_size` for your workload. Larger batches improve throughput but increase latency.

6. **Consumer lag**: Monitor `caatsm_nats_consumer_pending_messages` metric. High values indicate slow processing or insufficient consumer instances.

7. **DLQ routing**: Enable DLQ (`dlq.enabled = true`) to capture poison messages after max delivery attempts. Inspect DLQ subject (`caatsm.dlq`) for failed messages.

## Additional Documentation

- `docs/architecture.md` - Authoritative architecture design & migration roadmap
- `README.md` - Comprehensive user guide (setup, configuration, deployment)
- `AGENTS.md` - Quick reference for AI assistants (commands, style guide)
- `docs/prod-guide.md` - Production deployment guide
- `docs/nats.md` - NATS/JetStream configuration details
- `docs/observability.md` - Observability setup & metrics
- `docs/migrations.md` - Database migration strategy
- `docs/performance.md` - Performance tuning guidelines
