# Development Guide

This document describes how to run the full development stack—database, NATS, and observability tooling—using `docker-compose.dev.yml`. All commands assume you are at the repository root.

## Core Services (TimescaleDB + NATS)

Spin up PostgreSQL/TimescaleDB and NATS JetStream in the background:

```bash
docker compose -f docker-compose.dev.yml up -d postgres nats nats-box
```

> Development mode defaults to `nats.mode = "core"`, so the processor consumes directly from the configured subject (`subscription.topic`). **However, the publisher always targets JetStream for deduplicated fan-out, so the provided Taskfile (and most examples below) override the mode to `jetstream`.** If you truly need core mode, set `CAATSM_NATS_MODE=core` manually and ensure any publishers use core subjects.

- `postgres` seeds the `aviation` schema using `internal/repository/telegrams.ddl` and exposes port `5432`.
- `nats` enables JetStream with client port `4222` and monitoring/UI on `8222`.
- `nats-box` provides a toolbox container (`docker compose exec nats-box sh`) for publishing test messages or inspecting JetStream.

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

### Using Taskfile shortcuts

The `Taskfile.yml` includes helper targets that wrap the commands above:

- `task up` – starts PostgreSQL, NATS, and the observability stack (OpenTelemetry Collector, Jaeger, Prometheus, Grafana) using Docker Compose.
- `task dev-run` – ensures `task up` has run, exports the necessary `CAATSM_*` environment variables (including `CAATSM_NATS_MODE=jetstream`), and executes `go run ./cmd/main listen` with telemetry enabled.
- `task down` – stops the entire stack and removes containers/volumes.

Use these tasks if you prefer a one-command workflow instead of invoking `docker compose` and environment exports manually.

## Publishing Sample Telegrams

Use the helper CLI in `cmd/seed-telegrams` to push realistic payloads onto NATS (mirrors the fixtures in `internal/parsers/aviation_parser_test.go`):

```bash
# Insert rows into aviation.telegrams_raw and publish to NATS simultaneously
GO_ENV=dev go run ./cmd/seed-telegrams \
  --postgres-url postgres://caatsm:caatsm@localhost:5432/aviation?sslmode=disable \
  --nats-url nats://127.0.0.1:4222 \
  --subject telegram.serial \
  --count 20 \
  --category mixed \
  --status random
```

- `--postgres-url` controls database insertion (omit to skip DB writes); metadata lands in `aviation.telegrams_raw.metadata`.
- `--dry-run` prints telegrams without touching NATS/Postgres.
- `--category` chooses ARR/DEP/CNL/DLA/FPL or `mixed`.
- `--status` controls stored/published status (`parsed|header_error|body_error|publish_error|repository_error|random`).
- `--no-nats` disables publishing; `--jetstream`, `--stream`, `--js-subject` toggle JetStream publishing.
- Inspect deliveries with `docker compose exec nats-box nats sub 'telegram.>'`.
- When running in core mode (default), the seeder publishes via standard `nc.Publish` and sets `Nats-Msg-Id` headers so the processor can derive message IDs.

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
   CAATSM_NATS_MODE=jetstream \
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
  - Uses `configs/prometheus.dev.yml` to scrape the collector and NATS monitoring endpoint; UI available at <http://localhost:9090>
- `grafana`
  - Persists data in `grafana-data`, provisions datasources via `configs/grafana-datasources.dev.yml`, and listens on <http://localhost:3000> (login `admin` / `admin`)
  - Automatically loads dashboards from `configs/grafana-dashboards.dev/`, including OpenTelemetry Collector and NATS/JetStream overviews (find them under the **Dev Observability** folder)

### Customizing Collections & Dashboards

- Adjust `configs/prometheus.dev.yml` to add/remove scrape jobs—for example, include your application’s `/metrics` endpoint.
- Add more Grafana provisioning files (dashboards, alert rules) under `configs/` and mount them in `docker-compose.dev.yml`.
- To ingest telemetry from local services, configure their OTLP exporters to target `http://localhost:4318` (HTTP) or `grpc://localhost:4317`.

## Troubleshooting

- **PostgreSQL init errors**: ensure `internal/repository/telegrams.ddl` is valid SQL and the `postgres-data` volume is removed (`docker volume rm go-caatsm_postgres-data`) before restarting.
- **NATS connection failures**: confirm ports `4222/8222` are free and JetStream is enabled; use `docker compose logs nats`.
- **Prometheus scrape failures**: verify endpoints listed in `configs/prometheus.dev.yml` match the service names defined in Docker Compose.
- **Grafana provisioning issues**: check container logs (`docker compose logs grafana`) to ensure the datasources file was read; correct file permissions or YAML formatting if provisioning is skipped.

