package observability

import (
	"caatsm/internal/buildinfo"
	"caatsm/internal/config"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Server exposes /livez, /healthz, /readyz, and /metrics endpoints.
type Server struct {
	cfg        config.MonitoringConfig
	logger     *zap.Logger
	pool       *pgxpool.Pool
	conn       *nats.Conn
	httpServer *http.Server
}

// NewServer creates a monitoring HTTP server if enabled in configuration.
func NewServer(
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
		mux.HandleFunc("/livez", server.handleLive)
		mux.HandleFunc("/healthz", server.handleHealth)
		mux.HandleFunc("/readyz", server.handleHealth)
		routes++
	}
	if cfg.Monitoring.EnableMetrics {
		mux.Handle("/metrics", MetricsHandler())
		routes++
	}
	if routes == 0 {
		return nil, nil
	}

	server.httpServer = &http.Server{
		Addr:              cfg.Monitoring.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       cfg.Monitoring.ReadTimeout,
		WriteTimeout:      cfg.Monitoring.WriteTimeout,
	}
	return server, nil
}

// Start launches the monitoring HTTP server in the background.
func (s *Server) Start(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.Shutdown(shutCtx)
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
	deps := map[string]map[string]any{
		"postgres": {"status": "ok"},
		"nats":     {"status": "ok"},
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.healthTimeout())
	defer cancel()

	if s.pool == nil {
		deps["postgres"]["status"] = "unconfigured"
		status = http.StatusServiceUnavailable
	} else {
		start := time.Now()
		if err := s.pool.Ping(ctx); err != nil {
			deps["postgres"]["status"] = err.Error()
			status = http.StatusServiceUnavailable
		} else {
			deps["postgres"]["latency_ms"] = time.Since(start).Milliseconds()
		}
	}

	if s.conn == nil {
		deps["nats"]["status"] = "unconfigured"
		status = http.StatusServiceUnavailable
	} else if s.conn.Status() != nats.CONNECTED {
		deps["nats"]["status"] = s.conn.Status().String()
		status = http.StatusServiceUnavailable
	}

	payload := map[string]any{
		"status": httpStatusLabel(status),
		"build": map[string]any{
			"version":  buildinfo.Version,
			"rev":      buildinfo.Commit,
			"built_at": buildinfo.BuiltAt,
		},
		"dependencies": deps,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) healthTimeout() time.Duration {
	if s.cfg.HealthTimeout <= 0 {
		return 2 * time.Second
	}
	return s.cfg.HealthTimeout
}

func (s *Server) handleLive(w http.ResponseWriter, _ *http.Request) {
	payload := map[string]any{
		"status": "ok",
		"build": map[string]any{
			"version":  buildinfo.Version,
			"rev":      buildinfo.Commit,
			"built_at": buildinfo.BuiltAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func httpStatusLabel(code int) string {
	if code >= 200 && code < 300 {
		return "ok"
	}
	return "error"
}
