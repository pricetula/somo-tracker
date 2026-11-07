package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"somotracker/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	worker, err := app.NewWorker(cfg)
	if err != nil {
		slog.Error("worker create failed", "err", err)
		os.Exit(1)
	}

	if err := worker.Start(ctx); err != nil {
		slog.Error("worker start failed", "err", err)
		os.Exit(1)
	}

	<-ctx.Done()
	slog.Info("worker shutting down")
	if err := worker.Shutdown(ctx); err != nil {
		slog.Error("worker shutdown failed", "err", err)
	}
}