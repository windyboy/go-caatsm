package di

import (
	"caatsm/internal/adapter/parser"
	"caatsm/internal/app"
	"caatsm/internal/infra/config"
	"caatsm/internal/infra/log"
	"caatsm/internal/infra/nats"
	"caatsm/internal/infra/postgres"
	"errors"
)

// InitializeAppWithConfig wires dependencies using the provided config.
func InitializeAppWithConfig(cfg *config.Config) (*app.MessageProcessor, *nats.Consumer, error) {
	if cfg == nil {
		return nil, nil, errors.New("config is required")
	}

	logger, err := log.ProvideLogger(cfg)
	if err != nil {
		return nil, nil, err
	}

	pool, err := postgres.ProvideDB(cfg)
	if err != nil {
		return nil, nil, err
	}

	repo, err := postgres.ProvideRepository(pool, logger)
	if err != nil {
		return nil, nil, err
	}

	conn, err := nats.ProvideNATSConn(cfg, logger)
	{
		if err != nil {
			return nil, nil, err
		}
	}

	js, err := nats.ProvideJetStream(conn, cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	publisher, err := nats.ProvidePublisher(js, cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	messageParser := parser.ProvideParser()
	processor := app.NewMessageProcessor(messageParser, repo, publisher, logger)

	consumer, err := nats.ProvideConsumer(conn, js, processor, cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	return processor, consumer, nil
}
