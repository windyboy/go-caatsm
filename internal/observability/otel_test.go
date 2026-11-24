package observability

import (
	"caatsm/internal/config"
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("OTel helpers", func() {
	It("normalizes endpoints by stripping URL schemes and trailing slashes", func() {
		Expect(normalizeEndpoint(" http://localhost:4318/ ")).To(Equal("localhost:4318"))
		Expect(normalizeEndpoint("https://otel.prod:4318")).To(Equal("otel.prod:4318"))
	})

	It("returns a no-op shutdown when telemetry is disabled", func() {
		shut, err := InitOTEL(context.Background(), &config.Config{Telemetry: config.TelemetryConfig{Enabled: false}}, zap.NewNop())
		Expect(err).NotTo(HaveOccurred())
		Expect(shut(context.Background())).To(Succeed())
	})

	It("errors when telemetry is enabled without an endpoint", func() {
		_, err := InitOTEL(context.Background(), &config.Config{Telemetry: config.TelemetryConfig{Enabled: true}}, zap.NewNop())
		Expect(err).To(HaveOccurred())
	})

	It("selects environment-appropriate samplers", func() {
		Expect(getSamplerForEnvironment("prod").Description()).To(ContainSubstring("TraceIDRatioBased"))
		Expect(getSamplerForEnvironment("staging").Description()).To(ContainSubstring("TraceIDRatioBased"))
		Expect(getSamplerForEnvironment("dev").Description()).To(Equal("AlwaysOnSampler"))
	})
})
