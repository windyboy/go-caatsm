package nats

import (
	"caatsm/internal/app"
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// startCore starts the Core NATS consumer loop.
func (c *Consumer) startCore(ctx context.Context) error {
	queueGroup := c.cfg.Subscription.QueueGroup
	if queueGroup == "" {
		queueGroup = c.consumerName
	}

	handler := func(msg *nats.Msg) {
		if err := c.processMessage(ctx, msg); err != nil {
			isPermanent := app.IsPermanent(err)
			c.logger.Error("Failed to process message (core mode)",
				zap.String("subject", msg.Subject),
				zap.Error(err),
				zap.Bool("permanent", isPermanent),
			)
		}
	}

	sub, err := c.conn.QueueSubscribe(c.subject, queueGroup, handler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", c.subject, err)
	}
	if err := c.conn.Flush(); err != nil {
		return fmt.Errorf("failed to flush NATS connection: %w", err)
	}

	c.logger.Info("Started core NATS subscription",
		zap.String("subject", c.subject),
		zap.String("queue_group", queueGroup),
	)

	<-ctx.Done()
	c.logger.Info("Stopping core NATS consumer", zap.Error(ctx.Err()))

	if err := sub.Drain(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
		return fmt.Errorf("failed to drain core subscription: %w", err)
	}

	return ctx.Err()
}
