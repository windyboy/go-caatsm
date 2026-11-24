package observability

import (
	"caatsm/internal/buildinfo"
	"caatsm/internal/config"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Server", func() {
	It("returns nil when monitoring is disabled", func() {
		cfg := &config.Config{Monitoring: config.MonitoringConfig{Disabled: true, Addr: ":0"}}
		server, err := NewServer(cfg, zap.NewNop(), nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(server).To(BeNil())
	})

	It("fails synchronously when the monitoring address is already in use", func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = ln.Close() }()

		cfg := &config.Config{
			Monitoring: config.MonitoringConfig{
				Addr:          ln.Addr().String(),
				EnableHealth:  true,
				EnableMetrics: true,
			},
		}
		server, err := NewServer(cfg, zap.NewNop(), nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(server).NotTo(BeNil())

		err = server.Start(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("listen monitoring server"))
	})

	It("starts on an ephemeral port and shuts down cleanly", func() {
		cfg := &config.Config{
			Monitoring: config.MonitoringConfig{
				Addr:          "127.0.0.1:0",
				EnableHealth:  true,
				EnableMetrics: true,
			},
		}
		server, err := NewServer(cfg, zap.NewNop(), nil, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(server.Start(context.Background())).To(Succeed())

		shutCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		Expect(server.Shutdown(shutCtx)).To(Succeed())
	})

	Context("health and liveness handlers", func() {
		BeforeEach(func() {
			buildinfo.Version = "v-test"
			buildinfo.Commit = "abc"
			buildinfo.BuiltAt = "now"
		})

		It("returns build info on /livez", func() {
			server := &Server{cfg: config.MonitoringConfig{}, logger: zap.NewNop()}
			req := httptest.NewRequest(http.MethodGet, "/livez", nil)
			resp := httptest.NewRecorder()
			server.handleLive(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))
			payload := map[string]any{}
			Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
			Expect(payload["status"]).To(Equal("ok"))
		})

		It("reports unconfigured dependencies as 503", func() {
			server := &Server{
				cfg:    config.MonitoringConfig{EnableHealth: true, HealthTimeout: time.Second},
				logger: zap.NewNop(),
			}
			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			resp := httptest.NewRecorder()
			server.handleHealth(resp, req)

			Expect(resp.Code).To(Equal(http.StatusServiceUnavailable))
			payload := map[string]any{}
			Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
			deps := payload["dependencies"].(map[string]any)
			Expect(deps["postgres"].(map[string]any)["status"]).To(Equal("unconfigured"))
			Expect(deps["nats"].(map[string]any)["status"]).To(Equal("unconfigured"))
		})
	})
})
