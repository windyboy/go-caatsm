package nats

import (
	"caatsm/internal/infra/config"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ProvideNATSConn creates a basic NATS connection.
func ProvideNATSConn(cfg *config.Config, logger *zap.Logger) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.ReconnectWait(5 * time.Second),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected")
		}),
	}

	// Simple token authentication if provided
	if cfg.NATS.Auth.Token != "" {
		opts = append(opts, nats.Token(cfg.NATS.Auth.Token))
	}

	// Basic TLS support if enabled
	if cfg.NATS.Auth.TLSEnabled {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}

		// Load client certificate if provided
		if cfg.NATS.Auth.TLSCertFile != "" && cfg.NATS.Auth.TLSKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.NATS.Auth.TLSCertFile, cfg.NATS.Auth.TLSKeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		// Load CA certificate for server verification
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

	nc, err := nats.Connect(cfg.NATS.URL, opts...)
	if err != nil {
		logger.Error("failed to connect to NATS",
			zap.String("url", cfg.NATS.URL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return nc, nil
}

// ProvideJetStream creates a JetStream context from a NATS connection.
// Returns nil when mode is "core" to indicate JetStream should not be used.
func ProvideJetStream(nc *nats.Conn, cfg *config.Config, logger *zap.Logger) (nats.JetStreamContext, error) {
	// In core mode, return nil so that publishers/consumers use core NATS
	if cfg.NATS.Mode == "core" {
		logger.Debug("Skipping JetStream initialization (core mode)")
		return nil, nil
	}

	js, err := nc.JetStream()
	if err != nil {
		logger.Error("failed to create JetStream context", zap.Error(err))
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	return js, nil
}
