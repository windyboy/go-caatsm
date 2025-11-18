# Production Deployment Guide

This guide covers deploying and running the CAATSM application in production environments.

## Overview

Production deployments require:
- **JetStream mode** (mandatory) - for message reliability and persistence
- **Manual Stream/Consumer creation** - not auto-created in production
- **Proper configuration** - using `configs/config.prod.toml`
- **SSL/TLS connections** - for secure database and telemetry connections
- **High availability** - stream replicas set to 3+ for HA

## Prerequisites

Before deploying to production:

1. **Build the binary:**
   ```bash
   make build
   # Binary will be at bin/receiver
   ```

2. **NATS JetStream cluster** - must be running and accessible
3. **PostgreSQL/TimescaleDB** - database must be accessible with SSL
4. **OpenTelemetry collector** (optional) - for observability
5. **Prometheus** (optional) - for metrics scraping

## Configuration

### 1. Production Configuration File

The repository includes a production configuration template at `configs/config.prod.toml`. Copy and customize it:

```bash
cp configs/config.prod.toml configs/config.prod.toml.local
# Edit config.prod.toml.local with your production values
```

**Key configuration sections:**

```toml
[nats]
url = "nats://nats.prod:4222"
mode = "jetstream"  # Production MUST use JetStream
stream = "TELEGRAM"
consumer = "telegram-consumer"

[nats.stream_limits]
max_msgs = 1000000      # Adjust based on requirements
max_bytes = 1073741824  # 1GB
max_age = "168h"        # 7 days
discard = "old"
storage = "file"
replicas = 3            # Use 3+ for HA in production

[nats.consumer_rules]
max_deliver = 5
ack_wait = "30s"
max_ack_pending = 1024
deliver_policy = "new"  # Start from new messages in production
replay_policy = "instant"
backoff = ["5s", "30s", "2m", "5m"]

[subscription]
topic = "telegram.serial"
queue_group = "tele-queue"

[postgres]
url = "postgres://user:password@db.prod:5432/aviation?sslmode=require"
max_conns = 20
min_conns = 5

[app]
batch_size = 100
batch_timeout = "2s"
monitor_interval = "30s"

[log]
level = "info"
format = "json"
output = ["stdout"]

[telemetry]
enabled = true
endpoint = "otel-collector.prod:4318"
insecure = false  # Use TLS in production

[monitoring]
disabled = false
addr = ":2112"
enable_metrics = true
enable_health = true

[dlq]
enabled = true
subject = "caatsm.dlq"
```

### 2. Environment Variables

Alternatively, you can use environment variables instead of a config file:

```bash
export GO_ENV=prod
export CAATSM_NATS_URL=nats://nats.prod:4222
export CAATSM_NATS_MODE=jetstream
export CAATSM_POSTGRES_URL=postgres://user:pass@db:5432/aviation?sslmode=require
export CAATSM_LOG_LEVEL=info
export CAATSM_TELEMETRY_ENABLED=true
export CAATSM_TELEMETRY_ENDPOINT=otel-collector.prod:4318
export CAATSM_TELEMETRY_INSECURE=false
```

## JetStream Setup

### Pre-create Stream and Consumer

**IMPORTANT:** The application does NOT auto-create streams/consumers in production. You must create them manually before starting the application.

#### Using NATS CLI

```bash
# Create stream
nats stream add TELEGRAM \
  --subjects "telegram.serial,telegram.json" \
  --storage file \
  --replicas 3 \
  --max-msgs 1000000 \
  --max-bytes 1GB \
  --max-age 7d \
  --discard old

# Create consumer
nats consumer add TELEGRAM telegram-consumer \
  --filter "telegram.serial" \
  --ack explicit \
  --deliver new \
  --max-deliver 5 \
  --ack-wait 30s \
  --max-pending 1024
```

#### Using NATS Management API

You can also create streams/consumers programmatically using the NATS management API or configuration files.

### Verify Setup

```bash
# Check stream exists
nats stream info TELEGRAM

# Check consumer exists
nats consumer info TELEGRAM telegram-consumer

# Test connection
nats pub telegram.serial "ZCZC TEST 150631..."
```

## Running the Application

### Using Make

```bash
make run-prod   # GO_ENV=prod (requires config.prod.toml)
```

### Using Task

```bash
task run-prod   # GO_ENV=prod (requires config.prod.toml)
```

### Direct Execution

```bash
# Using binary with config file
GO_ENV=prod ./bin/receiver listen

# Or with environment variables (no config file needed)
GO_ENV=prod \
CAATSM_NATS_URL=nats://nats.prod:4222 \
CAATSM_NATS_MODE=jetstream \
CAATSM_POSTGRES_URL=postgres://user:pass@db:5432/aviation?sslmode=require \
./bin/receiver listen
```

## Production Checklist

Before deploying to production, verify:

- ✅ `nats.mode = "jetstream"` in config (mandatory)
- ✅ Stream and Consumer created manually
- ✅ Stream replicas set to 3+ for high availability
- ✅ PostgreSQL connection configured with SSL (`sslmode=require`)
- ✅ Log level set to `info` or `warn` (not `debug`)
- ✅ Log format set to `json` for log aggregation
- ✅ Telemetry endpoint configured (if using observability)
- ✅ Telemetry TLS enabled (`insecure = false`)
- ✅ Monitoring endpoints exposed for Prometheus scraping
- ✅ DLQ enabled for poison message handling
- ✅ Appropriate retention limits configured (max_msgs, max_bytes, max_age)
- ✅ Connection pool sizes appropriate for workload
- ✅ Batch sizes tuned for throughput

## Monitoring and Observability

### Health Endpoints

The application exposes health and readiness endpoints:

- `GET /livez` - Liveness endpoint (process health)
- `GET /readyz` - Readiness endpoint (checks PostgreSQL and NATS)
- `GET /healthz` - Alias for `/readyz`
- `GET /metrics` - Prometheus metrics

Configure Prometheus to scrape metrics:

```yaml
scrape_configs:
  - job_name: 'caatsm'
    static_configs:
      - targets: ['caatsm:2112']
```

### Key Metrics

Monitor these metrics in production:

- `caatsm_messages_total{stream,consumer,result}` - Message throughput and results
- `caatsm_handle_latency_seconds_bucket` - Processing latency
- `caatsm_retries_total{stream,consumer,reason}` - Retry counts
- `caatsm_db_queries_total{operation,result}` - Database activity
- `caatsm_dlq_messages_total` - Dead-letter queue messages
- `caatsm_nats_consumer_pending_messages` - Consumer backlog/lag

### Logging

Production logs are in JSON format for easy parsing by log aggregation systems:

```json
{
  "level": "info",
  "ts": 1234567890.123,
  "caller": "nats/consumer.go:123",
  "msg": "Started consuming messages",
  "subject": "telegram.serial",
  "consumer": "telegram-consumer",
  "stream": "TELEGRAM"
}
```

## High Availability

### Multiple Instances

Run multiple instances of the application for high availability:

- All instances use the same durable consumer name
- JetStream distributes messages across instances
- Each instance independently fetches messages
- If an instance fails, others continue processing

### Stream Replication

Configure stream with 3+ replicas for HA:

```toml
[nats.stream_limits]
replicas = 3  # Minimum 3 for HA, 5 for better distribution
```

### Database Connection Pooling

Configure appropriate connection pool sizes:

```toml
[postgres]
max_conns = 20  # Adjust based on number of instances
min_conns = 5
```

## Troubleshooting

### Stream Not Found

**Error:** `stream TELEGRAM not found`

**Solution:** Create the stream manually before starting the application (see "JetStream Setup" above).

### Consumer Not Found

**Error:** `consumer telegram-consumer not found in stream TELEGRAM`

**Solution:** Create the consumer manually before starting the application (see "JetStream Setup" above).

### Messages Not Being Consumed

**Symptoms:** High pending count, no messages processed

**Check:**
1. Verify consumer exists: `nats consumer info TELEGRAM telegram-consumer`
2. Check pending messages: `nats consumer next TELEGRAM telegram-consumer`
3. Verify application is running and connected
4. Check logs for errors

**Solutions:**
- Increase `batch_size` if processing is slow
- Add more consumer instances
- Check for processing errors in logs

### High Pending Count

**Symptoms:** Consumer has many pending messages

**Solutions:**
- Increase `batch_size` in config
- Add more application instances
- Check processing latency
- Verify database performance

### Messages Being Redelivered

**Symptoms:** Same messages processed multiple times

**Check:**
- Processing logs for errors
- `ack_wait` timeout may be too short
- Processing may be taking longer than `ack_wait`

**Solutions:**
- Increase `ack_wait` if processing takes longer
- Fix processing errors
- Check database connection and performance

### Connection Issues

**NATS Connection:**
- Verify NATS server is accessible
- Check network connectivity
- Verify NATS URL in config

**PostgreSQL Connection:**
- Verify database is accessible
- Check SSL certificate configuration
- Verify connection string format
- Check firewall rules

## Deployment Options

### Systemd Deployment

See `docs/deploy-systemd.md` for a complete systemd service deployment example.

### Kubernetes Deployment

See `docs/deploy-k8s.md` for Kubernetes deployment with ConfigMap/Secret and health probes.

## Performance Tuning

### Batch Processing

Adjust batch size based on message size and processing time:

```toml
[app]
batch_size = 100      # Increase for higher throughput
batch_timeout = "2s"  # Adjust based on latency requirements
```

### Connection Pools

Tune database connection pool:

```toml
[postgres]
max_conns = 20  # Total connections across all instances
min_conns = 5   # Keep-alive connections
```

### Stream Retention

Configure retention based on requirements:

```toml
[nats.stream_limits]
max_msgs = 1000000      # Maximum messages
max_bytes = 1073741824  # Maximum size (1GB)
max_age = "168h"        # Maximum age (7 days)
```

### Consumer Settings

Tune consumer for your workload:

```toml
[nats.consumer_rules]
max_ack_pending = 1024  # Increase for higher throughput
ack_wait = "30s"        # Adjust based on processing time
backoff = ["5s", "30s", "2m", "5m"]  # Retry delays
```

## Security Considerations

1. **Use SSL/TLS** for all connections:
   - PostgreSQL: `sslmode=require`
   - Telemetry: `insecure = false`

2. **Secure secrets** - Use environment variables or secret management:
   - Database passwords
   - NATS credentials
   - API keys

3. **Network security**:
   - Use private networks for internal services
   - Restrict access to monitoring endpoints
   - Use firewall rules appropriately

4. **Logging** - Avoid logging sensitive data:
   - Don't log message payloads in production
   - Use appropriate log levels

## NATS Authentication

The application supports multiple NATS authentication methods for secure connections. Configure authentication in the `[nats.auth]` section of your production configuration.

### Authentication Methods

Only one authentication method can be used at a time. Choose the method that best fits your infrastructure:

#### 1. Token Authentication

Simple token-based authentication suitable for service-to-service communication:

```toml
[nats.auth]
token = "your-nats-token-here"
tls_enabled = true
```

**When to use:**
- Simple service-to-service authentication
- Single token shared across services
- Quick setup for development/staging

**Security considerations:**
- Tokens should be rotated regularly
- Store tokens securely (use secret management)
- Use TLS to encrypt token transmission

#### 2. Credentials File (Recommended)

NATS credentials file authentication provides fine-grained access control:

```toml
[nats.auth]
credentials_file = "/etc/caatsm/nats.creds"
tls_enabled = true
```

**When to use:**
- Production environments requiring fine-grained permissions
- Multiple services with different access levels
- Integration with NATS account system

**Setup:**
1. Generate credentials file using NATS CLI:
   ```bash
   nats account creds -n caatsm-service > /etc/caatsm/nats.creds
   ```
2. Ensure the file is readable by the application user
3. Set appropriate file permissions (e.g., `chmod 600 /etc/caatsm/nats.creds`)

#### 3. User/Password Authentication

Traditional username/password authentication:

```toml
[nats.auth]
user = "caatsm-service"
password = "secure-password-here"
tls_enabled = true
```

**When to use:**
- Legacy NATS server configurations
- Simple authentication requirements
- Integration with existing user management systems

**Security considerations:**
- Use strong, unique passwords
- Store passwords securely (use secret management)
- Rotate passwords regularly

### TLS Configuration

TLS encryption is **required** for production deployments. Configure TLS in the `[nats.auth]` section:

```toml
[nats.auth]
credentials_file = "/etc/caatsm/nats.creds"
tls_enabled = true
tls_cert_file = "/etc/caatsm/tls/client.crt"  # Optional: client certificate
tls_key_file = "/etc/caatsm/tls/client.key"   # Optional: client private key
tls_ca_file = "/etc/caatsm/tls/ca.crt"        # Optional: CA certificate for server verification
```

**TLS Options:**
- `tls_enabled`: Enable TLS encryption (required for production)
- `tls_cert_file`: Client certificate file path (for mutual TLS)
- `tls_key_file`: Client private key file path (for mutual TLS)
- `tls_ca_file`: CA certificate file for server certificate verification

**Note:** If `tls_ca_file` is not specified, the system's default CA certificates are used. For production, it's recommended to specify a CA file for explicit certificate validation. The application will load and use the CA certificate file for server verification when provided.

### Environment Variable Configuration

You can also configure authentication via environment variables:

```bash
# Token authentication
export CAATSM_NATS_AUTH_TOKEN="your-token"

# Credentials file
export CAATSM_NATS_AUTH_CREDENTIALS_FILE="/etc/caatsm/nats.creds"

# User/Password
export CAATSM_NATS_AUTH_USER="caatsm-service"
export CAATSM_NATS_AUTH_PASSWORD="secure-password"

# TLS
export CAATSM_NATS_AUTH_TLS_ENABLED="true"
export CAATSM_NATS_AUTH_TLS_CERT_FILE="/etc/caatsm/tls/client.crt"
export CAATSM_NATS_AUTH_TLS_KEY_FILE="/etc/caatsm/tls/client.key"
export CAATSM_NATS_AUTH_TLS_CA_FILE="/etc/caatsm/tls/ca.crt"
```

### Testing Authentication

After configuring authentication, verify the connection:

```bash
# Test connection with authentication
./bin/receiver listen --nats-url nats://nats.prod:4222

# Check logs for authentication success
# Look for: "NATS reconnected" or connection errors
```

### Troubleshooting

**Connection failures:**
- Verify authentication credentials are correct
- Check NATS server logs for authentication errors
- Ensure TLS certificates are valid and accessible
- Verify file permissions on credentials/certificate files

**Common errors:**
- `authentication failed`: Check token/credentials/user-password
- `tls: bad certificate`: Verify TLS certificate configuration
- `permission denied`: Check file permissions on credentials/certificate files

## Backup and Recovery

### Database Backups

Ensure regular backups of PostgreSQL/TimescaleDB:
- Use pg_dump or TimescaleDB backup tools
- Test restore procedures regularly

### JetStream State

JetStream state is stored in NATS:
- Ensure NATS cluster has proper backup procedures
- Stream data is replicated across cluster nodes
- Test disaster recovery procedures

### Message Replay

If needed, messages can be replayed from JetStream:

```bash
# Replay from a specific sequence
./bin/receiver listen --replay-from seq:12345

# Replay from a specific time
./bin/receiver listen --replay-from time:2024-11-15T08:00:00Z
```

## Support

For issues or questions:
- Check logs: `journalctl -u caatsm` (systemd) or container logs
- Review metrics in Prometheus/Grafana
- Check health endpoints: `curl http://localhost:2112/readyz`
- Consult deployment-specific documentation

