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
	app, err := di.InitializeApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize worker: %v\n", err)
		os.Exit(1)
	}
	defer app.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := applog.Sugared()
	logger.Info("starting message worker")

	if err := app.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "worker exited with error: %v\n", err)
		os.Exit(1)
	}

	<-ctx.Done()
	logger.Info("worker stopped")
}
