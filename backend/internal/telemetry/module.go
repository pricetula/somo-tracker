// Package telemetry provides pluggable error telemetry sinks.
//
// Sinks can be registered to process errors asynchronously. Common sinks include:
//   - ZapSink: Sends errors to the structured logger
//   - SentrySink: Sends errors to Sentry (not included, implement your own)
//
// Usage:
//
//	import "somotracker/backend/internal/telemetry"
//
//	// In main.go or module initialization:
//	telemetry.RegisterSink(telemetry.NewZapSink(logger))
//
//	// Or for custom sinks:
//	telemetry.RegisterSink(myCustomSink)
//
// The registry sends errors to all registered sinks when HTTPError is called.
package telemetry

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/xerrors"
)

// Module is the fx module for telemetry.
// It provides the telemetry registry as a dependency.
//
// Usage in fx.New():
//
//	fx.Invoke(telemetry.Initialize)
var Module = fx.Module("telemetry",
	fx.Provide(NewTelemetryRegistry),
	fx.Invoke(registerDefaultSinks),
)

// TelemetryRegistry wraps the global registry for dependency injection.
type TelemetryRegistry struct {
	registry *RegistryType
}

// NewTelemetryRegistry creates a new telemetry registry for DI.
func NewTelemetryRegistry() *TelemetryRegistry {
	return &TelemetryRegistry{
		registry: Registry,
	}
}

// Register adds a sink to the registry.
func (t *TelemetryRegistry) Register(sink xerrors.TelemetrySink) {
	Registry.Register(sink)
}

// FlushAll flushes all sinks.
func (t *TelemetryRegistry) FlushAll(ctx context.Context) error {
	return Registry.FlushAll(ctx)
}

// registerDefaultSinks registers the default ZapSink.
// This is invoked via fx so that the logger is available.
func registerDefaultSinks(lc fx.Lifecycle, logger *zap.SugaredLogger) {
	// Register the Zap sink as the default
	zapSink := NewZapSink(logger)
	// Don't log validation errors at Error level - they're expected
	zapSink.SetFilter(func(de *xerrors.DomainError) bool {
		// Skip logging for 4xx errors at Error level
		return de.Status >= 400 && de.Status < 500
	})
	Registry.Register(zapSink)

	// Hook into shutdown to flush all sinks
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// Flush all telemetry sinks
			if err := Registry.FlushAll(ctx); err != nil {
				logger.Errorw("telemetry flush error", "error", err)
			}
			return nil
		},
	})
}

// RegisterSink is a convenience function for registering sinks without DI.
// Use this in module initialization or main.go.
func RegisterSink(sink xerrors.TelemetrySink) {
	Registry.Register(sink)
}

// RegisterSinkWithFilter registers a sink with a filter function.
func RegisterSinkWithFilter(sink xerrors.TelemetrySink, shouldSkip func(*xerrors.DomainError) bool) {
	if shouldSkip != nil {
		if zs, ok := sink.(*ZapSink); ok {
			zs.SetFilter(shouldSkip)
		}
	}
	Registry.Register(sink)
}
