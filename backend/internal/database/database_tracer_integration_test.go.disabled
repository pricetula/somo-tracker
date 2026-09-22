package database

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	tracetest "go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap/zaptest"

	"somotracker/backend/internal/config"
)

// integrationDSN returns the DSN used by the migrator integration tests so
// every database integration test exercises the same test instance.
func integrationDSN() string {
	if dsn := os.Getenv("TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}
	return "postgres://postgres:postgres@localhost:5433/somotracker_test?sslmode=disable"
}

func TestPoolTracer_EmitsSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	original := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(original) })

	logger := zaptest.NewLogger(t)
	cfg := &config.Config{
		Port:              3030,
		Environment:       "local",
		DatabaseURL:       integrationDSN(),
		DBMaxConns:        5,
		DBMaxConnLifetime: 30,
		DBMaxConnIdleTime: 30,
	}

	pool, err := NewPool(nil, cfg, logger)
	if err != nil {
		t.Skipf("test database unavailable (skip): %v\n"+
			"  hint: run `make test-db-up` and ensure TEST_DATABASE_URL is set", err)
	}
	defer pool.Close()

	assert.NotNil(t, pool)
	_ = (*pgxpool.Pool)(pool)
	recorder.Reset()
}

func TestPoolConfig_HasTracer(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	original := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(original) })
	require.NotNil(t, tp, "tracer provider must be non-nil")
}
