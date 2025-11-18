# NATS Integration Architecture

## Overview

The NATS integration provides a robust, production-ready message processing system built on Clean Architecture principles. It supports both JetStream (persistent) and Core NATS (fire-and-forget) modes with comprehensive error handling, observability, and resilience features.

## Architecture

### Clean Architecture Layers

```
┌─────────────────────────────────────┐
│         Port Interfaces             │
│   (Publisher, Consumer contracts)   │
├─────────────────────────────────────┤
│         Application Layer           │
│   (Message processing logic)        │
├─────────────────────────────────────┤
│         Infrastructure Layer        │
│   (NATS implementation details)     │
│                                     │
│   ┌─────────────────────────────┐   │
│   │        Consumer             │   │
│   │   ┌─────────────────────┐   │   │
│   │   │  MessageFetcher     │   │   │
│   │   │  MessageProcessor   │   │   │
│   │   │  ErrorHandler       │   │   │
│   │   │  DLQHandler         │   │   │
│   │   └─────────────────────┘   │   │
│   └─────────────────────────────┘   │
│                                     │
│   ┌─────────────────────────────┐   │
│   │       Publisher             │   │
│   │   ┌─────────────────────┐   │   │
│   │   │  MessageSerializer │   │   │
│   │   │  HeaderEnricher    │   │   │
│   │   └─────────────────────┘   │   │
│   └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

## Core Components

### Consumer

The consumer handles message consumption with the following features:

#### Modes
- **JetStream Mode**: Persistent, durable message processing with acknowledgments
- **Core Mode**: Fire-and-forget message processing for simple use cases

#### Key Features
- **Batch Processing**: Configurable batch sizes and timeouts for efficient processing
- **Backpressure**: Automatic backpressure when processing errors accumulate
- **Dead Letter Queue (DLQ)**: Automatic routing of failed messages to DLQ
- **Advisory DLQ**: Handles messages that exceed MaxDeliver limits
- **Self-Healing**: Automatic recreation of missing streams/consumers in dev environments
- **Graceful Shutdown**: Proper cleanup and draining of connections

#### Configuration
```toml
[NATS]
Mode = "jetstream"  # or "core"
Stream = "TELEGRAM"
Consumer = "telegram-consumer"

[NATS.ConsumerRules]
AckWait = "30s"
MaxDeliver = 3
MaxAckPending = 1000
DeliverPolicy = "all"
ReplayPolicy = "instant"
Backoff = ["1s", "2s", "5s", "10s"]

[DLQ]
Enabled = true
Subject = "caatsm.dlq"

[App]
BatchSize = 50
BatchTimeout = "2s"
MonitorInterval = "30s"
```

### Publisher

The publisher handles message publishing with deduplication and observability.

#### Features
- **Message Deduplication**: Automatic UUID-based deduplication headers
- **JetStream/Core Mode Support**: Adapts based on available JetStream context
- **Structured Logging**: Comprehensive logging of publish operations
- **Error Classification**: Distinguishes between transient and permanent errors

### Error Handling

#### Error Types
- **Transient Errors**: Network issues, temporary unavailability (retried with backoff)
- **Permanent Errors**: Message format issues, business logic failures (routed to DLQ)
- **Resource Errors**: Missing streams/consumers (auto-recovered in dev, fail in prod)

#### Recovery Strategies
- **Exponential Backoff**: Configurable backoff for transient failures
- **Circuit Breaker Pattern**: Prevents cascade failures
- **Resource Recreation**: Automatic recreation of missing JetStream resources
- **Graceful Degradation**: Continues processing other messages when one fails

### Dead Letter Queue (DLQ)

#### Features
- **Rich Metadata**: Includes original message, error details, delivery attempts
- **Stream Validation**: Validates DLQ stream exists at startup
- **Advisory Processing**: Handles MaxDeliver exhaustion automatically
- **Operational Visibility**: Comprehensive logging and metrics

#### DLQ Message Format
```json
{
  "transport_msg_id": "uuid",
  "subject": "original.subject",
  "stream": "TELEGRAM",
  "consumer": "telegram-consumer",
  "nats_sequence": 12345,
  "deliveries": 3,
  "error": "processing failed: invalid format",
  "received_at": "2024-01-01T12:00:00Z",
  "body": "original message data"
}
```

## Observability

### Metrics
- **Consumer Metrics**: ack_pending, redelivered, pending, delivered counts
- **Processing Metrics**: batch size, processing duration, error rates
- **DLQ Metrics**: messages routed to DLQ, publish failures
- **Connection Metrics**: connection health, reconnection events

### Tracing
- **End-to-End Tracing**: Request correlation through trace IDs
- **Span Attributes**: Consumer name, stream name, batch size, error details
- **Context Propagation**: Trace context passed through processing pipeline

### Logging
- **Structured Logs**: JSON format with correlation IDs
- **Log Levels**: Debug, Info, Warn, Error with appropriate detail levels
- **Operational Context**: Includes consumer, stream, and message metadata

## Resilience Patterns

### Backpressure
- **Error Accumulation**: Tracks consecutive processing errors
- **Adaptive Delay**: Increases delay based on error frequency
- **Circuit Breaking**: Stops processing when errors exceed threshold

### Connection Management
- **Auto-Reconnection**: Built-in NATS reconnection logic
- **Graceful Shutdown**: Proper draining with timeouts
- **Resource Cleanup**: Ensures subscriptions and connections are closed

### Self-Healing
- **Development Mode**: Auto-creates missing streams/consumers
- **Production Mode**: Fails fast on configuration issues
- **Recovery Logic**: Attempts to recreate resources on errors

## Configuration

### Environment Variables
```bash
CAATSM_NATS_URL=nats://localhost:4222
CAATSM_NATS_MODE=jetstream
CAATSM_DLQ_ENABLED=true
CAATSM_DLQ_SUBJECT=caatsm.dlq
```

### Runtime Configuration
- **Hot Reload**: Configuration changes applied without restart
- **Validation**: Comprehensive validation at startup
- **Defaults**: Sensible defaults for all configuration options

## Testing Strategy

### Unit Tests
- **Pure Functions**: Configuration normalization, policy mapping
- **Mock Dependencies**: NATS connections, JetStream contexts
- **Table-Driven Tests**: Comprehensive coverage of edge cases

### Integration Tests
- **Real NATS**: Testcontainers with actual NATS server
- **End-to-End**: Complete message processing pipelines
- **Failure Scenarios**: Network failures, resource unavailability

### Test Categories
- **Happy Path**: Normal operation scenarios
- **Error Recovery**: Various failure and recovery scenarios
- **Performance**: Load testing and resource usage
- **Configuration**: Different configuration combinations

## Usage Examples

### Basic Consumer Setup
```go
consumer, err := natsinfra.ProvideConsumer(
    natsConn,
    jetStream,
    messageProcessor,
    config,
    telemetryRecorder,
    logger,
)
if err != nil {
    return err
}

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

return consumer.Start(ctx)
```

### Publishing Messages
```go
publisher, err := natsinfra.ProvidePublisher(
    jetStream,
    natsConn,
    config,
    logger,
)
if err != nil {
    return err
}

err = publisher.Publish(&dto.ParsedTelegram{
    Uuid: uuid.NewString(),
    Data: telegramData,
})
```

### Custom Error Handling
```go
type CustomProcessor struct {
    // implementation
}

func (p *CustomProcessor) ProcessMessage(ctx context.Context, msg *nats.Msg) error {
    // Business logic here
    if shouldRetry := someCondition(); shouldRetry {
        return app.NewTransientError("temporary failure")
    }
    if isInvalid := validateMessage(msg); isInvalid {
        return app.NewPermanentError("invalid message format")
    }
    return nil
}
```

## Performance Considerations

### Optimization Strategies
- **Batch Processing**: Reduces per-message overhead
- **Connection Pooling**: Reuses connections efficiently
- **Memory Management**: Proper buffer sizing and cleanup
- **Concurrent Processing**: Parallel message processing within batches

### Monitoring Points
- **Throughput**: Messages processed per second
- **Latency**: End-to-end processing time
- **Resource Usage**: Memory, CPU, and network utilization
- **Error Rates**: Percentage of failed messages

## Operational Guide

### Deployment
1. **Configuration**: Set appropriate timeouts and limits
2. **Resource Provisioning**: Ensure sufficient NATS cluster capacity
3. **Monitoring Setup**: Configure alerts and dashboards
4. **DLQ Monitoring**: Set up DLQ message processing

### Troubleshooting
- **High Latency**: Check batch sizes and processing logic
- **Message Loss**: Verify consumer acks and DLQ configuration
- **Connection Issues**: Check NATS cluster health and network connectivity
- **Resource Exhaustion**: Monitor memory usage and connection counts

### Maintenance
- **Stream Cleanup**: Periodically clean up old streams
- **Consumer Recreation**: Recreate consumers for configuration changes
- **Performance Tuning**: Adjust batch sizes based on load patterns
- **Version Upgrades**: Test compatibility with NATS server versions

## Security Considerations

### Authentication
- **NATS Auth**: Use NATS built-in authentication mechanisms
- **TLS**: Enable TLS for encrypted communication
- **Token Auth**: Use NATS tokens for service authentication

### Authorization
- **Subject Permissions**: Restrict publish/subscribe permissions
- **Stream Access**: Control access to specific streams
- **DLQ Security**: Secure DLQ access to prevent data leakage

### Data Protection
- **Message Encryption**: Encrypt sensitive message data
- **Audit Logging**: Log all message operations for compliance
- **PII Handling**: Avoid logging sensitive information

## Future Enhancements

### Planned Features
- **Consumer Groups**: Horizontal scaling with multiple consumers
- **Message Filtering**: Subject-based and header-based filtering
- **Priority Queues**: High-priority message processing
- **Rate Limiting**: Per-consumer and per-subject rate limits
- **Message Transformation**: In-flight message modification
- **Multi-Region**: Cross-region message replication

### Extensibility Points
- **Custom Serializers**: Pluggable message serialization
- **Middleware**: Request/response middleware support
- **Hooks**: Pre/post processing hooks
- **Metrics Backends**: Support for additional metrics systems
- **Storage Backends**: Alternative storage for DLQ messages