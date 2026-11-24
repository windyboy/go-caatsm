package pipeline

import (
	"caatsm/internal/telegram"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type noopRecorder struct{}

func (noopRecorder) RecordProcessingResult(context.Context, string, string, time.Duration) {}
func (noopRecorder) RecordPublishFailure(context.Context, string)                          {}
func (noopRecorder) RecordFailure(string)                                                  {}
func (noopRecorder) RecordAFTNValidationError(context.Context, string)                     {}

// Processor orchestrates parsing, optional AFTN validation, persistence, and publishing.
type Processor struct {
	parser    Parser
	validator Validator
	store     Store
	publisher Publisher
	recorder  Recorder
	logger    *zap.Logger
	opts      Options
}

// NewProcessor constructs a Processor from consumer-defined interfaces.
func NewProcessor(
	parser Parser,
	validator Validator,
	store Store,
	publisher Publisher,
	recorder Recorder,
	logger *zap.Logger,
	opts Options,
) *Processor {
	if recorder == nil {
		recorder = noopRecorder{}
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Processor{
		parser:    parser,
		validator: validator,
		store:     store,
		publisher: publisher,
		recorder:  recorder,
		logger:    logger,
		opts:      opts,
	}
}

// Handle processes a raw telegram message end-to-end.
func (p *Processor) Handle(ctx context.Context, raw []byte, msgID string) error {
	if len(raw) == 0 {
		return Permanent(fmt.Errorf("empty message"))
	}

	tracer := otel.Tracer("caatsm/pipeline")
	ctx, span := tracer.Start(ctx, "MessageProcessor.Handle")
	defer span.End()

	span.SetAttributes(
		attribute.String("messaging.system", "nats"),
		attribute.String("messaging.operation", "receive"),
		attribute.String("messaging.message_id", msgID),
		attribute.String("caatsm.component", "processor"),
	)

	receivedAt := time.Now()

	parsed, parseErr := p.parser.Parse(string(raw))
	if parsed == nil {
		parsed = telegram.NewMessage()
		parsed.UUID = telegram.NewUUID()
		parsed.Content = string(raw)
		parsed.ErrorReason = "parser returned nil"
		parsed.Status = telegram.StatusBodyError
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
	if parsed.Status == telegram.StatusUnknown {
		if parseErr == nil {
			parsed.Status = telegram.StatusParsed
		} else {
			parsed.Status = telegram.StatusBodyError
		}
	}

	span.SetAttributes(
		attribute.String("telegram.message_id", parsed.MessageID),
		attribute.String("telegram.category", parsed.Category),
		attribute.String("telegram.status", string(parsed.Status)),
	)

	msgLogger := p.logger.With(
		zap.String("service", "caatsm-processor"),
		zap.String("transport_msg_id", msgID),
		zap.String("telegram_message_id", parsed.MessageID),
		zap.String("category", parsed.Category),
	)

	if parseErr != nil || !parsed.Parsed {
		if parseErr == nil {
			parseErr = errors.New("message marked as unparsed")
		}
		if parsed.ErrorReason == "" {
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
		p.recorder.RecordFailure("parser")
		p.recorder.RecordProcessingResult(ctx, string(parsed.Status), parsed.Category, time.Since(receivedAt))
		return Permanent(fmt.Errorf("parser error: %w", parseErr))
	}
	parsed.ErrorReason = ""

	if p.opts.ValidateAFTN && p.validator != nil {
		if err := p.validator.Validate(parsed); err != nil {
			errType := telegram.GetAFTNErrorType(err)
			parsed.Status = telegram.StatusAFTNError
			parsed.ErrorReason = err.Error()
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.SetAttributes(attribute.String("aftn.error_type", errType))
			p.recorder.RecordAFTNValidationError(ctx, errType)
			p.persistRaw(ctx, parsed)
			msgLogger.With(zap.String("status", string(parsed.Status))).
				Warn("AFTN validation failed",
					zap.String("error_type", errType),
					zap.String("content_preview", truncateContent(parsed.Content, 256)),
					zap.Error(err),
				)
			p.recorder.RecordFailure("aftn_validator")
			p.recorder.RecordProcessingResult(ctx, string(parsed.Status), parsed.Category, time.Since(receivedAt))
			return Permanent(fmt.Errorf("AFTN validation error: %w", err))
		}
	}

	span.SetAttributes(
		attribute.String("telegram.status", string(parsed.Status)),
		attribute.Bool("telegram.parsed", parsed.Parsed),
		attribute.String("telegram.category", parsed.Category),
	)

	msgLogger.Info("Message parsed successfully",
		zap.Time("received_at", parsed.ReceivedAt),
		zap.Time("parsed_at", parsed.ParsedAt),
	)

	if err := p.store.InsertOne(ctx, parsed); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		elapsed := time.Since(receivedAt)
		p.recorder.RecordFailure("repository")
		p.recorder.RecordProcessingResult(ctx, string(parsed.Status), parsed.Category, elapsed)
		msgLogger.Error("Failed to persist parsed message",
			zap.String("status", string(parsed.Status)),
			zap.Duration("latency", elapsed),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert message: %w", err)
	}

	pubCtx, pubSpan := tracer.Start(ctx, "Publisher.Publish")
	if err := p.publisher.Publish(pubCtx, parsed); err != nil {
		msgLogger.With(zap.String("status", string(parsed.Status))).
			Error("Failed to publish message", zap.Error(err))
		pubSpan.RecordError(err)
		pubSpan.SetStatus(codes.Error, err.Error())
		pubSpan.End()

		p.recorder.RecordPublishFailure(ctx, parsed.Category)
		p.recorder.RecordFailure("publisher")
		p.recorder.RecordProcessingResult(ctx, string(parsed.Status), parsed.Category, time.Since(receivedAt))

		if IsPermanent(err) {
			parsed.ErrorReason = err.Error()
			p.persistRaw(ctx, parsed)
			return err
		}
		return fmt.Errorf("transient publish error: %w", err)
	}
	pubSpan.End()

	p.recorder.RecordProcessingResult(ctx, string(parsed.Status), parsed.Category, time.Since(receivedAt))
	return nil
}

func (p *Processor) persistRaw(ctx context.Context, msg *telegram.Message) {
	if msg == nil || p.store == nil {
		return
	}
	if msg.Content == "" && msg.BodyData != nil {
		msg.Content = fmt.Sprintf("%v", msg.BodyData)
	}
	if msg.ReceivedAt.IsZero() {
		msg.ReceivedAt = time.Now()
	}
	if err := p.store.InsertRaw(ctx, msg); err != nil {
		if span := trace.SpanFromContext(ctx); span.IsRecording() {
			span.RecordError(err)
		}
		return
	}
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.AddEvent("raw telegram persisted",
			trace.WithAttributes(
				attribute.String("telegram.status", string(msg.Status)),
				attribute.String("telegram.message_id", msg.MessageID),
			))
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
