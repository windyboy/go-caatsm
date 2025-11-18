# OpenTelemetry Best Practices

This document outlines the OpenTelemetry implementation and best practices for the CAATSM receiver service.

## Architecture Overview

The CAATSM receiver implements a dual-telemetry approach:

1. **OpenTelemetry (OTEL)**: Business metrics and distributed tracing
2. **Prometheus**: Operational metrics and alerting

## OTEL Implementation

### SDK Configuration

The application uses a production-ready OTEL SDK setup with:

- **Environment-based sampling**: Cost-effective production monitoring
- **Comprehensive resource attributes**: Service identification and metadata
- **Optimized batching**: Efficient export with retry logic
- **TLS security**: Configurable secure connections

### Sampling Strategy

```go
Production: 1%   // Cost-effective, maintains observability
Staging:   10%   // Balanced observability for testing
Dev/Test:  100%  // Full debugging coverage
```

### Resource Attributes

All telemetry includes standardized resource metadata:

```yaml
# Service identification
service.name: caatsm
service.version: dev
service.namespace: airport
service.component: receiver

# Environment context
deployment.environment: prod|staging|dev

# Build information
build.commit: <git-hash>
build.built_at: <timestamp>

# Telemetry configuration
telemetry.endpoint: <collector-url>
telemetry.insecure: true|false
```

## Span Semantics

### Messaging Spans

**NATS Consumer Operations**:
```yaml
Span: Consumer.processMessage
Attributes:
  messaging.system: nats
  messaging.operation: receive
  messaging.destination: telegram.serial
  messaging.consumer.id: telegram-consumer
  caatsm.stream: TELEGRAM
```

**Application Processing**:
```yaml
Span: MessageProcessor.Handle
Attributes:
  messaging.system: nats
  messaging.operation: receive
  messaging.message_id: <nats-msg-id>
  caatsm.component: processor
  caatsm.message.category: ARR|DEP|FPL|etc
```

### Database Spans

**Repository Operations**:
```yaml
Span: Repository.InsertOne|InsertBatch|InsertRaw
Attributes:
  db.system: postgresql
  db.operation: insert
  db.name: aviation
  db.table: telegrams
  caatsm.message.id: <telegram-id>
```

## Metrics Strategy

### OTEL Metrics (Business Focus)

- `caatsm_messages_processed_total{message.status, message.category}`
- `caatsm_publish_failures_total{message.category}`
- `caatsm_parse_duration_seconds{message.status, message.category}`

### Prometheus Metrics (Operational Focus)

- `caatsm_messages_total{stream, consumer, result}`
- `caatsm_handle_latency_seconds{stream, consumer}`
- `caatsm_db_queries_total{operation, result}`
- `caatsm_nats_consumer_pending_messages{stream, consumer}`

## Collector Configuration

### Development Setup

```yaml
# configs/otel-collector.dev.yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318
        max_request_body_size: 20971520
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:
    send_batch_size: 1024
    timeout: 1s
  resource:
    attributes:
      - key: service.instance.id
        value: "${POD_NAME}"
        action: upsert

exporters:
  logging:
    sampling_initial: 10
    sampling_thereafter: 100
  otlphttp/jaeger:
    endpoint: http://jaeger:4318
    sending_queue:
      queue_size: 10000
    retry_on_failure:
      enabled: true
  prometheus:
    endpoint: "0.0.0.0:8889"

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resource, batch]
      exporters: [logging, otlphttp/jaeger]
    metrics:
      receivers: [otlp]
      processors: [resource, batch]
      exporters: [logging, prometheus]
```

### Production Setup

```yaml
# configs/otel-collector.prod.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
        tls:
          cert_file: /etc/ssl/certs/otel-collector.crt
          key_file: /etc/ssl/private/otel-collector.key
        auth:
          authenticator: bearer_token
      http:
        endpoint: 0.0.0.0:4318
        tls:
          cert_file: /etc/ssl/certs/otel-collector.crt
          key_file: /etc/ssl/private/otel-collector.key
        auth:
          authenticator: bearer_token

extensions:
  health_check:
    endpoint: 0.0.0.0:13133
  pprof:
    endpoint: :1888
  zpages:
    endpoint: :55679

exporters:
  otlphttp/jaeger:
    endpoint: https://jaeger.company.com:4318
    headers:
      authorization: "Bearer ${JAEGER_API_TOKEN}"
    tls:
      insecure: false
    sending_queue:
      queue_size: 10000
    retry_on_failure:
      enabled: true
  prometheusremotewrite:
    endpoint: https://prometheus.company.com/api/v1/write
    headers:
      authorization: "Bearer ${PROMETHEUS_API_TOKEN}"
    tls:
      insecure: false
    sending_queue:
      queue_size: 10000
    retry_on_failure:
      enabled: true

service:
  extensions: [health_check, pprof, zpages]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resource, batch]
      exporters: [otlphttp/jaeger]
    metrics:
      receivers: [otlp]
      processors: [resource, batch]
      exporters: [prometheusremotewrite]
```

## Configuration Examples

### Development Configuration

```toml
[telemetry]
enabled = true
endpoint = "localhost:4318"
insecure = true
```

### Production Configuration

```toml
[telemetry]
enabled = true
endpoint = "otel-collector.company.com:4318"
insecure = false
```

### CLI Overrides

```bash
# Enable telemetry
./bin/receiver listen --telemetry-enabled

# Custom endpoint
./bin/receiver listen --telemetry-endpoint https://otel-collector.prod:4318

# Insecure for development
./bin/receiver listen --telemetry-insecure
```

## Monitoring and Debugging

### Health Checks

```bash
# Collector health
curl http://otel-collector:13133

# Application metrics
curl http://localhost:2112/metrics

# OTEL collector metrics
curl http://otel-collector:8888/metrics
```

### Tracing Verification

```bash
# Jaeger UI
open http://localhost:16686

# Search for caatsm traces
Service: caatsm
Operation: Consumer.processMessage OR MessageProcessor.Handle
```

### Metrics Verification

```bash
# Prometheus queries
caatsm_messages_processed_total
caatsm_parse_duration_seconds
rate(caatsm_messages_total[5m])
```

## Best Practices

### 1. Sampling Strategy
- Use environment-appropriate sampling rates
- Monitor sampling effectiveness
- Adjust based on cost and observability needs

### 2. Resource Attributes
- Include comprehensive service metadata
- Use semantic conventions
- Add custom attributes for business context

### 3. Span Attributes
- Follow OpenTelemetry semantic conventions
- Include relevant business context
- Avoid high-cardinality attributes

### 4. Error Handling
- Always record errors on spans
- Set appropriate span status
- Include error context in attributes

### 5. Performance
- Use batching to reduce export overhead
- Configure appropriate queue sizes
- Monitor exporter performance

## Troubleshooting

### Common Issues

1. **No traces in Jaeger**
   - Check collector logs: `docker logs otel-collector`
   - Verify endpoint configuration
   - Check network connectivity

2. **High sampling rate**
   - Adjust sampling configuration
   - Monitor cost impact
   - Consider head-based sampling

3. **Missing metrics**
   - Verify collector pipeline configuration
   - Check Prometheus remote write configuration
   - Validate metric names and labels

4. **Performance impact**
   - Review sampling rates
   - Check batch configuration
   - Monitor exporter queue sizes

### Debug Commands

```bash
# View collector configuration
docker exec otel-collector cat /etc/otel/config.yaml

# Check collector metrics
curl -s http://otel-collector:8888/metrics | grep otel

# View application telemetry logs
./bin/receiver listen --log-level=debug 2>&1 | grep -i telemetry
```