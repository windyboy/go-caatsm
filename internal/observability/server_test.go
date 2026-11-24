package observability

import (
	"caatsm/internal/buildinfo"
	"caatsm/internal/config"
	"encoding/json"
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
