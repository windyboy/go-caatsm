package nats

import (
	"caatsm/internal/config"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Connect establishes a NATS connection with configured authentication, TLS, and reconnect settings.
func Connect(cfg *config.Config, logger *zap.Logger) (*nats.Conn, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	reconnectWait := cfg.Timeouts.ReconnectWait
	if reconnectWait <= 0 {
		reconnectWait = 5 * time.Second
	}

	opts := []nats.Option{
		nats.ReconnectWait(reconnectWait),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info("NATS reconnected")
		}),
	}

	switch {
	case cfg.NATS.Auth.CredentialsFile != "":
		opts = append(opts, nats.UserCredentials(cfg.NATS.Auth.CredentialsFile))
	case cfg.NATS.Auth.User != "":
		opts = append(opts, nats.UserInfo(cfg.NATS.Auth.User, cfg.NATS.Auth.Password))
	case cfg.NATS.Auth.Token != "":
		opts = append(opts, nats.Token(cfg.NATS.Auth.Token))
	}

	if cfg.NATS.Auth.TLSEnabled {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}

		if cfg.NATS.Auth.TLSCertFile != "" && cfg.NATS.Auth.TLSKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.NATS.Auth.TLSCertFile, cfg.NATS.Auth.TLSKeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		if cfg.NATS.Auth.TLSCAFile != "" {
			caCert, err := os.ReadFile(cfg.NATS.Auth.TLSCAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
		}

		opts = append(opts, nats.Secure(tlsConfig))
	}

	safeURL := sanitizeURLForLogging(cfg.NATS.URL)
	nc, err := nats.Connect(cfg.NATS.URL, opts...)
	if err != nil {
		logger.Error("failed to connect to NATS",
			zap.String("url", safeURL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Info("Connected to NATS", zap.String("url", safeURL))
	return nc, nil
}

// JetStream creates a JetStream context from an active NATS connection.
func JetStream(nc *nats.Conn, logger *zap.Logger) (nats.JetStreamContext, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if nc == nil {
		return nil, fmt.Errorf("NATS connection is nil")
	}

	js, err := nc.JetStream()
	if err != nil {
		logger.Error("failed to create JetStream context", zap.Error(err))
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	return js, nil
}
