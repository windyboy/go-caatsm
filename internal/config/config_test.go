package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config", func() {
	var (
		originalEnv string
		configFile  string
		envName     string
	)

	configContent := `
[nats]
client = "test-client"
url = "nats://localhost:4222"
cluster = "test-cluster"

[nats.jetstream]
enabled = true
stream_name = "TEST_STREAM"
subject = "Test.Subject"
auto_provision = true

[subscription]
topic = "example-topic"
queue_group = "example-group"

[publisher]
topic = "example-output"

[database]
host = "localhost"
port = 5432
user = "postgres"
password = "postgres"
database = "aviation"
ssl_mode = "disable"
max_conns = 10
min_conns = 1
max_conn_lifetime = "1h"

[api]
host = "0.0.0.0"
port = 8080
mode = "dev"

[timeouts]
server = "30s"
reconnect_wait = "10s"
close = "10s"
ack_wait = "5s"
`
	BeforeEach(func() {
		originalEnv = os.Getenv("GO_ENV")
		envName = fmt.Sprintf("test_%d", time.Now().UnixNano())
		os.Setenv("GO_ENV", envName)
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("config.%s.*.toml", envName))
		Expect(err).NotTo(HaveOccurred())
		configFile = tmpFile.Name()
		err = os.WriteFile(configFile, []byte(configContent), 0o600)
		Expect(err).NotTo(HaveOccurred())
		os.Setenv("CAATSM_CONFIG_FILE", configFile)
		MyConfig = nil
	})

	AfterEach(func() {
		os.Remove(configFile)
		os.Unsetenv("CAATSM_CONFIG_FILE")
		os.Setenv("GO_ENV", originalEnv)
	})

	Context("Loading configuration", func() {
		It("should load the configuration correctly", func() {
			cfg := GetMyConfig()
			Expect(cfg).NotTo(BeNil())
			Expect(cfg.Nats.Client).To(Equal("test-client"))
			Expect(cfg.Nats.URL).To(Equal("nats://localhost:4222"))
			Expect(cfg.Subscription.Topic).To(Equal("example-topic"))
			Expect(cfg.Subscription.QueueGroup).To(Equal("example-group"))
			Expect(cfg.Publisher.Topic).To(Equal("example-output"))
			Expect(cfg.Database.Host).To(Equal("localhost"))
			Expect(cfg.API.Port).To(Equal(8080))
		})
	})

	Context("Validating configuration", func() {
		It("should validate a valid configuration", func() {
			cfg := GetMyConfig()
			err := ValidateConfig(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error for missing NATS client", func() {
			cfg := GetMyConfig()
			cfg.Nats.Client = ""
			err := ValidateConfig(cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("nats client is required"))
		})

		It("should return an error for missing NATS URL", func() {
			cfg := GetMyConfig()
			cfg.Nats.URL = ""
			err := ValidateConfig(cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("nats URL is required"))
		})

		It("should return an error for missing subscription topic", func() {
			cfg := GetMyConfig()
			cfg.Subscription.Topic = ""
			err := ValidateConfig(cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("subscription topic is required"))
		})
	})

	Context("Environment overrides", func() {
		AfterEach(func() {
			os.Unsetenv("TELE_NATS_URL")
			os.Unsetenv("TELE_TIMEOUTS_ACK_WAIT")
		})

		It("should override simple keys via environment variables", func() {
			os.Setenv("TELE_NATS_URL", "nats://override:4222")
			cfg, err := LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Nats.URL).To(Equal("nats://override:4222"))
		})

		It("should override keys that contain underscores", func() {
			os.Setenv("TELE_TIMEOUTS_ACK_WAIT", "45s")
			cfg, err := LoadConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Timeouts.AckWait).To(Equal(45 * time.Second))
		})
	})
})
