//go:build wireinject

package di

import (
	"caatsm/internal/adapter/parser"
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	"caatsm/internal/infra/log"
	"caatsm/internal/infra/monitoring"
	"caatsm/internal/infra/nats"
	"caatsm/internal/infra/postgres"
	"caatsm/internal/infra/telemetry"

	"github.com/google/wire"
)

// InitializeApp initializes the application with all dependencies
func InitializeApp() (*app.MessageProcessor, *nats.Consumer, *monitoring.Server, error) {
	comps, err := buildAppComponents()
	if err != nil {
		return nil, nil, nil, err
	}
	return comps.Processor, comps.Consumer, comps.Monitoring, nil
}

// InitializeAppWithConfig wires dependencies using a pre-loaded configuration.
func InitializeAppWithConfig(cfg *config.Config) (*app.MessageProcessor, *nats.Consumer, *monitoring.Server, error) {
	comps, err := buildAppComponentsWithConfig(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	return comps.Processor, comps.Consumer, comps.Monitoring, nil
}

var runtimeSet = wire.NewSet(
	// Logger
	log.ProvideLogger,

	// Database
	postgres.ProvideDB,
	postgres.ProvideRepository,

	// NATS
	nats.ProvideNATSConn,
	nats.ProvideJetStream,
	nats.ProvidePublisher,

	// Parser
	parser.ProvideParser,

	// Telemetry
	telemetry.ProvideRecorder,

	// App
	app.NewMessageProcessor,

	// Consumer
	nats.ProvideConsumer,

	// Monitoring HTTP server
	monitoring.ProvideServer,
)

func buildAppComponents() (*appComponents, error) {
	wire.Build(
		config.ProvideConfig,
		runtimeSet,
		wire.Struct(new(appComponents), "*"),
	)
	return nil, nil
}

func buildAppComponentsWithConfig(cfg *config.Config) (*appComponents, error) {
	wire.Build(
		runtimeSet,
		wire.Struct(new(appComponents), "*"),
	)
	return nil, nil
}

type appComponents struct {
	Processor *app.MessageProcessor
	Consumer  *nats.Consumer
	Monitoring *monitoring.Server
}
