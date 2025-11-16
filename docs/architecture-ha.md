## High-Level Architecture and HA / Scaling

### Components

- **NATS / JetStream** – message broker providing durable storage and redelivery:
  - Stream: `TELEGRAM` (configurable via `nats.stream`).  
  - Consumer: durable pull consumer per deployment (`nats.consumer`).  
  - Subjects:
    - Inbound telegrams: `subscription.topic` (e.g. `telegram.serial`).  
    - Parsed telegrams (publisher): `publisher.topic` (e.g. `telegram.json`).  
    - Dead-letter: `dlq.subject` (e.g. `caatsm.dlq`).

- **Receiver service (`caatsm`)**:
  - NATS JetStream **pull consumer** (`internal/infra/nats/consumer.go`).  
  - Telegram parser and domain model (`internal/app`, `internal/parsers`, `internal/domain`).  
  - PostgreSQL repository (`internal/infra/postgres`).  
  - Monitoring/observability server (`internal/infra/monitoring`).

### HA and Failover

- NATS/JetStream is expected to run as a **cluster** with `replicas` configured on the stream to ensure message durability.  
- The receiver service is stateless aside from DB side effects and can be deployed with multiple replicas:
  - Each replica connects to the same NATS cluster and JetStream stream.  
  - Durability and at-least-once semantics are handled by JetStream.

Consumer behaviour:

- Pull-based consumption with configurable batch size/timeout (`app.batch_size`, `app.batch_timeout`).  
- When a receiver instance stops or crashes:
  - Its NATS connection is drained and closed.  
  - Remaining messages remain in the stream.  
  - Another healthy instance continues pulling from the durable consumer.

### Scaling and Rebalancing

Scaling out:

- Increase the number of receiver replicas.  
- All replicas share the same durable consumer name; for pull-based consumption, each instance independently fetches messages.  
- JetStream distributes messages across fetch calls; with more instances, aggregate throughput increases.

Scaling in / failure:

- When replicas are reduced or fail, the remaining instances continue to fetch messages.  
- No explicit rebalancing logic is required in the application; JetStream manages which messages are available for pull.

Tuning:

- **Per-instance throughput** is primarily influenced by:
  - `app.batch_size`  
  - `app.batch_timeout`  
  - the number of concurrent instances

- **Backpressure** is provided through:
  - JetStream `backoff` and `max_deliver` settings.  
  - Additional sleeps in the consumer when many consecutive errors occur.  
  - Readiness checks exposing DB/NATS health.

### Failure Scenarios

1. **DB outage**:
   - Insert operations fail and are treated as transient.  
   - Messages are NAKed with delay and the error streak causes additional consumer sleep.  
   - `/readyz` returns 503, signalling this instance should be removed from traffic.

2. **NATS outage**:
   - Connection events are logged via `ProvideNATSConn` callbacks.  
   - The consumer will stop fetching; once NATS is back and reconnected, consumption resumes.

3. **Single instance crash**:
   - Other instances continue consuming from JetStream.  
   - No messages are lost; unacked messages remain pending and will be fetched by surviving instances.


