package config

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LoadConfig", func() {
	var (
		originalWD string
	)

	BeforeEach(func() {
		Expect(os.Setenv("GO_ENV", "testdefaults")).To(Succeed())

		var err error
		originalWD, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())

		repoRoot := filepath.Clean(filepath.Join(originalWD, "..", "..", ".."))
		Expect(os.Chdir(repoRoot)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.Chdir(originalWD)).To(Succeed())
	})

	It("defaults ack waits when not provided", func() {
		cfg, err := LoadConfig()
		Expect(err).NotTo(HaveOccurred())

		want := 30 * time.Second
		Expect(cfg.Timeouts.AckWait).To(Equal(want))
		Expect(cfg.NATS.ConsumerRules.AckWait).To(Equal(want))
	})
})
