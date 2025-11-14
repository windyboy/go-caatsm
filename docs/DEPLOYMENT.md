# Deployment Guide

This document captures repeatable deployment practices for the Aviation Telegram Processing System across local, staging, and production environments.

## 1. Build & Publish Container Images

```bash
# API
docker build -f docker/Dockerfile.api -t ghcr.io/your-org/go-caatsm-api:latest .

# Worker
docker build -f docker/Dockerfile.worker -t ghcr.io/your-org/go-caatsm-worker:latest .

docker push ghcr.io/your-org/go-caatsm-api:latest
docker push ghcr.io/your-org/go-caatsm-worker:latest
```

Tag images with the semantic version used for releases (see `docs/RELEASING.md`).

## 2. Provision Dependencies

| Component | Notes |
| --- | --- |
| PostgreSQL / TimescaleDB | Enable TLS, create database `aviation`, run migrations in `migrations/`. Create roles `caatsm_api` (SELECT) and `caatsm_worker` (INSERT/UPDATE). |
| NATS + JetStream | Enable TLS/auth. Create stream `TELEGRAMS` (source) and `Telegram.Json` (downstream). Configure MaxDeliver, retention, and a dead-letter stream. |
| Redis (optional) | Required only if caching is enabled. Use password-authenticated or ACL-protected instances. |
| Object storage/logging | Point zap lumberjack output at your log shipper or mount a volume. |

## 3. Configuration & Secrets

1. Copy `configs/config.dev.toml` to an environment-specific file.
2. Override sensitive values via environment variables prefixed with `CAATSM_`.
3. Store credentials in Vault, AWS/GCP Secret Manager, or Kubernetes Secrets and mount them as env vars at runtime.
4. Set `database.ssl_mode=require` and provide TLS certs/keys for API endpoints when exposed publicly.

## 4. Deployment Targets

### Docker Compose (staging/local)

Extend `docker/docker-compose.yml` with environment-specific overrides:

```yaml
services:
  api:
    image: ghcr.io/your-org/go-caatsm-api:${VERSION}
    environment:
      - GO_ENV=prod
      - CAATSM_DATABASE_URL=${DATABASE_URL}
      - CAATSM_NATS_URL=${NATS_URL}
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health/ready"]
      interval: 10s
      retries: 5
  worker:
    image: ghcr.io/your-org/go-caatsm-worker:${VERSION}
    depends_on:
      - nats
```

### Kubernetes / Helm

If Helm is preferred, model the deployment around:

- `Deployment` per service with liveness (`/health/live`) and readiness (`/health/ready`) probes.
- `ConfigMap` for non-sensitive TOML configuration.
- `Secret` objects for database/NATS credentials and TLS material.
- `HorizontalPodAutoscaler` keyed to message throughput or CPU usage on the worker.
- `PodDisruptionBudget` to keep at least one consumer running during maintenance.

Example probe configuration:

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  periodSeconds: 10
```

## 5. Observability

- Scrape `/metrics` from both services. Alert on `message_processing_failures_total`, `message_processing_retries_total`, and JetStream consumer lag.
- Forward zap logs to an aggregator (ELK, Loki, Splunk). Each log line contains the `message ID` making end-to-end tracing straightforward.
- Expose dashboards for:
  - Processing latency buckets (`message_processing_duration_seconds`)
  - Worker batch timings (`worker_batch_duration_seconds`)
  - Database pool saturation (`database_connections_active`)
  - NATS publish/consume deltas

## 6. Failover & Backpressure

- Tune JetStream `MaxDeliver` and `AckWait` to match your retry policy. The default consumer in `internal/infrastructure/nats` sets `MaxDeliver=5`.
- Configure a dead-letter stream to capture poison messages after max retries.
- Use `message_processing_retries_total` as an early-warning signal for downstream failures (database outages, schema errors, etc.).

## 7. Rolling Upgrades

1. Deploy database migrations in `migrations/` (see `docs/UPGRADE.md`) before rolling the worker.
2. For the worker, drain JetStream consumers (`nats consumer info ... --ack-pending`) before terminating pods to avoid duplicate redelivery.
3. Roll API pods next, ensuring readiness probes pass before scaling down old replicas.
4. Monitor metrics/alerts for at least one processing interval before promoting the release.

## 8. Troubleshooting Checklist

| Symptom | Remediation |
| --- | --- |
| `publish error: ...` | Verify JetStream stream and subject exist, TLS certs are valid, and credentials have publish ACLs. |
| `repository error: failed to insert` | Ensure migrations ran, table exists, and DB user has INSERT privileges. |
| `/health/ready` returns 503 | Inspect dependency statuses in the JSON payload. Recreate DB pools or NATS connections as needed. |
| JetStream backlog increasing | Scale worker replicas, review `batch_size`/`batch_timeout`, or slow inbound publishers until backlog stabilizes. |

For advanced scenarios (geo-replication, multi-tenant isolation), capture the requirements in additional ADRs before diverging from this baseline.

