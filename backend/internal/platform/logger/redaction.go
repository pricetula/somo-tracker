package logger

import (
	"encoding/json"
	"log/slog"
)

// LogValuer is implemented by types that want to control their log representation.
// See slog.LogValuer interface.

// SensitiveString wraps a string that should be redacted in logs.
// Example: logger.Info("token", slog.Any("token", SensitiveString(token)))
type SensitiveString string

func (s SensitiveString) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

// SensitiveMap wraps a map that should have sensitive keys redacted.
type SensitiveMap map[string]string

func (m SensitiveMap) LogValue() slog.Value {
	redacted := make(map[string]string, len(m))
	for k, v := range m {
		if isSensitiveKey(k) {
			redacted[k] = "[REDACTED]"
		} else {
			redacted[k] = v
		}
	}
	return slog.AnyValue(redacted)
}

// Headers wraps HTTP headers for logging with redaction.
type Headers map[string]string

func (h Headers) LogValue() slog.Value {
	redacted := make(map[string]string, len(h))
	for k, v := range h {
		if isSensitiveKey(k) {
			redacted[k] = "[REDACTED]"
		} else {
			redacted[k] = v
		}
	}
	return slog.AnyValue(redacted)
}

// Cookie wraps a cookie for logging with redaction.
type Cookie struct {
	Name  string
	Value string
}

func (c Cookie) LogValue() slog.Value {
	if isSensitiveKey(c.Name) {
		return slog.GroupValue(
			slog.String("name", c.Name),
			slog.String("value", "[REDACTED]"),
		)
	}
	return slog.GroupValue(
		slog.String("name", c.Name),
		slog.String("value", c.Value),
	)
}

// RequestBody wraps a request body for logging with automatic redaction of sensitive fields.
type RequestBody []byte

func (b RequestBody) LogValue() slog.Value {
	var pretty map[string]interface{}
	if err := json.Unmarshal(b, &pretty); err != nil {
		return slog.StringValue(string(b))
	}
	redacted := redactMap(pretty)
	return slog.AnyValue(redacted)
}

// redactMap recursively redacts sensitive keys in a map
func redactMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if isSensitiveKey(k) {
			result[k] = "[REDACTED]"
			continue
		}
		switch v := v.(type) {
		case map[string]interface{}:
			result[k] = redactMap(v)
		case []interface{}:
			var arr []interface{}
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					arr = append(arr, redactMap(m))
				} else {
					arr = append(arr, item)
				}
			}
			result[k] = arr
		default:
			result[k] = v
		}
	}
	return result
}

// LogAttr creates a slog attribute with automatic redaction for sensitive keys.
func LogAttr(key string, value interface{}) slog.Attr {
	if isSensitiveKey(key) {
		return slog.String(key, "[REDACTED]")
	}
	return slog.Any(key, value)
}

// LogAttrs creates multiple slog attributes with automatic redaction.
func LogAttrs(kv ...interface{}) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		if i+1 >= len(kv) {
			break
		}
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, LogAttr(key, kv[i+1]))
	}
	return attrs
}

// ContextAttrs creates attributes from a context (request_id, user_id, tenant_id if present).
func ContextAttrs(ctx interface{ Get(string) interface{} }) []slog.Attr {
	var attrs []slog.Attr
	if v := ctx.Get("request_id"); v != nil {
		attrs = append(attrs, slog.String("request_id", v.(string)))
	}
	if v := ctx.Get("user_id"); v != nil {
		attrs = append(attrs, slog.String("user_id", v.(string)))
	}
	if v := ctx.Get("tenant_id"); v != nil {
		attrs = append(attrs, slog.String("tenant_id", v.(string)))
	}
	if v := ctx.Get("school_id"); v != nil {
		attrs = append(attrs, slog.String("school_id", v.(string)))
	}
	return attrs
}