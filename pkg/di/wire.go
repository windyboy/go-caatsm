//go:build wireinject
// +build wireinject

package di

import (
	"caatsm/internal/adapter/parser"
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	"caatsm/internal/infra/log"
	"caatsm/internal/infra/nats"
	"caatsm/internal/infra/postgres"

	"github.com/google/wire"
)

// InitializeApp initializes the application with all dependencies
func InitializeApp() (*app.MessageProcessor, *nats.Consumer, error) {
	comps, err := buildAppComponents()
	if err != nil {
		return nil, nil, err
	}
	return comps.Processor, comps.Consumer, nil
}

// InitializeAppWithConfig wires dependencies using a pre-loaded configuration.
func InitializeAppWithConfig(cfg *config.Config) (*app.MessageProcessor, *nats.Consumer, error) {
	comps, err := buildAppComponentsWithConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return comps.Processor, comps.Consumer, nil
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

	// App
	app.NewMessageProcessor,

	// Consumer
	nats.ProvideConsumer,
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
}
