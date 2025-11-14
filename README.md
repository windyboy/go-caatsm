# Aviation Telegram Processing System

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

See `configs/config.dev.toml` for example configuration.

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

### Code Generation

```bash
# Generate OpenAPI docs (if using swag)
swag init -g cmd/api/main.go
```

## Monitoring

### Metrics

The API service exposes Prometheus metrics at `/metrics`. Key metrics include:

- HTTP request count and duration
- Message processing count and duration
- Database query duration
- NATS message count
- Connection pool metrics

### Health Checks

- `/health` - Basic health check
- `/health/ready` - Readiness check (database, cache)
- `/health/live` - Liveness check

## Deployment

See `docs/UPGRADE.md` for detailed deployment instructions.

### Production Considerations

1. **Database**: Use SSL connections, configure connection pooling
2. **NATS**: Use TLS, configure appropriate retention policies
3. **API**: Implement authentication/authorization
4. **Monitoring**: Set up Prometheus and Grafana
5. **Logging**: Configure structured logging
6. **Secrets**: Use environment variables or secret management

## Documentation

- [Architecture Documentation](docs/ARCHITECTURE.md)
- [Upgrade Guide](docs/UPGRADE.md)
- [Architecture Decision Records](docs/ADR/)

## License

[Your License Here]

## Contributing

[Contributing Guidelines]
