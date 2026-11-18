package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger wraps slog with structured logging.
type Logger struct {
	*slog.Logger
}

// New creates a logger writing JSON to stdout.
func New(level string) *Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return &Logger{slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	}))}
}

// Info logs an info message with attributes.
func (l *Logger) Info(ctx context.Context, msg string, attrs ...any) {
	l.LogAttrs(ctx, slog.LevelInfo, msg, toAttrs(attrs)...)
}

// Error logs an error message with attributes.
func (l *Logger) Error(ctx context.Context, err error, msg string, attrs ...any) {
	all := append([]any{"err", err}, attrs...)
	l.LogAttrs(ctx, slog.LevelError, msg, toAttrs(all)...)
}

// Debug logs a debug message with attributes.
func (l *Logger) Debug(ctx context.Context, msg string, attrs ...any) {
	l.LogAttrs(ctx, slog.LevelDebug, msg, toAttrs(attrs)...)
}

func toAttrs(attrs []any) []slog.Attr {
	result := make([]slog.Attr, 0, len(attrs)/2)
	for i := 0; i+1 < len(attrs); i += 2 {
		key, ok := attrs[i].(string)
		if !ok {
			continue
		}
		result = append(result, slog.Any(key, attrs[i+1]))
	}
	return result
}

// Default returns the default slog-based logger.
func Default() *Logger {
	return New("info")
}
