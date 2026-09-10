// Package observability owns the OpenTelemetry TracerProvider / MeterProvider
// lifecycle for the Somotracker backend.
//
// The package exposes Fx-friendly constructors that:
//
//  1. Build a [sdktrace.TracerProvider] and [sdkmetric.MeterProvider] backed
//     by stdout exporters in non-production environments and a no-op in
//     tests. In production, swap the exporter for OTLP (HTTP or gRPC).
//  2. Register the providers as the global OTel providers via
//     [otel.SetTracerProvider] / [otel.SetMeterProvider] so libraries that
//     read the global provider (e.g. otelpgx) pick them up automatically.
//  3. Register an fx.OnStop hook that calls [sdktrace.TracerProvider.Shutdown]
//     and [sdkmetric.MeterProvider.Shutdown] with a bounded timeout so any
//     pending spans / metrics are flushed before the process exits.
package observability

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServiceName is the OTel resource.service.name reported to backends.
// Centralized here so all telemetry from the same process is grouped.
const ServiceName = "somotracker-api"

// ShutdownTimeout caps how long we wait for span / metric exporters to flush
// pending data during fx.OnStop. Keep it short enough that shutdown feels
// responsive; long enough for one round-trip to the collector.
const ShutdownTimeout = 5 * time.Second

// Config controls how [NewTracerProvider] / [NewMeterProvider] build their
// providers. Production deployments should set Exporter to "otlp" and supply
// the relevant endpoint environment variables.
type Config struct {
	// Exporter is one of: "stdout" (default, dev), "none" (disabled),
	// or "otlp" (production, requires OTEL_EXPORTER_OTLP_ENDPOINT).
	Exporter string

	// ServiceVersion populates resource.service.version. Default "dev".
	ServiceVersion string

	// Environment populates resource.deployment.environment.name.
	// Default "local".
	Environment string
}

// NewConfig builds a Config from environment variables with safe defaults
// for local development. Exposed for Fx providers that need to materialize
// a Config before constructing providers.
func NewConfig() Config {
	return Config{
		Exporter:       getEnv("OTEL_TRACES_EXPORTER", "stdout"),
		ServiceVersion: getEnv("SERVICE_VERSION", "dev"),
		Environment:    getEnv("APP_ENV", "local"),
	}
}

// NewTracerProvider builds and registers an [sdktrace.TracerProvider] using
// the supplied config. It returns the provider so callers can resolve a
// tracer directly; the provider is also set as the global OTel tracer
// provider, so libraries that call [otel.Tracer] (such as otelpgx) inherit
// it automatically.
//
// The function also wires an fx.OnStop hook to flush pending spans on
// shutdown. Failure to flush is logged but does not abort shutdown.
func NewTracerProvider(lc fx.Lifecycle, logger *zap.Logger) (*sdktrace.TracerProvider, error) {
	cfg := NewConfig()

	res, err := buildResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("observability.NewTracerProvider: build resource: %w", err)
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		// Always-sample in dev. Production should swap this for a
		// parent-based or ratio sampler.
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}

	if cfg.Exporter == "stdout" {
		exp, expErr := stdouttrace.New(
			stdouttrace.WithWriter(os.Stderr),
			stdouttrace.WithoutTimestamps(),
		)
		if expErr != nil {
			return nil, fmt.Errorf("observability.NewTracerProvider: build stdout exporter: %w", expErr)
		}
		tpOpts = append(tpOpts, sdktrace.WithBatcher(exp))
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)

	// Make the provider visible to libraries that read the global
	// (notably otelpgx.NewTracer, which defaults to otel.GetTracerProvider).
	otel.SetTracerProvider(tp)

	lc.Append(fx.Hook{
		OnStop: func(stopCtx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(stopCtx, ShutdownTimeout)
			defer cancel()
			if shutdownErr := tp.Shutdown(shutdownCtx); shutdownErr != nil {
				logger.Warn("tracer provider shutdown error",
					zap.Error(shutdownErr),
				)
				return shutdownErr
			}
			logger.Info("tracer provider shut down cleanly")
			return nil
		},
	})

	return tp, nil
}

// NewMeterProvider builds and registers an [sdkmetric.MeterProvider] and
// attaches an fx.OnStop hook to flush pending metric data. Like the tracer
// provider, it is also installed as the global OTel MeterProvider so
// otelpgx.RecordStats picks it up.
func NewMeterProvider(lc fx.Lifecycle, logger *zap.Logger) (*sdkmetric.MeterProvider, error) {
	cfg := NewConfig()

	res, err := buildResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("observability.NewMeterProvider: build resource: %w", err)
	}

	mpOpts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}

	if cfg.Exporter == "stdout" {
		exp, expErr := stdoutmetric.New()
		if expErr != nil {
			return nil, fmt.Errorf("observability.NewMeterProvider: build stdout metric exporter: %w", expErr)
		}
		mpOpts = append(mpOpts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)))
	}

	mp := sdkmetric.NewMeterProvider(mpOpts...)
	otel.SetMeterProvider(mp)

	lc.Append(fx.Hook{
		OnStop: func(stopCtx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(stopCtx, ShutdownTimeout)
			defer cancel()
			if shutdownErr := mp.Shutdown(shutdownCtx); shutdownErr != nil {
				logger.Warn("meter provider shutdown error",
					zap.Error(shutdownErr),
				)
				return shutdownErr
			}
			logger.Info("meter provider shut down cleanly")
			return nil
		},
	})

	return mp, nil
}

// buildResource centralizes [resource.Resource] construction so trace and
// metric providers report the same service identity.
func buildResource(cfg Config) (*resource.Resource, error) {
	host, _ := os.Hostname()
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			//nolint:staticcheck // deprecated semconv key, safe for internal telemetry.
			semconv.DeploymentEnvironment(cfg.Environment),
			semconv.HostName(host),
		),
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
