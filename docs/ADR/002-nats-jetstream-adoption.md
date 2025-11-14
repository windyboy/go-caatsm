# ADR-002: NATS JetStream Adoption

## Status
Accepted

## Context
The current system uses NATS with JetStream disabled. This means:
- Messages are not persisted
- No message replay capability
- No guaranteed delivery
- No message ordering guarantees
- Risk of message loss during service restarts

For a production aviation message processing system, we need:
- Message persistence for reliability
- Ability to replay messages for debugging and recovery
- Guaranteed message delivery
- Message ordering guarantees
- Better observability of message flow

## Decision
We will enable NATS JetStream for message persistence and reliability. JetStream provides:
- Persistent message storage
- At-least-once delivery guarantees
- Message replay capabilities
- Stream management and monitoring
- Consumer groups for load balancing

## Consequences

### Positive
- **Reliability**: Messages are persisted and won't be lost during restarts
- **Replay**: Ability to replay messages for debugging and recovery
- **Guaranteed Delivery**: At-least-once delivery semantics
- **Ordering**: Message ordering within a stream
- **Monitoring**: Better observability with stream and consumer metrics
- **Scalability**: Consumer groups enable horizontal scaling

### Negative
- **Storage**: Requires disk storage for message persistence
- **Complexity**: More complex configuration and management
- **Performance**: Slight performance overhead due to persistence
- **Resource Usage**: Additional memory and disk usage

### Mitigation
- Configure appropriate retention policies to manage storage
- Use compression for old messages
- Monitor stream sizes and consumer lag
- Set up alerts for consumer failures

## Implementation Notes
- Stream will be configured with 7-day retention
- Compression will be enabled for messages older than 7 days
- Consumer will use explicit ACK policy for reliable processing
- Max retry attempts will be set to 5
- Batch processing will be used for better throughput

