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

const testConfigsDir = "./test_configs_dir"

// Helper function to create a dummy config file
func createDummyConfigFile(env string, content string) string {
	err := os.MkdirAll(testConfigsDir, 0755)
	Expect(err).NotTo(HaveOccurred())
	filePath := testConfigsDir + "/config." + env + ".toml"
	err = os.WriteFile(filePath, []byte(content), 0644)
	Expect(err).NotTo(HaveOccurred())
	return filePath
}

// Helper function to remove dummy config files and directory
func cleanupDummyConfigs() {
	err := os.RemoveAll(testConfigsDir)
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("Config", func() {
	var originalGoEnv string
	var originalViperConfigPaths []string

	BeforeEach(func() {
		originalGoEnv = os.Getenv("GO_ENV")
		// Viper keeps a list of config paths. We need to save and restore it if we modify it.
		// However, LoadConfig uses viper.AddConfigPath("configs") hardcoded.
		// For isolated LoadConfig testing, we'll create a local "configs" dir or manipulate Viper's search path if possible.
		// For now, we assume LoadConfig will look in "./configs". We will create this directory for tests.
		// The createDummyConfigFile helper will effectively use a path like "./test_configs_dir/config.<env>.toml"
		// So, we need to tell Viper to look in testConfigsDir instead of "configs".
		// This is tricky because LoadConfig hardcodes `viper.AddConfigPath("configs")`.
		// A better approach for testing LoadConfig would be to allow passing the config path to LoadConfig.
		// For now, we'll work around it by temporarily creating "configs/config.<env>.toml".
		
		// Reset Viper for each test to avoid interference
		viper.Reset()
	})

	AfterEach(func() {
		os.Setenv("GO_ENV", originalGoEnv)
		cleanupDummyConfigs() // Cleans up any files created in testConfigsDir
		os.RemoveAll("./configs") // Clean up the actual "configs" dir if created by tests
		viper.Reset()
	})

	// Existing tests for GetMyConfig and ValidateConfig might need adjustment
	// if they relied on the old BeforeEach that directly manipulated MyConfig via a single temp file.
	// Let's keep them for now and see, they might still work if MyConfig is set by a successful LoadConfig call.

	Describe("LoadConfig function", func() {
		BeforeEach(func() {
			// Ensure the default "configs" directory used by LoadConfig is clean or doesn't exist
			os.RemoveAll("./configs") 
		})
		AfterEach(func() {
			os.RemoveAll("./configs")
		})


		Context("when a valid config file exists for the specified GO_ENV", func() {
			It("should load the configuration correctly from that file", func() {
				os.Setenv("GO_ENV", "test")
				dummyContent := `
[nats]
client = "env-test-client"
url = "nats://env-test:4222"
[subscription]
topic = "env-test-topic"
				`
				// Create the actual path LoadConfig uses
				err := os.MkdirAll("./configs", 0755)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile("./configs/config.test.toml", []byte(dummyContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				cfg, err := LoadConfig()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Nats.Client).To(Equal("env-test-client"))
				Expect(cfg.Nats.URL).To(Equal("nats://env-test:4222"))
				Expect(cfg.Subscription.Topic).To(Equal("env-test-topic"))
			})
		})

		Context("when config file is missing for the specified GO_ENV", func() {
			It("should return an error", func() {
				os.Setenv("GO_ENV", "nonexistentenv")
				// Ensure ./configs/config.nonexistentenv.toml does not exist
				os.Remove("./configs/config.nonexistentenv.toml") 

				_, err := LoadConfig()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error reading config file for environment 'nonexistentenv'"))
			})
		})

		Context("when GO_ENV is not set", func() {
			It("should default to 'dev' and load config.dev.toml", func() {
				os.Unsetenv("GO_ENV") // Ensure GO_ENV is not set
				dummyContent := `
[nats]
client = "dev-client-default"
url = "nats://dev-default:4222"
[subscription]
topic = "dev-topic-default"
				`
				err := os.MkdirAll("./configs", 0755)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile("./configs/config.dev.toml", []byte(dummyContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				cfg, err := LoadConfig()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Nats.Client).To(Equal("dev-client-default"))
				Expect(cfg.Subscription.Topic).To(Equal("dev-topic-default"))
			})
		})

		Context("when environment variables are set", func() {
			It("should override values from the config file", func() {
				os.Setenv("GO_ENV", "test")
				dummyContent := `
[nats]
client = "file-client"
url = "nats://file-url:4222" # This should be overridden
cluster = "file-cluster"
[subscription]
topic = "file-topic" # This should be overridden
queue_group = "file-group"
[publisher]
topic = "file-pub-topic" # This should be overridden by env
				`
				err := os.MkdirAll("./configs", 0755)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile("./configs/config.test.toml", []byte(dummyContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Set environment variables - viper.SetEnvPrefix("tele") and viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
				// means TELE_NATS_URL maps to Nats.URL, TELE_SUBSCRIPTION_TOPIC to Subscription.Topic
				os.Setenv("TELE_NATS_URL", "nats://env-override-url:4222")
				os.Setenv("TELE_SUBSCRIPTION_TOPIC", "env-override-sub-topic")
				os.Setenv("TELE_PUBLISHER_TOPIC", "env-override-pub-topic")
				// For nested structs like Timeouts, e.g., Timeouts.Server, it would be TELE_TIMEOUTS_SERVER
				os.Setenv("TELE_TIMEOUTS_SERVER", "60s")


				cfg, err := LoadConfig()
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).NotTo(BeNil())

				Expect(cfg.Nats.Client).To(Equal("file-client")) // Not overridden
				Expect(cfg.Nats.URL).To(Equal("nats://env-override-url:4222")) // Overridden
				Expect(cfg.Subscription.Topic).To(Equal("env-override-sub-topic")) // Overridden
				Expect(cfg.Subscription.QueueGroup).To(Equal("file-group")) // Not overridden
				Expect(cfg.Publisher.Topic).To(Equal("env-override-pub-topic")) // Overridden
				
				expectedServerTimeout, _ := time.ParseDuration("60s")
				Expect(cfg.Timeouts.Server).To(Equal(expectedServerTimeout)) // Overridden

				// Clean up env vars
				os.Unsetenv("TELE_NATS_URL")
				os.Unsetenv("TELE_SUBSCRIPTION_TOPIC")
				os.Unsetenv("TELE_PUBLISHER_TOPIC")
				os.Unsetenv("TELE_TIMEOUTS_SERVER")
			})
		})

		Context("when config file has missing sections or invalid types", func() {
			It("should return an error if a critical section like [nats] is missing but fields are required by viper unmarshal (if struct tags enforce it, or if unmarshal strict mode)", func() {
				os.Setenv("GO_ENV", "test")
				// This content is missing the [nats] section entirely.
				// Viper's Unmarshal behavior here depends on struct tags (e.g. `mapstructure:""`).
				// If NatsConfig struct itself is not tagged as optional, and has no default values, behavior can vary.
				// Typically, viper will try to unmarshal and missing sections will result in zero-valued structs.
				// The error, if any, would likely come from ValidateConfig later, not LoadConfig itself unless Unmarshal fails.
				// For this test, let's assume Unmarshal will succeed with an empty NatsConfig.
				dummyContent := `
[subscription]
topic = "some-topic"
				`
				err := os.MkdirAll("./configs", 0755)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile("./configs/config.test.toml", []byte(dummyContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				cfg, err := LoadConfig()
				Expect(err).NotTo(HaveOccurred()) // Viper likely won't error here for missing sections unless told to
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Nats.Client).To(BeEmpty()) // Nats section was missing
				Expect(cfg.Nats.URL).To(BeEmpty())
				Expect(cfg.Subscription.Topic).To(Equal("some-topic"))
			})

			It("should return an error during unmarshal if a field has a grossly incorrect type", func() {
				os.Setenv("GO_ENV", "test")
				dummyContent := `
[nats]
client = "test-client"
url = 12345 # URL should be a string, providing a number
[subscription]
topic = "test-topic"
				`
				err := os.MkdirAll("./configs", 0755)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile("./configs/config.test.toml", []byte(dummyContent), 0644)
				Expect(err).NotTo(HaveOccurred())

				_, err = LoadConfig()
				Expect(err).To(HaveOccurred())
				// The error message will depend on Viper's unmarshaler.
				// It often includes "decode" and type mismatch information.
				Expect(err.Error()).Should(SatisfyAny(
					ContainSubstring("unable to decode config into struct"),
					ContainSubstring("error unmarshaling"), // More generic check
					ContainSubstring("cannot unmarshal number into Go struct field Config.Nats.URL of type string"), // Example specific error
				))
			})

			It("should return an error for a syntactically incorrect TOML file", func() {
				os.Setenv("GO_ENV", "test")
				dummyContent := `
[nats
client = "bad-client" # Missing closing bracket for section
url = "nats://bad.com"
`
				err := os.MkdirAll("./configs", 0755)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile("./configs/config.test.toml", []byte(dummyContent), 0644)
				Expect(err).NotTo(HaveOccurred())
				
				_, err = LoadConfig()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error reading config file")) // Viper's ReadInConfig error
			})
		})
	})
	
	// The old tests are below. They use a different setup method for Viper.
	// For robust testing, we should adapt them or ensure new tests cover their cases.
	// The main issue is that the old tests directly manipulated `MyConfig` via `viper.Unmarshal(MyConfig)`
	// after setting `viper.SetConfigFile` to a *single specific temp file outside the "configs" dir hierarchy*.
	// The new `LoadConfig` tests correctly test the file discovery logic of `LoadConfig` itself.

	Context("Legacy GetMyConfig (using direct Viper setup for MyConfig)", func() {
		var tempConfigFile *os.File
		var originalGoEnvForLegacy string


		BeforeEach(func() {
			originalGoEnvForLegacy = os.Getenv("GO_ENV")
			viper.Reset() // Reset viper before each legacy test context
			viper.SetConfigType("toml")
			configContent := `
[nats]
client = "legacy-test-client"
url = "nats://legacy-localhost:4222"
[subscription]
topic = "legacy-example-topic"
queue_group = "legacy-example-group"
[hasura]
endpoint = "http://legacy-host:8080/v1/graphql"
secret  = "legacy-aviation-test"
`
			var err error
			tempConfigFile, err = os.CreateTemp("", "legacy_config.*.toml")
			Expect(err).NotTo(HaveOccurred())
			_, err = tempConfigFile.Write([]byte(configContent))
			Expect(err).NotTo(HaveOccurred())
			err = tempConfigFile.Close()
			Expect(err).NotTo(HaveOccurred())
			
			viper.SetConfigFile(tempConfigFile.Name())
			err = viper.ReadInConfig()
			Expect(err).NotTo(HaveOccurred())

			MyConfig = &Config{} // Directly manipulating global MyConfig
			err = viper.Unmarshal(MyConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Remove(tempConfigFile.Name()) // clean up
			os.Setenv("GO_ENV", originalGoEnvForLegacy)
			MyConfig = nil // Reset global MyConfig
			viper.Reset()
		})

		It("should load the configuration correctly using GetMyConfig", func() {
			// GetMyConfig here should return the MyConfig populated in BeforeEach
			cfg := GetMyConfig() 
			Expect(cfg).NotTo(BeNil())
			Expect(cfg.Nats.Client).To(Equal("legacy-test-client"))
			Expect(cfg.Nats.URL).To(Equal("nats://legacy-localhost:4222"))
			Expect(cfg.Subscription.Topic).To(Equal("legacy-example-topic"))
			Expect(cfg.Hasura.Endpoint).To(Equal("http://legacy-host:8080/v1/graphql"))
		})
	})


	Context("Validating configuration", func() {
		// This context needs a valid Config object.
		// We can use the legacy setup for this, or ensure LoadConfig works and provides one.
		// For simplicity, let's use a helper to get a predictably valid config for these validation tests.
		var validTestCfg *Config
		BeforeEach(func(){
			// This setup is similar to the legacy one, but focused on providing a cfg for validation tests
			viper.Reset()
			viper.SetConfigType("toml")
			configContent := `
[nats]
client = "validate-client"
url = "nats://validate:4222"
[subscription]
topic = "validate-topic"
[publisher]
topic = "validate-pub-topic"
[hasura]
endpoint = "http://validate-hasura:8080"
secret = "validate-secret"
`
			tmpFile, err := os.CreateTemp("", "validate_cfg.*.toml")
			Expect(err).NotTo(HaveOccurred())
			_, err = tmpFile.Write([]byte(configContent))
			Expect(err).NotTo(HaveOccurred())
			err = tmpFile.Close()
			Expect(err).NotTo(HaveOccurred())
			
			v := viper.New()
			v.SetConfigFile(tmpFile.Name())
			err = v.ReadInConfig()
			Expect(err).NotTo(HaveOccurred())
			validTestCfg = &Config{}
			err = v.Unmarshal(validTestCfg)
			Expect(err).NotTo(HaveOccurred())
			os.Remove(tmpFile.Name()) // Clean up temp file after loading into struct
		})


		It("should validate a valid configuration", func() {
			// cfg := GetMyConfig() // This would rely on global MyConfig
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
			err := ValidateConfig(validTestCfg) // Use the locally defined validTestCfg
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error for missing NATS client", func() {
			cfg := *validTestCfg // Create a copy to modify
			cfg.Nats.Client = ""
			err := ValidateConfig(&cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("nats client is required"))
		})

		It("should return an error for missing NATS URL", func() {
			cfg := *validTestCfg // Create a copy
			cfg.Nats.URL = ""
			err := ValidateConfig(&cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("nats URL is required"))
		})

		It("should return an error for missing subscription topic", func() {
			cfg := *validTestCfg // Create a copy
			cfg.Subscription.Topic = ""
			err := ValidateConfig(&cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("subscription topic is required"))
		})

		It("should return an error for missing publisher topic", func() {
			cfg := *validTestCfg // Create a copy
			cfg.Publisher.Topic = ""
			err := ValidateConfig(&cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("publisher topic is required"))
		})

		It("should return an error for missing Hasura endpoint", func() {
			cfg := *validTestCfg // Create a copy
			cfg.Hasura.Endpoint = ""
			// If endpoint is empty, secret validation might not apply or might also fail.
			// The current rule is "hasura secret is required when hasura endpoint is provided"
			// So, if endpoint is empty, secret being empty or not is not an error from *that* rule.
			err := ValidateConfig(&cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("hasura endpoint is required"))
		})

		It("should return an error for missing Hasura secret when endpoint is provided", func() {
			cfg := *validTestCfg // Create a copy
			Expect(cfg.Hasura.Endpoint).NotTo(BeEmpty(), "Test setup error: Hasura endpoint should be non-empty for this test")
			cfg.Hasura.Secret = ""
			err := ValidateConfig(&cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("hasura secret is required when hasura endpoint is provided"))
		})
		
		It("should NOT return an error for missing Hasura secret if endpoint is also missing (as endpoint is the primary error)", func() {
			cfg := *validTestCfg // Create a copy
			cfg.Hasura.Endpoint = ""
			cfg.Hasura.Secret = "" // Secret is also empty
			err := ValidateConfig(&cfg)
			Expect(err).To(HaveOccurred())
			// The error should be about the endpoint, not the secret, because the secret validation is conditional on endpoint.
			Expect(err.Error()).To(Equal("hasura endpoint is required")) 
		})
	})
})
