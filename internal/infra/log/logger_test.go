package log

import (
	"os"
	"path/filepath"
	"testing"

	configpkg "caatsm/internal/infra/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Log Suite")
}

var _ = Describe("ProvideLogger", func() {
	Context("when running in development console mode", func() {
		It("returns a functional logger", func() {
			cfg := &configpkg.Config{
				Log: configpkg.LogConfig{
					Level:       "debug",
					Format:      "console",
					Development: true,
					Output:      []string{"stdout"},
				},
			}

			logger, err := ProvideLogger(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(logger).ToNot(BeNil())
		})
	})

	Context("when file output is configured", func() {
		It("creates the directory before writing logs", func() {
			tmpDir := filepath.Join(os.TempDir(), "caatsm-log-test")
			defer func() {
				if err := os.RemoveAll(tmpDir); err != nil {
					// Cleanup errors in tests are not critical
					_ = err
				}
			}()
			logPath := filepath.Join(tmpDir, "child", "app.log")
			cfg := &configpkg.Config{
				Log: configpkg.LogConfig{
					Level:       "info",
					Format:      "json",
					Development: false,
					Output:      []string{"file"},
					File:        logPath,
					MaxSize:     1,
					MaxBackups:  1,
					MaxAge:      1,
					Compress:    false,
				},
			}

			logger, err := ProvideLogger(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(logger).ToNot(BeNil())
			Expect(filepath.Dir(logPath)).To(BeADirectory())
		})
	})
})
