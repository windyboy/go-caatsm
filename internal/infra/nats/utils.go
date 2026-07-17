package nats

import (
	"errors"
	"net/url"
	"strings"

	"github.com/nats-io/nats.go"
)

// isJetStreamResourceNotFound checks if an error indicates missing JetStream resources.
func isJetStreamResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, nats.ErrStreamNotFound) || errors.Is(err, nats.ErrConsumerNotFound) {
		return true
	}

	// Some JetStream API errors are only exposed via error strings.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "stream not found") || strings.Contains(msg, "consumer not found")
}

// sanitizeURLForLogging removes credentials from URLs for safe logging.
func sanitizeURLForLogging(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	// Parse URL
	u, err := url.Parse(rawURL)
	if err != nil {
		// If parsing fails, return a safe placeholder
		return "***"
	}

	// Rebuild without credentials
	u.User = nil
	return u.String()
}

// mapDeliverPolicy maps string configuration to NATS DeliverPolicy.
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

// mapReplayPolicy maps string configuration to NATS ReplayPolicy.
func mapReplayPolicy(value string) nats.ReplayPolicy {
	switch strings.ToLower(value) {
	case "original":
		return nats.ReplayOriginalPolicy
	default:
		return nats.ReplayInstantPolicy
	}
}

// dedupeSubjects removes duplicate and empty subjects from a list.
func dedupeSubjects(subjects []string) []string {
	seen := make(map[string]struct{})
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
