# NATS Integration

## Overview

The NATS integration provides a streamlined, production-ready message processing system focused on essential functionality. It supports JetStream persistent messaging with basic error handling, TLS security, and observability.

### Key Concepts

1. **Consumer**: Pulls messages from NATS JetStream in batches and processes them
2. **Publisher**: Publishes messages to NATS with deduplication
3. **Batch Processing**: Fetches multiple messages for efficiency
4. **Error Classification**: Distinguishes transient vs permanent errors
5. **Dead Letter Queue (DLQ)**: Routes permanent errors to DLQ
6. **TLS Support**: Secure connections with client certificates
7. **Basic Monitoring**: Essential metrics and logging

### Quick Start Flow

```
1. Configure NATS connection and consumer settings
2. Create Consumer with dependencies (processor, logger, telemetry)
3. Start Consumer - begins fetching and processing messages
4. Messages flow: Fetch → Process → ACK/NAK/DLQ
5. Errors handled automatically with retries and backoff
6. Graceful shutdown on context cancellation
```

## Architecture

> See [`docs/architecture.md`](architecture.md) for the authoritative 3-layer architecture blueprint, stream topology, and consumer-side interface design.

### Architecture Overview

In `internal/nats/`, message consumption and publishing are consolidated with consumer-side `MessageHandler` and `Recorder` interfaces:

```
┌───────────────────────────────────────────────────────────┐
│                Pipeline Layer (internal/pipeline)         │
│                    Processor.Handle()                     │
└──────────────────────────▲────────────────────────────────┘
                           │ implements nats.MessageHandler
┌──────────────────────────┴────────────────────────────────┐
│                  NATS Layer (internal/nats)               │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                     Consumer                        │  │
│  │  - Inbound/Outbound/DLQ 3-stream topology           │  │
│  │  - ShouldAutoProvision() prod read-only check       │  │
│  │  - Start(ctx) / processBatch() / processMsg()       │  │
│  │  - ConsumerMonitor (lag, message gap, seq gap)      │  │
│  │  - defaultDLQHandler (RouteToDLQ)                   │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                     Publisher                       │  │
│  │  - Implements pipeline.Publisher                    │  │
│  │  - Injects W3C TraceContext headers                 │  │
│  └─────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────┘
```

### Component Interaction

```
┌──────────────┐
│   Publisher  │
│ 1. Marshal   │
│ 2. Publish   │
└──────┬───────┘
       │
       │ Publish to Subject
       ▼
┌─────────────────────────────────────┐
│         NATS JetStream              │
│                                     │
│  ┌──────────────┐                   │
│  │   Stream     │                   │
│  │  (TELEGRAM)  │                   │
│  └──────┬───────┘                   │
│         │                           │
│  ┌──────▼───────┐                   │
│  │  Consumer    │                   │
│  │  (Pull Sub)  │                   │
│  └──────┬───────┘                   │
└─────────┼───────────────────────────┘
          │
          │ Fetch Batch
          ▼
┌─────────────────────────────────────┐
│        Consumer (consumer.go)       │
│  - PullSubscribe + Fetch loop       │
│  - ConsumerMonitor.TrackMessage     │
│  - processMsg -> processor.Handle() │
│  - Ack / DLQ+Ack / NakWithDelay     │
└─────────────────────────────────────┘
```

*(Note: Currently `Consumer.ensureInfrastructure()` binds inbound `subscription.topic`, outbound `publisher.topic`, and `dlq.subject` to the single `WorkQueuePolicy` stream `TELEGRAM`. Splitting these into 3 dedicated streams is planned in `docs/architecture.md`.)*

### Design Goals

1. **Cohesion**: Pull subscription, batch processing, backoff, and DLQ routing live inside `Consumer` with focused helpers (`ConsumerMonitor`, `DLQHandler`).
2. **Separation of Concerns**:
   - `Consumer`: Stream/consumer setup, pull loop, ACK/NAK/DLQ handling.
   - `ConsumerMonitor`: Pending backlog, message gap, and sequence gap metrics.
   - `defaultDLQHandler`: Dead-letter payload construction and publishing.
   - `Publisher`: Downstream JSON publishing with `Nats-Msg-Id` deduplication.
3. **Target Decoupling**: Decoupling `Consumer` from concrete `*app.MessageProcessor` via a `MessageHandler` interface is planned in `docs/architecture.md` (Phase 1).

## Logic Flow

### Consumer Processing Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Consumer Start                           │
│  1. ensureInfrastructure() (AddStream / AddConsumer)        │
│  2. Start ConsumerMonitor goroutine                         │
│  3. Create PullSubscribe + record initial NumPending        │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    Main Loop                                │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Step 1: Check Context                               │   │
│  │  - If cancelled, return nil                          │   │
│  └──────────────────┬───────────────────────────────────┘   │
│                     │                                       │
│  ┌──────────────────▼───────────────────────────────────┐   │
│  │  Step 2: Fetch Batch                                 │   │
│  │  - sub.Fetch(batchSize, MaxWait(batchTimeout))       │   │
│  │  - nats.ErrTimeout -> continue                       │   │
│  │  - Other error -> warn log + 100ms sleep + continue  │   │
│  └──────────────────┬───────────────────────────────────┘   │
│                     │                                       │
│  ┌──────────────────▼───────────────────────────────────┐   │
│  │  Step 3: Process Batch                               │   │
│  │  - For each message in batch:                        │   │
│  │    * monitor.TrackMessage(msg)                       │   │
│  │    * processMsg -> processor.Handle(ctx, data, id)   │   │
│  │    * Handle result (ACK / DLQ+ACK / NAK with delay)  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### Message Processing Logic

#### Single Message Processing Flow

```
Message Received
      │
      ▼
┌─────────────────────┐
│ Extract Message ID  │
│ - Check header      │
│ - Fallback to meta  │
│ - Generate if none  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Create Trace Span   │
│ - Add attributes    │
│ - Propagate context │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Process Message     │
│ - Call processor    │
│ - Business logic    │
└──────────┬──────────┘
           │
    ┌──────┴──────┐
    │             │
Success        Error
    │             │
    │      ┌──────▼──────────┐
    │      │ Classify Error  │
    │      └──────┬──────────┘
    │             │
    │      ┌──────┴──────┐
    │      │             │
    │  Permanent    Transient
    │      │             │
    │  ┌───▼───┐    ┌────▼────┐
    │  │  DLQ  │    │   NAK   │
    │  │  ACK  │    │ Backoff │
    │  └───┬───┘    └────┬────┘
    │      │             │
    └──────┴─────────────┘
           │
           ▼
    End Processing
```

#### Error Handling Logic

```
Error Occurred
      │
      ▼
┌─────────────────────┐
│ Is Permanent Error? │
│ - app.IsPermanent() │
└──────┬──────────────┘
       │
   ┌───┴───┐
   │       │
  Yes     No
   │       │
   │   ┌───▼──────────────────────┐
   │   │ Increment Error Streak   │
   │   └───┬──────────────────────┘
   │       │
   │   ┌───▼──────────────────────┐
   │   │ Streak >= Threshold?     │
   │   │ (default: 10 errors)     │
   │   └───┬──────────────────────┘
   │       │
   │   ┌───┴───┐
   │   │       │
   │  Yes     No
   │   │       │
   │   │   ┌───▼──────────────┐
   │   │   │ NAK with delay   │
   │   │   │ - Use backoff    │
   │   │   │ - Request retry  │
   │   │   └──────────────────┘
   │   │
   │   ▼
   │ ┌──────────────────────┐
   │ │ Apply Backpressure   │
   │ │ - Sleep: errors*100ms│
   │ │ - Max: 5 seconds     │
   │ └───┬──────────────────┘
   │     │
   │     ▼
   │ ┌──────────────────────┐
   │ │ NAK with delay       │
   │ └──────────────────────┘
   │
   ▼
┌──────────────────────┐
│ Route to DLQ         │
│ - Enrich metadata    │
│ - Publish to DLQ     │
│ - ACK original msg   │
└──────────────────────┘
```

### Publisher Logic Flow

```
Publish Request
      │
      ▼
┌─────────────────────┐
│ Validate Topic      │
│ - Check config      │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Serialize Message   │
│ - JSON marshal      │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Extract/Generate ID │
│ - From message.Uuid │
│ - Or generate UUID  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Set Header          │
│ - Nats-Msg-Id       │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ Publish to NATS     │
│ - js.PublishMsg()   │
└──────────┬──────────┘
           │
    ┌──────┴──────┐
    │             │
Success        Error
    │             │
    │      ┌──────▼──────────┐
    │      │ Classify Error  │
    │      └──────┬──────────┘
    │             │
    │      ┌──────┴──────┐
    │      │             │
    │  Transient    Permanent
    │      │             │
    │  ┌───▼───┐    ┌────▼────┐
    │  │ Retry │    │  Fail   │
    │  │ Later │    │  Fast   │
    │  └───────┘    └─────────┘
    │
    ▼
Success
```

## Core Components

### Consumer

The consumer handles message consumption with the following features:

The consumer uses JetStream mode exclusively for persistent, durable message processing with acknowledgments.

#### Key Features
- **Batch Processing**: Configurable batch sizes for efficient processing
- **Error Handling**: Distinguishes transient vs permanent errors
- **Dead Letter Queue (DLQ)**: Routes permanent errors to DLQ
- **TLS Support**: Secure connections with client certificates
- **Basic Monitoring**: Essential metrics collection

#### Component Details (`internal/nats/consumer.go`)

**Fetch Loop (`Consumer.Start`)**
- Binds via `PullSubscribe(..., nats.Bind(stream, consumer))` and pulls batches with `sub.Fetch(batchSize, nats.Context(fetchCtx))`
- Resets error streak and continues on `nats.ErrTimeout`/`context.DeadlineExceeded`; exits on context cancellation and NAKs remaining unprocessed batch messages on shutdown
- On transient fetch errors, waits with exponential backoff (`500ms` to `5s`) and re-binds/recovers the JetStream subscription

**Batch & Message Handling (`Consumer.processBatch` / `Consumer.processMsg`)**
- Processes messages sequentially within a batch
- Calls `ConsumerMonitor.TrackMessage(msg)` for gap/sequence tracking
- Extracts upstream W3C `TraceContext` from NATS headers via `otel.GetTextMapPropagator().Extract`
- Extracts message IDs (`Nats-Msg-Id` header -> `js-<stream_seq>` fallback -> `"unknown"`)
- Calls `handler.Handle(ctx, msg.Data, msgID)` and handles ACK / NAK-with-backoff / DLQ (falling back to NAK+backoff if `RouteToDLQ` fails)

**DLQHandler (`defaultDLQHandler` in `internal/nats/dlq.go`)**
- Routes permanent errors (`pipeline.IsPermanent(err)`) to `dlq.subject` (`nats.dlq_stream`) with JSON payload (`subject`, `stream`, `consumer`, `error`, `received_at`, `body`)
- `ValidateDLQ()` checks stream binding for `dlq.subject` during startup when DLQ is enabled

#### Configuration
```toml
[nats]
url = "nats://localhost:4222"
stream = "TELEGRAM"
consumer = "telegram-consumer"

[nats.auth]
token = "your-token"  # Optional token authentication
tls_enabled = true     # Enable TLS
tls_cert_file = "/path/to/client.crt"  # Client certificate
tls_key_file = "/path/to/client.key"   # Client private key
tls_ca_file = "/path/to/ca.crt"        # CA certificate

[nats.consumer_rules]
ack_wait = "30s"
max_deliver = 3
max_ack_pending = 1000

[dlq]
enabled = true
subject = "caatsm.dlq"

[app]
batch_size = 50
batch_timeout = "2s"
```

### Publisher

The publisher handles message publishing with deduplication and observability.

#### Features
- **Message Deduplication**: Automatic UUID-based deduplication headers
- **JetStream Publishing**: Uses JetStream for reliable message delivery
- **Structured Logging**: Comprehensive logging of publish operations
- **Error Classification**: Distinguishes between transient and permanent errors

#### Component Logic

**Publishing Flow**
1. **Validation**: Checks that publisher topic is configured and `msg != nil`
2. **Serialization**: Marshals `*telegram.Message` to JSON using `json.Marshal()`
3. **ID Extraction**: Uses `msg.UUID` if non-empty or generates a new UUID
4. **Header Attachment**: Sets `Nats-Msg-Id` for deduplication and injects W3C TraceContext via `otel.GetTextMapPropagator().Inject`
5. **Publishing**: Calls `js.PublishMsg(jsMsg, nats.Context(ctx))` to publish to JetStream
6. **Error Handling**: Classifies `ErrNoResponders`, `ErrTimeout`, connection loss, and context errors as transient; others as `pipeline.Permanent`

**Deduplication Strategy**
- Uses `Nats-Msg-Id` header for JetStream deduplication
- Extracts UUID from `msg.UUID` (`*telegram.Message`) if available
- Falls back to generating new UUID if not present
- JetStream uses this header to prevent duplicate message processing

**Error Classification**
- **Transient**: `nats.ErrNoResponders`, `nats.ErrTimeout`, connection closed/disconnected, context deadline/cancellation - wrapped without `pipeline.Permanent` for retry
- **Permanent**: Empty topic, marshaling failure, or non-transient broker errors - wrapped with `pipeline.Permanent`

### Error Handling

#### Error Types
- **Transient Errors**: Network issues, temporary unavailability (retried with backoff)
- **Permanent Errors**: Message format issues, business logic failures (routed to DLQ)

#### Recovery Strategies
- **Simple Backoff**: Exponential backoff for transient failures
- **Graceful Degradation**: Continues processing other messages when one fails

### Dead Letter Queue (DLQ)

#### Features
- **Error Context**: Includes `subject`, `stream`, `consumer`, `error`, `received_at`, and raw `body`
- **Operational Visibility**: Structured error logging and Prometheus metrics (`caatsm_dlq_messages_total`, `caatsm_dlq_publish_failures_total`)

#### DLQ Message Format
```json
{
  "subject": "telegram.serial",
  "stream": "TELEGRAM",
  "consumer": "telegram-consumer",
  "error": "parser error: invalid telegram header",
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



## Configuration

### Environment Variables
```bash
CAATSM_NATS__URL=nats://localhost:4222
CAATSM_NATS__AUTH__TOKEN=your-token  # Optional
CAATSM_DLQ__ENABLED=true
CAATSM_DLQ__SUBJECT=caatsm.dlq
```

### TLS Configuration
For production deployments with TLS:

```toml
[NATS.Auth]
TLSEnabled = true
TLSCertFile = "/etc/ssl/certs/client.crt"
TLSKeyFile = "/etc/ssl/private/client.key"
TLSCAFile = "/etc/ssl/certs/ca.crt"
```

### Runtime Configuration
- **Validation**: Basic validation at startup
- **Defaults**: Sensible defaults for essential options

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
- **Error Handling**: Basic error scenarios
- **Configuration**: Configuration validation

## Simple Examples

### Example 1: Complete Consumer Setup and Start

This example shows how to set up and start a consumer from scratch:

```go
package main

import (
    "context"
    "time"

    "caatsm/internal/config"
    "caatsm/internal/nats"
    "caatsm/internal/pipeline"
    "go.uber.org/zap"
)

func main() {
    // 1. Load configuration
    cfg := &config.Config{
        NATS: config.NATSConfig{
            URL:           "nats://localhost:4222",
            Stream:        "TELEGRAM",
            PublishStream: "TELEGRAM_OUT",
            DLQStream:     "TELEGRAM_DLQ",
            Consumer:      "telegram-consumer",
            Auth: config.NATSAuthConfig{
                Token: "your-token", // Optional
            },
            ConsumerRules: config.ConsumerRulesConfig{
                AckWait:       30 * time.Second,
                MaxDeliver:    3,
                MaxAckPending: 1000,
            },
        },
        App: config.AppConfig{
            BatchSize:    50,
            BatchTimeout: 2 * time.Second,
        },
        DLQ: config.DLQConfig{
            Enabled: true,
            Subject: "caatsm.dlq",
        },
    }

    logger := zap.NewNop()

    // 2. Create NATS connection
    nc, err := nats.Connect(cfg, logger)
    if err != nil {
        panic(err)
    }
    defer nc.Close()

    // 3. Get JetStream context
    js, err := nats.JetStream(nc, logger)
    if err != nil {
        panic(err)
    }

    // 4. Create pipeline processor (your business logic implementing nats.MessageHandler)
    handler := pipeline.NewProcessor(/* parser, validator, store, publisher, recorder, logger, opts */)

    // 5. Create telemetry recorder
    rec := /* observability.NewRecorder() or nil for noop */

    // 6. Create consumer
    consumer, err := nats.NewConsumer(
        nc,
        js,
        handler,
        cfg,
        rec,
        logger,
    )
    if err != nil {
        panic(err)
    }

    // 7. Start consumer with context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // 8. Start consuming (blocks until context cancelled)
    if err := consumer.Start(ctx); err != nil {
        logger.Error("Consumer stopped", zap.Error(err))
    }
}
```

### Example 2: Publishing a Message

Simple example of publishing a message:

```go
package main

import (
    "context"
    "fmt"

    "caatsm/internal/pipeline"
    "caatsm/internal/telegram"
)

func publishMessage(ctx context.Context, publisher pipeline.Publisher) error {
    // Create message with UUID
    message := telegram.NewMessage()
    message.UUID = telegram.NewUUID() // Used for Nats-Msg-Id deduplication
    message.Content = "telegram message data"

    // Publish - automatically handles:
    // - JSON serialization
    // - UUID header attachment
    // - W3C TraceContext injection
    // - Transient vs Permanent error classification
    if err := publisher.Publish(ctx, message); err != nil {
        return fmt.Errorf("failed to publish: %w", err)
    }

    return nil
}
```

### Example 3: Message Processing Flow

Step-by-step what happens when a message is processed:

```go
// Step 1: Consumer fetches batch of messages
msgs, err := subscription.Fetch(50, nats.MaxWait(2*time.Second))
// Result: Up to 50 messages, or timeout after 2 seconds

// Step 2: For each message in batch
for _, msg := range msgs {
    // Step 2a: Extract message ID
    msgID := msg.Header.Get("Nats-Msg-Id")
    if msgID == "" {
        // Fallback: use JetStream sequence
        meta, _ := msg.Metadata()
        msgID = fmt.Sprintf("js-%d", meta.Sequence.Stream)
    }
    
    // Step 2b: Create tracing span
    ctx, span := tracer.Start(ctx, "process.message")
    span.SetAttributes(
        attribute.String("messaging.system", "nats"),
        attribute.String("messaging.destination.name", msg.Subject),
    )
    
    // Step 2c: Process message (your business logic)
    err := handler.Handle(ctx, msg.Data, msgID)
    
    // Step 2d: Handle result
    if err != nil {
        if pipeline.IsPermanent(err) {
            // Permanent error: route to DLQ; only ACK if DLQ publish succeeds
            if dlqErr := dlqHandler.RouteToDLQ(ctx, msg, err); dlqErr != nil {
                msg.NakWithDelay(calculateBackoff(msg))
            } else {
                msg.Ack()
            }
        } else {
            // Transient error: NAK with backoff
            msg.NakWithDelay(calculateBackoff(msg))
        }
    } else {
        // Success: ACK message
        msg.Ack()
    }
    
    span.End()
}
```

### Example 4: Error Handling Scenarios

Different error scenarios and how they're handled:

```go
// Scenario 1: Transient Error (Network / DB Issue)
func handleMsg(ctx context.Context, raw []byte) error {
    // Return a normal wrapped error when downstream dependency is temporarily unavailable
    if networkDown {
        return fmt.Errorf("network unavailable")
    }
    // Result: Message is NAK'd, will be redelivered with backoff
    return nil
}

// Scenario 2: Permanent Error (Invalid Format)
func handleMsgPermanent(ctx context.Context, raw []byte) error {
    var data MyStruct
    if err := json.Unmarshal(raw, &data); err != nil {
        return pipeline.Permanent(fmt.Errorf("invalid JSON format: %w", err))
    }
    // Result: Message routed to DLQ (if enabled), original message ACK'd
    return nil
}

// Scenario 3: Backpressure Trigger
// When 10+ consecutive errors occur:
// - Processing pauses
// - Sleep duration = min(consecutiveErrors * 100ms, 5s)
// - Prevents overwhelming the system

// Scenario 4: MaxDeliver Exhausted
// When message fails MaxDeliver times (default: 3):
// - JetStream stops redelivering the message
// - Consider monitoring JetStream consumer info for failed deliveries
```

### Example 5: DLQ Message Structure

What a DLQ message looks like (matching `internal/nats/dlq.go`):

```json
{
  "subject": "telegram.serial",
  "stream": "TELEGRAM",
  "consumer": "telegram-consumer",
  "error": "parser error: invalid telegram header",
  "received_at": "2024-01-15T10:30:00Z",
  "body": "raw telegram payload"
}
```

### Example 6: Configuration Examples

Basic configuration with TLS:

```toml
[NATS]
URL = "nats://secure.nats.server:4222"
Stream = "TELEGRAM"
Consumer = "telegram-consumer"

[NATS.Auth]
TLSEnabled = true
TLSCertFile = "/etc/ssl/certs/client.crt"
TLSKeyFile = "/etc/ssl/private/client.key"
TLSCAFile = "/etc/ssl/certs/ca.crt"

[NATS.ConsumerRules]
AckWait = "30s"
MaxDeliver = 3
MaxAckPending = 1000

[DLQ]
Enabled = true
Subject = "caatsm.dlq"

[App]
BatchSize = 50
BatchTimeout = "2s"
```

Development configuration:

```toml
[NATS]
URL = "nats://localhost:4222"
Stream = "TELEGRAM"
Consumer = "telegram-consumer"

[DLQ]
Enabled = true
Subject = "caatsm.dlq"
```

### Example 7: Observability Integration

How to monitor the consumer:

```go
// Metrics are automatically collected:
// - ack_pending: Messages waiting for ACK
// - redelivered: Messages being redelivered
// - pending: Messages in stream
// - delivered: Total messages delivered

// Tracing spans are created for:
// - Each message processing
// - DLQ routing
// - Error handling

// Logs include:
// - Message processing events
// - Error details with context
// - DLQ routing events
// - Connection status changes
```

## Usage Examples

### Basic Consumer Setup
```go
consumer, err := nats.NewConsumer(
    natsConn,
    jetStream,
    processor,
    config,
    recorder,
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
publisher, err := nats.NewPublisher(
    jetStream,
    config,
    recorder,
    logger,
)
if err != nil {
    return err
}

err = publisher.Publish(ctx, &telegram.Message{
    UUID:    telegram.NewUUID(),
    Content: telegramText,
})
```

### Custom Error Handling
```go
type CustomProcessor struct {
    // implementation
}

func (p *CustomProcessor) Handle(ctx context.Context, raw []byte, msgID string) error {
    // Business logic here
    if shouldRetry := someCondition(); shouldRetry {
        return fmt.Errorf("temporary failure")
    }
    if isInvalid := validateMessage(raw); isInvalid {
        return pipeline.Permanent(fmt.Errorf("invalid message format"))
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
- **Token Auth**: Use NATS tokens for simple authentication
- **TLS**: Enable TLS with client certificates for secure communication

### TLS Configuration
```toml
[NATS.Auth]
TLSEnabled = true
TLSCertFile = "/path/to/client.crt"
TLSKeyFile = "/path/to/client.key"
TLSCAFile = "/path/to/ca.crt"
```

### Data Protection
- **TLS Encryption**: All communication is encrypted
- **Basic Logging**: Avoid logging sensitive message content

## Future Enhancements

### Future Enhancements
- **Additional Auth Methods**: Support for more authentication mechanisms if needed
- **Advanced Monitoring**: Enhanced metrics and tracing if required
- **Performance Tuning**: Batch size and timeout optimizations