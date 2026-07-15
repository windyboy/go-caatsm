## Reliability and Fault Handling

This document summarises how the service handles failures, provides resilience, and avoids data loss or duplication.

### Dead-Letter Queue (DLQ) and Poison Messages

Configuration is defined under `dlq`:

```toml
[dlq]
enabled = true
subject = "caatsm.dlq"
```

- When `dlq.enabled` is `true` and `dlq.subject` is non-empty, **permanent** failures are routed to the DLQ subject.  
- A permanent failure is indicated by wrapping an error with `app.Permanent` and is treated as a **poison message**.

Behaviour:

1. The NATS consumer calls `processor.Handle`.  
2. If an error is returned and `app.IsPermanent(err)` is true:
   - The original NATS message is copied into a DLQ payload with metadata:
     - `subject`, `stream`, `consumer`
     - `error` (stringified cause)
     - `received_at` (DLQ event time)
     - `body` (raw message body)
     - `transport_msg_id` (NATS `Nats-Msg-Id` header, if set)
     - `nats_sequence` (JetStream stream sequence number)
     - `deliveries` (JetStream delivery count)
     - `reply` (NATS reply subject, if present)
     - `headers` (NATS message headers, if any)
   - The payload is published to `dlq.subject` using JetStream.  
   - The original message is **ACKed**, so it will not be redelivered.

The DLQ subject should be consumed by an offline repair/analysis tool or operational dashboard that can:

- Inspect poison messages.  
- Decide whether to fix and re-publish, or discard with justification.  
- Track DLQ volume over time for alerting (e.g. via `caatsm_dlq_messages_total`
  and `caatsm_dlq_publish_failures_total` metrics).

### Transient Errors and Backoff

Transient errors (not marked permanent) result in:

- Negative acknowledgements with delay (`NakWithDelay`) according to `nats.consumer_rules.backoff`.  
- A retry streak counter inside the consumer:
  - Each transient error increases `consecutiveProcessErrors`.  
  - After 10 or more consecutive errors, the consumer applies an additional **sleep**:
    - `backoff = min(consecutive_errors * 100ms, 5s)`.  
    - A warning log with the sleep duration and error count is emitted.

This combination provides **backpressure** when downstream systems (especially the DB) are in trouble, slowing down consumption instead of aggressively retrying.

### Persistence and Idempotency

The primary persistence path is `Repository.InsertOne` into `aviation.telegrams`. Business-message idempotency is enforced atomically via the **`aviation.telegram_keys` gate table**:

- If both `message_id` and `date_time` are non-empty:
  - `InsertOne` begins a transaction and reserves the business key in `aviation.telegram_keys` (PK: `message_id, date_time`).
  - A primary-key violation means the same business message was already persisted (possibly by a concurrent writer), so the telegram insert is **skipped** and the transaction is rolled back.
  - Otherwise, the telegram is inserted and the transaction is committed, ensuring both the gate row and the telegram are atomically persisted.

This makes repeated delivery of the same telegram (same `message_id`/`date_time`) safe from a business perspective, even under concurrent inserts or JetStream redeliveries.

**Partial records:** When either `message_id` or `date_time` is empty (legacy/partial telegrams), `InsertOne` falls back to a direct insert without the gate, preserving the previous behaviour. Duplicates are still possible for these records.

**Migration:** The `aviation.telegram_keys` table is defined in `internal/infra/postgres/telegrams.ddl`. For existing production databases, apply the following DDL:

```sql
CREATE TABLE IF NOT EXISTS aviation.telegram_keys (
    message_id TEXT NOT NULL,
    date_time TEXT NOT NULL,
    PRIMARY KEY (message_id, date_time)
);
```

### DB Degradation and Backpressure

Database write failures in `Repository.InsertOne` and related methods are treated as **transient** by default:

- Errors propagate back to the NATS consumer.  
- The consumer issues a NAK (with delay) and increases the transient error counter.  
- When errors persist, the added sleep in the consumer reduces message throughput and gives the DB time to recover.

DB health also feeds into readiness:

- The monitoring server hits `pgxpool.Pool.Ping` on `/readyz` and `/healthz`.  
- If the DB is not reachable, the endpoints return `503`, signalling to orchestrators that this instance should be drained from traffic.

Together, this yields:

- **Backpressure** via reduced consumption rate and NATS-level backoff.  
- **Degradation signalling** via health probes for external systems to act upon.

### Retry and Max Deliver

JetStream consumer configuration (via `nats.consumer_rules`) controls:

- `max_deliver` – maximum number of redeliveries before JetStream gives up.  
- `ack_wait` – how long JetStream waits for an ACK before considering the message pending.  
- `backoff` – per-attempt delays for `NakWithDelay`.

Recommended pattern:

- Keep `max_deliver` modest (e.g. 5).  
- Use a backoff array such as `[5s, 30s, 2m]`.  
- Treat messages that still fail after `max_deliver` as candidates for DLQ, via the permanent error/poison message path where applicable.

### JetStream Availability and Auto-Recovery (Dev vs Prod)

- When the JetStream API is temporarily unavailable (for example, NATS has just
  restarted and returns `ErrNoResponders`), the consumer uses an exponential
  backoff when retrying `Fetch` calls (roughly `1s, 2s, 4s, ...` up to
  around `30s`) to avoid log spam while allowing the system to recover.
- At startup the receiver always calls `StreamManager.EnsureStream` and
  `ConsumerManager.EnsureConsumer`. If the JetStream account allows it,
  missing streams are created with the configured retention limits
  (`max_msgs`, `max_bytes`, `max_age`, discard/storage policy, replicas) before
  the durable consumer is created. This keeps dev/test clusters self-healing
  after `docker compose down -v` and removes the race where a consumer was
  created without its stream.
- When the JetStream account lacks permissions to create protected resources
  (a common production posture), the same code path fails fast with a clear
  error message so operators know they must provision the stream out-of-band.
- On the publishing side, JetStream `ErrNoResponders` and similar errors are
  treated as temporary by the processor:
  - Such errors cause the consumer to NAK messages and rely on the configured
    backoff for retries.
  - Permanent configuration/permission errors remain mapped to permanent
    failures and follow the DLQ + ACK flow.

### Alerts and Dashboards

Prometheus alert suggestions:

- High failure rate:
  - `rate(caatsm_messages_total{result!="ok"}[5m])` above a small threshold.  
  - `rate(caatsm_retries_total[5m])` above a threshold.

- DLQ growth:
  - Alerts on DLQ stream message count, using NATS/JetStream exporter metrics.

- Readiness / health:
  - Alert when `/readyz` fails or when DB/NATS checks start failing consistently.

Dashboards should combine:

- Message rates, error rates, and DLQ rates.  
- NATS consumer statistics (pending, redelivered, ack_pending).  
- DB health indicators (latency, error counts, connection usage).

