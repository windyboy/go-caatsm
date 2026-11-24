package config

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var (
		originalWD  string
		originalEnv string
		hadEnv      bool
	)

	BeforeEach(func() {
		originalEnv, hadEnv = os.LookupEnv("GO_ENV")
		Expect(os.Setenv("GO_ENV", "testdefaults")).To(Succeed())

		var err error
		originalWD, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())

		repoRoot := filepath.Clean(filepath.Join(originalWD, "..", ".."))
		Expect(os.Chdir(repoRoot)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.Chdir(originalWD)).To(Succeed())
		if hadEnv {
			Expect(os.Setenv("GO_ENV", originalEnv)).To(Succeed())
		} else {
			Expect(os.Unsetenv("GO_ENV")).To(Succeed())
		}
	})

	It("defaults ack waits and derived streams when not provided", func() {
		cfg, err := LoadConfig()
		Expect(err).NotTo(HaveOccurred())

		want := 30 * time.Second
		Expect(cfg.Timeouts.AckWait).To(Equal(want))
		Expect(cfg.NATS.ConsumerRules.AckWait).To(Equal(want))
		Expect(cfg.EffectivePublishStream()).To(Equal("TELEGRAM_OUT"))
		Expect(cfg.EffectiveDLQStream()).To(Equal("TELEGRAM_DLQ"))
		Expect(cfg.ShouldAutoProvision()).To(BeTrue())
	})

	It("maps snake_case and double-underscore CAATSM_ environment variables correctly", func() {
		DeferCleanup(func() {
			_ = os.Unsetenv("CAATSM_MONITORING_ENABLE_METRICS")
			_ = os.Unsetenv("CAATSM_MONITORING__ENABLE_HEALTH")
			_ = os.Unsetenv("CAATSM_APP_BATCH_SIZE")
			_ = os.Unsetenv("CAATSM_NATS_CONSUMER_RULES_ACK_WAIT")
			_ = os.Unsetenv("CAATSM_AFTN_VALIDATION_ENABLED")
		})

		Expect(os.Setenv("CAATSM_MONITORING_ENABLE_METRICS", "true")).To(Succeed())
		Expect(os.Setenv("CAATSM_MONITORING__ENABLE_HEALTH", "true")).To(Succeed())
		Expect(os.Setenv("CAATSM_APP_BATCH_SIZE", "75")).To(Succeed())
		Expect(os.Setenv("CAATSM_NATS_CONSUMER_RULES_ACK_WAIT", "45s")).To(Succeed())
		Expect(os.Setenv("CAATSM_AFTN_VALIDATION_ENABLED", "true")).To(Succeed())

		cfg, err := LoadConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Monitoring.EnableMetrics).To(BeTrue())
		Expect(cfg.Monitoring.EnableHealth).To(BeTrue())
		Expect(cfg.App.BatchSize).To(Equal(75))
		Expect(cfg.NATS.ConsumerRules.AckWait).To(Equal(45 * time.Second))
		Expect(cfg.AFTN.ValidationEnabled).To(BeTrue())
	})

	It("supports pure-environment loading when config file does not exist", func() {
		Expect(os.Setenv("GO_ENV", "nonexistent_env")).To(Succeed())
		Expect(os.Setenv("CAATSM_NATS_URL", "nats://env-nats:4222")).To(Succeed())
		Expect(os.Setenv("CAATSM_POSTGRES_URL", "postgres://u:p@db:5432/aviation?sslmode=disable")).To(Succeed())
		Expect(os.Setenv("CAATSM_PUBLISHER_TOPIC", "telegram.json")).To(Succeed())
		DeferCleanup(func() {
			_ = os.Unsetenv("CAATSM_NATS_URL")
			_ = os.Unsetenv("CAATSM_POSTGRES_URL")
			_ = os.Unsetenv("CAATSM_PUBLISHER_TOPIC")
		})

		cfg, err := LoadConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.NATS.URL).To(Equal("nats://env-nats:4222"))
		Expect(cfg.Postgres.URL).To(Equal("postgres://u:p@db:5432/aviation?sslmode=disable"))
		Expect(cfg.Publisher.Topic).To(Equal("telegram.json"))
		Expect(cfg.App.BatchSize).To(Equal(50))
	})

	It("disables auto provisioning by default in prod", func() {
		Expect(os.Setenv("GO_ENV", "prod")).To(Succeed())
		cfg := &Config{NATS: NATSConfig{Stream: "TELEGRAM"}}
		Expect(cfg.ShouldAutoProvision()).To(BeFalse())

		override := true
		cfg.NATS.AutoProvision = &override
		Expect(cfg.ShouldAutoProvision()).To(BeTrue())
	})

	It("rejects unsupported nats.mode values", func() {
		cfg := &Config{
			NATS: Config{}.NATS,
			Postgres: PostgresConfig{
				URL: "postgres://u:p@localhost:5432/db",
			},
			Publisher: PublisherConfig{Topic: "out"},
			App: AppConfig{
				BatchSize:       1,
				BatchTimeout:    time.Second,
				MonitorInterval: time.Second,
			},
		}
		cfg.NATS.URL = "nats://localhost:4222"
		cfg.NATS.Mode = "core"
		cfg.NATS.Stream = "TELEGRAM"
		cfg.NATS.StreamLimits.Replicas = 1
		cfg.NATS.ConsumerRules.MaxDeliver = 3
		cfg.NATS.ConsumerRules.AckWait = time.Second

		err := cfg.Validate()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("nats.mode"))
	})

	It("enables health and metrics by default when only CAATSM_MONITORING_ADDR is overridden", func() {
		Expect(os.Setenv("GO_ENV", "nonexistent_env")).To(Succeed())
		Expect(os.Setenv("CAATSM_NATS_URL", "nats://env-nats:4222")).To(Succeed())
		Expect(os.Setenv("CAATSM_POSTGRES_URL", "postgres://u:p@db:5432/aviation?sslmode=disable")).To(Succeed())
		Expect(os.Setenv("CAATSM_PUBLISHER_TOPIC", "telegram.json")).To(Succeed())
		Expect(os.Setenv("CAATSM_MONITORING_ADDR", ":9099")).To(Succeed())
		DeferCleanup(func() {
			_ = os.Unsetenv("CAATSM_NATS_URL")
			_ = os.Unsetenv("CAATSM_POSTGRES_URL")
			_ = os.Unsetenv("CAATSM_PUBLISHER_TOPIC")
			_ = os.Unsetenv("CAATSM_MONITORING_ADDR")
		})

		cfg, err := LoadConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Monitoring.Addr).To(Equal(":9099"))
		Expect(cfg.Monitoring.EnableHealth).To(BeTrue())
		Expect(cfg.Monitoring.EnableMetrics).To(BeTrue())
	})

	It("rejects invalid auth, TLS pairs, streams, postgres pool bounds, DLQ, and log settings", func() {
		validBase := func() *Config {
			return &Config{
				NATS: NATSConfig{
					URL:           "nats://localhost:4222",
					Mode:          "jetstream",
					Stream:        "TELEGRAM",
					StreamLimits:  StreamLimitsConfig{Replicas: 1, Discard: "old", Storage: "file"},
					ConsumerRules: ConsumerRulesConfig{MaxDeliver: 3, AckWait: time.Second},
				},
				Subscription: SubscriptionConfig{Topic: "telegram.serial"},
				Publisher:    PublisherConfig{Topic: "telegram.json"},
				Postgres:     PostgresConfig{URL: "postgres://u:p@localhost:5432/db", MinConns: 1, MaxConns: 5},
				App:          AppConfig{BatchSize: 1, BatchTimeout: time.Second, MonitorInterval: time.Second},
				Log:          LogConfig{Level: "info", Format: "json"},
			}
		}

		// Conflicting auth
		c := validBase()
		c.NATS.Auth.Token = "tok"
		c.NATS.Auth.CredentialsFile = "/tmp/creds"
		Expect(c.Validate()).To(HaveOccurred())

		// Mismatched TLS cert/key
		c = validBase()
		c.NATS.Auth.TLSCertFile = "/tmp/cert.pem"
		Expect(c.Validate()).To(HaveOccurred())

		// Invalid discard / storage
		c = validBase()
		c.NATS.StreamLimits.Discard = "drop"
		Expect(c.Validate()).To(HaveOccurred())

		c = validBase()
		c.NATS.StreamLimits.Storage = "ssd"
		Expect(c.Validate()).To(HaveOccurred())

		// Sequence / Time deliver policies without start args
		c = validBase()
		c.NATS.ConsumerRules.DeliverPolicy = "sequence"
		Expect(c.Validate()).To(HaveOccurred())

		c = validBase()
		c.NATS.ConsumerRules.DeliverPolicy = "time"
		Expect(c.Validate()).To(HaveOccurred())

		// MinConns > MaxConns
		c = validBase()
		c.Postgres.MinConns = 10
		c.Postgres.MaxConns = 2
		Expect(c.Validate()).To(HaveOccurred())

		// DLQ enabled without subject
		c = validBase()
		c.DLQ.Enabled = true
		Expect(c.Validate()).To(HaveOccurred())

		// Overlapping stream or subject
		c = validBase()
		c.NATS.PublishStream = "TELEGRAM"
		Expect(c.Validate()).To(HaveOccurred())

		c = validBase()
		c.Publisher.Topic = "telegram.serial"
		Expect(c.Validate()).To(HaveOccurred())

		// Invalid log level/format
		c = validBase()
		c.Log.Level = "verbose"
		Expect(c.Validate()).To(HaveOccurred())

		c = validBase()
		c.Log.Format = "xml"
		Expect(c.Validate()).To(HaveOccurred())
	})
})
