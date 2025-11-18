package nats

import (
	"caatsm/internal/infra/config"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ProvideNATSConn creates a reusable NATS connection with optional authentication.
func ProvideNATSConn(cfg *config.Config, logger *zap.Logger) (*nats.Conn, error) {
	opts := []nats.Option{
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
	}

	// Apply authentication options
	authOpts, err := buildAuthOptions(&cfg.NATS.Auth, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build auth options: %w", err)
	}
	opts = append(opts, authOpts...)

	nc, err := nats.Connect(cfg.NATS.URL, opts...)
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

// buildAuthOptions builds NATS connection options based on authentication configuration.
func buildAuthOptions(auth *config.NATSAuthConfig, logger *zap.Logger) ([]nats.Option, error) {
	var opts []nats.Option
	authMethods := 0

	// Token authentication (highest priority)
	if auth.Token != "" {
		authMethods++
		logger.Debug("Using NATS token authentication")
		opts = append(opts, nats.Token(auth.Token))
	}

	// Credentials file authentication
	if auth.CredentialsFile != "" {
		authMethods++
		if authMethods > 1 {
			return nil, fmt.Errorf("multiple authentication methods specified: only one of token, credentials_file, or user/password can be used")
		}
		logger.Debug("Using NATS credentials file authentication", zap.String("file", auth.CredentialsFile))
		opts = append(opts, nats.UserCredentials(auth.CredentialsFile))
	}

	// User/Password authentication
	if auth.User != "" || auth.Password != "" {
		authMethods++
		if authMethods > 1 {
			return nil, fmt.Errorf("multiple authentication methods specified: only one of token, credentials_file, or user/password can be used")
		}
		if auth.User == "" || auth.Password == "" {
			return nil, fmt.Errorf("both user and password must be specified for user/password authentication")
		}
		logger.Debug("Using NATS user/password authentication", zap.String("user", auth.User))
		opts = append(opts, nats.UserInfo(auth.User, auth.Password))
	}

	// TLS configuration
	if auth.TLSEnabled {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		// Load client certificate and key if provided
		if auth.TLSCertFile != "" && auth.TLSKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(auth.TLSCertFile, auth.TLSKeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
			logger.Debug("Loaded TLS client certificate", zap.String("cert", auth.TLSCertFile))
		}

		// Load CA certificate for server verification if provided
		if auth.TLSCAFile != "" {
			caCert, err := os.ReadFile(auth.TLSCAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate from %s", auth.TLSCAFile)
			}
			tlsConfig.RootCAs = caCertPool
			logger.Debug("Loaded TLS CA certificate", zap.String("ca_file", auth.TLSCAFile))
		}

		opts = append(opts, nats.Secure(tlsConfig))
		logger.Debug("TLS enabled for NATS connection")
	}

	return opts, nil
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

	// Ensure the stream exists using StreamManager
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
		nc.Close()
		return nil, fmt.Errorf("no subjects configured for JetStream stream %s", streamName)
	}

	streamManager := NewStreamManagerWithConfig(js, streamName, streamSubjects, &cfg.NATS.StreamLimits, logger)
	if err := streamManager.EnsureStream(); err != nil {
		nc.Close()
		return nil, err
	}

	return js, nil
}
