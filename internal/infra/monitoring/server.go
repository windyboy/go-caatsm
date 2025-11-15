package monitoring

import (
	"caatsm/internal/infra/config"
	obsmetrics "caatsm/internal/observability/metrics"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Server exposes /healthz and /metrics endpoints for basic operations and observability checks.
type Server struct {
	cfg        config.MonitoringConfig
	logger     *zap.Logger
	pool       *pgxpool.Pool
	conn       *nats.Conn
	httpServer *http.Server
}

// ProvideServer wires a monitoring server if enabled in configuration.
func ProvideServer(
	cfg *config.Config,
	logger *zap.Logger,
	pool *pgxpool.Pool,
	conn *nats.Conn,
) (*Server, error) {
	if cfg == nil || logger == nil || cfg.Monitoring.Disabled {
		return nil, nil
	}
	if cfg.Monitoring.Addr == "" {
		return nil, nil
	}

	mux := http.NewServeMux()
	server := &Server{
		cfg:    cfg.Monitoring,
		logger: logger,
		pool:   pool,
		conn:   conn,
	}

	routes := 0
	if cfg.Monitoring.EnableHealth {
		mux.HandleFunc("/healthz", server.handleHealth)
		routes++
	}
	if cfg.Monitoring.EnableMetrics {
		mux.Handle("/metrics", obsmetrics.Handler())
		routes++
	}

	if routes == 0 {
		return nil, nil
	}

	httpServer := &http.Server{
		Addr:              cfg.Monitoring.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       cfg.Monitoring.ReadTimeout,
		WriteTimeout:      cfg.Monitoring.WriteTimeout,
	}
	server.httpServer = httpServer

	return server, nil
}

// Start launches the monitoring HTTP server in the background.
func (s *Server) Start(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}

	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()

	go func() {
		s.logger.Info("Monitoring server listening",
			zap.String("addr", s.httpServer.Addr),
			zap.Bool("metrics", s.cfg.EnableMetrics),
			zap.Bool("health", s.cfg.EnableHealth),
		)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("Monitoring server exited", zap.Error(err))
		}
	}()

	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	result := map[string]interface{}{
		"postgres": "ok",
		"nats":     "ok",
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.healthTimeout())
	defer cancel()

	if s.pool == nil {
		result["postgres"] = "unconfigured"
		status = http.StatusServiceUnavailable
	} else if err := s.pool.Ping(ctx); err != nil {
		result["postgres"] = err.Error()
		status = http.StatusServiceUnavailable
	}

	if s.conn == nil {
		result["nats"] = "unconfigured"
		status = http.StatusServiceUnavailable
	} else if s.conn.Status() != nats.CONNECTED {
		result["nats"] = s.conn.Status().String()
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) healthTimeout() time.Duration {
	timeout := s.cfg.HealthTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return timeout
}
