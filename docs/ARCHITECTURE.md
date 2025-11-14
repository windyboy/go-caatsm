# Architecture Documentation

## System Overview

The Aviation Telegram Processing System is designed to receive, parse, store, and query aviation telegram messages in real-time. The system is built using Go and follows a layered architecture pattern.

## Architecture Diagram

```
┌─────────────┐
│   NATS      │
│  JetStream  │
└──────┬──────┘
       │
       │ Messages
       ▼
┌─────────────────┐
│  Worker Service │
│  (cmd/worker)   │
└──────┬──────────┘
       │
       ├──► Parse Message
       ├──► Validate
       ├──► Store (PostgreSQL)
       └──► Publish (NATS)
              │
              ▼
       ┌──────────────┐
       │ TimescaleDB  │
       │  (PostgreSQL)│
       └──────┬───────┘
              │
              │ Query
              ▼
       ┌──────────────┐
       │  API Service │
       │  (cmd/api)   │
       └──────┬───────┘
              │
              │ HTTP/REST
              ▼
       ┌──────────────┐
       │   Clients    │
       │  (Frontend)  │
       └──────────────┘
```

## Technology Stack

### Core Technologies
- **Language**: Go 1.22.5+
- **Database**: PostgreSQL 14+ with TimescaleDB extension
- **Message Queue**: NATS with JetStream
- **API Framework**: Echo v4
- **Database Driver**: pgx/v5
- **Caching**: Redis (optional)

### Supporting Technologies
- **Monitoring**: Prometheus + Grafana
- **Logging**: zap
- **Configuration**: viper
- **Migration**: golang-migrate
- **API Documentation**: OpenAPI/Swagger

## Component Architecture

### 1. Worker Service (`cmd/worker`)

**Responsibility**: Process incoming messages from NATS JetStream

**Components**:
- NATS JetStream Consumer
- Message Handler
- Repository (PostgreSQL)
- Publisher (NATS)

**Flow**:
1. Subscribe to NATS JetStream subject
2. Receive raw message
3. Parse message using parser
4. Validate parsed message
5. Store in TimescaleDB
6. Publish parsed message to output topic

### 2. API Service (`cmd/api`)

**Responsibility**: Provide REST API for querying messages

**Components**:
- Echo HTTP Server
- Handlers
- Service Layer
- Repository (PostgreSQL)
- Middleware (logging, recovery, CORS, metrics)

**Endpoints**:
- `GET /api/v1/telegrams` - List messages with filters
- `GET /api/v1/telegrams/:id` - Get message by ID
- `GET /api/v1/telegrams/time-range` - Get messages by time range
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

### 3. Data Layer

#### Repository Pattern
- **Interface**: `internal/iface/TelegramRepository`
- **Implementation**: `internal/repository/postgres/TelegramRepository`
- **Operations**: Create, CreateBatch, GetByID, List, GetByTimeRange, Count

#### Database Schema
- **Table**: `aviation.telegrams` (hypertable)
- **Partitioning**: By `received_at` (1 day chunks)
- **Compression**: Data older than 7 days
- **Indexes**: On message_id, category, primary_address, received_at

#### Continuous Aggregates
- `aviation.telegrams_hourly` - Hourly statistics
- `aviation.telegrams_daily` - Daily statistics

### 4. Service Layer

**Purpose**: Business logic and orchestration

**Services**:
- `TelegramService`: CRUD operations for telegrams
- `MessageService`: Message processing logic

### 5. Infrastructure Layer

#### NATS JetStream
- **Stream**: `TELEGRAMS`
- **Subject**: `Telegram.Serial`
- **Retention**: 7 days
- **Storage**: File-based

#### Database Connection Pool
- **Max Connections**: 25
- **Min Connections**: 5
- **Max Lifetime**: 1 hour
- **Health Check**: Every 1 minute

#### Redis Cache (Optional)
- Used for hot data caching
- Can be disabled via configuration

## Data Flow

### Message Ingestion Flow
```
NATS JetStream → Worker → Parser → Validator → Repository → TimescaleDB
                                      │
                                      └──► Publisher → NATS (parsed messages)
```

### Query Flow
```
Client → API → Service → Repository → TimescaleDB → Response
```

## Deployment Architecture

### Development
- Single instance of each service
- Local PostgreSQL with TimescaleDB
- Local NATS with JetStream
- Optional Redis

### Production
- Multiple worker instances (horizontal scaling)
- API service behind load balancer
- PostgreSQL with replication
- NATS cluster
- Redis cluster (if enabled)
- Prometheus + Grafana for monitoring

## Security Considerations

1. **Database**: Use SSL connections in production
2. **API**: Implement authentication/authorization (future)
3. **NATS**: Use TLS for connections (future)
4. **Secrets**: Use environment variables, not config files
5. **Input Validation**: Validate all inputs
6. **Rate Limiting**: Implement rate limiting (future)

## Performance Considerations

1. **Database**:
   - Connection pooling
   - Query optimization
   - Index usage
   - Compression for old data

2. **NATS**:
   - Batch processing
   - Consumer groups for load balancing
   - Stream retention policies

3. **API**:
   - Response caching (Redis)
   - Pagination
   - Query optimization

## Monitoring and Observability

### Metrics (Prometheus)
- HTTP request count and duration
- Message processing count and duration
- Database query duration
- NATS message count
- Connection pool metrics

### Logging
- Structured logging with zap
- Request/response logging
- Error logging with stack traces

### Health Checks
- `/health` - Basic health
- `/health/ready` - Readiness (database, cache)
- `/health/live` - Liveness

## Scalability

### Horizontal Scaling
- **Workers**: Scale by adding more worker instances
- **API**: Scale by adding more API instances behind load balancer
- **Database**: Use read replicas for queries

### Vertical Scaling
- Increase database connection pool
- Increase NATS stream size
- Increase API server resources

## Future Enhancements

1. **Authentication/Authorization**: JWT or OAuth2
2. **Rate Limiting**: Per-client rate limits
3. **WebSockets**: Real-time updates
4. **GraphQL**: Optional GraphQL endpoint
5. **Multi-tenancy**: Support for multiple organizations
6. **Advanced Analytics**: More aggregation views
7. **Message Replay UI**: Web interface for message replay

## References

- [ADR-001: Migration from Hasura to pgx](ADR/001-migration-from-hasura-to-pgx.md)
- [ADR-002: NATS JetStream Adoption](ADR/002-nats-jetstream-adoption.md)
- [ADR-003: TimescaleDB Integration](ADR/003-timescaledb-integration.md)
- [ADR-004: REST API Design](ADR/004-rest-api-design.md)

