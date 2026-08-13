package logger

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module registers the *zap.Logger and *zap.SugaredLogger dependencies into FX.
//
// The structured *zap.Logger is used by code that logs with typed fields
// (zap.String, zap.Error, …). The *zap.SugaredLogger is the zap equivalent of
// the old global log/slog key-value style and is used by code migrated from
// slog; both wrap the same underlying core, so levels and sinks stay
// consistent across the application.
var Module = fx.Provide(NewLogger, NewSugaredLogger)

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

// NewSugaredLogger derives the shared sugared view of the structured logger
// so both flavors report through the same core and lifecycle.
func NewSugaredLogger(log *zap.Logger) *zap.SugaredLogger {
	return log.Sugar()
}
