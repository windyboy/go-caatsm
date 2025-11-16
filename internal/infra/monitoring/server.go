package monitoring

import (
	"caatsm/internal/infra/buildinfo"
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
		// Liveness: cheap process check that does not hit external dependencies.
		mux.HandleFunc("/livez", server.handleLive)
		// Backward-compatible health endpoint. For now this keeps the same
		// semantics as readiness but will remain stable for existing users.
		mux.HandleFunc("/healthz", server.handleHealth)
		// Readiness: dependency-aware check intended for load balancers and
		// orchestrators.
		mux.HandleFunc("/readyz", server.handleHealth)
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
	deps := map[string]map[string]interface{}{
		"postgres": {
			"status": "ok",
		},
		"nats": {
			"status": "ok",
		},
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

	payload := map[string]interface{}{
		"status": httpStatusLabel(status),
		"build": map[string]interface{}{
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
	timeout := s.cfg.HealthTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return timeout
}

// handleLive reports basic process liveness and build information without
// consulting external dependencies. It is suitable for liveness probes.
func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	payload := map[string]interface{}{
		"status": "ok",
		"build": map[string]interface{}{
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

