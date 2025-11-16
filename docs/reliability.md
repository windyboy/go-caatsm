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

Behaviour (JetStream mode):

1. The NATS consumer calls `processor.Handle`.  
2. If an error is returned and `app.IsPermanent(err)` is true:
   - The original NATS message is copied into a DLQ payload with metadata:
     - `transport_msg_id` (NATS message ID)  
     - `subject`, `stream`, `consumer`  
     - `nats_sequence`, `deliveries`  
     - `error` (stringified cause)  
     - `received_at` (DLQ event time)  
     - `body` (raw message body)
   - The payload is published to `dlq.subject` using JetStream.  
   - The original message is **ACKed**, so it will not be redelivered.

The DLQ subject should be consumed by an offline repair/analysis tool or operational dashboard that can:

- Inspect poison messages.  
- Decide whether to fix and re-publish, or discard with justification.  
- Track DLQ volume over time for alerting.

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

The primary persistence path is `Repository.InsertOne` into `aviation.telegrams`. To avoid applying the same business event multiple times, a **minimal idempotency check** is implemented:

- If both `message_id` and `date_time` are non-empty:
  - `InsertOne` first calls `messageExists(message_id, date_time)`.  
  - If a row already exists, the insert is **skipped** and an informational log is written.  
  - Otherwise, the insert proceeds.

This makes repeated delivery of the same telegram (same `message_id`/`date_time`) safe from a business perspective, even if JetStream redelivers messages or upstream replays.

For higher guarantees in production environments, you may:

- Add a unique index on `(message_id, date_time)` at the DB level, and treat any conflict as a duplicate.  
- Extend the idempotency key with additional fields (e.g. originator, category) if required by the business model.

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


