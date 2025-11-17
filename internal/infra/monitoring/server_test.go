package monitoring

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"caatsm/internal/infra/buildinfo"
	"caatsm/internal/infra/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

func TestMonitoring(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Monitoring Suite")
}

var _ = Describe("Server", func() {
	Context("when monitoring is disabled", func() {
		It("returns nil from ProvideServer", func() {
			cfg := &config.Config{
				Monitoring: config.MonitoringConfig{
					Disabled: true,
					Addr:     ":0",
				},
			}

			server, err := ProvideServer(cfg, zap.NewNop(), nil, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).To(BeNil())
		})
	})

	Context("health and metrics handlers", func() {
		BeforeEach(func() {
			buildinfo.Version = "v-test"
			buildinfo.Commit = "abc"
			buildinfo.BuiltAt = "now"
		})

		It("returns build info on /livez", func() {
			server := &Server{
				cfg:    config.MonitoringConfig{},
				logger: zap.NewNop(),
			}

			req := httptest.NewRequest(http.MethodGet, "/livez", nil)
			resp := httptest.NewRecorder()

			server.handleLive(resp, req)

			Expect(resp.Code).To(Equal(http.StatusOK))

			payload := map[string]interface{}{}
			Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())
			Expect(payload["status"]).To(Equal("ok"))
		})

		It("reports unconfigured dependencies as unhealthy", func() {
			server := &Server{
				cfg: config.MonitoringConfig{
					EnableHealth:  true,
					HealthTimeout: time.Second,
				},
				logger: zap.NewNop(),
			}

			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			resp := httptest.NewRecorder()

			server.handleHealth(resp, req)

			Expect(resp.Code).To(Equal(http.StatusServiceUnavailable))

			payload := map[string]interface{}{}
			Expect(json.NewDecoder(resp.Body).Decode(&payload)).To(Succeed())

			deps := payload["dependencies"].(map[string]interface{})
			Expect(deps["postgres"].(map[string]interface{})["status"]).To(Equal("unconfigured"))
			Expect(deps["nats"].(map[string]interface{})["status"]).To(Equal("unconfigured"))
		})
	})

	Describe("httpStatusLabel", func() {
		It("classifies status codes", func() {
			Expect(httpStatusLabel(200)).To(Equal("ok"))
			Expect(httpStatusLabel(502)).To(Equal("error"))
		})
	})
})
