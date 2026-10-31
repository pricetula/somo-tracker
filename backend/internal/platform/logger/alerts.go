package logger

import (
	"context"
	"log/slog"
	"time"
)

// AlertLevel represents the severity of an alert
type AlertLevel string

const (
	AlertLevelCritical AlertLevel = "critical"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelInfo     AlertLevel = "info"
)

// Alert represents an alert to be sent to the alerting system
type Alert struct {
	Level       AlertLevel
	Title       string
	Message     string
	Service     string
	Environment string
	RequestID   string
	Timestamp   time.Time
	Metadata    map[string]string
}

// AlertHandler is an interface for sending alerts
type AlertHandler interface {
	Send(ctx context.Context, alert Alert) error
}

// DefaultAlertHandler is a no-op handler (logs to slog)
type DefaultAlertHandler struct{}

func (h *DefaultAlertHandler) Send(ctx context.Context, alert Alert) error {
	attrs := []any{
		"alert_level", alert.Level,
		"alert_title", alert.Title,
		"request_id", alert.RequestID,
	}
	for k, v := range alert.Metadata {
		attrs = append(attrs, k, v)
	}

	switch alert.Level {
	case AlertLevelCritical:
		slog.ErrorContext(ctx, alert.Message, attrs...)
	case AlertLevelWarning:
		slog.WarnContext(ctx, alert.Message, attrs...)
	default:
		slog.InfoContext(ctx, alert.Message, attrs...)
	}
	return nil
}

// AlertManager manages alert sending with deduplication
type AlertManager struct {
	handler AlertHandler
	// deduplication could be added here
}

func NewAlertManager(handler AlertHandler) *AlertManager {
	if handler == nil {
		handler = &DefaultAlertHandler{}
	}
	return &AlertManager{handler: handler}
}

// Fire sends an alert
func (m *AlertManager) Fire(ctx context.Context, alert Alert) {
	alert.Timestamp = time.Now()
	if alert.Service == "" {
		alert.Service = "somotracker"
	}
	if alert.Environment == "" {
		alert.Environment = "development"
	}
	_ = m.handler.Send(ctx, alert)
}

// Predefined alert constructors

// Alert5xx fires a critical alert for 5xx errors
func (m *AlertManager) Alert5xx(ctx context.Context, requestID, route string, err error) {
	m.Fire(ctx, Alert{
		Level:       AlertLevelCritical,
		Title:       "5xx Error",
		Message:     "5xx error on " + route + ": " + err.Error(),
		RequestID:   requestID,
		Metadata:    map[string]string{"route": route},
	})
}

// AlertHighErrorRate fires a warning alert for high error rate
func (m *AlertManager) AlertHighErrorRate(ctx context.Context, rate float64, window time.Duration) {
	m.Fire(ctx, Alert{
		Level:    AlertLevelWarning,
		Title:    "High Error Rate",
		Message:  "Error rate is " + rateString(rate) + " over " + window.String(),
		Metadata: map[string]string{"rate": rateString(rate), "window": window.String()},
	})
}

// AlertPanic fires a critical alert for recovered panics
func (m *AlertManager) AlertPanic(ctx context.Context, requestID string, panicValue interface{}) {
	m.Fire(ctx, Alert{
		Level:       AlertLevelCritical,
		Title:       "Panic Recovered",
		Message:     "Panic recovered: " + toString(panicValue),
		RequestID:   requestID,
		Metadata:    map[string]string{"panic": toString(panicValue)},
	})
}

// AlertBackgroundJobFailure fires a warning for failed background jobs
func (m *AlertManager) AlertBackgroundJobFailure(ctx context.Context, jobName string, err error) {
	m.Fire(ctx, Alert{
		Level:    AlertLevelWarning,
		Title:    "Background Job Failed",
		Message:  "Job " + jobName + " failed: " + err.Error(),
		Metadata: map[string]string{"job": jobName},
	})
}

func rateString(r float64) string {
	if r >= 1 {
		return "100%"
	}
	return toFixed(r*100, 2) + "%"
}

func toFixed(num float64, precision int) string {
	mult := 1
	for i := 0; i < precision; i++ {
		mult *= 10
	}
	return string(rune(int(num*float64(mult) + 0.5)))
}

func toString(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	switch v := v.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		return "unknown"
	}
}