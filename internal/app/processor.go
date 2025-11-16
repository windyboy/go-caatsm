package app

import (
	"caatsm/internal/adapter"
	"caatsm/internal/adapter/parser"
	"caatsm/internal/model"
	obslogging "caatsm/internal/observability/logging"
	"caatsm/internal/observability/telemetry"
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// MessageProcessor handles message processing
type MessageProcessor struct {
	parser     parser.Parser
	repository adapter.Repository
	publisher  adapter.Publisher
	logger     *zap.Logger
	telemetry  telemetry.Recorder
}

// ProcessingStatus represents the outcome of the processing pipeline
// (persistence, publishing, etc.), independent from the parsing status
// captured in model.MessageStatus.
type ProcessingStatus string

const (
	ProcessingStatusOK            ProcessingStatus = "ok"
	ProcessingStatusPersistFailed ProcessingStatus = "persist_failed"
	ProcessingStatusPublishFailed ProcessingStatus = "publish_failed"
)

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(
	parser parser.Parser,
	repository adapter.Repository,
	publisher adapter.Publisher,
	rec telemetry.Recorder,
	logger *zap.Logger,
) *MessageProcessor {
	return &MessageProcessor{
		parser:     parser,
		repository: repository,
		publisher:  publisher,
		logger:     logger,
		telemetry:  rec,
	}
}

// Handle processes a message
func (p *MessageProcessor) Handle(ctx context.Context, raw []byte, msgID string) error {
	if len(raw) == 0 {
		return Permanent(fmt.Errorf("empty message"))
	}

	tracer := otel.Tracer("caatsm/app")
	ctx, span := tracer.Start(ctx, "MessageProcessor.Handle")
	defer span.End()
	span.SetAttributes(attribute.String("nats.msg_id", msgID))

	receivedAt := time.Now()

	parsed, parseErr := p.parser.Parse(string(raw))
	if parsed == nil {
		parsed = model.NewParsedTelegram()
		parsed.Content = string(raw)
		parsed.ErrorReason = "parser returned nil"
		parsed.Status = model.MessageStatusBodyError
		parseErr = fmt.Errorf("parser returned nil")
	}

	if msgID != "" {
		if parsed.Comments == "" {
			parsed.Comments = fmt.Sprintf("nats_msg_id=%s", msgID)
		} else if !strings.Contains(parsed.Comments, "nats_msg_id=") {
			parsed.Comments = fmt.Sprintf("%s; nats_msg_id=%s", parsed.Comments, msgID)
		}
	}
	if parsed.ReceivedAt.IsZero() {
		parsed.ReceivedAt = receivedAt
	}
	if parsed.ParsedAt.IsZero() {
		parsed.ParsedAt = time.Now()
	}
	if parsed.Status == model.MessageStatusUnknown {
		if parseErr == nil {
			parsed.Status = model.MessageStatusParsed
		} else {
			parsed.Status = model.MessageStatusBodyError
		}
	}

	// Enrich span with parsed telegram information as soon as we have it.
	span.SetAttributes(
		attribute.String("telegram.message_id", parsed.MessageID),
		attribute.String("telegram.category", parsed.Category),
		attribute.String("telegram.status", string(parsed.Status)),
	)

	msgLogger := obslogging.WithMessageContext(p.logger, obslogging.MessageFields{
		Service:        "caatsm-processor",
		TransportMsgID: msgID,
		BusinessMsgID:  parsed.MessageID,
		Category:       parsed.Category,
	})

	if parseErr != nil || !parsed.Parsed {
		if parsed.ErrorReason == "" && parseErr != nil {
			parsed.ErrorReason = parseErr.Error()
		}
		span.RecordError(parseErr)
		span.SetStatus(codes.Error, parseErr.Error())
		p.persistRaw(ctx, parsed)
		msgLogger.With(zap.String("status", string(parsed.Status))).
			Warn("Message failed to parse",
				zap.String("content_preview", truncateContent(parsed.Content, 256)),
				zap.Error(parseErr),
			)
		latency := parsed.ParsedAt.Sub(receivedAt)
		p.telemetry.RecordFailure("parser")
		p.telemetry.RecordProcessingResult(ctx, string(parsed.Status), parsed.Category, latency)
		return Permanent(fmt.Errorf("parser error: %w", parseErr))
	}
	parsed.ErrorReason = ""

	// Log parsing result
	span.SetAttributes(
		attribute.String("telegram.status", string(parsed.Status)),
		attribute.Bool("telegram.parsed", parsed.Parsed),
		attribute.String("telegram.category", parsed.Category),
	)

	msgLogger.Info("Message parsed successfully",
		zap.Time("received_at", parsed.ReceivedAt),
		zap.Time("parsed_at", parsed.ParsedAt),
	)

	// Insert into database
	if err := p.repository.InsertOne(ctx, parsed); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		latency := parsed.ParsedAt.Sub(receivedAt)
		p.telemetry.RecordFailure("repository")
		p.telemetry.RecordProcessingResult(ctx, string(parsed.Status), parsed.Category, latency)
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Publish parsed message
	_, pubSpan := tracer.Start(ctx, "Publisher.Publish")
	if err := p.publisher.Publish(parsed); err != nil {
		// Log error but don't fail the entire operation
		msgLogger.With(zap.String("status", string(parsed.Status))).
			Error("Failed to publish message",
				zap.Error(err),
			)
		pubSpan.RecordError(err)
		pubSpan.SetStatus(codes.Error, err.Error())
		parsed.ErrorReason = err.Error()
		p.telemetry.RecordPublishFailure(ctx, parsed.Category)
		latency := parsed.ParsedAt.Sub(receivedAt)
		p.telemetry.RecordFailure("publisher")
		p.telemetry.RecordProcessingResult(ctx, string(parsed.Status), parsed.Category, latency)
		p.persistRaw(ctx, parsed)
		// Mark as permanent so the consumer will ack instead of retrying
		pubSpan.End()
		return Permanent(fmt.Errorf("failed to publish message: %w", err))
	}
	pubSpan.End()

	latency := parsed.ParsedAt.Sub(receivedAt)
	p.telemetry.RecordProcessingResult(ctx, string(parsed.Status), parsed.Category, latency)

	return nil
}

func (p *MessageProcessor) persistRaw(ctx context.Context, msg *model.ParsedTelegram) {
	if msg == nil || p.repository == nil {
		return
	}
	if msg.Content == "" && msg.BodyData != nil {
		msg.Content = fmt.Sprintf("%v", msg.BodyData)
	}
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = time.Now()
	}
	if err := p.repository.InsertRaw(ctx, msg); err != nil {
		p.logger.Error("Failed to persist raw telegram",
			zap.String("message_id", msg.MessageID),
			zap.String("status", string(msg.Status)),
			zap.Error(err),
		)
	} else {
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.AddEvent("raw telegram persisted",
				trace.WithAttributes(
					attribute.String("telegram.status", string(msg.Status)),
					attribute.String("telegram.message_id", msg.MessageID),
				))
		}
	}
}

func truncateContent(content string, limit int) string {
	if limit <= 0 || len(content) <= limit {
		return content
	}
	if limit <= 3 {
		return content[:limit]
	}
	return content[:limit-3] + "..."
}
