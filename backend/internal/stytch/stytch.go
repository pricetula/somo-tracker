// Package stytch provides a secure, resilient B2B integration with the
// Stytch authentication platform. It is registered as an Fx module so
// other services and controllers can inject *Client cleanly.
//
// Security design:
//   - Never leaks internal error details to clients — all responses are
//     mapped through SanitizedError which strips stacks / DB context.
//   - All external operations are protected by a circuit breaker (gobreaker)
//     to prevent cascading failures during Stytch outages.
//   - Retries (exponential backoff + jitter) are applied ONLY to idempotent
//     read / verification calls. Non-idempotent writes are never retried.
//   - All failures (Stytch error codes, breaker state, retry exhaustion)
//     are logged securely via the shared Zap logger.
package stytch

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/sony/gobreaker"
	b2bstytchapi "github.com/stytchauth/stytch-go/v18/stytch/b2b/b2bstytchapi"
	stytchconfig "github.com/stytchauth/stytch-go/v18/stytch/config"
	"github.com/stytchauth/stytch-go/v18/stytch/stytcherror"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
)

// Module is the Fx module that exposes the Stytch client.
// Import with: fx.Options(config.Module, stytch.Module)
var Module = fx.Module(
	"stytch",
	fx.Provide(NewClient),
)

const (
	// Breaker settings tuned for auth verification traffic (low volume,
	// high sensitivity to outages). Fail fast after 5 errors in 30s.
	breakerFailureThreshold = 5
	breakerTimeout          = 30 * time.Second
	breakerResetTimeout     = 30 * time.Second

	// Retry settings — only for idempotent reads/verification.
	retryMaxAttempts = 3
	retryBaseDelay   = 200 * time.Millisecond
	retryMaxDelay    = 2 * time.Second
)

// Client wraps the official Stytch B2B SDK with circuit-breaker protection,
// idempotent retry logic, secure error mapping, and structured logging.
type Client struct {
	api    *b2bstytchapi.API
	cb     *gobreaker.CircuitBreaker
	logger *zap.Logger
}

// NewClient initializes the Stytch B2B SDK client from Config and registers
// it with Fx. The SDK performs JWKS initialization during construction;
// failure here aborts Fx startup so the app never runs without auth.
func NewClient(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("stytch.NewClient: config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("stytch.NewClient: logger is required")
	}
	if cfg.StytchProjectID == "" {
		return nil, fmt.Errorf("stytch.NewClient: STYTCH_PROJECT_ID is required")
	}
	if cfg.StytchSecret == "" {
		return nil, fmt.Errorf("stytch.NewClient: STYTCH_SECRET is required")
	}

	// Initialize with base URI override matching STYTCH_ENV.
	baseURI := stytchconfig.BaseURITest
	if cfg.StytchEnv == "live" {
		baseURI = stytchconfig.BaseURILive
	}

	api, err := b2bstytchapi.NewClient(cfg.StytchProjectID, cfg.StytchSecret,
		b2bstytchapi.WithBaseURI(string(baseURI)),
	)
	if err != nil {
		logger.Error("stytch: SDK initialization failed",
			zap.String("project_id", cfg.StytchProjectID),
			zap.String("env", cfg.StytchEnv),
			zap.Error(err),
		)
		return nil, fmt.Errorf("stytch.NewClient: SDK init: %w", err)
	}

	cbSettings := gobreaker.Settings{
		Name:        "stytch-b2b",
		MaxRequests: breakerFailureThreshold,
		Interval:    breakerTimeout,
		Timeout:     breakerTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= breakerFailureThreshold && failureRatio > 0.5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("stytch: circuit breaker state changed",
				zap.String("name", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()),
			)
		},
	}
	cb := gobreaker.NewCircuitBreaker(cbSettings)

	client := &Client{
		api:    api,
		cb:     cb,
		logger: logger.With(zap.String("service", "stytch")),
	}

	logger.Info("stytch: B2B client initialized",
		zap.String("project_id", cfg.StytchProjectID),
		zap.String("env", cfg.StytchEnv),
	)

	return client, nil
}

// SanitizedError maps Stytch API errors to secure, sanitized HTTP-level
// responses. It never leaks stack traces, DB context, or internal SDK paths.
func (c *Client) SanitizedError(err error) error {
	if err == nil {
		return nil
	}

	var stErr *stytcherror.Error
	if errors.As(err, &stErr) {
		c.logger.Warn("stytch: API error",
			zap.Int("status_code", stErr.StatusCode),
			zap.String("error_type", string(stErr.ErrorType)),
			zap.String("message", string(stErr.ErrorMessage)),
			zap.String("request_id", string(stErr.RequestID)),
		)

		switch {
		// Invalid token / magic link patterns.
		case stErr.StatusCode == 400 || strings.Contains(string(stErr.ErrorType), "invalid") ||
			strings.Contains(string(stErr.ErrorMessage), "invalid") ||
			strings.Contains(string(stErr.ErrorMessage), "magic_link") ||
			strings.Contains(string(stErr.ErrorMessage), "token"):
			return fmt.Errorf("bad_request: the magic link is invalid or has already been used, please request a new one")

		// Session expired / missing session.
		case stErr.StatusCode == 401 || strings.Contains(string(stErr.ErrorType), "session") ||
			strings.Contains(string(stErr.ErrorMessage), "expired") ||
			strings.Contains(string(stErr.ErrorMessage), "session"):
			return fmt.Errorf("unauthorized: your session has expired, please log in again")

		// Rate-limit patterns.
		case stErr.StatusCode == 429 || strings.Contains(string(stErr.ErrorType), "rate_limit") ||
			strings.Contains(string(stErr.ErrorMessage), "rate"):
			return fmt.Errorf("too_many_requests: too many requests, please wait a few minutes before trying again")

		default:
			c.logger.Error("stytch: unexpected API error",
				zap.Int("status_code", stErr.StatusCode),
				zap.String("error_type", string(stErr.ErrorType)),
				zap.Error(err),
			)
			return fmt.Errorf("internal_error: an unexpected error occurred, please try again later")
		}
	}

	// Non-Stytch errors (network, breaker open, retries exhausted).
	c.logger.Error("stytch: non-API failure",
		zap.String("error", err.Error()),
	)
	return fmt.Errorf("internal_error: an unexpected error occurred, please try again later")
}

// RetryWithBackoff executes the given operation with exponential backoff
// and full-jitter. It is intended ONLY for idempotent read/verification
// operations. Never call this for write requests.
func (c *Client) RetryWithBackoff(ctx context.Context, op func(context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < retryMaxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff with full jitter.
			delay := retryBaseDelay * time.Duration(1<<uint(attempt-1))
			if delay > retryMaxDelay {
				delay = retryMaxDelay
			}
			jitter := time.Duration(rand.Int63n(int64(delay)))
			sleep := delay + jitter

			c.logger.Info("stytch: retrying read/verification",
				zap.Int("attempt", attempt+1),
				zap.Duration("backoff", sleep),
			)

			select {
			case <-time.After(sleep):
			case <-ctx.Done():
				return fmt.Errorf("stytch: retry cancelled: %w", ctx.Err())
			}
		}

		lastErr = op(ctx)
		if lastErr == nil {
			return nil
		}
		// Only retry if the error looks retryable (network / temporary).
		if !isRetryable(lastErr) {
			return lastErr
		}
	}
	c.logger.Warn("stytch: retry exhausted for read/verification",
		zap.Int("max_attempts", retryMaxAttempts),
		zap.Error(lastErr),
	)
	return fmt.Errorf("stytch: retry exhausted: %w", lastErr)
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// Retry on temporary errors, network issues, rate limits (too many
	// requests may recover quickly), and breaker half-open states.
	msg := strings.ToLower(err.Error())
	retryable := []string{"timeout", "temporary", "connection", "network", "refused", "reset", "too many"}
	for _, s := range retryable {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// ReadCall executes the operation through the circuit breaker with safe
// retry logic (exponential backoff + jitter). Used for idempotent
// verification/authentication calls ONLY.
func (c *Client) ReadCall(op func(context.Context) error) error {
	_, err := c.cb.Execute(func() (any, error) {
		return nil, c.RetryWithBackoff(context.Background(), op)
	})
	if err != nil {
		return c.SanitizedError(err)
	}
	return nil
}

// WriteCall executes through the circuit breaker WITHOUT retries. Non-
// idempotent writes must never retry to avoid duplicate state mutations.
func (c *Client) WriteCall(op func(context.Context) error) error {
	_, err := c.cb.Execute(func() (any, error) {
		return nil, op(context.Background())
	})
	if err != nil {
		return c.SanitizedError(err)
	}
	return nil
}
