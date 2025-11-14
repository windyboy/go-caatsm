# Aviation Telegram Processing System

[![CI](https://github.com/your-org/go-caatsm/actions/workflows/ci.yml/badge.svg)](https://github.com/your-org/go-caatsm/actions/workflows/ci.yml)
![Coverage](https://img.shields.io/badge/coverage-%E2%89%A580%25-brightgreen)

_Update badge URLs to match your GitHub namespace._

A high-performance system for processing, storing, and querying aviation telegram messages in real-time.

## Architecture

The system follows a layered architecture pattern with the following components:

- **Worker Service**: Processes incoming messages from NATS JetStream, parses them, and stores them in TimescaleDB
- **API Service**: Provides REST API for querying messages
- **Database**: TimescaleDB (PostgreSQL extension) for time-series data storage
- **Message Queue**: NATS with JetStream for reliable message processing
- **Cache**: Redis (optional) for hot data caching

### Technology Stack

- **Language**: Go 1.22.5+
- **Database**: PostgreSQL 14+ with TimescaleDB
- **Message Queue**: NATS with JetStream
- **API Framework**: Echo v4
- **Database Driver**: pgx/v5
- **UI Templates**: templ
- **Monitoring**: Prometheus + Grafana
- **Caching**: Redis (optional)

## Quick Start

### Prerequisites

- Go 1.22.5 or later
- PostgreSQL 14+ with TimescaleDB extension
- NATS Server 2.9+ with JetStream enabled
- Docker and Docker Compose (optional)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd go-caatsm
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up database**
   ```bash
   # Start PostgreSQL with TimescaleDB (using Docker)
   docker run -d --name postgres \
     -e POSTGRES_PASSWORD=postgres \
     -e POSTGRES_DB=aviation \
     -p 5432:5432 \
     timescale/timescaledb:latest-pg14

   # Run migrations
   psql -U postgres -d aviation -f migrations/001_create_hypertable.up.sql
   psql -U postgres -d aviation -f migrations/002_add_continuous_aggregate.up.sql
   ```

4. **Start NATS with JetStream**
   ```bash
   docker run -d --name nats \
     -p 4222:4222 \
     -p 8222:8222 \
     nats:latest \
     -js
   ```

5. **Configure the application**
   ```bash
   # Copy and edit configuration
   cp configs/config.dev.toml configs/config.local.toml
   # Edit configs/config.local.toml with your settings
   ```

6. **Build and run**

   **Worker Service:**
   ```bash
   go build -o bin/worker cmd/worker/main.go
   ./bin/worker
   ```

   **API Service:**
   ```bash
   go build -o bin/api cmd/api/main.go
   ./bin/api
   ```

### Using Docker Compose

```bash
docker-compose up -d
```

## Configuration

Configuration is managed through TOML files in the `configs/` directory. The application looks for `config.{GO_ENV}.toml` where `GO_ENV` defaults to `dev`.

### Key Configuration Sections

- `[database]`: PostgreSQL connection settings
- `[nats]`: NATS connection and JetStream configuration
- `[api]`: API server settings
- `[redis]`: Redis cache settings (optional)
- `[timeouts]`: Dial, reconnect, close, and ack-wait tuning for worker/runtime clients

See `configs/config.dev.toml` for example configuration.

### Environment Variables

All configuration keys can be overridden with a `CAATSM_` environment variable. Frequently tuned settings are summarized below:

| Variable | Description | Default |
| --- | --- | --- |
| `GO_ENV` | Selects `config.<GO_ENV>.toml` | `dev` |
| `CAATSM_DATABASE_URL` | Full PostgreSQL/TimescaleDB DSN (include `sslmode=require` in prod) | Derived from `[database]` section |
| `CAATSM_NATS_URL` | NATS connection string, e.g. `nats://user:pass@host:4222` | `nats://localhost:4222` |
| `CAATSM_JETSTREAM_STREAM` | JetStream stream name used by the worker | `TELEGRAMS` |
| `CAATSM_SUBSCRIPTION_QUEUE` | Consumer queue group for the worker | `tele-queue` |
| `CAATSM_PUBLISHER_TOPIC` | Topic used by the JSON publisher | `Telegram.Json` |
| `CAATSM_BATCH_SIZE` | Worker batch size when persisting telegrams | `25` |
| `CAATSM_BATCH_TIMEOUT` | Maximum wait before flushing a batch | `1s` |
| `CAATSM_REDIS_ADDR` | Redis address if caching is enabled | `localhost:6379` |
| `CAATSM_TLS_CERT`, `CAATSM_TLS_KEY` | Optional TLS material for API/worker servers | unset |

Document and store secrets (database passwords, NATS credentials) in a secret manager such as Vault or Kubernetes Secrets instead of committing them to the repository.

#### Timeout Defaults

If the `[timeouts]` block omits any value, the worker falls back to safe defaults to keep JetStream consumers predictable:

| Field | Description | Default |
| --- | --- | --- |
| `server` | NATS connect timeout | `5s` |
| `reconnect_wait` | Delay between reconnect attempts | `2s` |
| `close` | Grace period when closing client connections | `5s` |
| `ack_wait` | Maximum duration JetStream waits for an ACK before redelivery | `30s` |

## API Usage

### Health Check

```bash
curl http://localhost:8080/health
```

### List Telegrams

```bash
# List all telegrams
curl http://localhost:8080/api/v1/telegrams

# List with filters
curl "http://localhost:8080/api/v1/telegrams?category=ARR&limit=10&offset=0"

# List by time range
curl "http://localhost:8080/api/v1/telegrams/time-range?start=2024-01-01T00:00:00Z&end=2024-01-02T00:00:00Z"
```

### Get Telegram by ID

```bash
curl http://localhost:8080/api/v1/telegrams/{uuid}
```

### Prometheus Metrics

```bash
curl http://localhost:8080/metrics
```

## API Documentation

Full API documentation is available in OpenAPI format:
- OpenAPI Spec: `api/openapi.yaml`
- Swagger UI: Available at `/swagger/` (when enabled)

## Development

### Project Structure

```
.
├── cmd/
│   ├── api/          # API service entry point
│   └── worker/       # Worker service entry point
├── internal/
│   ├── api/          # API layer (handlers, middleware, DTOs)
│   ├── service/      # Business logic layer
│   ├── repository/   # Data access layer
│   ├── infrastructure/ # Infrastructure (NATS, DB, Cache)
│   ├── domain/       # Domain models
│   └── config/       # Configuration
├── migrations/       # Database migrations
├── docs/            # Documentation
│   ├── ADR/         # Architecture Decision Records
│   ├── ARCHITECTURE.md
│   └── UPGRADE.md
└── api/             # OpenAPI specifications
```

### Running Tests

```bash
go test ./...
```

To enforce the project-wide coverage gate used in CI:

```bash
go test ./... -coverprofile=cover.out -covermode=atomic
go tool cover -func=cover.out
```

Integration tests can be executed locally with Docker:

```bash
./tests/integration/run.sh up
INTEGRATION_DATABASE_URL=postgres://postgres:postgres@localhost:55432/aviation?sslmode=disable \
INTEGRATION_NATS_URL=nats://localhost:54222 \
go test ./tests/integration -tags=integration -count=1
./tests/integration/run.sh down
```

> The CI workflow runs integration tests only when the `run_integration=true` input is supplied to the workflow_dispatch trigger.

### Code Generation

```bash
# Generate OpenAPI docs (if using swag)
swag init -g cmd/api/main.go
```

## Monitoring & Alerting

### Metrics

Both API and worker services expose Prometheus metrics at `/metrics`. Key series include:

- `http_requests_total`, `http_request_duration_seconds`
- `messages_processed_total`, `message_processing_duration_seconds`
- `message_processing_failures_total{stage=parser|repository|publisher}`
- `nats_messages_consumed_total`, `nats_messages_published_total`, `message_processing_retries_total`
- `worker_batch_duration_seconds`
- `database_query_duration_seconds`, `database_connections_active`

Recommended alerts:

- Processing failures or retries exceed rolling baseline
- JetStream consumer lag/backlog grows beyond recovery window
- `/health/ready` returns `503` for longer than 2 consecutive probes
- Database connection utilization > 85% for sustained periods

### Health Checks

- `/health` - Basic heartbeat
- `/health/ready` - Dependency readiness (database, Redis, NATS) with JSON status per dependency
- `/health/live` - Liveness check

## Deployment

Detailed deployment playbooks, including container images, Helm/Kubernetes manifests, and rolling-upgrade workflows, are available in `docs/DEPLOYMENT.md`. `docs/UPGRADE.md` covers schema/data migrations and zero-downtime strategies.

### Production Considerations

1. **Database**: Enforce TLS, create least-privilege roles (read-only API, insert-only worker), and size pgx pools based on workload.
2. **NATS/JetStream**: Enable authentication/TLS, configure retention, dead-letter streams, and max delivery attempts to match your SLOs.
3. **API**: Front with an ingress that terminates TLS and injects auth (mTLS, OAuth2, or API keys).
4. **Monitoring**: Scrape `/metrics` from API and worker pods, alert on failure counters and backlog depth.
5. **Logging**: Ship structured zap logs to your aggregator, capturing `message_id`, `batch_id`, and processing duration.
6. **Secrets**: Load secrets from Vault or Kubernetes Secrets rather than plaintext config files.

## Documentation

- [Architecture Documentation](docs/ARCHITECTURE.md)
- [Upgrade Guide](docs/UPGRADE.md)
- [Deployment Guide](docs/DEPLOYMENT.md)
- [Releasing Guide](docs/RELEASING.md)
- [Architecture Decision Records](docs/ADR/)

## Security & Secrets

- **Transport security**: enable TLS for PostgreSQL (`sslmode=require`) and NATS (certs or NKey/JWT). Expose JetStream APIs only on private networks.
- **Secret management**: project configuration supports the `CAATSM_` prefix so you can load secrets from Vault, AWS/GCP Secret Manager, or Kubernetes Secrets via environment variables.
- **Least privilege**: create separate DB roles for API reads and worker inserts. Limit JetStream permissions to the specific subjects (`Telegram.Serial`, `Telegram.Json`).
- **Auditing**: zap logs already include `message_id`; forward them to your SIEM to trace suspect traffic.

## License

Distributed under the MIT License. See `LICENSE` for details.

