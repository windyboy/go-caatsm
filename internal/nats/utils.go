package nats

import (
	"errors"
	"net/url"
	"strings"

	"github.com/nats-io/nats.go"
)

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
	switch strings.ToLower(value) {
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
	switch strings.ToLower(value) {
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
