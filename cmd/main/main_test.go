package main

import (
	"flag"
	"time"

	appconfig "caatsm/internal/infra/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v2"
)

var _ = Describe("CLI overrides", func() {
	var (
		cfg     *appconfig.Config
		flagSet *flag.FlagSet
		args    []string
	)

	BeforeEach(func() {
		cfg = &appconfig.Config{
			NATS: appconfig.NATSConfig{
				URL:      "nats://base:4222",
				Stream:   "BASE_STREAM",
				Consumer: "base-consumer",
			},
			Subscription: appconfig.SubscriptionConfig{
				Topic: "base.subject",
			},
			Publisher: appconfig.PublisherConfig{
				Topic: "base.publisher",
			},
			Postgres: appconfig.PostgresConfig{
				URL: "postgres://base",
			},
			Log: appconfig.LogConfig{
				Level: "info",
			},
			Telemetry: appconfig.TelemetryConfig{
				Enabled:  false,
				Endpoint: "",
				Insecure: false,
			},
		}

		flagSet = flag.NewFlagSet("test", flag.ContinueOnError)
		flagSet.String("nats-url", "", "")
		flagSet.String("subject", "", "")
		flagSet.String("stream", "", "")
		flagSet.String("consumer", "", "")
		flagSet.String("publisher-topic", "", "")
		flagSet.String("postgres-url", "", "")
		flagSet.String("log-level", "", "")
		flagSet.String("replay-from", "", "")
		flagSet.Duration("ack-wait", 0, "")
		flagSet.Bool("telemetry-enabled", false, "")
		flagSet.String("telemetry-endpoint", "", "")
		flagSet.Bool("telemetry-insecure", false, "")

		args = []string{
			"--nats-url", "nats://override:4222",
			"--subject", "override.subject",
			"--stream", "STREAM",
			"--consumer", "CONSUMER",
			"--publisher-topic", "pub.topic",
			"--postgres-url", "postgres://override",
			"--log-level", "debug",
			"--replay-from", "seq:42",
			"--ack-wait", "45s",
			"--telemetry-enabled",
			"--telemetry-endpoint", "http://otel:4318",
			"--telemetry-insecure",
		}
	})

	JustBeforeEach(func() {
		err := flagSet.Parse(args)
		Expect(err).NotTo(HaveOccurred())

		ctx := cli.NewContext(cli.NewApp(), flagSet, nil)
		applyCLIOverrides(cfg, ctx)
	})

	It("applies overrides for all critical flags", func() {
		Expect(cfg.NATS.URL).To(Equal("nats://override:4222"))
		Expect(cfg.Subscription.Topic).To(Equal("override.subject"))
		Expect(cfg.NATS.Stream).To(Equal("STREAM"))
		Expect(cfg.NATS.Consumer).To(Equal("CONSUMER"))
		Expect(cfg.Publisher.Topic).To(Equal("pub.topic"))
		Expect(cfg.Postgres.URL).To(Equal("postgres://override"))
		Expect(cfg.Log.Level).To(Equal("debug"))
		Expect(cfg.NATS.ConsumerRules.DeliverPolicy).To(Equal("sequence"))
		Expect(cfg.NATS.ConsumerRules.StartSequence).To(Equal(uint64(42)))
		Expect(cfg.NATS.ConsumerRules.AckWait).To(Equal(45 * time.Second))
		Expect(cfg.Telemetry.Enabled).To(BeTrue())
		Expect(cfg.Telemetry.Endpoint).To(Equal("http://otel:4318"))
		Expect(cfg.Telemetry.Insecure).To(BeTrue())
	})

	Context("with time-based replay override", func() {
		BeforeEach(func() {
			args = []string{
				"--replay-from", "time:2024-11-15T08:00:00Z",
			}
		})

		It("configures deliver policy and start time", func() {
			Expect(cfg.NATS.ConsumerRules.DeliverPolicy).To(Equal("time"))
			Expect(cfg.NATS.ConsumerRules.StartTime).To(Equal("2024-11-15T08:00:00Z"))
			Expect(cfg.NATS.ConsumerRules.StartSequence).To(Equal(uint64(0)))
		})
	})
})
