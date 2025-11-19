## Observability

### Metrics

The service exposes Prometheus metrics via the monitoring HTTP server (default `:2112`):

- `caatsm_messages_total{stream,consumer,result}`  
  Total number of messages handled by the receiver, labelled by stream/consumer and result (`ok`, `fail`, `permanent_fail`, `retry`).

- `caatsm_handle_latency_seconds{stream,consumer}`  
  End-to-end handling latency from NATS receive to handler completion.

- `caatsm_retries_total{stream,consumer,reason}`  
  Number of retries (NAKs) issued by the consumer, labelled by reason (e.g. `processor_error`).

- `caatsm_js_api_calls_total{operation}`  
  JetStream API calls performed by the service.

- `caatsm_db_queries_total{operation,result}`  
  Database operations grouped by operation (`insert_one`, `insert_batch`, `insert_raw`) and result (`ok`, `error`).

- `caatsm_db_query_latency_seconds{operation}`  
  DB operation latency.

- `caatsm_dlq_messages_total{stream,consumer}`  
  Count of messages successfully routed to the DLQ.

- `caatsm_dlq_publish_failures_total{stream,consumer}`  
  Count of failures when attempting to publish messages to the DLQ.

- `caatsm_publish_failures_total{category}`  
  Count of general publish failures (not DLQ-specific), labelled by message category.

- `caatsm_nats_consumer_pending_messages{stream,consumer}`  
  Current pending message count for each JetStream consumer (useful for lag/backlog alerts). A zero sample is emitted only once at startup so the series exists; if JetStream stats queries fail later, the last known value is preserved rather than force-setting the gauge to `0`, which prevents false “queue cleared” alerts.

Additional OTEL metrics are emitted via the configured OTEL endpoint, including:

- `caatsm_messages_processed_total{message_status,message_category}`  
  Total number of telegrams processed, labelled by status and category.

- `caatsm_parse_duration_seconds{message_status,message_category}`  
  Latency of parsing a telegram, in seconds, labelled by status and category.

- `caatsm_publish_failures_total{message_category}`  
  Total number of telegram publish failures, labelled by category (also available in Prometheus as `caatsm_publish_failures_total{category}`).

These metrics are intended to be scraped by Prometheus (either directly or via the OTEL collector) and visualised in Grafana dashboards. Recommended dashboard panels include:

- Per-stream/consumer message rate and error rate.  
- Handling latency P50/P95/P99.  
- NATS consumer backlog and redelivery counts.  
- DB query rates and latencies.

#### Prometheus scrape configuration

In the local dev environment, metrics are typically scraped by the Prometheus
container defined in `docker-compose.dev.yml` using `configs/prometheus.dev.yml`.

A recommended scrape configuration for the receiver is:

```yaml
scrape_configs:
  - job_name: "otel-collector"
    static_configs:
      - targets:
          - "otel-collector:8888"

  - job_name: "nats-exporter"
    static_configs:
      - targets:
          - "nats-exporter:7777"

  - job_name: "caatsm-receiver"
    static_configs:
      - targets:
          # go-caatsm running on host/WSL, Prometheus in Docker
          - "host.docker.internal:2112"
```

When you run the receiver directly on the host/WSL, ensure the monitoring
server listens on all interfaces so that Docker can reach it, for example via:

```bash
export CAATSM_MONITORING_ADDR=0.0.0.0:2112
export CAATSM_MONITORING_ENABLE_METRICS=true
export CAATSM_MONITORING_ENABLE_HEALTH=true
```

Alternative topologies:

- **Receiver and Prometheus in the same Docker network**  
  Expose the monitoring server via a container port and use the container
  name as the scrape target, e.g. `caatsm-receiver:2112`.
- **Receiver behind a reverse proxy / load balancer**  
  Point Prometheus at the proxy address and path that forwards to `/metrics`.

#### CAATSM – Receiver Overview Dashboard

The `caatsm-overview` Grafana dashboard (provisioned from `configs/grafana-dashboards.dev/caatsm-overview.json`) focuses on the CAATSM receiver service and surfaces:

**Prometheus Metrics (operational focus):**
- **Message throughput by result** – derived from `caatsm_messages_total{result}`.  
- **Per stream/consumer rates** – `caatsm_messages_total{stream,consumer}`.  
- **End-to-end handle latency** – P50/P95/P99 from `caatsm_handle_latency_seconds_bucket`.  
- **DB query rate and latency** – from `caatsm_db_queries_total` and `caatsm_db_query_latency_seconds_bucket`.  
- **Retry and permanent failure rates** – from `caatsm_retries_total` and `caatsm_messages_total{result="permanent_fail"}`.  
- **Publish failures (Prometheus)** – from `caatsm_publish_failures_total{category}`.
- **NATS consumer pending messages** – from `caatsm_nats_consumer_pending_messages`.

**OTEL Metrics (business focus, scraped from collector):**
- **Messages Processed** – `caatsm_messages_processed_total` by `message_status` and `message_category`.
- **Parse Duration** – P50/P95/P99 percentiles from `caatsm_parse_duration_seconds_bucket`.
- **Publish Failures (OTEL)** – `caatsm_publish_failures_total` by `message_category`.

All CAATSM metrics are consolidated in this dashboard for comprehensive service monitoring.

To validate that the dashboard is receiving data:

1. Check the monitoring endpoint directly:

   ```bash
   curl -s http://localhost:2112/metrics | grep caatsm_messages_total || true
   ```

2. In Prometheus (`http://localhost:9090`), run:

   ```text
   caatsm_messages_total
   ```

   and

   ```text
   rate(caatsm_messages_total[5m])
   ```

3. In Grafana, open the **CAATSM – Receiver Overview** dashboard and
   verify that:

   - “Messages by result (5m rate)” shows time series for `ok`, `fail`,
     and `permanent_fail`.
   - “Messages per stream/consumer” shows series labelled by `stream`
     and `consumer`.
   - DB-related panels show non-zero values based on
     `caatsm_db_queries_total` and `caatsm_db_query_latency_seconds`.

### Health and Readiness

The monitoring server exposes:

- `/livez` – lightweight liveness endpoint that reports process/build information without checking dependencies.  
- `/healthz` – backward-compatible health endpoint used by existing deploys; currently shares logic with `/readyz`.  
- `/readyz` – readiness endpoint that checks critical dependencies and should be used by load balancers / orchestrators.

Checks performed:

- PostgreSQL: `pgxpool.Pool.Ping` with configurable timeout (`monitoring.health_timeout`), reporting `status` and `latency_ms`.  
- NATS: connection status must be `CONNECTED`; otherwise the dependency is marked as unavailable.

Responses include build metadata and a dependency map, for example:

```json
{
  "status": "ok",
  "build": {
    "version": "v0.4.3",
    "rev": "abc1234",
    "built_at": "2025-11-16T08:35:00Z"
  },
  "dependencies": {
    "postgres": {"status": "ok", "latency_ms": 4},
    "nats": {"status": "CONNECTED"}
  }
}
```

A non-2xx response indicates the service is not healthy/ready and should be removed from traffic.

### Tracing

The application implements production-ready OpenTelemetry tracing with comprehensive span coverage and semantic attributes.

#### Configuration

Tracing is configured via the `telemetry` section:

- `telemetry.enabled` – enables OTEL exporters (default: `false` in dev, `true` in prod)
- `telemetry.endpoint` – OTLP HTTP endpoint (e.g. `localhost:4318` for dev, `otel-collector.company.com:4318` for prod)
- `telemetry.insecure` – disables TLS for local/dev (default: `true` in dev, `false` in prod)

#### Sampling Strategy

Environment-based sampling ensures cost-effective production monitoring:

- **Production**: 1% sampling (cost-effective, maintains observability)
- **Staging**: 10% sampling (balanced observability for testing)
- **Development/Test**: 100% sampling (full debugging coverage)

#### Resource Attributes

All spans include comprehensive resource metadata:


#### Key Spans with Semantic Attributes

**NATS Consumer (`caatsm/nats`)**:
- `Consumer.processMessage`
  - `messaging.system: nats`
  - `messaging.operation: receive`
  - `messaging.destination: <subject>`
  - `messaging.consumer.id: <consumer-name>`
  - `caatsm.stream: <stream-name>`

**Application Processor (`caatsm/app`)**:
- `MessageProcessor.Handle`
  - `messaging.system: nats`
  - `messaging.operation: receive`
  - `messaging.message_id: <msg-id>`
  - `caatsm.component: processor`
  - `caatsm.message.category: <ARR|DEP|FPL|etc>`

**Database Operations (`caatsm/postgres`)**:
- `Repository.InsertOne`, `Repository.InsertBatch`, `Repository.InsertRaw`
  - `db.system: postgresql`
  - `db.operation: insert`
  - `db.name: aviation`
  - `db.table: telegrams`
  - `caatsm.message.id: <telegram-id>`

#### OTEL vs Prometheus Metrics

The receiver reports complementary metrics through both systems:

**Prometheus metrics via `/metrics`** (operational focus):
- End-to-end message handling (`caatsm_messages_total`, `caatsm_handle_latency_seconds`, `caatsm_retries_total`)
- DB activity (`caatsm_db_queries_total`, `caatsm_db_query_latency_seconds`)
- NATS consumer metrics (`caatsm_nats_consumer_pending_messages`)
- DLQ operations (`caatsm_dlq_messages_total`, `caatsm_dlq_publish_failures_total`)
- Publish failures (`caatsm_publish_failures_total{category}`)

**OpenTelemetry metrics via OTLP** (business focus):
- Message processing results (`caatsm_messages_processed_total{message_status,message_category}`)
- Parse performance (`caatsm_parse_duration_seconds{message_status,message_category}`)
- Publish reliability (`caatsm_publish_failures_total{message_category}`)

**Note**: `caatsm_publish_failures_total` is available in both Prometheus (with `category` label) and OTEL (with `message_category` label). The OTEL version is exported via the collector and scraped by Prometheus, where attribute keys are converted to label names (dots become underscores: `message_status`, `message_category`).

#### Collector Integration

OTEL metrics and traces are exported to the configured collector:

- **Development**: `configs/otel-collector.dev.yaml` (batching, resource processing, retry logic)
- **Production**: `configs/otel-collector.prod.yaml` (TLS, authentication, high availability)

The collector exports OTEL metrics to Prometheus via the `prometheus` exporter (default endpoint: `:8889`), making them available for Grafana dashboards.

#### OpenTelemetry Collector Dashboard

The `otel-collector-dev` Grafana dashboard (provisioned from `configs/grafana-dashboards.dev/otel-collector.json`) focuses on the OTEL collector infrastructure:

- **OTEL Collector metrics** – receiver/exporter throughput, queue sizes, process metrics
- **Collector health** – memory, CPU, uptime, and queue capacity

**Note**: CAATSM application metrics exported via OTEL are displayed in the **CAATSM – Receiver Overview** dashboard for consolidated service monitoring. The OTEL collector dashboard focuses solely on collector infrastructure metrics.

### Structured Logging Contract

Logging is done with Zap. The `internal/infra/log` package standardises fields via `MessageFields`:

- `service` – logical component (`caatsm-consumer`, `caatsm-processor` etc.).  
- `transport_msg_id` – NATS/envelope message ID (derived from `Nats-Msg-Id` or JetStream sequence).  
- `telegram_message_id` – business telegram message ID from the payload.  
- `category` – telegram category (ARR, DEP, FPL, etc.).  
- `stream`, `consumer`, `subject` – JetStream context.  
- `nats_sequence` – JetStream stream sequence, when available.  
- `request_id`, `trace_id` – correlation identifiers.  
- `error_type` – high-level classification:
  - `business` – payload/validation/domain issues; not suitable for retry.
  - `transient` – network/DB/NATS glitches that may succeed on retry.
  - `fatal` – programming errors, schema mismatches, or configuration issues requiring operator attention.

Handler and consumer logs should always be emitted through `WithMessageContext` to ensure these fields are present where applicable.

