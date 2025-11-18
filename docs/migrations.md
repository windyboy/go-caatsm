# Database Migrations Guide

This document describes the database schema management and migration strategy for the CAATSM application.

## Current Approach

The application currently uses DDL (Data Definition Language) files for schema management:

- **Schema file**: `internal/infra/postgres/telegrams.ddl`
- **Manual execution**: Schema changes are applied manually using `psql` or similar tools
- **Version control**: DDL files are version-controlled in the repository

### Current Schema Structure

The application uses TimescaleDB (PostgreSQL extension) with the following key components:

- **Schema**: `aviation`
- **Main table**: `aviation.telegrams` (hypertable for time-series data)
- **Raw table**: `aviation.telegrams_raw` (for unparsed/failed messages)
- **Indexes**: Multiple indexes on key fields for query performance

## Recommended Migration Tools

For production deployments, we recommend using a dedicated migration tool for better schema management:

### Option 1: golang-migrate (Recommended)

[golang-migrate](https://github.com/golang-migrate/migrate) is a popular Go-based migration tool with excellent PostgreSQL support.

**Installation:**
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**Setup:**
1. Create migrations directory:
   ```bash
   mkdir -p migrations
   ```

2. Create initial migration from existing schema:
   ```bash
   migrate create -ext sql -dir migrations -seq initial_schema
   ```

3. Copy DDL content to migration files:
   - `migrations/000001_initial_schema.up.sql` - Create schema
   - `migrations/000001_initial_schema.down.sql` - Drop schema

**Usage:**
```bash
# Apply all migrations
migrate -path migrations -database "postgres://user:pass@localhost:5432/aviation?sslmode=disable" up

# Rollback last migration
migrate -path migrations -database "postgres://user:pass@localhost:5432/aviation?sslmode=disable" down 1

# Check migration version
migrate -path migrations -database "postgres://user:pass@localhost:5432/aviation?sslmode=disable" version
```

### Option 2: migrate (by golang-migrate, different package)

Similar to golang-migrate but distributed as a separate package.

### Option 3: Custom Migration Scripts

For simple deployments, you can create custom migration scripts that:
- Check current schema version
- Apply migrations sequentially
- Track migration state in a `schema_migrations` table

## Migration Workflow

### Development

1. **Create migration file:**
   ```bash
   migrate create -ext sql -dir migrations -seq add_new_column
   ```

2. **Write up migration** (`migrations/XXXXXX_add_new_column.up.sql`):
   ```sql
   ALTER TABLE aviation.telegrams
   ADD COLUMN new_field TEXT;
   
   CREATE INDEX idx_telegrams_new_field ON aviation.telegrams (new_field);
   ```

3. **Write down migration** (`migrations/XXXXXX_add_new_column.down.sql`):
   ```sql
   DROP INDEX IF EXISTS idx_telegrams_new_field;
   ALTER TABLE aviation.telegrams
   DROP COLUMN IF EXISTS new_field;
   ```

4. **Test migration:**
   ```bash
   # Apply
   migrate -path migrations -database "$DATABASE_URL" up
   
   # Rollback
   migrate -path migrations -database "$DATABASE_URL" down 1
   ```

### Production

1. **Backup database** before applying migrations:
   ```bash
   pg_dump -U postgres -d aviation > backup_$(date +%Y%m%d_%H%M%S).sql
   ```

2. **Test migration on staging** environment first

3. **Apply migration** during maintenance window:
   ```bash
   migrate -path migrations -database "$DATABASE_URL" up
   ```

4. **Verify migration** success:
   ```bash
   migrate -path migrations -database "$DATABASE_URL" version
   ```

5. **Monitor application** for any issues

## Schema Evolution Best Practices

### 1. Backward Compatibility

- **Additive changes** (new columns, indexes) are generally safe
- **Removing columns** requires application code changes first
- **Changing column types** requires careful planning and data migration

### 2. TimescaleDB Considerations

- **Hypertables**: Be careful when modifying hypertable structure
- **Retention policies**: Consider impact on existing data
- **Compression**: Test compression policies with schema changes

### 3. Index Management

- **Create indexes concurrently** in production to avoid locking:
  ```sql
  CREATE INDEX CONCURRENTLY idx_telegrams_new_field ON aviation.telegrams (new_field);
  ```

- **Drop unused indexes** to improve write performance

### 4. Data Migrations

For data transformations, use separate migration steps:

1. **Add new column** (nullable)
2. **Backfill data** in application or migration script
3. **Add constraints** (NOT NULL, etc.) after backfill
4. **Remove old column** in separate migration

### 5. Rollback Procedures

Always provide rollback migrations:

- **Test rollback** on staging before production
- **Document rollback steps** in migration comments
- **Consider data loss** implications of rollbacks

## Example Migration

### Adding a New Index

**Up migration:**
```sql
-- Add index for querying by category and date
CREATE INDEX CONCURRENTLY idx_telegrams_category_date 
ON aviation.telegrams (category, received_at DESC);
```

**Down migration:**
```sql
-- Remove index
DROP INDEX IF EXISTS idx_telegrams_category_date;
```

### Adding a New Column

**Up migration:**
```sql
-- Add processing_metadata column for additional metadata
ALTER TABLE aviation.telegrams
ADD COLUMN processing_metadata JSONB;

-- Add index for JSONB queries
CREATE INDEX CONCURRENTLY idx_telegrams_processing_metadata_gin
ON aviation.telegrams USING GIN (processing_metadata);
```

**Down migration:**
```sql
-- Remove index and column
DROP INDEX IF EXISTS idx_telegrams_processing_metadata_gin;
ALTER TABLE aviation.telegrams
DROP COLUMN IF EXISTS processing_metadata;
```

## Migration State Management

### Schema Version Tracking

Migration tools typically use a `schema_migrations` table to track applied migrations:

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT NOT NULL PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);
```

### Checking Migration Status

```bash
# Check current version
migrate -path migrations -database "$DATABASE_URL" version

# Check for pending migrations
migrate -path migrations -database "$DATABASE_URL" up
```

## CI/CD Integration

### Automated Migration Testing

Add migration tests to CI pipeline:

```yaml
# Example GitHub Actions workflow
- name: Test migrations
  run: |
    # Start test database
    docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=test postgres:15
    
    # Wait for database
    sleep 5
    
    # Apply migrations
    migrate -path migrations -database "postgres://postgres:test@localhost:5432/test?sslmode=disable" up
    
    # Verify schema
    psql "postgres://postgres:test@localhost:5432/test?sslmode=disable" -c "\d aviation.telegrams"
```

### Deployment Automation

For production deployments, integrate migrations into deployment process:

1. **Pre-deployment**: Backup database
2. **Deployment**: Apply migrations
3. **Post-deployment**: Verify migration success
4. **Rollback**: If migration fails, rollback application and database

## Troubleshooting

### Migration Failures

**Common issues:**
- **Lock conflicts**: Use `CONCURRENTLY` for index creation
- **Timeout errors**: Increase migration timeout for large tables
- **Dirty state**: Manually fix `schema_migrations` table if migration fails mid-way

**Recovery:**
```sql
-- Check migration state
SELECT * FROM schema_migrations;

-- Fix dirty state (if needed)
UPDATE schema_migrations SET dirty = false WHERE version = X;
```

### Performance Considerations

- **Large tables**: Test migrations on production-sized data
- **Downtime**: Plan for maintenance windows for major schema changes
- **Replication lag**: Consider impact on read replicas

## Future Improvements

Consider implementing:

1. **Automated migration testing** in CI/CD
2. **Migration rollback automation** in deployment pipeline
3. **Schema validation** before applying migrations
4. **Migration dry-run** mode for testing
5. **Migration status monitoring** and alerting

## References

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [TimescaleDB Best Practices](https://docs.timescale.com/timescaledb/latest/how-to-guides/migrate-data/)
- [PostgreSQL Migration Guide](https://www.postgresql.org/docs/current/ddl-alter.html)

