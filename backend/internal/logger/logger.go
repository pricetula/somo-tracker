package logger

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module registers the *zap.Logger dependency into FX
var Module = fx.Provide(NewLogger)

func NewLogger(lc fx.Lifecycle) (*zap.Logger, error) {
	// Use zap.NewProduction() or zap.NewDevelopment()
	log, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	// Flush buffered logs on application shutdown
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			_ = log.Sync()
			return nil
		},
	})

	return log, nil
}
