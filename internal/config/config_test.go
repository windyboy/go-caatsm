package config

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config", func() {
	var originalEnv string

	BeforeEach(func() {
		// Save the original GO_ENV value
		originalEnv = os.Getenv("GO_ENV")
		// Set up a temporary configuration file for testing
		viper.Reset()
		viper.SetConfigType("toml")
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
		tmpFile, err := os.CreateTemp("", "config.*.toml")
		Expect(err).NotTo(HaveOccurred())
		_, err = tmpFile.Write([]byte(configContent))
		Expect(err).NotTo(HaveOccurred())
		err = tmpFile.Close()
		Expect(err).NotTo(HaveOccurred())

		viper.SetConfigFile(tmpFile.Name())
		err = viper.ReadInConfig()
		Expect(err).NotTo(HaveOccurred())

		// Load the configuration
		MyConfig = &Config{}
		err = viper.Unmarshal(MyConfig)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Restore the original GO_ENV value
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
})
