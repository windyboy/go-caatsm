# Upgrade Guide

## Overview
This guide describes the steps to upgrade from the legacy Hasura-based deployment to the current pgx + TimescaleDB + JetStream stack. Hasura support has been fully removed from the codebase, so following this guide ensures your environment, configuration, and data pipelines match the new default architecture.

## Prerequisites
- Go 1.22.5 or later
- PostgreSQL 14+ with TimescaleDB extension
- NATS Server 2.9+ with JetStream enabled
- Docker and Docker Compose (for containerized deployment)

## Pre-Upgrade Checklist
- [ ] Backup existing database
- [ ] Backup existing configuration files
- [ ] Review and understand breaking changes
- [ ] Test upgrade in staging environment
- [ ] Prepare rollback plan

## Database Migration

### 1. Install TimescaleDB Extension
```sql
CREATE EXTENSION IF NOT EXISTS timescaledb;
```

### 2. Run Migration Scripts
```bash
# Using golang-migrate
migrate -path migrations -database "postgres://user:password@localhost/aviation?sslmode=disable" up

# Or manually
psql -U postgres -d aviation -f migrations/001_create_hypertable.up.sql
psql -U postgres -d aviation -f migrations/002_add_continuous_aggregate.up.sql
```

### 3. Verify Migration
```sql
-- Check hypertable
SELECT * FROM timescaledb_information.hypertables;

-- Check continuous aggregates
SELECT * FROM timescaledb_information.continuous_aggregates;
```

## Configuration Changes

### Removed Configuration
- `[hasura]` section - No longer needed

### New Configuration
Update `configs/config.dev.toml`:

```toml
[database]
host = "localhost"
port = 5432
user = "postgres"
password = "postgres"
database = "aviation"
ssl_mode = "disable"
max_conns = 25
min_conns = 5
max_conn_lifetime = "1h"

[nats.jetstream]
enabled = true
stream_name = "TELEGRAMS"
subject = "Telegram.Serial"
max_pending = 256
auto_provision = true

[api]
host = "0.0.0.0"
port = 8080
mode = "dev"

[redis]
addr = "localhost:6379"
password = ""
db = 0
enabled = false
```

## Code Changes

### Dependencies
Run `go mod tidy` to update dependencies. New dependencies include:
- `github.com/jackc/pgx/v5`
- `github.com/labstack/echo/v4`
- `github.com/prometheus/client_golang`
- `github.com/redis/go-redis/v9`

### Environment Variables
- Remove: `GRAPHQL_TOKEN` (no longer needed)
- Add: Database connection environment variables (optional, can use config file)

## Deployment Steps

### 1. Stop Existing Services
```bash
# Stop Hasura
docker-compose down hasura

# Stop old worker
pkill -f "go-caatsm"
```

### 2. Deploy New Services
```bash
# Build new binaries
go build -o bin/api cmd/api/main.go
go build -o bin/worker cmd/worker/main.go

# Or use Docker
docker-compose up -d
```

### 3. Verify Deployment
```bash
# Check API health
curl http://localhost:8080/health

# Check worker logs
tail -f logs/worker.log
```

## Data Migration

If you have existing data in Hasura/PostgreSQL:

### Option 1: Export and Import
```bash
# Export from old database
pg_dump -U postgres -d old_database -t aviation_telegrams > backup.sql

# Import to new database
psql -U postgres -d aviation < backup.sql
```

### Option 2: Direct Migration
```sql
-- If table structure is similar, you can use INSERT ... SELECT
INSERT INTO aviation.telegrams 
SELECT * FROM old_schema.aviation_telegrams;
```

## Rollback Plan

### If Upgrade Fails

1. **Stop New Services**
   ```bash
   docker-compose down
   ```

2. **Restore Database Backup**
   ```bash
   psql -U postgres -d aviation < backup.sql
   ```

3. **Revert Configuration**
   ```bash
   git checkout configs/config.dev.toml
   ```

4. **Restart Old Services**
   ```bash
   # Restore old deployment
   ```

## Post-Upgrade Verification

### 1. Verify API Endpoints
```bash
# Health check
curl http://localhost:8080/health

# List telegrams
curl http://localhost:8080/api/v1/telegrams

# Get specific telegram
curl http://localhost:8080/api/v1/telegrams/{uuid}
```

### 2. Verify Message Processing
- Send test message to NATS
- Verify message is stored in database
- Verify message appears in API

### 3. Verify Metrics
```bash
curl http://localhost:8080/metrics
```

## Common Issues

### Issue: TimescaleDB extension not found
**Solution**: Install TimescaleDB extension:
```sql
CREATE EXTENSION IF NOT EXISTS timescaledb;
```

### Issue: NATS JetStream not enabled
**Solution**: Enable JetStream in NATS configuration:
```yaml
jetstream {
  store_dir: "/data/jetstream"
}
```

### Issue: Database connection errors
**Solution**: Check database credentials and connection string in config file.

### Issue: Migration fails
**Solution**: Check PostgreSQL logs and ensure TimescaleDB extension is installed.

## Performance Tuning

### Database
- Adjust connection pool size based on load
- Monitor query performance
- Review and optimize indexes

### NATS JetStream
- Adjust stream retention policy
- Monitor stream size
- Configure compression policy

### API
- Configure rate limiting if needed
- Monitor response times
- Scale horizontally if needed

## Support

For issues or questions:
1. Check logs: `logs/api.log` and `logs/worker.log`
2. Review documentation: `docs/ARCHITECTURE.md`
3. Check ADRs: `docs/ADR/`

