# NATS Integration Architecture

## Overview

The NATS integration provides a robust, production-ready message processing system built on Clean Architecture principles. It supports both JetStream (persistent) and Core NATS (fire-and-forget) modes with comprehensive error handling, observability, and resilience features.

### Key Concepts

1. **Consumer**: Pulls messages from NATS JetStream in batches, processes them, and handles ACKs/NAKs
2. **Publisher**: Publishes messages to NATS with automatic deduplication via UUID headers
3. **Batch Processing**: Fetches multiple messages at once (configurable size) for efficiency
4. **Error Classification**: Distinguishes between transient (retry) and permanent (DLQ) errors
5. **Dead Letter Queue (DLQ)**: Routes failed messages to a separate queue for analysis
6. **Backpressure**: Automatically slows down processing when errors accumulate
7. **Self-Healing**: Automatically recreates missing streams/consumers in development
8. **Observability**: Built-in metrics, tracing, and structured logging

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

### Clean Architecture Layers

The NATS integration follows Clean Architecture principles, separating concerns into distinct layers:

```
┌─────────────────────────────────────┐
│         Port Interfaces             │
│   (Publisher, Consumer contracts)   │
│   - Define contracts, not impl      │
│   - Enable dependency inversion     │
├─────────────────────────────────────┤
│         Application Layer           │
│   (Message processing logic)        │
│   - Business logic                  │
│   - Use case orchestration          │
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

### Component Interaction Diagram

```
┌──────────────┐
│   Publisher  │
│              │
│ 1. Serialize │
│ 2. Add UUID  │
│ 3. Publish   │
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
│         Consumer                    │
│                                     │
│  ┌──────────────────────────────┐   │
│  │   MessageFetcher             │   │
│  │   - FetchBatch()             │   │
│  │   - HandleFetchError()       │   │
│  └──────────┬───────────────────┘   │
│             │                       │
│  ┌──────────▼───────────────────┐   │
│  │   MessageProcessor           │   │
│  │   - ProcessBatch()           │   │
│  │   - ProcessSingleMessage()   │   │
│  └──────────┬───────────────────┘   │
│             │                       │
│  ┌──────────▼───────────────────┐   │
│  │   ErrorHandler               │   │
│  │   - Classify errors          │   │
│  │   - Apply backpressure       │   │
│  └──────────┬───────────────────┘   │
│             │                       │
│  ┌──────────▼───────────────────┐   │
│  │   DLQHandler                 │   │
│  │   - RouteToDLQ()             │   │
│  │   - AdvisoryDLQHandler       │   │
│  └──────────────────────────────┘   │
└─────────────────────────────────────┘
          │
          │ ACK/NAK
          ▼
┌─────────────────────────────────────┐
│    Application Processor            │
│    (Business Logic)                 │
└─────────────────────────────────────┘
```

### Architecture Principles

1. **Dependency Inversion**: High-level modules (Consumer, Publisher) depend on abstractions (interfaces), not concrete implementations
2. **Separation of Concerns**: Each component has a single responsibility:
   - `MessageFetcher`: Handles message retrieval
   - `MessageProcessor`: Handles message processing logic
   - `ErrorHandler`: Handles error classification and recovery
   - `DLQHandler`: Handles dead letter queue routing
3. **Testability**: All components can be mocked and tested independently
4. **Extensibility**: New implementations can be added without modifying existing code

## Logic Flow

### Consumer Processing Flow

The consumer follows a well-defined processing loop with error handling at each stage:

```
┌─────────────────────────────────────────────────────────────┐
│                    Consumer Start                           │
│  1. Initialize components (Fetcher, Processor, DLQ)        │
│  2. Create/validate JetStream resources                     │
│  3. Start advisory DLQ handler (if enabled)                 │
│  4. Start metrics collection goroutine                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    Main Loop                                │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Step 1: Check Context                               │  │
│  │  - If cancelled, exit gracefully                     │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│  ┌──────────────────▼───────────────────────────────────┐  │
│  │  Step 2: Fetch Batch                                 │  │
│  │  - Fetch up to BatchSize messages                    │  │
│  │  - Wait up to BatchTimeout                           │  │
│  │  - Handle fetch errors with recovery                 │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│         ┌───────────┴───────────┐                          │
│         │                       │                          │
│    Success                  Error                          │
│         │                       │                          │
│         │              ┌────────▼────────┐                │
│         │              │ Handle Error    │                │
│         │              │ - Classify type │                │
│         │              │ - Apply backoff │                │
│         │              │ - Recover if dev│                │
│         │              └────────┬────────┘                │
│         │                       │                          │
│         │              ┌────────▼────────┐                │
│         │              │ Continue?       │                │
│         │              └────────┬────────┘                │
│         │                       │                          │
│         │              Yes      │ No                       │
│         │              │        │                          │
│         │              └───┬────┴───┐                      │
│         │                  │        │                      │
│         │              Continue   Exit                    │
│         │                  │                              │
│         └──────────────────┘                              │
│                     │                                       │
│  ┌──────────────────▼───────────────────────────────────┐  │
│  │  Step 3: Process Batch                               │  │
│  │  - For each message in batch:                        │  │
│  │    * Check context                                   │  │
│  │    * Extract message ID                              │  │
│  │    * Create tracing span                             │  │
│  │    * Call processor.Handle()                         │  │
│  │    * Handle result (ACK/NAK/DLQ)                     │  │
│  └──────────────────────────────────────────────────────┘  │
│                     │                                       │
│         ┌───────────┴───────────┐                          │
│         │                       │                          │
│    Success                  Error                          │
│         │                       │                          │
│         │              ┌────────▼────────┐                │
│         │              │ Classify Error  │                │
│         │              └────────┬────────┘                │
│         │                       │                          │
│         │         ┌─────────────┴─────────────┐          │
│         │         │                           │          │
│         │    Permanent                    Transient      │
│         │         │                           │          │
│         │  ┌──────▼──────┐          ┌────────▼──────┐   │
│         │  │ Route to DLQ│          │ NAK with delay│   │
│         │  │ ACK message │          │ Apply backpres│   │
│         │  └──────┬──────┘          └────────┬──────┘   │
│         │         │                           │          │
│         └─────────┴───────────────────────────┘          │
│                     │                                       │
│  ┌──────────────────▼───────────────────────────────────┐  │
│  │  Step 4: Reset Error Streak (if successful)          │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│                     └─────────── Loop ─────────────────────┘
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

#### Component Logic

**MessageFetcher (`defaultMessageFetcher`)**
- Fetches batches of messages using `sub.Fetch(batchSize, MaxWait(timeout))`
- Handles fetch errors with exponential backoff
- Recovers subscriptions when connection issues occur
- Context-aware: respects cancellation signals

**MessageProcessor (`defaultBatchProcessor`)**
- Processes messages sequentially within a batch
- Extracts message IDs (header → metadata → generated)
- Creates OpenTelemetry spans for tracing
- Calls application processor for business logic
- Handles ACK/NAK based on processing results

**ErrorHandler**
- Classifies errors as transient or permanent using `app.IsPermanent()`
- Tracks consecutive error streaks
- Applies backpressure when streak exceeds threshold (default: 10)
- Calculates backoff delays for retries

**DLQHandler (`defaultDLQHandler`)**
- Routes permanent errors to DLQ with enriched metadata
- Validates DLQ stream exists at startup
- Publishes DLQ messages with error context

**AdvisoryDLQHandler**
- Subscribes to JetStream advisory events: `$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.*`
- Handles messages that exhaust MaxDeliver attempts
- Retrieves original message from stream using `GetMsg()`
- Routes to DLQ with advisory metadata

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

#### Component Logic

**Publishing Flow**
1. **Validation**: Checks that publisher topic is configured
2. **Serialization**: Marshals message to JSON using `json.Marshal()`
3. **ID Extraction**: Extracts UUID from message (if `ParsedTelegram` type) or generates new UUID
4. **Header Attachment**: Sets `Nats-Msg-Id` header for deduplication
5. **Publishing**: Calls `js.PublishMsg()` to publish to JetStream
6. **Error Handling**: Classifies errors as transient (`ErrNoResponders`) or permanent

**Deduplication Strategy**
- Uses `Nats-Msg-Id` header for JetStream deduplication
- Extracts UUID from `ParsedTelegram.Uuid` field if available
- Falls back to generating new UUID if not present
- JetStream uses this header to prevent duplicate message processing

**Error Classification**
- **Transient**: `nats.ErrNoResponders` - JetStream temporarily unavailable, should retry
- **Permanent**: Other errors - configuration issues, should fail fast

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

## Simple Examples

### Example 1: Complete Consumer Setup and Start

This example shows how to set up and start a consumer from scratch:

```go
package main

import (
    "context"
    "time"
    
    "caatsm/internal/infra/config"
    "caatsm/internal/infra/nats"
    "caatsm/internal/app"
    "go.uber.org/zap"
)

func main() {
    // 1. Load configuration
    cfg := &config.Config{
        NATS: config.NATSConfig{
            URL:  "nats://localhost:4222",
            Mode: "jetstream",
            Stream: "TELEGRAM",
            Consumer: "telegram-consumer",
            ConsumerRules: config.ConsumerRules{
                AckWait:     30 * time.Second,
                MaxDeliver:  3,
                Backoff:     []time.Duration{1*time.Second, 2*time.Second, 5*time.Second},
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
    
    // 2. Create NATS connection
    nc, _ := nats.Connect(cfg.NATS.URL)
    defer nc.Close()
    
    // 3. Get JetStream context
    js, _ := nc.JetStream()
    
    // 4. Create message processor (your business logic)
    processor := app.NewMessageProcessor(/* dependencies */)
    
    // 5. Create logger
    logger, _ := zap.NewProduction()
    
    // 6. Create telemetry recorder
    telemetry := /* your telemetry implementation */
    
    // 7. Create consumer
    consumer, err := natsinfra.ProvideConsumer(
        nc,
        js,
        processor,
        cfg,
        telemetry,
        logger,
    )
    if err != nil {
        logger.Fatal("Failed to create consumer", zap.Error(err))
    }
    
    // 8. Start consumer with context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Handle graceful shutdown
    go func() {
        // Wait for interrupt signal
        <-ctx.Done()
        shutdownCtx, _ := context.WithTimeout(context.Background(), 5*time.Second)
        consumer.Shutdown(shutdownCtx)
    }()
    
    // 9. Start consuming (blocks until context cancelled)
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
    "caatsm/internal/adapter/dto"
    "caatsm/internal/infra/nats"
    "github.com/google/uuid"
)

func publishMessage(publisher port.Publisher) error {
    // Create message with UUID
    message := &dto.ParsedTelegram{
        Uuid: uuid.NewString(), // Used for deduplication
        Data: []byte("telegram message data"),
        // ... other fields
    }
    
    // Publish - automatically handles:
    // - JSON serialization
    // - UUID header attachment
    // - Error classification
    if err := publisher.Publish(message); err != nil {
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
    err := processor.Handle(ctx, msg.Data, msgID)
    
    // Step 2d: Handle result
    if err != nil {
        if app.IsPermanent(err) {
            // Permanent error: route to DLQ and ACK
            dlqHandler.RouteToDLQ(ctx, msg, err)
            msg.Ack()
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
// Scenario 1: Transient Error (Network Issue)
func processMessage(msg *nats.Msg) error {
    // Simulate network error
    if networkDown {
        return app.NewTransientError("network unavailable")
    }
    // Result: Message is NAK'd, will be redelivered with backoff
}

// Scenario 2: Permanent Error (Invalid Format)
func processMessage(msg *nats.Msg) error {
    var data MyStruct
    if err := json.Unmarshal(msg.Data, &data); err != nil {
        return app.NewPermanentError("invalid JSON format")
    }
    // Result: Message routed to DLQ, original message ACK'd
}

// Scenario 3: Backpressure Trigger
// When 10+ consecutive errors occur:
// - Processing pauses
// - Sleep duration = min(consecutiveErrors * 100ms, 5s)
// - Prevents overwhelming the system

// Scenario 4: MaxDeliver Exhausted
// When message fails MaxDeliver times (default: 3):
// - JetStream publishes advisory event
// - AdvisoryDLQHandler catches event
// - Retrieves original message
// - Routes to DLQ with metadata
```

### Example 5: DLQ Message Structure

What a DLQ message looks like:

```json
{
  "transport_msg_id": "550e8400-e29b-41d4-a716-446655440000",
  "subject": "telegram.orders.12345",
  "stream": "TELEGRAM",
  "consumer": "telegram-consumer",
  "nats_sequence": 12345,
  "deliveries": 3,
  "error": "permanent error: invalid message format",
  "received_at": "2024-01-15T10:30:00Z",
  "body": "{\"order_id\":123,\"invalid\":\"data\"}",
  "advisory_source": false
}
```

### Example 6: Configuration Examples

Different configuration scenarios:

```toml
# Example 1: High Throughput Configuration
[NATS]
Mode = "jetstream"
Stream = "TELEGRAM"
Consumer = "telegram-consumer"

[NATS.ConsumerRules]
AckWait = "60s"
MaxDeliver = 5
MaxAckPending = 5000
Backoff = ["1s", "2s", "5s", "10s", "30s"]

[App]
BatchSize = 100        # Larger batches
BatchTimeout = "5s"    # Longer timeout

# Example 2: Low Latency Configuration
[App]
BatchSize = 10         # Smaller batches
BatchTimeout = "500ms" # Shorter timeout

# Example 3: Development Mode (Self-Healing)
[NATS]
Mode = "jetstream"
# Missing streams/consumers auto-created

# Example 4: Production Mode (Fail Fast)
[NATS]
Mode = "jetstream"
# Missing streams/consumers cause startup failure
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