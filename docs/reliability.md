## Reliability and Fault Handling

This document summarises how the service handles failures, provides resilience, and avoids data loss or duplication.

> For the full reliability gap analysis and target JetStream/pipeline design, see [`docs/architecture.md`](architecture.md).

### Dead-Letter Queue (DLQ) and Poison Messages

Configuration is defined under `dlq`:

```toml
[dlq]
enabled = true
subject = "caatsm.dlq"
```

- When `dlq.enabled` is `true` and `dlq.subject` is non-empty, **permanent** failures are routed to the DLQ stream (`nats.dlq_stream`, default `TELEGRAM_DLQ`).  
- A permanent failure is indicated by wrapping an error with `pipeline.Permanent` and is treated as a **poison message**.

Behaviour:

1. The NATS consumer calls `handler.Handle` (`*pipeline.Processor`).
2. If an error is returned and `pipeline.IsPermanent(err)` is true:
   - The original NATS message is copied into a JSON DLQ payload (`internal/nats/dlq.go`) with fields:
     - `subject`, `stream`, `consumer`
     - `error` (stringified cause)
     - `received_at` (DLQ event time in UTC)
     - `body` (raw message body)
   - The payload is published to `dlq.subject` (`nats.dlq_stream`, `LimitsPolicy`) using JetStream.
   - If DLQ publishing succeeds, the original message is **ACKed**, so it will not be redelivered.
   - If `RouteToDLQ` fails, `Consumer.handleFailure` logs/records `caatsm_dlq_publish_failures_total` and `caatsm_retries_total{reason="dlq_publish_error"}`, applies backpressure, and issues **NAK with backoff** instead of ACKing so no message is silently lost.

The DLQ subject should be consumed by an offline repair/analysis tool or operational dashboard that can:

- Inspect poison messages.  
- Decide whether to fix and re-publish, or discard with justification.  
- Track DLQ volume over time for alerting (e.g. via `caatsm_dlq_messages_total`
  and `caatsm_dlq_publish_failures_total` metrics).

### Transient Errors and Backoff

Transient errors (not marked permanent) result in:

- Negative acknowledgements with delay (`NakWithDelay`) according to `nats.consumer_rules.backoff`.
- A retry streak counter inside the consumer:
  - Each transient error increases `consecutiveErrors` (`internal/nats/consumer.go`).
  - After 10 or more consecutive errors (`consecutiveErrors >= 10`), `applyBackpressure` waits (context-aware timer) before issuing NAK:
    - `delay = min(consecutiveErrors * 100ms, 5s)`.

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

### JetStream Availability and Auto-Recovery (Dev vs Prod)

- When `sub.Fetch(batchSize, nats.Context(ctx), nats.MaxWait(batchTimeout))` encounters a transient fetch error other than `nats.ErrTimeout` or context cancellation, the consumer logs a warning, waits with exponential backoff (`500ms` to `5s`), and attempts JetStream subscription recovery before retrying.
- At startup, `NewConsumer` checks `cfg.ShouldAutoProvision()` (`auto_provision` override, or `false` when `GO_ENV=prod` and `true` otherwise):
  - When `true` (dev/test): creates or verifies the inbound (`nats.stream`, `WorkQueuePolicy`), outbound (`nats.publish_stream`, `LimitsPolicy`), and DLQ (`nats.dlq_stream`, `LimitsPolicy`) streams and the durable consumer.
  - When `false` (`prod`): performs read-only `StreamInfo` / `ConsumerInfo` verification and fails fast if required infrastructure has not been pre-provisioned out-of-band.
- On the publishing side, `internal/nats/publisher.go` classifies `nats.ErrNoResponders`, `nats.ErrTimeout`, connection errors (`nats.ErrConnectionClosed`/`ErrDisconnected`), and `context.Canceled`/`context.DeadlineExceeded` as transient errors (wrapped without `pipeline.Permanent`), while configuration/marshaling and other broker errors are wrapped with `pipeline.Permanent`.
  - Transient errors cause the consumer to NAK messages with configured backoff for retries.
  - Permanent errors are routed through `pipeline.Processor` (`internal/pipeline/processor.go`) to DLQ + ACK.

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

