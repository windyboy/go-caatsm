# Performance Tuning Guide

This document provides guidelines for optimizing the performance of the CAATSM application.

## Performance Metrics

Key performance indicators to monitor:

- **Message throughput**: Messages processed per second
- **Latency**: End-to-end processing time (NATS receive → DB insert → publish)
- **Database query time**: Time spent on database operations
- **Memory usage**: Application memory consumption
- **CPU usage**: CPU utilization
- **Connection pool utilization**: Database and NATS connection usage

## Batch Processing Configuration

### Current Implementation

The application processes messages in batches for efficiency:

```toml
[app]
batch_size = 50          # Number of messages per batch
batch_timeout = "2s"     # Maximum wait time for a batch
monitor_interval = "30s" # Consumer metrics reporting interval
```

### Tuning Guidelines

**Batch Size:**
- **Small batches (10-50)**: Lower latency, higher overhead
- **Medium batches (50-200)**: Balanced latency and throughput
- **Large batches (200-1000)**: Higher throughput, higher latency

**Recommendations:**
- **Development**: 10-50 messages (faster feedback)
- **Production (low latency)**: 50-100 messages
- **Production (high throughput)**: 200-500 messages

**Batch Timeout:**
- **Low latency**: 500ms-1s (process quickly even with small batches)
- **Balanced**: 2-5s (good balance)
- **High throughput**: 5-10s (wait for larger batches)

### Example Configuration

```toml
# High-throughput production configuration
[app]
batch_size = 200
batch_timeout = "5s"
monitor_interval = "30s"

# Low-latency production configuration
[app]
batch_size = 50
batch_timeout = "1s"
monitor_interval = "30s"
```

## Database Connection Pooling

### Configuration

```toml
[postgres]
url = "postgres://user:pass@db:5432/aviation?sslmode=require"
max_conns = 20    # Maximum connections in pool
min_conns = 5     # Minimum connections in pool
```

### Tuning Guidelines

**Connection Pool Size:**
- **Formula**: `max_conns = (expected_concurrent_requests * avg_query_time) / target_latency`
- **Minimum**: 2-5 connections (small deployments)
- **Recommended**: 10-20 connections (medium deployments)
- **Maximum**: 50-100 connections (high-throughput deployments)

**Considerations:**
- Each connection consumes memory (~2-5MB)
- PostgreSQL has a maximum connection limit (default: 100)
- Too many connections can degrade performance
- Use connection pooler (PgBouncer) for high concurrency

### Example Configurations

```toml
# Small deployment (single instance)
[postgres]
max_conns = 10
min_conns = 2

# Medium deployment (2-3 instances)
[postgres]
max_conns = 20
min_conns = 5

# Large deployment (5+ instances, use PgBouncer)
[postgres]
max_conns = 10  # Per instance
min_conns = 2
# Use PgBouncer with pool_mode=transaction
```

### Connection Pool Monitoring

Monitor connection pool metrics:
- Active connections
- Idle connections
- Connection wait time
- Connection errors

## Database Indexing Strategy

### Current Indexes

The application creates indexes on key fields:

```sql
CREATE INDEX idx_telegrams_message_id ON aviation.telegrams (message_id);
CREATE INDEX idx_telegrams_date_time ON aviation.telegrams (date_time);
CREATE INDEX idx_telegrams_priority_indicator ON aviation.telegrams (priority_indicator);
CREATE INDEX idx_telegrams_primary_address ON aviation.telegrams (primary_address);
CREATE INDEX idx_telegrams_received_at ON aviation.telegrams (received_at);
CREATE INDEX idx_telegrams_uuid ON aviation.telegrams (uuid);
```

### Index Optimization

**Query Patterns:**
- **Time-range queries**: Index on `received_at` (already exists)
- **Message lookup**: Index on `message_id` (already exists)
- **Category filtering**: Consider index on `category` if frequently queried
- **Composite indexes**: For multi-column queries

**Example Composite Index:**
```sql
-- For queries filtering by category and date range
CREATE INDEX idx_telegrams_category_received_at 
ON aviation.telegrams (category, received_at DESC);
```

### Index Maintenance

- **Monitor index usage**: Use `pg_stat_user_indexes` to identify unused indexes
- **Rebuild indexes**: Periodically rebuild indexes to reduce bloat
- **Concurrent creation**: Use `CREATE INDEX CONCURRENTLY` in production

## NATS JetStream Performance Tuning

### Stream Configuration

```toml
[nats.stream_limits]
max_msgs = 1000000      # Maximum messages in stream
max_bytes = 1073741824  # Maximum size (1GB)
max_age = "168h"        # Retention period (7 days)
discard = "old"         # Discard policy
storage = "file"        # Storage type (file or memory)
replicas = 3            # Number of replicas
```

### Tuning Guidelines

**Storage Type:**
- **File storage**: Persistent, slower (recommended for production)
- **Memory storage**: Faster, ephemeral (suitable for high-throughput temporary streams)

**Replicas:**
- **Single node**: 1 replica (development)
- **Production**: 3+ replicas (high availability)

**Retention:**
- **Short retention**: Lower storage, faster cleanup
- **Long retention**: More storage, replay capability

### Consumer Configuration

```toml
[nats.consumer_rules]
max_deliver = 5              # Maximum redelivery attempts
ack_wait = "30s"             # ACK wait time
max_ack_pending = 1024       # Maximum unacknowledged messages
deliver_policy = "new"       # Delivery policy
backoff = ["5s", "30s", "2m"] # Retry delays
```

**Tuning:**
- **ack_wait**: Set based on processing time (processing_time * 2-3)
- **max_ack_pending**: Increase for high-throughput (1024-4096)
- **backoff**: Adjust based on failure patterns

## Memory Optimization

### Garbage Collection Tuning

Set Go GC environment variables for production:

```bash
# Balanced GC (default)
export GOGC=100

# Aggressive GC (lower memory, higher CPU)
export GOGC=50

# Conservative GC (higher memory, lower CPU)
export GOGC=200
```

### Memory Profiling

Use `pprof` to identify memory issues:

```bash
# Enable memory profiling
go tool pprof http://localhost:2112/debug/pprof/heap

# Generate memory profile
go tool pprof -alloc_space http://localhost:2112/debug/pprof/heap
```

## CPU Optimization

### Goroutine Management

- **Limit goroutines**: Use worker pools for concurrent processing
- **Context cancellation**: Properly cancel goroutines to prevent leaks
- **Monitor goroutine count**: Use `runtime.NumGoroutine()`

### CPU Profiling

```bash
# Enable CPU profiling
go tool pprof http://localhost:2112/debug/pprof/profile

# 30-second CPU profile
go tool pprof http://localhost:2112/debug/pprof/profile?seconds=30
```

## Monitoring and Profiling

### Prometheus Metrics

Key metrics to monitor:

- `caatsm_messages_total`: Message throughput
- `caatsm_handle_latency_seconds`: Processing latency
- `caatsm_db_query_latency_seconds`: Database query time
- `caatsm_nats_consumer_pending_messages`: Consumer lag

### Grafana Dashboards

Create dashboards for:
- Message throughput over time
- Latency percentiles (P50, P95, P99)
- Error rates
- Resource utilization (CPU, memory, connections)

### Profiling Endpoints

The application exposes profiling endpoints (if enabled):

```bash
# Heap profile
curl http://localhost:2112/debug/pprof/heap > heap.prof

# CPU profile
curl http://localhost:2112/debug/pprof/profile?seconds=30 > cpu.prof

# Goroutine profile
curl http://localhost:2112/debug/pprof/goroutine > goroutine.prof
```

## Performance Testing

### Load Testing

Use tools like `k6`, `wrk`, or `vegeta` for load testing:

```bash
# Example: Generate load with seed-telegrams
go run ./cmd/seed-telegrams \
  --count=10000 \
  --mode=burst \
  --category=mixed
```

### Benchmark Tests

Run built-in benchmarks:

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkParser -benchmem ./internal/adapter/parser
```

### Performance Baselines

Establish performance baselines:
- **Throughput**: Messages per second
- **Latency**: P50, P95, P99 percentiles
- **Resource usage**: CPU, memory, connections

## Optimization Checklist

### Application Level

- [ ] Optimize batch size for workload
- [ ] Tune connection pool sizes
- [ ] Review and optimize database queries
- [ ] Add missing indexes for query patterns
- [ ] Enable query result caching where appropriate

### Infrastructure Level

- [ ] Use connection pooler (PgBouncer) for high concurrency
- [ ] Configure database connection limits appropriately
- [ ] Use read replicas for query-heavy workloads
- [ ] Optimize NATS JetStream stream configuration
- [ ] Scale horizontally (multiple instances)

### Monitoring

- [ ] Set up performance dashboards
- [ ] Configure alerts for performance degradation
- [ ] Regular performance profiling
- [ ] Monitor resource utilization
- [ ] Track performance trends over time

## Troubleshooting Performance Issues

### High Latency

**Symptoms:**
- Slow message processing
- High P95/P99 latencies

**Investigation:**
1. Check database query times
2. Review NATS consumer lag
3. Profile CPU and memory usage
4. Check for connection pool exhaustion

**Solutions:**
- Optimize slow database queries
- Increase batch size
- Add database indexes
- Scale horizontally

### Low Throughput

**Symptoms:**
- Low messages per second
- High CPU usage

**Investigation:**
1. Check for bottlenecks (DB, NATS, CPU)
2. Review batch processing configuration
3. Profile application code

**Solutions:**
- Increase batch size
- Optimize hot code paths
- Scale horizontally
- Use connection pooling

### High Memory Usage

**Symptoms:**
- Memory leaks
- High memory consumption

**Investigation:**
1. Heap profiling
2. Check for goroutine leaks
3. Review batch sizes

**Solutions:**
- Fix memory leaks
- Reduce batch sizes
- Tune GC settings
- Limit concurrent operations

## References

- [Go Performance Best Practices](https://go.dev/doc/effective_go#performance)
- [PostgreSQL Performance Tuning](https://www.postgresql.org/docs/current/performance-tips.html)
- [TimescaleDB Performance Tuning](https://docs.timescale.com/timescaledb/latest/how-to-guides/performance/)
- [NATS JetStream Performance](https://docs.nats.io/nats-concepts/jetstream/performance)
- [Go Profiling Guide](https://go.dev/blog/pprof)

