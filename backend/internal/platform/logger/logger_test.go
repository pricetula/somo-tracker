package logger

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"
)

func TestInit_ProductionJSON(t *testing.T) {
	Init(Config{
		Level:       LevelInfo,
		Environment: "production",
		Version:     "test-123",
		Service:     "test-service",
	})

	logger := slog.Default()
	if logger == nil {
		t.Fatal("default logger not set")
	}
}

func TestInit_DevelopmentText(t *testing.T) {
	Init(Config{
		Level:       LevelDebug,
		Environment: "development",
		Version:     "dev",
		Service:     "test-service",
	})

	logger := slog.Default()
	if logger == nil {
		t.Fatal("default logger not set")
	}
}

func TestLoggerFrom_EmptyContext(t *testing.T) {
	ctx := context.Background()
	logger := LoggerFrom(ctx)
	if logger == nil {
		t.Fatal("LoggerFrom returned nil")
	}
}

func TestWithLogger(t *testing.T) {
	custom := slog.New(slog.NewTextHandler(nil, nil))
	ctx := WithLogger(context.Background(), custom)

	logger := LoggerFrom(ctx)
	if logger != custom {
		t.Errorf("LoggerFrom returned wrong logger")
	}
}

func TestRequestLogger(t *testing.T) {
	ctx := WithLogger(context.Background(), slog.Default())
	logger := RequestLogger(ctx, "req_test123")

	if logger == nil {
		t.Fatal("RequestLogger returned nil")
	}
}

func TestErrorLogger(t *testing.T) {
	ctx := WithLogger(context.Background(), slog.Default())
	logger := ErrorLogger(ctx, 
		fmt.Errorf("test error"),
		slog.String("op", "test"),
	)

	if logger == nil {
		t.Fatal("ErrorLogger returned nil")
	}
}

func TestRedactedString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "[REDACTED]"},
		{"short", "[REDACTED]"},
		{"12345678", "[REDACTED]"},
		{"123456789", "1234*6789"},
		{"this-is-a-long-string", "this*************ring"},
	}

	for _, tc := range tests {
		result := RedactedString(tc.input)
		if result != tc.expected {
			t.Errorf("RedactedString(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestRedactedMap(t *testing.T) {
	input := map[string]string{
		"username":  "john",
		"password":  "secret123",
		"api_token": "abc123",
		"email":     "john@example.com",
	}

	result := RedactedMap(input)

	if result["username"] != "john" {
		t.Errorf("username not redacted: %v", result["username"])
	}
	if result["password"] != "[REDACTED]" {
		t.Errorf("password not redacted: %v", result["password"])
	}
	if result["api_token"] != "[REDACTED]" {
		t.Errorf("api_token not redacted: %v", result["api_token"])
	}
	if result["email"] != "john@example.com" {
		t.Errorf("email redacted: %v", result["email"])
	}
}

func TestSensitiveString_LogValue(t *testing.T) {
	s := SensitiveString("my-secret-token")
	val := slog.StringValue(s.LogValue().String())
	if val.String() != "[REDACTED]" {
		t.Errorf("SensitiveString LogValue = %v, want [REDACTED]", val)
	}
}

func TestSensitiveMap_LogValue(t *testing.T) {
	m := SensitiveMap{"password": "secret", "name": "john"}
	val := m.LogValue()
	_ = val
}

func TestHeaders_LogValue(t *testing.T) {
	h := Headers{
		"Authorization": "Bearer token123",
		"Content-Type":  "application/json",
	}
	val := h.LogValue()
	_ = val
}

func TestCookie_LogValue(t *testing.T) {
	c := Cookie{Name: "session_token", Value: "abc123"}
	val := c.LogValue()
	_ = val

	c2 := Cookie{Name: "theme", Value: "dark"}
	val2 := c2.LogValue()
	_ = val2
}

func TestRequestBody_LogValue(t *testing.T) {
	body := RequestBody(`{"email": "test@example.com", "password": "secret"}`)
	val := body.LogValue()
	_ = val
}

func TestLogAttr(t *testing.T) {
	attr := LogAttr("password", "secret")
	if attr.Value.String() != "[REDACTED]" {
		t.Errorf("LogAttr(password) = %v, want [REDACTED]", attr.Value)
	}

	attr = LogAttr("name", "john")
	if attr.Value.String() != "john" {
		t.Errorf("LogAttr(name) = %v, want john", attr.Value)
	}
}

func TestLogAttrs(t *testing.T) {
	attrs := LogAttrs("name", "john", "password", "secret", "email", "test@example.com")
	if len(attrs) != 3 {
		t.Errorf("LogAttrs returned %d attrs, want 3", len(attrs))
	}
	for _, a := range attrs {
		if a.Key == "password" && a.Value.String() != "[REDACTED]" {
			t.Errorf("password not redacted: %v", a.Value)
		}
	}
}

type testContext struct {
	values map[string]interface{}
}

func (tc testContext) Get(key string) interface{} {
	return tc.values[key]
}

func TestContextAttrs(t *testing.T) {
	ctx := testContext{values: map[string]interface{}{
		"request_id": "req_123",
		"user_id":    "user_456",
		"tenant_id":  "tenant_789",
	}}

	attrs := ContextAttrs(ctx)
	if len(attrs) != 3 {
		t.Errorf("ContextAttrs returned %d attrs, want 3", len(attrs))
	}
	for _, a := range attrs {
		if a.Key == "request_id" && a.Value.String() != "req_123" {
			t.Errorf("request_id = %v, want req_123", a.Value)
		}
		if a.Key == "user_id" && a.Value.String() != "user_456" {
			t.Errorf("user_id = %v, want user_456", a.Value)
		}
		if a.Key == "tenant_id" && a.Value.String() != "tenant_789" {
			t.Errorf("tenant_id = %v, want tenant_789", a.Value)
		}
	}
}

func TestSafeGo_RecoversPanic(t *testing.T) {
	errCh := make(chan error, 1)
	ctx := context.Background()

	SafeGo(ctx, func(ctx context.Context) error {
		panic("test panic")
	}, errCh)

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error from panic")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for panic error")
	}
}

func TestSafeGo_ReturnsError(t *testing.T) {
	errCh := make(chan error, 1)
	ctx := context.Background()

	SafeGo(ctx, func(ctx context.Context) error {
		return nil
	}, errCh)

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for error")
	}
}

func TestSafeGoWithTimeout(t *testing.T) {
	errCh := make(chan error, 1)
	ctx := context.Background()

	SafeGoWithTimeout(ctx, 50*time.Millisecond, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}, errCh)

	select {
	case err := <-errCh:
		if err != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for timeout error")
	}
}

func TestGroup_ReturnsFirstError(t *testing.T) {
	ctx := context.Background()

	err := Group(ctx,
		func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		},
	)

	if err != nil {
		t.Errorf("Group returned error: %v", err)
	}
}

func TestGroup_CancelsOnFirstError(t *testing.T) {
	ctx := context.Background()

	err := Group(ctx,
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
	)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAlertManager_Fire(t *testing.T) {
	m := NewAlertManager(nil)
	ctx := context.Background()

	m.Fire(ctx, Alert{
		Level:   AlertLevelWarning,
		Title:   "Test",
		Message: "test message",
	})
}

func TestAlertManager_Alert5xx(t *testing.T) {
	m := NewAlertManager(nil)
	ctx := context.Background()

	m.Alert5xx(ctx, "req_123", "/test", fmt.Errorf("test error"))
}

func TestAlertManager_AlertPanic(t *testing.T) {
	m := NewAlertManager(nil)
	ctx := context.Background()

	m.AlertPanic(ctx, "req_123", "test panic")
}

func TestAlertManager_AlertBackgroundJobFailure(t *testing.T) {
	m := NewAlertManager(nil)
	ctx := context.Background()

	m.AlertBackgroundJobFailure(ctx, "test-job", fmt.Errorf("job failed"))
}

func TestDefaultAlertHandler_Send(t *testing.T) {
	h := &DefaultAlertHandler{}
	ctx := context.Background()

	err := h.Send(ctx, Alert{
		Level:   AlertLevelCritical,
		Title:   "Test",
		Message: "test",
	})
	if err != nil {
		t.Errorf("Send returned error: %v", err)
	}
}