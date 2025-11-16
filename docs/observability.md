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

Additional OTEL metrics are emitted via the configured OTEL endpoint, including:

- `caatsm_messages_processed_total`  
- `caatsm_parse_duration_ms`  
- `caatsm_publish_failures_total`  
- `caatsm_nats_consumer_ack_pending`  
- `caatsm_nats_consumer_redelivered`  
- `caatsm_nats_consumer_pending`  
- `caatsm_nats_consumer_delivered`

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

- **Message throughput by result** – derived from `caatsm_messages_total{result}`.  
- **Per stream/consumer rates** – `caatsm_messages_total{stream,consumer}`.  
- **End-to-end handle latency** – P50/P95/P99 from `caatsm_handle_latency_seconds_bucket`.  
- **DB query rate and latency** – from `caatsm_db_queries_total` and `caatsm_db_query_latency_seconds_bucket`.  
- **Retry and permanent failure rates** – from `caatsm_retries_total` and `caatsm_messages_total{result="permanent_fail"}`.  
- **Publish failures** – from `caatsm_publish_failures_total`.

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

- `/healthz` – basic liveness and dependency check.  
- `/readyz` – readiness endpoint with the same logic as `/healthz`, intended for load balancers / orchestrators.

Checks performed:

- PostgreSQL: `pgxpool.Pool.Ping` with configurable timeout (`monitoring.health_timeout`).  
- NATS: connection status must be `CONNECTED`.

A non-200 response indicates the service is not healthy/ready and should be removed from traffic.

### Tracing

Tracing is configured via the `telemetry` section:

- `telemetry.enabled` – enables OTEL exporters.  
- `telemetry.endpoint` – OTLP HTTP endpoint (e.g. `localhost:4318`).  
- `telemetry.insecure` – disables TLS for local/dev.

#### OTEL vs Prometheus metrics

The receiver reports two complementary sets of metrics:

- **Prometheus metrics via `/metrics`**  
  Implemented in `internal/observability/metrics`, covering:
  - End-to-end message handling (`caatsm_messages_total`,
    `caatsm_handle_latency_seconds`, `caatsm_retries_total`)
  - DB activity (`caatsm_db_queries_total`,
    `caatsm_db_query_latency_seconds`)
  - Legacy per-telegram metrics

- **OpenTelemetry metrics via OTLP**  
  Implemented using `otel.Meter` in the NATS consumer and app processor,
  including:
  - `caatsm_messages_processed_total`
  - `caatsm_parse_duration_ms`
  - `caatsm_publish_failures_total`
  - `caatsm_nats_consumer_ack_pending`
  - `caatsm_nats_consumer_redelivered`
  - `caatsm_nats_consumer_pending`
  - `caatsm_nats_consumer_delivered`

Prometheus only sees the metrics exposed on `/metrics`. OTEL metrics are
exported to the configured OTEL collector (`telemetry.endpoint`) via OTLP and
are, by default, forwarded to Jaeger (traces) and logs (metrics) according to
`configs/otel-collector.dev.yaml`. If you want OTEL metrics to appear in
Prometheus as well, you can extend the collector configuration with a
`prometheus` or `prometheusremotewrite` exporter and add a corresponding
scrape or remote-write configuration.

Key spans:

- `caatsm/nats`  
  - `Consumer.processMessage`
- `caatsm/app`  
  - `MessageProcessor.Handle`  
  - `Publisher.Publish`
- `caatsm/postgres`  
  - `Repository.InsertOne`  
  - `Repository.InsertBatch`  
  - `Repository.InsertRaw`

Important attributes:

- `nats.subject`, `nats.msg_id`, `nats.js.stream_seq`, `nats.js.consumer_seq`  
- `telegram.message_id`, `telegram.category`, `telegram.status`  
- `db.table`, `db.inserted`

### Structured Logging Contract

Logging is done with Zap. The `internal/observability/logging` package standardises fields via `MessageFields`:

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


