// Package telemetry provides pluggable error telemetry sinks.
//
// This package provides pluggable error telemetry sinks for processing errors
// and sending them to various monitoring platforms. Sinks are designed to be
// asynchronous and non-blocking to ensure error processing never delays response.
//
// This file defines the concrete telemetry sink implementations.
package telemetry

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"somotracker/backend/internal/xerrors"
)

// ─── Sink Registry ────────────────────────────────────────────────────────

// Registry holds all registered telemetry sinks.
// This acts as a global registry pattern that can be configured at startup.
// Usage: telemetry.GetRegistry().Register(mySink)
var Registry *RegistryType

func init() {
	Registry = NewRegistry()
}

// RegistryType manages telemetry sinks and provides unified access.
type RegistryType struct {
	mu     sync.RWMutex
	nsinks []xerrors.TelemetrySink
	logger *zap.SugaredLogger
}

// NewRegistry creates a new telemetry sink registry.
func NewRegistry() *RegistryType {
	return &RegistryType{
		logger: zap.NewNop().Sugar(),
	}
}

// Register adds a telemetry sink to the registry.
// Thread-safe.
func (r *RegistryType) Register(sink xerrors.TelemetrySink) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nsinks = append(r.nsinks, sink)
	r.logger.Infow("telemetry sink registered", "name", sink.Name())
}

// ProcessAll sends an error to all registered sinks.
// Non-blocking for each sink to avoid delaying the error response.
func (r *RegistryType) ProcessAll(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) {
	r.mu.RLock()
	sinks := make([]xerrors.TelemetrySink, len(r.nsinks))
	copy(sinks, r.nsinks)
	r.mu.RUnlock()

	// Process each sink in its own goroutine to avoid blocking
	for _, sink := range sinks {
		go func(s xerrors.TelemetrySink) {
			tryProcess(s, ctx, err, req)
		}(sink)
	}
}

func tryProcess(sink xerrors.TelemetrySink, ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) {
	defer func() {
		if r := recover(); r != nil {
			// Log but don't crash - sinks should be resilient
			Registry.logger.Errorw("telemetry sink panic",
				"sink", sink.Name(),
				"error", r,
				"domain_error", err,
			)
		}
	}()
	sink.ProcessError(ctx, err, req)
}

// FlushAll flushes all sinks to ensure pending data is sent.
func (r *RegistryType) FlushAll(ctx context.Context) error {
	r.mu.RLock()
	sinks := make([]xerrors.TelemetrySink, len(r.nsinks))
	copy(sinks, r.nsinks)
	r.mu.RUnlock()

	var lastErr error
	for _, sink := range sinks {
		if err := sink.Flush(ctx); err != nil {
			r.logger.Errorw("sink flush error",
				"sink", sink.Name(),
				"error", err,
			)
			lastErr = err
		}
	}
	return lastErr
}

// ─── Concrete Sinks ──────────────────────────────────────────────────────

// ZapSink sends errors to the existing logging infrastructure.
type ZapSink struct {
	logger *zap.SugaredLogger
	// Optional filter to skip certain error codes
	filterFunc func(*xerrors.DomainError) bool
}

func NewZapSink(logger *zap.SugaredLogger) *ZapSink {
	return &ZapSink{logger: logger}
}

func (z *ZapSink) ProcessError(ctx context.Context, err *xerrors.DomainError, req *xerrors.TelemetryRequest) {
	if z.filterFunc != nil && z.filterFunc(err) {
		return
	}

	fields := []interface{}{
		"error_code", err.Code,
		"error_message", err.Message,
		"status", err.Status,
		"method", req.Method,
		"path", req.Path,
	}

	if req.RequestID != "" {
		fields = append(fields, "request_id", req.RequestID)
	}
	if req.UserID != "" {
		fields = append(fields, "user_id", req.UserID)
	}
	if req.TenantID != "" {
		fields = append(fields, "tenant_id", req.TenantID)
	}

	// Add metadata fields if present
	if err.Meta != nil {
		for k, v := range err.Meta {
			// Only add simple types to avoid complex serialization issues
			switch v.(type) {
			case string, int, float64, bool:
				fields = append(fields, k, v)
			}
		}
	}

	// Add source information if available
	if err.Source != nil {
		fields = append(fields,
			"error_source_package", err.Source.Package,
			"error_source_function", err.Source.Function,
			"error_source_file", err.Source.File,
			"error_source_line", err.Source.Line,
		)
	}

	if err.Status >= 500 {
		z.logger.Errorw("error processed via ZapSink", fields...)
	} else {
		z.logger.Infow("error processed via ZapSink", fields...)
	}
}

func (z *ZapSink) Flush(ctx context.Context) error {
	// Zap logger doesn't have a flush method, but we can sync underlying logger
	return nil
}

func (z *ZapSink) Name() string {
	return "zap_sink"
}

// SetFilter allows filtering which errors are sent to the sink.
func (z *ZapSink) SetFilter(fn func(*xerrors.DomainError) bool) {
	z.filterFunc = fn
}

// ─── Sink Interface Validation ──────────────────────────────────────────────

// Ensure our concrete sinks implement the interface correctly.
var _ xerrors.TelemetrySink = (*ZapSink)(nil)

// ─── Telemetry Request ────────────────────────────────────────────────────

// TelemetryRequest contains contextual information about the error occurrence.
type TelemetryRequest struct {
	Method    string
	Path      string
	Query     map[string][]string
	Headers   map[string]string
	UserID    string
	TenantID  string
	RequestID string
	// Context is additional structured data from the request
	Context map[string]any
}
