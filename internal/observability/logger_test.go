package observability

import (
	"caatsm/internal/config"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logger", func() {
	It("creates a functional logger in console dev mode", func() {
		cfg := &config.Config{
			Log: config.LogConfig{
				Level:       "debug",
				Format:      "console",
				Development: true,
				Output:      []string{"stdout"},
			},
		}
		logger, err := NewLogger(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(logger).ToNot(BeNil())
	})

	It("creates the log directory when file output is configured", func() {
		tmpDir := filepath.Join(os.TempDir(), "caatsm-obs-log-test")
		DeferCleanup(func() { _ = os.RemoveAll(tmpDir) })

		logPath := filepath.Join(tmpDir, "child", "app.log")
		cfg := &config.Config{
			Log: config.LogConfig{
				Level:      "info",
				Format:     "json",
				Output:     []string{"file"},
				File:       logPath,
				MaxSize:    1,
				MaxBackups: 1,
				MaxAge:     1,
			},
		}
		logger, err := NewLogger(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(logger).ToNot(BeNil())
		Expect(filepath.Dir(logPath)).To(BeADirectory())
	})

	It("attaches structured MessageFields via WithMessageContext", func() {
		base, err := NewLogger(&config.Config{Log: config.LogConfig{Level: "info", Format: "json", Output: []string{"stdout"}}})
		Expect(err).NotTo(HaveOccurred())
		enriched := WithMessageContext(base, MessageFields{
			Service:        "caatsm",
			TransportMsgID: "id-1",
			BusinessMsgID:  "TMQ1",
			Category:       "ARR",
			JSSequence:     42,
			ErrorType:      ErrorTypeTransient,
		})
		Expect(enriched).NotTo(BeNil())
		Expect(WithMessageContext(nil, MessageFields{})).NotTo(BeNil())
	})
})
