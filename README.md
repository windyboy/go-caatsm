# go-caatsm

Civil Aviation Authority Telegram Message Processor.

High-performance processing for aviation telegrams using NATS JetStream and PostgreSQL/TimescaleDB.

> **Architecture Status:** See [`docs/architecture.md`](docs/architecture.md) for the authoritative 3-layer Go-idiomatic architecture specification.

## Architecture

3-layer high-cohesion package layout:

- `cmd/receiver`: CLI entrypoint (`listen` command), explicit dependency wiring, and graceful shutdown
- `cmd/seed-telegrams`: synthetic aviation telegram generator
- `internal/telegram`: pure domain envelope (`Message`), `Router` (composite routing), `AFTNValidator`, plus `aviation/` (ARR/DEP/CNL/DLA/FPL parsers & models) and `weather/` (METAR/SPECI/TAF parsers & models)
- `internal/pipeline`: `Processor` orchestration pipeline, consumer-side small interfaces (`ports.go`), and permanent/transient error classification
- `internal/nats`: JetStream pull consumer (inbound/outbound/DLQ 3-stream topology, `ShouldAutoProvision()` prod guard), publisher (OTel TraceContext propagation), DLQ handler, and monitor
- `internal/postgres`: PostgreSQL/TimescaleDB repository (`pgx`), time-bounded deduplication query, 17-column row mapper, and idempotent schema DDL (`telegrams.ddl`)
- `internal/config`: Koanf TOML + `CAATSM_*` env configuration with strict production & stream guards
- `internal/observability`: Zap structured logging, consolidated OpenTelemetry SDK & TraceContext propagator, Prometheus metrics, and HTTP health/metrics server
- `internal/buildinfo`: build version, commit, and timestamp metadata

## Features

- JetStream ingestion with retries and DLQ support
- PostgreSQL/TimescaleDB persistence
- Structured logging (Zap)
- OpenTelemetry tracing and Prometheus metrics
- Optional AFTN protocol validation
- Batch processing and health monitoring
- Weather report parsing (METAR, SPECI, TAF)

## Prerequisites

- Go 1.24+
- PostgreSQL 12+
- NATS Server with JetStream enabled
- Docker (integration tests)

## Quick Start

1. Start dependencies:
   ```bash
   docker compose -f docker-compose.dev.yml up -d postgres nats nats-box
   ```

2. Configure the application:
   - Copy `configs/config.dev.toml` and edit as needed
   - Or set environment variables with the `CAATSM_` prefix

3. Run in dev mode:
   ```bash
   make run-dev
   ```

## Build and Run

- Build: `make build`
- Run (dev): `make run-dev`
- Run (prod): `make run-prod`
- Run (local go run): `make run-local`

## Configuration

Configuration loads from `configs/config.{env}.toml`, where `{env}` is `GO_ENV` (default: `dev`).

Required values:
- `nats.url`
- `postgres.url`
- `publisher.topic`

Defaults:
- `nats.stream` defaults to `TELEGRAM`
- `subscription.topic` defaults to `telegram.serial`
- `nats.mode` must be `jetstream` or empty (defaults to JetStream)

Minimal example:

```toml
[nats]
url = "nats://localhost:4222"
mode = "jetstream"
stream = "TELEGRAM"
consumer = "telegram-consumer"

[publisher]
topic = "telegram.json"

[subscription]
topic = "telegram.serial"

[postgres]
url = "postgres://user:password@localhost:5432/aviation?sslmode=disable"

[app]
batch_size = 50
batch_timeout = "2s"
monitor_interval = "30s"

[log]
level = "info"
format = "json"
```

### Timeouts and Ack Wait

`[timeouts]` is optional. To tune redelivery, set `timeouts.ack_wait` and/or `nats.consumer_rules.ack_wait`. When neither is specified the application defaults to `30s`.

## JetStream Notes

- JetStream is required; other modes are not supported (`nats.mode` must be `jetstream` or empty).
- Stream topology separates inbound (`nats.stream`, `WorkQueuePolicy`), outbound (`nats.publish_stream`, `LimitsPolicy`), and DLQ (`nats.dlq_stream`, `LimitsPolicy`).
- Provisioning behavior is controlled by `nats.auto_provision` (defaults to auto-create/verify in non-prod, and read-only stream/consumer verification in `prod` via `ShouldAutoProvision()`).
- Configure retention and delivery behavior under `[nats.stream_limits]` and `[nats.consumer_rules]`.

## AFTN Validation

AFTN validation is optional and disabled by default. Enable it with:

```toml
[aftn]
validation_enabled = true
message_gap_threshold = "2m"
enable_sequence_gap_detection = true
```

When enabled, invalid telegrams are logged, recorded with error details, and can be routed to a DLQ if configured.

## Observability

- Metrics: `GET /metrics`
- Liveness: `GET /livez`
- Readiness: `GET /readyz`

Monitoring server settings are under `[monitoring]`. Tracing is configured via `[telemetry]`.

## CLI

```bash
./bin/receiver listen --help
```

Common flags:
- `--nats-url`
- `--subject`
- `--stream`
- `--consumer`
- `--publisher-topic`
- `--postgres-url`
- `--log-level`
- `--replay-from`
- `--ack-wait`
- `--telemetry-enabled`, `--telemetry-endpoint`, `--telemetry-insecure`
- `--monitoring-addr`, `--monitoring-disabled`

## Testing

- Unit tests (Ginkgo): `make test`
- Integration tests (Docker): `make test-int`
- All tests: `make test-all`
- Lint: `make lint`
- Coverage: `make coverage`

Single test example:

```bash
ginkgo -r -v --focus "Test Description" ./path/to/package
```

## Seed Tool

`cmd/seed-telegrams` publishes synthetic telegrams for development and testing.

Example:

```bash
go run ./cmd/seed-telegrams \
  --nats-url nats://localhost:4222 \
  --jetstream \
  --stream TELEGRAM \
  --js-subject telegram.serial \
  --count 10 \
  --category mixed \
  --status random
```

## Documentation

- `docs/architecture.md` (authoritative architecture baseline & target design)
- `docs/architecture-ha.md`
- `docs/dev-guide.md`
- `docs/prod-guide.md`
- `docs/nats.md`
- `docs/observability.md`
- `docs/performance.md`
- `docs/migrations.md`
- `docs/secret-management.md`
- `docs/reliability.md`
- `docs/weather-parser.md`

## Contributing

See `AGENTS.md` for coding standards, testing expectations, and release hygiene.

## License

This repository has not declared a public license yet.
