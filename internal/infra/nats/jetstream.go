package nats

import (
	"caatsm/internal/infra/config"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ProvideNATSConn creates a reusable NATS connection.
func ProvideNATSConn(cfg *config.Config, logger *zap.Logger) (*nats.Conn, error) {
	nc, err := nats.Connect(
		cfg.NATS.URL,
		nats.RetryOnFailedConnect(true),
		nats.Timeout(cfg.Timeouts.Server),
		nats.ReconnectWait(cfg.Timeouts.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return nc, nil
}

// ProvideJetStream creates a NATS JetStream context using an existing connection.
func ProvideJetStream(nc *nats.Conn, cfg *config.Config, logger *zap.Logger) (nats.JetStreamContext, error) {
	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Create stream if it doesn't exist
	streamName := cfg.NATS.Stream
	consumerSubject := cfg.EffectiveSubscriptionTopic()
	publisherSubject := strings.TrimSpace(cfg.Publisher.Topic)

	streamSubjects := dedupeSubjects([]string{consumerSubject, publisherSubject})
	if len(streamSubjects) == 0 {
		nc.Close()
		return nil, fmt.Errorf("no subjects configured for JetStream stream %s", streamName)
	}

	streamLimits := cfg.NATS.StreamLimits
	storage := nats.FileStorage
	switch strings.ToLower(streamLimits.Storage) {
	case "memory":
		storage = nats.MemoryStorage
	case "file":
		storage = nats.FileStorage
	}

	discard := nats.DiscardOld
	if strings.EqualFold(streamLimits.Discard, "new") {
		discard = nats.DiscardNew
	}

	streamConfig := &nats.StreamConfig{
		Name:      streamName,
		Subjects:  streamSubjects,
		Retention: nats.LimitsPolicy,
		MaxMsgs:   streamLimits.MaxMsgs,
		MaxBytes:  streamLimits.MaxBytes,
		MaxAge:    streamLimits.MaxAge,
		Discard:   discard,
		Storage:   storage,
		Replicas:  streamLimits.Replicas,
	}

	info, err := js.StreamInfo(streamName)
	if err != nil {
		if errors.Is(err, nats.ErrStreamNotFound) {
			if shouldBootstrapStream() {
				if _, err = js.AddStream(streamConfig); err != nil {
					nc.Close()
					return nil, fmt.Errorf("failed to create stream: %w", err)
				}
				logger.Info("Created JetStream",
					zap.String("stream", streamName),
					zap.Strings("subjects", streamSubjects),
				)
			} else {
				nc.Close()
				return nil, fmt.Errorf("stream %s not found and auto-creation disabled", streamName)
			}
		} else {
			nc.Close()
			return nil, fmt.Errorf("failed to fetch stream info: %w", err)
		}
	} else {
		validateStreamConfig(info, streamSubjects, logger)
	}

	return js, nil
}

func shouldBootstrapStream() bool {
	switch strings.ToLower(os.Getenv("GO_ENV")) {
	case "", "dev", "development", "test", "testing":
		return true
	default:
		return false
	}
}

func validateStreamConfig(info *nats.StreamInfo, expectedSubjects []string, logger *zap.Logger) {
	if info == nil {
		return
	}
	defer func() {
		if len(expectedSubjects) == 0 {
			expectedSubjects = []string{"<none>"}
		}
	}()

	missing := make([]string, 0)
	for _, subj := range expectedSubjects {
		if subj == "" {
			continue
		}
		if !containsSubject(info.Config.Subjects, subj) {
			missing = append(missing, subj)
		}
	}
	if len(missing) > 0 {
		logger.Warn("JetStream stream subjects missing expected entries",
			zap.String("stream", info.Config.Name),
			zap.Strings("stream_subjects", info.Config.Subjects),
			zap.Strings("missing_subjects", missing),
		)
	}
}

func containsSubject(subjects []string, target string) bool {
	for _, s := range subjects {
		if s == target {
			return true
		}
	}
	return false
}

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
