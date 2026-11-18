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
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	a, err := app.New(cfg)
	if err != nil {
		slog.Error("failed to create app", "err", err)
		os.Exit(1)
	}

	if err := a.Start(ctx); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}
