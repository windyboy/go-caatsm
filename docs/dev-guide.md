# Development Guide

This document describes how to run the full development stack—database, NATS, and observability tooling—using `docker-compose.dev.yml`. All commands assume you are at the repository root.

## Core Services (TimescaleDB + NATS)

Spin up PostgreSQL/TimescaleDB and NATS JetStream in the background:

```bash
docker compose -f docker-compose.dev.yml up -d postgres nats nats-box
```

> **NATS Mode:** The application uses JetStream mode exclusively for message persistence, ACK/NAK, automatic retries, and DLQ support. This ensures consistent behavior across development, testing, and production environments.

- `postgres` seeds the `aviation` schema using `internal/infra/postgres/telegrams.ddl` and exposes port `5432`.
- `nats` enables JetStream with client port `4222` and monitoring/UI on `8222`.
- `nats-box` provides a toolbox container (`docker compose exec nats-box sh`) for publishing test messages or inspecting JetStream.
- `nats-exporter` scrapes the monitoring endpoints (`/varz`, `/connz`, `/routez`, `/subz`) and exposes them as Prometheus metrics on port `7777` for the Grafana dashboards.

Prefer to run the Go application on your host for quick iteration while keeping infra in Docker:

```bash
GO_ENV=dev \
CAATSM_POSTGRES_URL=postgres://caatsm:caatsm@localhost:5432/aviation?sslmode=disable \
go run ./cmd/main listen
```

Stop and clean the stack when finished:

```bash
docker compose -f docker-compose.dev.yml down -v
```

In development (`GO_ENV=dev` or unset), if you run the application while
stopping and recreating the NATS/JetStream containers (for example:

```bash
docker compose -f docker-compose.dev.yml down -v
docker compose -f docker-compose.dev.yml up -d postgres nats nats-box
```

), the JetStream state will be reset. The processor behaves as follows:

- The NATS client keeps retrying the connection and automatically reconnects
  when NATS is back.
- The JetStream consumer expects streams and consumers to exist.
- In development, you may need to create them manually or ensure they exist before starting the application.

### Using Taskfile shortcuts

The `Taskfile.yml` includes helper targets that wrap the commands above:

- `task up` – starts PostgreSQL, NATS (JetStream, toolbox, and Prometheus exporter), and the observability stack (OpenTelemetry Collector, Jaeger, Prometheus, Grafana) using Docker Compose.
- `task dev-run` – ensures `task up` has run, exports the necessary `CAATSM_*` environment variables, and executes `go run ./cmd/main listen` with telemetry enabled.
- `task down` – stops the entire stack and removes containers/volumes.

Use these tasks if you prefer a one-command workflow instead of invoking `docker compose` and environment exports manually.

## Publishing Sample Telegrams

Use the helper CLI in `cmd/seed-telegrams` to push realistic payloads onto NATS. The tool supports both aviation telegrams (FPL, ARR, DEP, etc.) and weather reports (METAR, SPECI, TAF).

### Publishing to JetStream (Recommended)

When using JetStream mode, publish messages to the JetStream stream:

```bash
# Publish to JetStream stream (messages are persisted)
GO_ENV=dev go run ./cmd/seed-telegrams \
  --nats-url nats://127.0.0.1:4222 \
  --jetstream \
  --stream TELEGRAM \
  --js-subject telegram.serial \
  --count 20 \
  --category mixed \
  --status random
```

### Using Makefile/Taskfile Tasks

For convenience, you can use the provided tasks:

```bash
# Quick seed (10 messages, JetStream)
make seed
# or
task seed

# Continuous slow seeding (until Ctrl-C, JetStream)
make seed-slow
# or
task seed-slow

# Customize slow seeding
make seed-slow INTERVAL_MIN=3s INTERVAL_MAX=8s CATEGORY=ARR
task seed-slow INTERVAL_MIN=1s INTERVAL_MAX=2s STATUS=parsed
```

### Continuous Slow Seeding

For long-running tests and monitoring, use the `seed-slow` task to continuously send messages at a configurable interval until you press Ctrl-C:

```bash
# Default: 2-5 second intervals, JetStream mode
make seed-slow

# Custom interval and category
make seed-slow INTERVAL_MIN=5s INTERVAL_MAX=10s CATEGORY=DEP

```

This is equivalent to running:
```bash
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

Setting `--count 0` makes it run indefinitely until interrupted.

### Common Options

- `--postgres-url`: Insert rows into `aviation.telegrams_raw` (omit to skip DB writes)
- `--dry-run`: Print telegrams without publishing to NATS/Postgres
- `--category`: Choose message type (`ARR|DEP|CNL|DLA|FPL|mixed`)
- `--status`: Control stored/published status (`parsed|header_error|body_error|publish_error|repository_error|random`)
- `--no-nats`: Disable publishing to NATS
- `--jetstream`: Enable JetStream publishing (requires `--stream` and `--js-subject`)
- `--stream`: JetStream stream name (default: `TELEGRAM`)
- `--js-subject`: Subject within the JetStream stream
- `--mode`: Seed mode (`burst|interval|mixed`)
- `--interval-min` / `--interval-max`: Time interval between messages in interval/mixed modes
- `--count`: Number of messages to send (0 = infinite, until Ctrl-C)

### Inspecting Messages

```bash
# View messages in JetStream stream
docker compose exec nats-box nats stream view TELEGRAM

# Subscribe to messages (JetStream)
docker compose exec nats-box nats sub 'telegram.>'

# View consumer status and pending messages
docker compose exec nats-box nats consumer info TELEGRAM telegram-consumer
```

The main processor keeps consuming `subscription.topic` (defaults to `telegram.>`). Use the seeder to simulate parser failures, publish errors, or replay raw telegrams directly from the database.

## Tracing with Jaeger

1. **Start the observability stack**
   ```bash
   docker compose -f docker-compose.dev.yml up -d otel-collector jaeger prometheus grafana
   ```
   - Jaeger UI runs at <http://localhost:16686>.
   - The OTLP HTTP collector endpoint is available at `http://localhost:4318`.

2. **Run the processor with telemetry enabled**
   ```bash
   CAATSM_TELEMETRY_ENABLED=true \
   CAATSM_TELEMETRY_ENDPOINT=localhost:4318 \
   CAATSM_TELEMETRY_INSECURE=true \
   GO_ENV=dev \
   CAATSM_POSTGRES_URL=postgres://caatsm:caatsm@localhost:5432/aviation?sslmode=disable \
   go run ./cmd/main listen
   ```
   - The service name reported to Jaeger is `caatsm`.

3. **Generate traffic**
   ```bash
   task seed COUNT=5
   ```
   or publish manually with `go run ./cmd/seed-telegrams`.

4. **Inspect traces**
   - Open <http://localhost:16686>, choose the `caatsm` service, and click “Find Traces”.
   - Filter by operation name (e.g., `Consumer.processMessage`) or by time range to drill into individual telegram processing flows.

## Observability Dashboard Stack

The dev compose file also includes OpenTelemetry Collector, Jaeger, Prometheus, and Grafana so you can inspect traces and metrics emitted by the processor.

```bash
docker compose -f docker-compose.dev.yml up -d \
  postgres nats otel-collector jaeger prometheus grafana
```

Services:

- `otel-collector`
  - Loads `configs/otel-collector.dev.yaml`
  - Ports: OTLP gRPC `4317`, OTLP HTTP `4318`, Prometheus scrape `8888`, Prometheus exporter `8889`, health `13133`, zPages `55679`
  - Exports traces to Jaeger via the built-in OTLP gRPC exporter (secured with `tls.insecure: true`)
- `jaeger`
  - Receives OTLP traffic forwarded from the collector on `14250` gRPC and serves the UI at <http://localhost:16686>
- `prometheus`
  - Uses `configs/prometheus.dev.yml` to scrape the collector, `nats-exporter` (`http://nats-exporter:7777/metrics`), and application OTLP metrics forwarded via the collector; UI available at <http://localhost:9090>
- `grafana`
  - Persists data in `grafana-data`, provisions datasources via `configs/grafana-datasources.dev.yml`, and listens on <http://localhost:3000> (login `admin` / `admin`)
  - Automatically loads dashboards from `configs/grafana-dashboards.dev/`, including OpenTelemetry Collector and NATS/JetStream overviews (find them under the **Dev Observability** folder)
  - The OpenTelemetry dashboard also charts the CAATSM-specific metrics `caatsm_messages_processed_total`, `caatsm_publish_failures_total`, and `caatsm_parse_duration_ms` (percentiles) so you can track throughput and parsing latency.
  - Note: `caatsm_parse_duration_ms` has been renamed to `caatsm_parse_duration_seconds` to align with Prometheus `_seconds` conventions.

### Customizing Collections & Dashboards

- Adjust `configs/prometheus.dev.yml` to add/remove scrape jobs—for example, include your application’s `/metrics` endpoint.
- Add more Grafana provisioning files (dashboards, alert rules) under `configs/` and mount them in `docker-compose.dev.yml`.
- To ingest telemetry from local services, configure their OTLP exporters to target `http://localhost:4318` (HTTP) or `grpc://localhost:4317`.

## Troubleshooting

- **PostgreSQL init errors**: ensure `internal/infra/postgres/telegrams.ddl` is valid SQL and the `postgres-data` volume is removed (`docker volume rm go-caatsm_postgres-data`) before restarting.
- **NATS connection failures**: confirm ports `4222/8222` are free and JetStream is enabled; use `docker compose logs nats`.
- **Prometheus scrape failures**: verify endpoints listed in `configs/prometheus.dev.yml` match the service names defined in Docker Compose.
- **Grafana provisioning issues**: check container logs (`docker compose logs grafana`) to ensure the datasources file was read; correct file permissions or YAML formatting if provisioning is skipped.

