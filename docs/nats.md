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

### Architecture

The NATS integration follows simplified Clean Architecture with focused components:

```
┌─────────────────────────────────────┐
│         Application Layer           │
│   (Business logic & processing)     │
├─────────────────────────────────────┤
│         Infrastructure Layer        │
│   (NATS implementation)             │
│                                     │
│   ┌─────────────────────────────┐   │
│   │        Consumer             │   │
│   │   ┌─────────────────────┐   │   │
│   │   │  MessageFetcher     │   │   │
│   │   │  MessageProcessor   │   │   │
│   │   │  DLQHandler         │   │   │
│   │   └─────────────────────┘   │   │
│   └─────────────────────────────┘   │
│                                     │
│   ┌─────────────────────────────┐   │
│   │       Publisher             │   │
│   └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Component Interaction

```
┌──────────────┐
│   Publisher  │
│              │
│ 1. Serialize │
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
│  │   - ProcessMessage()         │   │
│  └──────────┬───────────────────┘   │
│             │                       │
│  ┌──────────▼───────────────────┐   │
│  │   DLQHandler                 │   │
│  │   - RouteToDLQ()             │   │
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
    - `DLQHandler`: Handles dead letter queue routing
3. **Testability**: All components can be mocked and tested independently
4. **Extensibility**: New implementations can be added without modifying existing code

## Logic Flow

### Consumer Processing Flow

The consumer follows a simplified processing loop:

```
┌─────────────────────────────────────────────────────────────┐
│                    Consumer Start                           │
│  1. Initialize components (Fetcher, Processor, DLQ)        │
│  2. Start main processing loop                              │
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
│  │  - Handle fetch errors with backoff                  │  │
│  └──────────────────┬───────────────────────────────────┘  │
│                     │                                       │
│         ┌───────────┴───────────┐                          │
│         │                       │                          │
│    Success                  Error                          │
│         │                       │                          │
│         │              ┌────────▼────────┐                │
│         │              │ Apply Backoff   │                │
│         │              │ Continue Loop   │                │
│         │              └─────────────────┘                │
│         │                                                 │
│         └──────────────────┬──────────────────────────────┘
│                            │
│  ┌─────────────────────────▼─────────────────────────────┐  │
│  │  Step 3: Process Batch                               │  │
│  │  - For each message in batch:                        │  │
│  │    * Extract message ID                              │  │
│  │    * Call processor.Handle()                         │  │
│  │    * Handle result (ACK/NAK/DLQ)                     │  │
│  └──────────────────┬───────────────────────────────────┘  │
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
│         │  │ ACK message │          └────────────────┘   │
│         │  └──────┬──────┘                              │
│         │         │                                       │
│         └─────────┴───────────────────────────────────────┘
│                     │
│                     └─────────── Loop ─────────────────────┘
└─────────────────────────────────────────────────────────────┘
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
- **Batch Processing**: Configurable batch sizes for efficient processing
- **Error Handling**: Distinguishes transient vs permanent errors
- **Dead Letter Queue (DLQ)**: Routes permanent errors to DLQ
- **TLS Support**: Secure connections with client certificates
- **Basic Monitoring**: Essential metrics collection

#### Component Logic

**MessageFetcher (`defaultMessageFetcher`)**
- Fetches batches of messages using `sub.Fetch(batchSize, MaxWait(timeout))`
- Handles fetch errors with simple exponential backoff
- Context-aware: respects cancellation signals

**MessageProcessor (`defaultBatchProcessor`)**
- Processes messages sequentially within a batch
- Extracts message IDs (header → metadata → generated)
- Calls application processor for business logic
- Handles ACK/NAK based on processing results

**DLQHandler (`defaultDLQHandler`)**
- Routes permanent errors to DLQ with basic metadata
- Validates DLQ stream exists at startup
- Publishes DLQ messages with error context

#### Configuration
```toml
[NATS]
URL = "nats://localhost:4222"
Stream = "TELEGRAM"
Consumer = "telegram-consumer"

[NATS.Auth]
Token = "your-token"  # Optional token authentication
TLSEnabled = true     # Enable TLS
TLSCertFile = "/path/to/client.crt"  # Client certificate
TLSKeyFile = "/path/to/client.key"   # Client private key
TLSCAFile = "/path/to/ca.crt"        # CA certificate

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

#### Recovery Strategies
- **Simple Backoff**: Exponential backoff for transient failures
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



## Configuration

### Environment Variables
```bash
CAATSM_NATS_URL=nats://localhost:4222
CAATSM_NATS_TOKEN=your-token  # Optional
CAATSM_DLQ_ENABLED=true
CAATSM_DLQ_SUBJECT=caatsm.dlq
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

    "caatsm/internal/infra/config"
    "caatsm/internal/infra/nats"
    "caatsm/internal/app"
    "go.uber.org/zap"
)

func main() {
    // 1. Load configuration
    cfg := &config.Config{
        NATS: config.NATSConfig{
            URL:    "nats://localhost:4222",
            Stream: "TELEGRAM",
            Consumer: "telegram-consumer",
            Auth: config.NATSAuthConfig{
                Token: "your-token", // Optional
            },
            ConsumerRules: config.ConsumerRules{
                AckWait:    30 * time.Second,
                MaxDeliver: 3,
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

    // 2. Create NATS connection
    nc, err := nats.ProvideNATSConn(cfg, zap.NewNop())
    if err != nil {
        panic(err)
    }
    defer nc.Close()

    // 3. Get JetStream context
    js, err := nats.ProvideJetStream(nc, zap.NewNop())
    if err != nil {
        panic(err)
    }

    // 4. Create message processor (your business logic)
    processor := app.NewMessageProcessor(/* dependencies */)

    // 5. Create logger
    logger := zap.NewNop()

    // 6. Create telemetry recorder
    telemetry := /* your telemetry implementation */

    // 7. Create consumer
    consumer, err := nats.ProvideConsumer(
        nc,
        js,
        processor,
        cfg,
        telemetry,
        logger,
    )
    if err != nil {
        panic(err)
    }

    // 8. Start consumer with context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

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
// - Message is not automatically handled
// - Consider monitoring JetStream consumer info for failed deliveries
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