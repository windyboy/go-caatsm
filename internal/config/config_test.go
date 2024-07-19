package config

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	})

	AfterEach(func() {
		// Restore the original GO_ENV value
		os.Setenv("GO_ENV", originalEnv)
	})

	// Helper function to create temporary TOML files in the "configs" directory
	createTempConfigFile := func(env, content string) string {
		dir := "configs"
		err := os.MkdirAll(dir, 0755)
		Expect(err).NotTo(HaveOccurred(), "failed to create config directory")

		filename := filepath.Join(dir, "config."+env+".toml")
		tmpfile, err := os.Create(filename)
		Expect(err).NotTo(HaveOccurred(), "failed to create temp config file")
		_, err = tmpfile.Write([]byte(content))
		Expect(err).NotTo(HaveOccurred(), "failed to write to temp config file")
		err = tmpfile.Close()
		Expect(err).NotTo(HaveOccurred(), "failed to close temp config file")
		return filename
	}

	Context("Loading and validating a valid TOML file", func() {
		It("should load and validate the configuration correctly", func() {
			content := `
[nats]
client = "test-client"
url = "nats://localhost:4222"
cluster = "test-cluster"

[subscription]
topic = "example-topic"
queue_group = "example-group"

[timeouts]
server_timeout = "30s"
reconnect_wait = "10s"
close_timeout = "10s"
ack_wait_timeout = "5s"

[[body]]
name = "FPL"
[[body.patterns]]
pattern = "^\\((?P<type>[A-Z]{3})\\-(?P<number>[A-Z]+\\d+)\\-(?P<indicator>[A-Z]{2})(?:.*\\s*)?\\-(?P<aircraft>[A-Z]+\\d+/?[A-Z]?)\\s*\\-(?P<surve>.*)\\s*\\-(?P<departure>[A-Z]{4})(?P<departure_time>\\d{4})\\s*\\-(?P<speed>[A-Z]+\\d+)(?P<level>[A-Z0-9]+)\\s(?P<route>.*)\\s*\\-(?P<destination>[A-Z]{4})(?P<estt>\\d{4})\\s(?P<alter>[A-Z]{4})\\s*\\-(?P<pbn>PBN\\/[A-Z0-9]+)\\s(?P<nav>NAV\\/\\w+)\\sREG\\/(?P<reg>[A-Z0-9]+)\\sEET\\/(?P<eet>\\w{4}\\d{4})\\sSEL\\/(?P<sel>\\w+)\\sPER\\/(?P<performance>\\w)\\sRIF\\/(?P<rif>\\w+\\s[A-Z0-9]+\\s[A-Z]+)\\s*RMK\\/(?P<remark>.*)\\)$"
comments = "FPL multi line expression"
`
			os.Setenv("GO_ENV", "test")
			createTempConfigFile("test", content)
			defer os.RemoveAll("configs")

			config, err := LoadConfig()
			Expect(err).NotTo(HaveOccurred(), "failed to load valid config")
			Expect(config).NotTo(BeNil(), "config should not be nil")
			Expect(config.Nats.Client).To(Equal("test-client"), "nats.client should be 'test-client'")
			Expect(config.Subscription.Topic).To(Equal("example-topic"), "subscription.topic should be 'example-topic'")

			// Validate the config
			err = ValidateConfig(config)
			Expect(err).NotTo(HaveOccurred(), "validation should pass for valid config")
		})
	})

	Context("Loading and validating a non-existent file", func() {
		It("should return an error", func() {
			os.Setenv("GO_ENV", "nonexistent")
			defer os.RemoveAll("configs")

			_, err := LoadConfig()
			Expect(err).To(HaveOccurred(), "expected error for non-existent config file")
		})
	})

	Context("Loading and validating a file with invalid TOML format", func() {
		It("should return an error", func() {
			content := `
    invalid TOML content
    `
			os.Setenv("GO_ENV", "invalid")
			createTempConfigFile("invalid", content)
			defer os.RemoveAll("configs")

			_, err := LoadConfig()
			Expect(err).To(HaveOccurred(), "expected error for invalid TOML format")
		})
	})

	Context("Validating an invalid config structure", func() {
		It("should return an error for missing required fields", func() {
			invalidConfig := &Config{
				Nats: NatsConfig{
					Client: "",
					URL:    "",
				},
				Subscription: SubscriptionConfig{
					Topic: "",
				},
				Body: []BodyConfig{},
			}

			err := ValidateConfig(invalidConfig)
			Expect(err).To(HaveOccurred(), "expected validation error for invalid config")
			Expect(err.Error()).To(ContainSubstring("nats client is required"), "expected error for missing nats client")
		})
	})
})
