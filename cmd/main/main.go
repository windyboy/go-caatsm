package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	applog "caatsm/internal/infra/log"
	"caatsm/pkg/di"
)

func main() {
	logger := applog.Sugared()

	app, err := di.InitializeApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting telegram worker")
	if err := app.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "consumer exited with error: %v\n", err)
		os.Exit(1)
	}

	<-ctx.Done()
	logger.Info("telegram worker stopped")
}
