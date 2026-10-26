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

	app, err := app.New(cfg)
	if err != nil {
		slog.Error("app create failed", "err", err)
		os.Exit(1)
	}

	if err := app.Start(ctx); err != nil {
		slog.Error("app start failed", "err", err)
		os.Exit(1)
	}

	<-ctx.Done()
	slog.Info("shutting down")
	if err := app.Shutdown(ctx); err != nil {
		slog.Error("shutdown failed", "err", err)
	}
}