# go-caatsm

Civil Aviation Authority Telegram Message Processor

A high-performance, production-ready message processing system for aviation telegrams using Clean Architecture, NATS JetStream, and PostgreSQL.

## Architecture

This project follows Clean Architecture principles with clear separation of concerns:

```
/cmd/main/main.go              # Application entry point
/internal
    /port                      # Port layer (interfaces/contracts)
        repository.go          # Repository interface
        publisher.go           # Publisher interface
    /domain                    # Domain models (pure Go types, no external dependencies)
        aviation.go            # Aviation domain types (FPL, DEP, ARR, etc.)
    /app                       # Application layer (business logic orchestration)
        processor.go           # Message processor
    /adapter                   # Adapter layer (implementations)
        /parser                # Message parsing adapters
        /mapper                # Data mapping (domain ↔ infrastructure)
        /dto                   # Data Transfer Objects
            telegram.go        # ParsedTelegram and MessageStatus
    /infra                     # Infrastructure layer
        /config                # Configuration management (Koanf)
        /nats                  # NATS JetStream client
        /postgres              # PostgreSQL repository (pgx)
        /log                   # Logging (Zap)
        /metrics               # Prometheus metrics
        /telemetry             # OpenTelemetry tracing
/pkg/di                        # Dependency injection (Wire)
```

## Features

- **Clean Architecture**: Clear separation between domain, application, and infrastructure layers
- **NATS JetStream**: Reliable message streaming with automatic retries and dead-letter queues
- **TimescaleDB (PostgreSQL)**: High-performance persistence using pgx with batch operations and hypertables
- **Dependency Injection**: Google Wire for compile-time dependency injection
- **Configuration Management**: Koanf for flexible configuration loading (file + environment variables)
- **Structured Logging**: Zap logger with configurable levels and formats
- **Batch Processing**: Efficient batch message processing and database inserts

## Contributor Guide

For coding standards, test expectations, and release hygiene, read [AGENTS.md](AGENTS.md).

## Prerequisites

- Go 1.22+
- PostgreSQL 12+
- NATS Server with JetStream enabled

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd go-caatsm
```

2. Install dependencies:
```bash
go mod download
```

3. Set up PostgreSQL database:
```bash
psql -U postgres -f internal/infra/postgres/telegrams.ddl
```

4. Configure the application:
   - Copy `configs/config.dev.toml` and modify as needed
   - Or set environment variables with `CAATSM_` prefix

## Configuration

Configuration is loaded from TOML files and environment variables. The configuration file should be located at `configs/config.{env}.toml` where `{env}` is determined by the `GO_ENV` environment variable (defaults to `dev`).

- `nats.url` and `nats.stream` are required (the latter defaults to `TELEGRAM` when omitted).
- `subscription.topic` is optional; when not provided the application subscribes to `telegram.>`.

### Configuration Structure

```toml
[nats]
url = "nats://localhost:4222"
# mode: "jetstream" (default) or "core"
#   See "NATS Mode Selection" section below for detailed comparison
mode = "jetstream"
stream = "TELEGRAM"
consumer = "telegram-consumer"
client = "serial-client"
cluster = "tele-cluster"

[nats.stream_limits]
# These settings only apply when mode = "jetstream"
max_msgs = 100000
max_bytes = 67108864
max_age = "24h"
discard = "old"
storage = "file"
replicas = 1

[nats.consumer_rules]
# Consumer delivery rules (only applies when mode = "jetstream")
# max_deliver: Maximum number of delivery attempts before giving up
max_deliver = 3
# ack_wait: Time to wait for ACK before redelivering message
ack_wait = "30s"
# max_ack_pending: Maximum number of unacknowledged messages before pausing delivery
max_ack_pending = 1000

[subscription]
# Optional. Defaults to "telegram.>" when omitted.
topic = "telegram.serial"
# queue_group: Used in both core and jetstream modes for load balancing
queue_group = "tele-queue"

[publisher]
topic = "telegram.json"

[postgres]
url = "postgres://user:password@localhost:5432/aviation?sslmode=disable"
max_conns = 10
min_conns = 2

[app]
batch_size = 50
batch_timeout = "2s"
monitor_interval = "30s"

[log]
level = "info"
format = "json"

[telemetry]
enabled = false
endpoint = "http://otel-collector:4318"
insecure = true

# Production configuration example:
# enabled = true
# endpoint = "otel-collector.company.com:4318"
# insecure = false  # Use TLS in production

### Timeouts and Ack Wait

`[timeouts]` is optional, but if you plan to tune JetStream redelivery you should set `timeouts.ack_wait` and/or `[nats.consumer].ack_wait`. When neither is specified the application defaults both values to `30s`, ensuring predictable redelivery timing.
```

### Environment Variables

You can override any configuration value using environment variables with the `CAATSM_` prefix:

```bash
export CAATSM_NATS_URL="nats://nats-server:4222"
export CAATSM_POSTGRES_URL="postgres://user:pass@db:5432/aviation"
export CAATSM_LOG_LEVEL="debug"
```

Environment variable names are converted from `CAATSM_NATS_URL` to `nats.url` in the configuration.

### NATS Mode Selection

The application supports two NATS consumption modes, controlled by `nats.mode`:

#### JetStream Mode (Recommended for Production)

**Configuration:** `nats.mode = "jetstream"` (default)

**Features:**
- ✅ **Message Persistence**: Messages are stored in a Stream, allowing replay and recovery
- ✅ **ACK/NAK Mechanism**: Explicit message acknowledgment ensures guaranteed delivery
- ✅ **Error Handling**: Failed messages are handled with simple backoff
- ✅ **Dead-Letter Queue**: Poison messages can be routed to a DLQ for inspection
- ✅ **Batch Processing**: Efficient batch fetching and processing
- ✅ **Consumer Monitoring**: Real-time metrics for consumer lag and pending messages
- ✅ **At-Least-Once Delivery**: Messages are guaranteed to be delivered at least once

**Use Cases:**
- Production environments requiring message reliability
- Scenarios where message loss is unacceptable
- Systems needing message replay capabilities
- Applications requiring retry logic for transient failures

**Configuration Requirements:**
- Requires JetStream to be enabled on the NATS server
- Stream must be created (auto-created in dev/test environments)
- Consumer configuration via `[nats.consumer_rules]` section

**How to Use JetStream Mode:**

1. **Prerequisites:**
   - Ensure NATS server has JetStream enabled (default in `docker-compose.dev.yml`)
   - Set `nats.mode = "jetstream"` in your config file (or use `CAATSM_NATS_MODE=jetstream`)

2. **Start NATS with JetStream:**
   ```bash
   # Using Docker Compose (recommended for development)
   docker compose -f docker-compose.dev.yml up -d nats
   
   # Or start NATS server manually with JetStream enabled:
   # nats-server -js
   ```

3. **Configure Stream and Consumer:**
   The application automatically creates the Stream and Consumer on startup in dev/test environments (`GO_ENV=dev` or `GO_ENV=test`). In production, you may need to create them manually or ensure they exist.

   **Stream Configuration** (`[nats.stream_limits]`):
   - `max_msgs`: Maximum number of messages in the stream (default: 100000)
   - `max_bytes`: Maximum total size of messages (default: 64MB)
   - `max_age`: Maximum age of messages before deletion (default: 24h)
   - `storage`: "file" (persistent) or "memory" (ephemeral)
   - `replicas`: Number of stream replicas for HA (default: 1, use 3+ for production)

   **Consumer Configuration** (`[nats.consumer_rules]`):
   - `max_deliver`: Maximum delivery attempts before giving up (default: 3)
   - `ack_wait`: Time to wait for ACK before redelivery (default: 30s)

4. **Start the Application:**
   ```bash
   # Development mode (auto-creates stream/consumer)
   GO_ENV=dev \
   CAATSM_NATS_MODE=jetstream \
   CAATSM_POSTGRES_URL=postgres://user:pass@localhost:5432/aviation \
     go run ./cmd/main listen
   
   # Production mode (requires stream/consumer to exist)
   GO_ENV=prod \
   CAATSM_NATS_MODE=jetstream \
     ./bin/receiver listen
   ```

5. **Publish Messages to JetStream:**
   ```bash
   # Using nats-box (included in docker-compose.dev.yml)
   docker compose exec nats-box nats pub telegram.serial "ZCZC TEST123 150631..."
   
   # Or use the seed-telegrams tool with JetStream
   go run ./cmd/seed-telegrams \
     --nats-url nats://localhost:4222 \
     --jetstream \
     --stream TELEGRAM \
     --js-subject telegram.serial \
     --count 10
   ```

6. **Monitor JetStream:**
   ```bash
   # View stream info
   docker compose exec nats-box nats stream info TELEGRAM
   
   # View consumer info
   docker compose exec nats-box nats consumer info TELEGRAM telegram-consumer
   
   # View pending messages
   docker compose exec nats-box nats consumer next TELEGRAM telegram-consumer
   
   # Or use NATS monitoring UI at http://localhost:8222
   ```

7. **Message Processing Flow:**
   - Messages are published to the configured subject (e.g., `telegram.serial`)
   - Stream stores messages according to retention policy
   - Consumer pulls messages in batches (configurable via `app.batch_size`)
   - Each message is processed and ACKed on success
    - Failed messages are NAKed and redelivered with simple backoff
   - After `max_deliver` attempts, permanent failures are routed to DLQ (if enabled)

8. **Replay Messages:**
   ```bash
   # Replay from a specific sequence
   ./bin/receiver listen --replay-from seq:12345
   
   # Replay from a specific time
   ./bin/receiver listen --replay-from time:2024-11-15T08:00:00Z
   ```

9. **Dead-Letter Queue (DLQ):**
   Enable DLQ in config to route poison messages:
   ```toml
   [dlq]
   enabled = true
   subject = "caatsm.dlq"
   ```
   Messages that fail after `max_deliver` attempts are published to the DLQ subject for manual inspection.

10. **Troubleshooting:**
    - **Stream not found**: Ensure `GO_ENV=dev` for auto-creation, or create manually in production
    - **Consumer not found**: Application auto-creates consumer on startup
    - **Messages not being consumed**: Check consumer info for pending messages and delivery status
    - **High pending count**: Increase `batch_size` or add more consumer instances
    - **Messages being redelivered**: Check processing logs for errors; adjust `ack_wait` if processing takes longer

#### Core NATS Mode (Default for Development)

**Configuration:** `nats.mode = "core"` (default in `config.dev.toml`)

**Features:**
- ⚡ **Simple Pub/Sub**: Basic publish/subscribe messaging
- ⚡ **Queue Groups**: Load balancing across multiple consumers
- ⚡ **Low Latency**: No persistence overhead
- ⚡ **Fast Startup**: No stream/consumer setup required
- ❌ **No Persistence**: Messages are lost if no consumer is available
- ❌ **No ACK**: No delivery guarantees
- ❌ **No Retry**: Processing failures are logged but not retried
- ❌ **No DLQ**: Failed messages cannot be routed to a dead-letter queue

**Use Cases:**
- **Local development** (recommended default)
- Quick testing and iteration
- Real-time monitoring/logging where message loss is acceptable
- Simple pub/sub scenarios without reliability requirements
- Performance testing without persistence overhead

**Configuration Requirements:**
- Works with any NATS server (JetStream not required)
- Only `[subscription]` settings are used (queue_group for load balancing)
- `[nats.consumer_rules]` and `[nats.stream_limits]` are ignored

**How to Use Core Mode:**

1. **Start NATS Server** (JetStream not required, but can be enabled):
   ```bash
   # Simple NATS server
   nats-server
   
   # Or with Docker Compose (JetStream enabled but not required for core mode)
   docker compose -f docker-compose.dev.yml up -d nats
   ```

2. **Start the Application** (Core mode is default in dev config):
   ```bash
   # Core mode is default, no need to specify
   GO_ENV=dev \
   CAATSM_POSTGRES_URL=postgres://user:pass@localhost:5432/aviation \
     go run ./cmd/main listen
   ```

3. **Publish Messages** (use standard NATS publish):
   ```bash
   # Using nats-box
   docker compose exec nats-box nats pub telegram.serial "ZCZC TEST123 150631..."
   
   # Or use seed-telegrams without --jetstream flag
   go run ./cmd/seed-telegrams \
     --nats-url nats://localhost:4222 \
     --subject telegram.serial \
     --count 10
   ```

**Switching Modes:**

```bash
# Use Core NATS mode (default for development)
CAATSM_NATS_MODE=core go run ./cmd/main listen
# Or simply (core is default in config.dev.toml)
go run ./cmd/main listen

# Use JetStream mode (for production or integration testing)
CAATSM_NATS_MODE=jetstream go run ./cmd/main listen
```

**Note:** The publisher always uses JetStream for deduplicated fan-out, regardless of the consumer mode. If you need pure Core NATS, ensure publishers also use Core NATS subjects.

## Usage

### Build

The build process automatically injects build information (version, commit, build time) into the binary. This information is available via the `/livez` and `/readyz` health endpoints.

Using Make (writes `bin/receiver`):

```bash
make build
# Or with custom version:
VERSION=v1.0.0 make build
```

Using Task:

```bash
task build
# Or with custom version:
VERSION=v1.0.0 task build
```

Or directly with Go:

```bash
# With build info injection:
go build -ldflags "-X 'caatsm/internal/infra/buildinfo.Version=dev' -X 'caatsm/internal/infra/buildinfo.Commit=$(git rev-parse --short HEAD)' -X 'caatsm/internal/infra/buildinfo.BuiltAt=$$(go run - <<'EOF'
package main
import (
  \"fmt\"
  \"time\"
)
func main() {
  fmt.Print(time.Now().UTC().Format(time.RFC3339))
}
EOF)'" -o bin/receiver ./cmd/main
```

Build information is automatically populated from:
- **Version**: `VERSION` environment variable (defaults to "dev")
- **Commit**: Git commit hash (short format)
- **BuiltAt**: UTC timestamp of build time

### Run

#### Development Mode

Use Make targets (binary mode):

```bash
make run-dev    # GO_ENV=dev (uses core NATS mode by default)
make run-local  # go run ./cmd/main listen (honors GO_ENV)
```

Task equivalents:

```bash
task run-dev
task run-local         # go run ./cmd/main listen
task dev-run           # boots docker-compose dev stack + go run (core NATS mode)
```

#### Production Mode

For production deployment, see the comprehensive guide: **[Production Deployment Guide](docs/prod-guide.md)**

Quick start:

```bash
# Build the binary with version information
VERSION=v1.0.0 make build

# Run in production mode
make run-prod   # GO_ENV=prod (requires config.prod.toml)
```

**Key requirements:**
- JetStream mode (mandatory)
- Stream and Consumer must be created manually
- Production configuration file: `configs/config.prod.toml`
- SSL/TLS for secure connections
- NATS authentication configured (see `docs/prod-guide.md#nats-authentication`)

See `docs/prod-guide.md` for complete production deployment instructions, including:
- NATS authentication setup
- Database migrations (see `docs/migrations.md`)
- Secret management (see `docs/secret-management.md`)
- Performance tuning (see `docs/performance.md`)

### Command Line Options

```bash
./bin/receiver listen --help

Flags:
  -n, --nats-url string          NATS server address
  -t, --subject string           NATS subject to listen to
      --stream string            JetStream stream name
      --consumer string          JetStream durable consumer
      --publisher-topic string   Subject used by the publisher
      --postgres-url string      PostgreSQL connection URL
      --log-level string         Logger level (debug|info|warn|error)
      --replay-from string       Deliver policy override (all|new|last|seq:<n>|time:<RFC3339>)
      --ack-wait duration        Ack wait override (e.g. 45s)
      --telemetry-enabled        Enable OpenTelemetry exporters
      --telemetry-endpoint string
                                 OTLP collector endpoint
      --telemetry-insecure       Send OTLP traffic without TLS
```

Critical overrides stay available through CLI flags; configuration is managed via the TOML file or `CAATSM_` environment variables.

| CLI flag            | Config key             | Purpose                                |
|---------------------|------------------------|----------------------------------------|
| `--nats-url`        | `nats.url`             | Point to a different NATS cluster      |
| `--subject`         | `subscription.topic`   | Change the subscribed subject filter   |
| `--stream`          | `nats.stream`          | Bind to another JetStream stream       |
| `--consumer`        | `nats.consumer`        | Override the durable consumer name     |
| `--publisher-topic` | `publisher.topic`      | Publish parsed output to a new subject |
| `--postgres-url`    | `postgres.url`         | Redirect persistence to another DB     |
| `--log-level`       | `log.level`            | Adjust runtime logging verbosity       |
| `--telemetry-*`     | `telemetry.*`          | Toggle tracing/metrics exporters       |

#### Replay & Backoff

- `--replay-from seq:12345` replays from a specific JetStream sequence, while `--replay-from time:2024-11-15T08:00:00Z` starts at a timestamp.
- Configure retry behavior with `[nats.consumer_rules]` settings.

### Observability

The processor exposes three complementary observability surfaces with production-ready OpenTelemetry implementation:

1. **OpenTelemetry (traces + metrics)**
    - **Production-ready setup** with environment-based sampling, comprehensive resource attributes, and optimized batching.
    - Enable via `[telemetry] enabled = true` and set `endpoint` to your OTLP/HTTP collector (e.g., `http://otel-collector:4318`).
    - **Sampling strategy**:
      - Production: 1% sampling (cost-effective)
      - Staging: 10% sampling (balanced observability)
      - Development/Test: 100% sampling (full debugging)
    - CLI overrides:
      - `--telemetry-enabled` toggles exporters on/off.
      - `--telemetry-endpoint` and `--telemetry-insecure` adjust the OTLP HTTP endpoint and TLS behavior.
    - **Comprehensive traces** with semantic attributes:
      - `caatsm/app`: Message processing spans with `messaging.system`, `messaging.operation`, `caatsm.message.category`
      - `caatsm/postgres`: Database operations with `db.system`, `db.operation`, `db.table`
      - `caatsm/nats`: NATS operations with `messaging.destination`, `messaging.consumer.id`
    - **Business metrics** (focused set for OTEL):
      - `caatsm_messages_processed_total` (counter, by `message.status` / `message.category`)
      - `caatsm_publish_failures_total` (counter)
      - `caatsm_parse_duration_seconds` (histogram)
    - **Resource attributes** include service metadata, environment, build info, and infrastructure details.
    - Application code records telemetry via a thin `telemetry.Recorder` abstraction, which fans out to OTEL and Prometheus backends as configured.

2. **Prometheus metrics (`/metrics`)**
   - Implemented in `internal/infra/metrics` and considered the primary source for SRE PromQL/SLOs.
   - Key metric families:
     - `caatsm_messages_total{stream,consumer,result}` – per-stream/consumer throughput and results.
     - `caatsm_handle_latency_seconds_bucket{stream,consumer}` – end-to-end handling latency from NATS receive to handler completion.
     - `caatsm_retries_total{stream,consumer,reason}` – JetStream retry/NAK counts.
     - `caatsm_db_queries_total{operation,result}` and `caatsm_db_query_latency_seconds_bucket{operation}` – DB activity and latency.
     - `caatsm_dlq_messages_total{stream,consumer}` and `caatsm_dlq_publish_failures_total{stream,consumer}` – DLQ routing success/failures.
     - `caatsm_nats_consumer_pending_messages{stream,consumer}` – JetStream consumer backlog/lag.
   - Prometheus scrapes `GET /metrics` on the monitoring server; Grafana dashboards under `configs/grafana-dashboards.dev` are wired to these series.

3. **Health and readiness endpoints**
   - A lightweight monitoring server exposes:
     - `GET /livez` – liveness endpoint: reports process and build information, does not call external dependencies.
     - `GET /readyz` – readiness endpoint: pings PostgreSQL and checks NATS connection status within `monitoring.health_timeout`, returning 503 on failure.
     - `GET /healthz` – backward-compatible alias currently sharing logic with `/readyz`.
   - Responses include build metadata and dependency status/latency (see `docs/observability.md` for examples).
   - Configure the server via the `[monitoring]` block (defaults shown):

```toml
[monitoring]
addr = ":2112"
enable_metrics = true
enable_health = true
read_timeout = "5s"
write_timeout = "5s"
health_timeout = "2s"
```

Set `monitoring.disabled = true` (or `addr = ""`) if you need to turn the HTTP server off, e.g., during certain integration tests.

## Development

See `docs/dev-guide.md` for the full development workflow, including Docker Compose instructions, observability tooling, and troubleshooting tips.

Quick start:

```bash
# Start database + messaging
docker compose -f docker-compose.dev.yml up -d postgres nats nats-box

# Start observability stack (optional)
docker compose -f docker-compose.dev.yml up -d otel-collector jaeger prometheus grafana
```

Run the processor locally while the infra runs in Docker:

```bash
# Core NATS mode (default for development, fast and lightweight)
GO_ENV=dev \
CAATSM_POSTGRES_URL=postgres://caatsm:caatsm@localhost:5432/aviation?sslmode=disable \
  go run ./cmd/main listen

# JetStream mode (for integration testing or production-like behavior)
GO_ENV=dev \
CAATSM_NATS_MODE=jetstream \
CAATSM_POSTGRES_URL=postgres://caatsm:caatsm@localhost:5432/aviation?sslmode=disable \
  go run ./cmd/main listen
```

Tear everything down with `docker compose -f docker-compose.dev.yml down -v`.

## Deployment Examples

For production deployment, see the comprehensive guide: **[Production Deployment Guide](docs/prod-guide.md)**

Additional deployment-specific guides:

- **`docs/prod-guide.md`** - Complete production deployment guide with configuration, setup, and operations
- **`docs/deploy-systemd.md`** - Systemd service deployment with environment file configuration
- **`docs/deploy-k8s.md`** - Kubernetes deployment with ConfigMap/Secret and health probes

## Additional Documentation

- **`docs/migrations.md`** - Database migration strategy and best practices
- **`docs/secret-management.md`** - Secret management best practices and integration guides
- **`docs/performance.md`** - Performance tuning guidelines and optimization strategies
- **`docs/observability.md`** - Observability setup and metrics documentation
- **`docs/nats.md`** - NATS/JetStream configuration and usage guide
- **`docs/reliability.md`** - Reliability patterns and error handling

### Project Structure

- **Domain Layer** (`internal/domain`): Pure business logic and domain models
- **Application Layer** (`internal/app`): Orchestrates business flows
- **Adapter Layer** (`internal/adapter`): Interfaces and adapters between layers
- **Infrastructure Layer** (`internal/infra`): External concerns (NATS, PostgreSQL, config, logging)

### Adding New Features

1. **Domain Changes**: Add to `internal/domain` (no external dependencies)
2. **Business Logic**: Add to `internal/app`
3. **External Integrations**: Add to `internal/infra`
4. **Adapters**: Add to `internal/adapter` to bridge between layers

### Dependency Injection

Dependencies are managed using Google Wire. To add a new dependency:

1. Create a provider function in the appropriate package
2. Add it to `pkg/di/wire.go`
3. Run `wire ./pkg/di` to regenerate `wire_gen.go`

> If the `wire` binary is missing, install it with `go install github.com/google/wire/cmd/wire@v0.7.0` and ensure `$GOPATH/bin` is on your `PATH` (or run it directly via the absolute path).

### Testing

The project keeps tests close to the code that they exercise:

- **Domain/adapter/app unit tests** live under `internal/**` and cover parsing, validation, orchestration, and adapters. Run them all with `task test` (or `make test`), which now uses the Ginkgo CLI to run unit test suites in verbose mode (`ginkgo -r -v ./cmd ./internal`).
- **Integration tests** under `test/integration` spin up disposable TimescaleDB and NATS JetStream instances (via `testcontainers-go`) and execute a full ingestion flow. Use `task test-int` after ensuring Docker is running.
- **Coverage goals** are tracked via `task coverage`, which produces both a coverage profile and an HTML report under `coverage/coverage.html`.
- **Benchmark tests** are available for performance-critical components:
  ```bash
  # Run parser benchmarks
  go test -bench=BenchmarkParse -benchmem ./internal/adapter/parser
  
  # Run repository benchmarks (requires DB connection)
  go test -bench=BenchmarkMapper -benchmem ./internal/infra/postgres
  
  # Run processor benchmarks
  go test -bench=BenchmarkHandle -benchmem ./internal/app
  ```

| Purpose                    | Make command        | Task command        |
|----------------------------|---------------------|---------------------|
| Run unit tests (Ginkgo)    | `make test`         | `task test`         |
| Run integration tests      | `make test-int`     | `task test-int`     |
| Run unit+integration tests | `make test-all`     | `task test-all`     |
| Generate coverage html     | `make coverage`     | `task coverage`     |
| Lint (golangci-lint)       | `make lint`         | `task lint`         |

> Integration tests need Docker available on the host. Ginkgo-based unit tests or lint targets require the respective binaries (`go install github.com/onsi/ginkgo/v2/ginkgo@latest`, [golangci-lint install guide](https://golangci-lint.run/)). Use `task install-test` to bootstrap Ginkgo tooling before running `task test`, `task test-all`, or their `make` equivalents. `make test` / `task test` run Ginkgo in verbose mode (`-v`) over `./cmd` and `./internal`, showing each spec for easier debugging.

## Message Flow

1. **NATS Consumer** receives raw telegram messages from NATS:
   - **JetStream mode** (default): Uses durable pull consumer with batch processing, ACK/NAK, and retry logic
   - **Core mode**: Uses `QueueSubscribe` for simple pub/sub with queue group load balancing (no persistence or retries)
2. **MessageProcessor** orchestrates the processing:
   - Parses the message using the Parser adapter
   - Stores the parsed message in PostgreSQL via Repository
   - Publishes the parsed message to the output topic via Publisher
3. **ACK/NAK** is sent based on processing success/failure
4. **Retry Logic** handles transient failures automatically

### Error Handling & Retries

- **Parser failures** (invalid headers/body) are treated as permanent: the raw payload is stored in `aviation.telegrams_raw`, the message is ACKed, and no JetStream retries are attempted.
- **Repository failures** are transient: the consumer returns an error, the message is `NAK`ed, and JetStream redelivers it with simple backoff.
- **Publisher failures** are logged and persisted as raw records, but they are marked permanent to avoid hammering downstream topics; the deduplicated output can be replayed from the raw table later.
- Tune JetStream retry behavior via `[nats.consumer_rules.max_deliver]`, `[nats.consumer_rules.backoff]`, and CLI overrides like `--ack-wait`. The monitoring server plus Prometheus counters provide visibility into each failure bucket.

### Failure Buckets

Messages that cannot be parsed or fail to publish are written to `aviation.telegrams_raw` with a status:

| Status                | Description                                      |
|-----------------------|--------------------------------------------------|
| `parsed`              | Successfully parsed and stored                   |
| `header_error`        | Header invalid (missing start indicator, etc.)   |
| `body_error`          | Body pattern did not match any known format      |
| `publish_error`       | Downstream publisher returned an error           |

Each entry stores the raw payload, received timestamp, and metadata to aid replay or manual inspection.

## Message Parsing

The system supports parsing of aviation telegram messages in the standard ICAO format. All messages follow a common header structure, followed by a message body that varies by message type.

### Message Format

All telegrams follow this general structure:

```
ZCZC <MessageID> <DateTime>
<PriorityIndicator> <PrimaryAddress>
<SecondaryAddresses>
<Originator>
<Body>
NNNN
```

**Header Fields:**
- `ZCZC`: Start indicator
- `MessageID`: Unique message identifier (e.g., "TMQ1324")
- `DateTime`: Message date and time (e.g., "150631")
- `PriorityIndicator`: Message priority (e.g., "FF", "DD")
- `PrimaryAddress`: Primary recipient address (ICAO code)
- `SecondaryAddresses`: Additional recipient addresses
- `Originator`: Message originator (optional)

### Supported Message Types

The parser supports the following message categories:

#### 1. ARR - Arrival Message

Arrival messages report aircraft arrival information.

**Format:**
```
(ARR-<FlightNumber>[/<SSR>]-<DepartureAirport>-<ArrivalAirport><ArrivalTime>)
```

**Parsed Fields:**
- `category`: "ARR"
- `aircraft_id`: Aircraft identification/flight number
- `ssr_mode_and_code`: SSR mode and code (optional)
- `departure_airport`: Departure airport ICAO code
- `departure_time`: Departure time
- `arrival_airport`: Arrival airport ICAO code
- `arrival_time`: Arrival time
- `estimated_elapsed_time`: Estimated flight duration (optional)
- `alternate_airport`: Alternate airport (optional)
- `other_info`: Additional information (optional)

**Example:**
```
ZCZC ARR1234 150631
FF ZBTJZPZX
150630 ZBACZQZX
(ARR-CCA1234-A1234-ZBTJ1500-ZGGG0135)
NNNN
```

#### 2. DEP - Departure Message

Departure messages report aircraft departure information.

**Format:**
```
(DEP-<FlightNumber>[/<SSR>]-<DepartureAirport><DepartureTime>-<Destination>)
```

**Parsed Fields:**
- `category`: "DEP"
- `aircraft_id`: Aircraft identification/flight number
- `ssr_mode_and_code`: SSR mode and code (optional)
- `departure_airport`: Departure airport ICAO code
- `departure_time`: Departure time
- `destination`: Destination airport ICAO code
- `estimated_elapsed_time`: Estimated flight duration
- `alternate_airport`: Alternate airport (optional)
- `other_info`: Additional information (optional)

**Example:**
```
ZCZC DEP5678 120915
DD KLAXZPZX
120914 KSFOZQZX
(DEP-ABC5678-A1234-ZBTJ1440-ZGGG)
NNNN
```

#### 3. CNL - Cancellation Message

Cancellation messages indicate flight cancellations.

**Format:**
```
(CNL-<FlightNumber>-<DepartureAirport>-<DestinationAirport>)
```

**Parsed Fields:**
- `category`: "CNL"
- `aircraft_id`: Aircraft identification/flight number
- `departure_airport`: Departure airport ICAO code
- `destination_airport`: Destination airport ICAO code
- `other_info`: Additional information (optional)

**Example:**
```
ZCZC CNL9012 150631
FF ZBTJZPZX
(CNL-CCA9012-ZBTJ-ZGGG)
NNNN
```

#### 4. DLA - Delay Message

Delay messages report flight delays with new departure times.

**Format:**
```
(DLA-<FlightNumber>[/<SSR>]-<DepartureAirport>[<NewDepartureTime>]-<ArrivalAirport>[<ArrivalTime>])
```

**Parsed Fields:**
- `category`: "DLA"
- `aircraft_id`: Aircraft identification/flight number
- `ssr_mode_and_code`: SSR mode and code (optional)
- `departure_airport`: Departure airport ICAO code
- `new_departure_time`: New departure time (optional)
- `arrival_airport`: Arrival airport ICAO code
- `arrival_time`: Estimated arrival time (optional)
- `other_info`: Additional information (optional)

**Example:**
```
ZCZC DLA3456 150631
FF ZBTJZPZX
(DLA-CCA3456-A1234-ZBTJ1600-ZGGG0200)
NNNN
```

#### 5. FPL - Flight Plan Message

Flight plan messages contain detailed flight planning information.

**Format:**
```
(FPL-<FlightNumber>-<Indicator>
-<AircraftID>/<SSR>
-<DepartureAirport><DepartureTime>
-<Speed><Level> <Route>
-<Destination><EstimatedTime> <AlternateAirport>
-<OtherInfo>)
```

## Seed Tool: `cmd/seed-telegrams`

`seed-telegrams` 是一个开发/测试用的报文发生器，用来向 NATS/JetStream 持续或突发地发送合成电报（ARR/DEP/CNL/DLA/FPL），用于驱动解析与下游流水线。

### 支持的类别与内容

生成的电报遵循与解析器相同的格式约定：

- ARR / DEP：支持无 SSR、合法简单 SSR，以及刻意构造为“当前正则无法解析”的复杂 SSR。
- CNL / DLA：与 `internal/adapter/parser/aviation_parser_test.go` 中的测试样例同一类结构。
- FPL：生成包含多行 route 与 `OtherInfo` 字段的完整 FPL，`OtherInfo` 中会随机组合 `PBN/`, `NAV/`, `REG/`, `EET/`, `SEL/`, `PER/`, `RIF/`, `RMK/` 等片段，以覆盖解析逻辑。

### 命令行参数

常用参数：

- `--nats-url`：NATS 地址（默认 `nats://127.0.0.1:4222`，为空字符串则完全不连接 NATS）。
- `--subject`：普通 NATS 发布 subject（默认 `telegram.raw`）。
- `--jetstream`：是否使用 JetStream 发布。
- `--stream` / `--js-subject`：JetStream 相关选项。
- `--count`：要发送的电报数量（`mode=burst` 或有上限的 interval/mixed 时生效）。
- `--category`：`ARR|DEP|CNL|DLA|FPL|mixed`，`mixed` 表示在五类中随机选择。
- `--status`：`parsed|header_error|body_error|publish_error|repository_error|random`。
  - `random` 模式下，合法报文偏向标记为 `parsed`，刻意非法报文偏向标记为 `body_error`。
- `--error-reason`：错误原因说明，将写入元数据 header（默认 `synthetic test payload`）。
- `--dry-run`：只打印电报内容，不真正发布到 NATS。
- `--header-format`：`json|none`，控制是否以 JSON 形式附加元数据 header。
- `--mode`：seed 模式：
  - `burst`：一次性快速发送完 `count` 条。
  - `interval`：按照给定时间间隔持续发送。
  - `mixed`：先按 interval 发送一部分，再以 burst 方式发送剩余。
- `--interval-min` / `--interval-max`：`interval`/`mixed` 模式下两条电报之间的最小/最大间隔（默认 `1s` / `2s`）。
- `--duration`：`interval`/`mixed` 模式下的总持续时间，`0` 表示仅按 `count` 控制停止条件。

### 使用示例

#### 1. 一次性快速打 100 条（突发流量）

```bash
go run ./cmd/seed-telegrams \
  --count=100 \
  --mode=burst \
  --category=mixed \
  --status=random
```

#### 2. 模拟真实流量：每 1–2 秒发一条，持续 5 分钟

```bash
go run ./cmd/seed-telegrams \
  --mode=interval \
  --interval-min=1s \
  --interval-max=2s \
  --duration=5m \
  --status=random \
  --category=mixed
```

#### 3. 慢热 + 突发：前半段慢慢发，后半段瞬间打完

```bash
go run ./cmd/seed-telegrams \
  --count=200 \
  --mode=mixed \
  --interval-min=500ms \
  --interval-max=1500ms \
  --status=random
```

#### 4. 只打印合成电报，不发送（本地调试报文格式）

```bash
go run ./cmd/seed-telegrams \
  --count=5 \
  --mode=burst \
  --dry-run \
  --category=FPL
```

#### 5. 持续缓慢发送（直到 Ctrl-C）

```bash
# 使用 Makefile/Taskfile 任务（推荐）
make seed-slow
# 或
task seed-slow

# 自定义间隔时间
make seed-slow INTERVAL_MIN=3s INTERVAL_MAX=8s

# 直接使用命令行
go run ./cmd/seed-telegrams \
  --nats-url nats://localhost:4222 \
  --jetstream \
  --stream TELEGRAM \
  --js-subject telegram.serial \
  --mode interval \
  --interval-min 2s \
  --interval-max 5s \
  --count 0
```

**注意**：设置 `--count 0` 会持续发送直到手动停止（Ctrl-C）。这对于长时间测试和监控系统行为非常有用。

该工具专门为开发与测试设计，不影响生产服务逻辑，推荐在本地或测试环境配合解析与存储流水线一起使用，用于回归测试、吞吐量观察和错误场景演练。

**Parsed Fields:**
- `category`: "FPL"
- `flight_number`: Flight number
- `reference_data`: Reference data (optional)
- `aircraft_id`: Aircraft identification
- `ssr_mode_and_code`: SSR mode and code
- `flight_rules_and_type`: Flight rules and type
- `cruising_speed_and_level`: Cruising speed and flight level
- `departure_airport`: Departure airport ICAO code
- `departure_time`: Departure time
- `route`: Flight route
- `destination_and_total_time`: Destination and estimated total time
- `alternate_airport`: Alternate airport (optional)
- `estimated_arrival_time`: Estimated arrival time
- `pbn`: Performance-based navigation equipment
- `navigation_equipment`: Navigation equipment
- `estimated_elapsed_time`: Estimated elapsed time
- `selcal_code`: SELCAL code
- `register`: Aircraft registration (optional)
- `performance_category`: Performance category
- `reroute_information`: Reroute information (optional)
- `remarks`: Remarks (optional)

**Example:**
```
ZCZC FPL7890 150631
FF ZBTJZPZX
(FPL-JAE7433-IS
-B744/H-SXIRPZJWY/S
-ZBTJ1755
-K0926S0920 CG A326 VYK W80 HUR B339 GM A575 MANSA/K0919S0980
-EDDF0948 EDDK
-EET/ZMUB0100 UNKL0236
REG/B2422 SEL/JLAD
NAV/RNAV1 RNAV5 RNP4
RMK/AGCS EQUIPPED)
NNNN
```

### Parsing Process

1. **Header Parsing**: The parser extracts header information including message ID, date/time, priority, and addresses
2. **Category Detection**: The parser identifies the message category from the body content
3. **Body Parsing**: Based on the category, the parser applies the appropriate regex pattern to extract structured data
4. **Data Mapping**: Extracted data is mapped to domain model structures (ARR, DEP, CNL, DLA, or FPL)
5. **Storage**: The parsed message is stored in PostgreSQL with:
   - Raw content in `content` field
   - Parsed structured data in `body_data` field (JSONB)
   - Metadata in dedicated columns

### Parsed Message Structure

All parsed messages are stored in the `ParsedMessage` domain model:

```go
type ParsedMessage struct {
    Uuid               string      // Unique identifier
    MessageID          string      // Telegram message ID
    DateTime           string      // Message date/time
    PriorityIndicator  string      // Priority level
    PrimaryAddress     string      // Primary recipient
    SecondaryAddresses string      // Secondary recipients
    Originator         string      // Message originator
    OriginatorDateTime string      // Originator date/time
    Category           string      // Message category (ARR, DEP, CNL, DLA, FPL)
    Content            string      // Raw message content
    BodyData           interface{} // Parsed body data (ARR, DEP, CNL, DLA, or FPL struct)
    ReceivedAt         time.Time   // Reception timestamp
    ParsedAt           time.Time   // Parsing timestamp
    DispatchedAt       time.Time   // Dispatch timestamp
    NeedDispatch       bool        // Dispatch flag
    Parsed             bool        // Parsing success flag
    Comments           string      // Parsing comments/errors
}
```

### Error Handling

- **Invalid Format**: Messages that don't match expected formats are stored with `Parsed = false` and error details in `Comments`
- **Partial Parsing**: Header parsing failures result in storing raw content only
- **Category Mismatch**: Unsupported categories are logged and stored with parsing errors

## Database Schema

The application uses the `aviation.telegrams` table. See `internal/infra/postgres/telegrams.ddl` for the schema definition.

Key fields:
- `uuid`: Primary key (UUID)
- `message_id`: Telegram message ID
- `content`: Raw message content
- `body_data`: Parsed body data (JSONB)
- `received_at`, `parsed_at`, `dispatched_at`: Timestamps

## Logging

Logging uses Zap with structured logging. Log levels and format can be configured:

- **Levels**: `debug`, `info`, `warn`, `error`
- **Formats**: `json` (production) or `console` (development)

Logs include contextual information:
- Message IDs
- Subject names
- Stream names
- Processing attempts

## Performance

- **Batch Processing**: Messages are processed in configurable batches (default: 50)
- **Database Inserts**: Uses PostgreSQL `COPY FROM` for efficient batch inserts
  via `Repository.InsertBatch`. The default processor issues single inserts,
  but you can switch to buffered batches in high-throughput deployments.
- **Connection Pooling**: Configurable PostgreSQL connection pool
- **JetStream**: Reliable message delivery with automatic retries

## Troubleshooting

### Connection Issues

- **NATS**: Check that NATS server is running and JetStream is enabled
- **PostgreSQL**: Verify database connection string and that the schema exists

### Message Processing Issues

- Check logs for parsing errors
- Verify message format matches expected telegram format
- Check database constraints and indexes

### Configuration Issues

- Ensure `GO_ENV` is set correctly
- Verify configuration file exists at `configs/config.{env}.toml`
- Check environment variable names use `CAATSM_` prefix

## Migration from Legacy System

This project was refactored from:
- **Watermill** → **nats.go JetStream**
- **Hasura GraphQL** → **PostgreSQL pgx**
- **Viper** → **Koanf**
- **Manual DI** → **Google Wire**

The legacy code has been removed. See the project history for migration details.

## License

This repository has not declared a public license yet.

## Contributing

Contribution guidelines are not published; please coordinate changes via pull requests or direct maintainers.
