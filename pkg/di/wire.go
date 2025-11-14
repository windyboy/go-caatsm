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
	wire.Build(
		// Config
		config.ProvideConfig,

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
	return nil, nil, nil
}
