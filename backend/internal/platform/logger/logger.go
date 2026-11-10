package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Level is a wrapper around slog.Level for config
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

func (l Level) SlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Config holds logger configuration
type Config struct {
	Level       Level
	Environment string // development, staging, production
	Version     string // git SHA
	Service     string
}

// DefaultConfig returns default logger configuration
func DefaultConfig() Config {
	return Config{
		Level:       LevelInfo,
		Environment: "development",
		Version:     "dev",
		Service:     "somotracker",
	}
}

// Init initializes the global slog logger with JSON handler
// Call once at application startup
func Init(cfg Config) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: cfg.Level.SlogLevel(),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Redact sensitive fields
			if isSensitiveKey(a.Key) {
				return slog.String(a.Key, "[REDACTED]")
			}
			return a
		},
	}

	if cfg.Environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		// Human-readable text for development
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(
		slog.String("service", cfg.Service),
		slog.String("env", cfg.Environment),
		slog.String("version", cfg.Version),
	)
	slog.SetDefault(logger)
}

// isSensitiveKey returns true if the key should be redacted
func isSensitiveKey(key string) bool {
	sensitive := []string{
		"password", "token", "secret", "authorization",
		"cookie", "set-cookie", "x-api-key", "x-auth-token",
		"credit_card", "ssn", "passport",
	}
	lower := strings.ToLower(key)
	for _, s := range sensitive {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// Context keys for request-scoped values
type ctxKey string

const (
	loggerKey ctxKey = "logger"
)

// LoggerFrom extracts the request-scoped logger from context
// Falls back to slog.Default() if not set
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// WithLogger adds a logger to context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// RequestLogger creates a logger with request context
func RequestLogger(ctx context.Context, requestID string, attrs ...slog.Attr) *slog.Logger {
	base := LoggerFrom(ctx)
	allAttrs := make([]any, 0, len(attrs)+1)
	allAttrs = append(allAttrs, slog.String("request_id", requestID))
	for _, a := range attrs {
		allAttrs = append(allAttrs, a)
	}
	return base.With(allAttrs...)
}

// ErrorLogger creates a logger with error context
func ErrorLogger(ctx context.Context, err error, attrs ...slog.Attr) *slog.Logger {
	base := LoggerFrom(ctx)
	allAttrs := make([]any, 0, len(attrs)+1)
	allAttrs = append(allAttrs, slog.String("error", err.Error()))
	for _, a := range attrs {
		allAttrs = append(allAttrs, a)
	}
	return base.With(allAttrs...)
}

// Error logs an error at Error level with the given message
func Error(ctx context.Context, err error, msg string, attrs ...slog.Attr) {
	logger := LoggerFrom(ctx)
	args := make([]any, 0, len(attrs)+1)
	args = append(args, slog.String("error", err.Error()))
	for _, a := range attrs {
		args = append(args, a)
	}
	logger.Log(ctx, slog.LevelError, msg, args...)
}

// RedactedString returns a redacted string for logging
func RedactedString(s string) string {
	if len(s) <= 8 {
		return "[REDACTED]"
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}

// RedactedMap returns a copy of the map with sensitive keys redacted
func RedactedMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		if isSensitiveKey(k) {
			result[k] = "[REDACTED]"
		} else {
			result[k] = v
		}
	}
	return result
}