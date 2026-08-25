# Telemetry Plugin — Implementation Notes

Status: Architecture defined, integration deferred. See `ERROR_HANDLING_OVERHAUL_PLAN.md`.

## What's Ready
- `internal/xerrors`: `TelemetrySink` interface + `DomainError.Meta`/`Source`
- `internal/telemetry`: Registry, `ZapSink`, `ErrorPolicy`, `ErrorEnricher`
- `internal/middleware/errors.go`: `HTTPError` passes errors to telemetry registry (async)

## What's Not Wired (do not touch until telemetry sprint)
- `telemetry.RequestIDResolver` / `SessionResolver` never initialized
- `telemetry.EnrichWithRequestContext` depends on middleware vars
- No concrete 3rd-party sinks (Sentry, Datadog, Prometheus)
- No metrics/observability hooks
- No tests for telemetry package

## When Implementing
1. Initialize resolvers in `middleware/register.go` (after session resolver runs)
2. Add concrete sink implementations under `internal/telemetry/sinks/`
3. Register sinks in `telemetry.Module` via `fx.Invoke`
4. Add metrics counters to `ZapSink.ProcessError`
5. Add `telemetry/` tests
6. Update `internal/middleware/errors.go`: wire enrichment, fix `handleSpecialErrors` duplication

## Plugin Pattern Example (for future)
```go
type SentrySink struct{ dsn string }
func (s *SentrySink) ProcessError(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) { /* ... */ }
func (s *SentrySink) Flush(ctx context.Context) error { return nil }
func (s *SentrySink) Name() string { return "sentry" }
// Register: telemetry.Registry.Register(&SentrySink{dsn: os.Getenv("SENTRY_DSN")})
```
