# ADR-003: TimescaleDB Integration

## Status
Accepted

## Context
The system processes aviation telegram messages which are inherently time-series data:
- Messages arrive continuously over time
- Queries are often time-range based
- Historical data needs to be retained for analysis
- Storage efficiency is important for long-term retention
- Aggregations over time periods are common

Standard PostgreSQL, while capable, is not optimized for time-series workloads:
- No automatic partitioning by time
- No built-in compression for time-series data
- No continuous aggregates
- Less efficient queries for time-range operations

## Decision
We will use TimescaleDB, a PostgreSQL extension designed for time-series data, as our primary database.

## Consequences

### Positive
- **Performance**: Optimized for time-series queries with automatic partitioning
- **Storage Efficiency**: Built-in compression can reduce storage by 90%+
- **Continuous Aggregates**: Pre-computed aggregations for faster queries
- **Hypertables**: Automatic time-based partitioning
- **PostgreSQL Compatibility**: Full PostgreSQL feature set
- **Query Optimization**: Better query plans for time-range operations

### Negative
- **Dependency**: Additional extension to install and maintain
- **Learning Curve**: Team needs to understand TimescaleDB concepts
- **Migration**: Need to migrate existing data (if any)
- **Backup/Restore**: Slightly more complex backup procedures

### Mitigation
- Use Docker images with TimescaleDB pre-installed
- Provide comprehensive documentation
- Create migration scripts for data migration
- Document backup and restore procedures

## Implementation Notes
- Main table will be converted to a hypertable partitioned by `received_at`
- Compression policy will be set for data older than 7 days
- Continuous aggregates will be created for hourly and daily statistics
- Chunk time interval will be set to 1 day
- Indexes will be created on commonly queried fields

