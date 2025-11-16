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

#### CAATSM – Receiver Overview Dashboard

The `caatsm-overview` Grafana dashboard (provisioned from `configs/grafana-dashboards.dev/caatsm-overview.json`) focuses on the CAATSM receiver service and surfaces:

- **Message throughput by result** – derived from `caatsm_messages_total{result}`.  
- **Per stream/consumer rates** – `caatsm_messages_total{stream,consumer}`.  
- **End-to-end handle latency** – P50/P95/P99 from `caatsm_handle_latency_seconds_bucket`.  
- **DB query rate and latency** – from `caatsm_db_queries_total` and `caatsm_db_query_latency_seconds_bucket`.  
- **Retry and permanent failure rates** – from `caatsm_retries_total` and `caatsm_messages_total{result="permanent_fail"}`.  
- **Publish failures** – from `caatsm_publish_failures_total`.

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


