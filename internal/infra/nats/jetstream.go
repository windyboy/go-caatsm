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
		// Use infinite reconnects so the app survives long NATS outages (e.g. docker compose down/up).
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			safeURL := sanitizeURLForLogging(nc.ConnectedUrl())
			logger.Info("NATS reconnected", zap.String("url", safeURL))
		}),
	)
	if err != nil {
		safeURL := sanitizeURLForLogging(cfg.NATS.URL)
		logger.Error("failed to connect to NATS",
			zap.String("url", safeURL),
			zap.Duration("timeout", cfg.Timeouts.Server),
			zap.Duration("reconnect_wait", cfg.Timeouts.ReconnectWait),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return nc, nil
}

// ProvideJetStream creates a NATS JetStream context using an existing connection.
// Returns nil, nil when cfg.NATS.Mode == "core" to support plain NATS servers without JetStream.
func ProvideJetStream(nc *nats.Conn, cfg *config.Config, logger *zap.Logger) (nats.JetStreamContext, error) {
	mode := strings.ToLower(cfg.NATS.Mode)
	if mode == "core" {
		logger.Info("Skipping JetStream initialization for core NATS mode")
		return nil, nil
	}

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		safeURL := sanitizeURLForLogging(cfg.NATS.URL)
		logger.Error("failed to get JetStream context",
			zap.String("url", safeURL),
			zap.Error(err),
		)
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Ensure the stream exists and is minimally aligned with configuration.
	if err := EnsureStream(js, cfg, logger); err != nil {
		nc.Close()
		return nil, err
	}

	return js, nil
}

// EnsureStream ensures that the configured JetStream stream exists and has
// at least the expected subjects bound. It is safe to call multiple times.
//
// In dev/test environments (see shouldBootstrapStream), the stream will be
// auto-created if it does not exist. In production, a missing stream results
// in an error so that operators can intervene.
func EnsureStream(js nats.JetStreamContext, cfg *config.Config, logger *zap.Logger) error {
	streamName := cfg.NATS.Stream
	consumerSubject := cfg.EffectiveSubscriptionTopic()
	publisherSubject := strings.TrimSpace(cfg.Publisher.Topic)

	streamSubjects := dedupeSubjects([]string{consumerSubject, publisherSubject})
	if len(streamSubjects) == 0 {
		logger.Error("no subjects configured for JetStream stream",
			zap.String("stream", streamName),
			zap.String("consumer_subject", consumerSubject),
			zap.String("publisher_subject", publisherSubject),
		)
		return fmt.Errorf("no subjects configured for JetStream stream %s", streamName)
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
					logger.Error("failed to create stream",
						zap.String("stream", streamName),
						zap.Strings("subjects", streamSubjects),
						zap.Error(err),
					)
					return fmt.Errorf("failed to create stream %s: %w", streamName, err)
				}
				logger.Info("Created JetStream stream",
					zap.String("stream", streamName),
					zap.Strings("subjects", streamSubjects),
				)
				return nil
			}
			logger.Error("stream not found and auto-creation disabled",
				zap.String("stream", streamName),
				zap.Strings("expected_subjects", streamSubjects),
			)
			return fmt.Errorf("stream %s not found and auto-creation disabled", streamName)
		}
		logger.Error("failed to fetch stream info",
			zap.String("stream", streamName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to fetch stream info for %s: %w", streamName, err)
	}

	// Stream exists: validate subjects but do not fail hard if they differ.
	validateStreamConfig(info, streamSubjects, logger)
	return nil
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
