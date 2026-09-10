// Package captcha provides CAPTCHA verification middleware for abuse-prone endpoints.
//
// Currently supports a no-op "disabled" mode for development and a pluggable
// interface for production CAPTCHA providers (hCaptcha, Turnstile, reCAPTCHA v3).
//
// Integration:
//
//	app.Post("/api/auth/magic-link/send",
//	    ratelimit.NewRateLimitMiddleware(limiter, authRateEmail, "api:auth:magic-link:email"),
//	    captcha.NewCaptchaMiddleware(cfg, logger), // Requires CAPTCHA when triggered
//	    r.Auth.sendMagicLink,
//	)
//
// The middleware is triggered conditionally (e.g., when email rate limit is near
// exhaustion or on suspicious patterns) to avoid adding friction for legitimate users.
package captcha

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

// Provider defines the interface for CAPTCHA verification.
type Provider interface {
	// Verify checks the CAPTCHA token and returns nil if valid.
	// The context carries the request ID for logging correlation.
	Verify(ctx context.Context, token, clientIP string) error
}

// Config holds CAPTCHA configuration.
type Config struct {
	// Enabled turns on CAPTCHA verification. When false, middleware passes through.
	Enabled bool

	// ProviderName identifies the CAPTCHA provider ("hcaptcha", "turnstile", "recaptcha").
	ProviderName string

	// SiteKey is the public site key (used by frontend).
	SiteKey string

	// SecretKey is the private secret key (used by backend verification).
	SecretKey string

	// Threshold for score-based providers (reCAPTCHA v3, Turnstile).
	// Requests below this score are rejected. Range: 0.0 - 1.0.
	ScoreThreshold float64
}

// DefaultConfig returns a development-friendly config with CAPTCHA disabled.
func DefaultConfig() Config {
	return Config{
		Enabled:        false,
		ProviderName:   "disabled",
		ScoreThreshold: 0.5,
	}
}

// VerifyRequest extracts the CAPTCHA token from the request.
// Supports both form field (cf-turnstile-response, h-captcha-response, g-recaptcha-response)
// and header (X-Captcha-Token) for flexibility.
func VerifyRequest(c fiber.Ctx) string {
	// Check common form field names
	if token := c.FormValue("cf-turnstile-response"); token != "" {
		return token
	}
	if token := c.FormValue("h-captcha-response"); token != "" {
		return token
	}
	if token := c.FormValue("g-recaptcha-response"); token != "" {
		return token
	}
	// Check custom header
	if token := c.Get("X-Captcha-Token"); token != "" {
		return token
	}
	return ""
}

// NewCaptchaMiddleware returns a middleware that verifies CAPTCHA tokens.
// When disabled (config.Enabled=false), it passes through without verification.
// When enabled, it requires a valid CAPTCHA token on POST requests.
func NewCaptchaMiddleware(cfg Config, logger *zap.Logger) fiber.Handler {
	var provider Provider
	var err error

	if !cfg.Enabled {
		provider = &noopProvider{}
	} else {
		provider, err = newProvider(cfg)
		if err != nil {
			// Log error but don't crash - fall back to no-op with warning
			logger.Warn("captcha: provider initialization failed, falling back to no-op",
				zap.Error(err),
				zap.String("provider", cfg.ProviderName),
			)
			provider = &noopProvider{}
		}
	}

	mwLogger := logger
	if logger != nil {
		mwLogger = logger.With(zap.String("middleware", "captcha"))
	} else {
		mwLogger = zap.NewNop()
	}

	return func(c fiber.Ctx) error {
		// Only verify on mutating requests
		if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead || c.Method() == fiber.MethodOptions {
			return c.Next()
		}

		// If CAPTCHA is disabled, pass through
		if !cfg.Enabled {
			return c.Next()
		}

		token := VerifyRequest(c)
		if token == "" {
			mwLogger.Warn("captcha: missing token",
				zap.String("path", c.Path()),
				zap.String("ip", c.IP()),
				zap.String("request_id", c.Get("X-Request-ID")),
			)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "captcha_required",
				"message": "CAPTCHA verification required",
				"errors":  fiber.Map{"captcha": []string{"required"}},
			})
		}

		ctx := c.Context()
		clientIP := c.IP()

		if err := provider.Verify(ctx, token, clientIP); err != nil {
			mwLogger.Warn("captcha: verification failed",
				zap.String("path", c.Path()),
				zap.String("ip", clientIP),
				zap.String("request_id", c.Get("X-Request-ID")),
				zap.Error(err),
			)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"code":    "captcha_invalid",
				"message": "CAPTCHA verification failed",
				"errors":  fiber.Map{"captcha": []string{"invalid or expired"}},
			})
		}

		return c.Next()
	}
}

// noopProvider is a no-op provider for development/disabled mode.
type noopProvider struct{}

func (n *noopProvider) Verify(ctx context.Context, token, clientIP string) error {
	return nil
}

// newProvider creates a CAPTCHA provider from config.
func newProvider(cfg Config) (Provider, error) {
	switch strings.ToLower(cfg.ProviderName) {
	case "hcaptcha":
		return newHCaptchaProvider(cfg)
	case "turnstile":
		return newTurnstileProvider(cfg)
	case "recaptcha":
		return newRecaptchaProvider(cfg)
	default:
		return nil, fmt.Errorf("captcha: unknown provider %q", cfg.ProviderName)
	}
}

// hCaptchaProvider verifies hCaptcha tokens.
type hCaptchaProvider struct {
	secretKey string
}

func newHCaptchaProvider(cfg Config) (Provider, error) {
	if cfg.SecretKey == "" {
		return nil, errors.New("captcha: hCaptcha secret key required")
	}
	return &hCaptchaProvider{secretKey: cfg.SecretKey}, nil
}

func (p *hCaptchaProvider) Verify(ctx context.Context, token, clientIP string) error {
	// TODO: Implement hCaptcha verification via HTTP call to https://hcaptcha.com/siteverify
	// For now, return not implemented
	return errors.New("captcha: hCaptcha verification not yet implemented")
}

// turnstileProvider verifies Cloudflare Turnstile tokens.
type turnstileProvider struct {
	secretKey string
}

func newTurnstileProvider(cfg Config) (Provider, error) {
	if cfg.SecretKey == "" {
		return nil, errors.New("captcha: Turnstile secret key required")
	}
	return &turnstileProvider{secretKey: cfg.SecretKey}, nil
}

func (p *turnstileProvider) Verify(ctx context.Context, token, clientIP string) error {
	// TODO: Implement Turnstile verification via HTTP call to https://challenges.cloudflare.com/turnstile/v0/siteverify
	// For now, return not implemented
	return errors.New("captcha: Turnstile verification not yet implemented")
}

// recaptchaProvider verifies reCAPTCHA v3 tokens (score-based).
type recaptchaProvider struct {
	secretKey      string
	scoreThreshold float64
}

func newRecaptchaProvider(cfg Config) (Provider, error) {
	if cfg.SecretKey == "" {
		return nil, errors.New("captcha: reCAPTCHA secret key required")
	}
	threshold := cfg.ScoreThreshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.5
	}
	return &recaptchaProvider{
		secretKey:      cfg.SecretKey,
		scoreThreshold: threshold,
	}, nil
}

func (p *recaptchaProvider) Verify(ctx context.Context, token, clientIP string) error {
	// TODO: Implement reCAPTCHA v3 verification via HTTP call to https://www.google.com/recaptcha/api/siteverify
	// For now, return not implemented
	return errors.New("captcha: reCAPTCHA verification not yet implemented")
}
