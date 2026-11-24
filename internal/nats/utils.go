package nats

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	resultOK            = "ok"
	resultFail          = "fail"
	resultPermanentFail = "permanent_fail"

	retryReasonProcessorError = "processor_error"
	retryReasonDLQError       = "dlq_publish_error"
)

type permanentIndicator interface {
	IsPermanent() bool
}

type permanentError struct {
	err error
}

func (e *permanentError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *permanentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func (e *permanentError) IsPermanent() bool {
	return e != nil
}

func markPermanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

func isPermanentError(err error) bool {
	var indicator permanentIndicator
	return errors.As(err, &indicator) && indicator.IsPermanent()
}

type noopRecorder struct{}

func (noopRecorder) RecordMessageHandled(context.Context, string, string, string, time.Duration) {}
func (noopRecorder) RecordRetry(context.Context, string, string, string)                         {}
func (noopRecorder) RecordDLQMessage(context.Context, string, string)                            {}
func (noopRecorder) RecordDLQPublishFailure(context.Context, string, string)                     {}
func (noopRecorder) RecordJSAPICall(string)                                                      {}
func (noopRecorder) RecordNATSConsumerPending(string, string, uint64)                            {}
func (noopRecorder) RecordMessageGap(string, string, float64)                                    {}
func (noopRecorder) RecordSequenceGap(string, string, uint64)                                    {}
func (noopRecorder) RecordSerialReaderHealth(string, string, bool)                               {}

func isJetStreamResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, nats.ErrStreamNotFound) || errors.Is(err, nats.ErrConsumerNotFound) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "stream not found") || strings.Contains(msg, "consumer not found")
}

func sanitizeURLForLogging(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "***"
	}
	u.User = nil
	return u.String()
}

func mapDeliverPolicy(value string) nats.DeliverPolicy {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "new":
		return nats.DeliverNewPolicy
	case "last":
		return nats.DeliverLastPolicy
	case "last_per_subject":
		return nats.DeliverLastPerSubjectPolicy
	case "sequence":
		return nats.DeliverByStartSequencePolicy
	case "time":
		return nats.DeliverByStartTimePolicy
	default:
		return nats.DeliverAllPolicy
	}
}

func mapReplayPolicy(value string) nats.ReplayPolicy {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "original":
		return nats.ReplayOriginalPolicy
	default:
		return nats.ReplayInstantPolicy
	}
}

func dedupeSubjects(subjects []string) []string {
	seen := make(map[string]struct{}, len(subjects))
	result := make([]string, 0, len(subjects))
	for _, subj := range subjects {
		subj = strings.TrimSpace(subj)
		if subj == "" {
			continue
		}
		if _, ok := seen[subj]; ok {
			continue
		}
		seen[subj] = struct{}{}
		result = append(result, subj)
	}
	return result
}

// natsHeaderCarrier adapts nats.Header to OpenTelemetry's TextMapCarrier.
type natsHeaderCarrier nats.Header

func (c natsHeaderCarrier) Get(key string) string {
	return nats.Header(c).Get(key)
}

func (c natsHeaderCarrier) Set(key, value string) {
	nats.Header(c).Set(key, value)
}

func (c natsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
