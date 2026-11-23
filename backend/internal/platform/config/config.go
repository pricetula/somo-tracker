package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds all configuration loaded from environment variables.
// Secrets are injected by Doppler at runtime; none are read from the repo.
type Config struct {
	AppEnv        string
	LogLevel      string
	ServerPort    string
	DatabaseURL   string
	RedisURL      string

	// Brevo email provider
	BrevoAPIKey    string
	BrevoSenderEmail string
	BrevoSenderName  string

	// OTP security
	EmailHMACKey string

	// CORS
	CORSOrigins string

	// Turnstile
	TurnstileSecretKey     string
	TurnstileExpectedHost  string

	// Rate limits
	RateLimitOTPRequestIPPerMin   int
	RateLimitOTPRequestEmailPer15Min int

	// Session cookie name (REQUIRED, no default). Must be "session" in dev, "__Host-session" in prod.
	SessionCookieName string
}

// firstNonEmpty returns the first non-empty environment variable value.
func firstNonEmpty(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

// Load reads configuration from environment variables.
// All values must be provided via Doppler at runtime — no fallbacks.
// Missing required values are caught by validate().
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:        firstNonEmpty("APP_ENV"),
		LogLevel:      firstNonEmpty("LOG_LEVEL"),
		ServerPort:    firstNonEmpty("PORT"),
		DatabaseURL:   firstNonEmpty("DATABASE_URL"),
		RedisURL:      firstNonEmpty("REDIS_URL"),

		BrevoAPIKey:    firstNonEmpty("BREVO_API_KEY"),
		BrevoSenderEmail: firstNonEmpty("BREVO_SENDER_EMAIL"),
		BrevoSenderName:  firstNonEmpty("BREVO_SENDER_NAME"),

		EmailHMACKey: firstNonEmpty("EMAIL_HMAC_KEY"),

		CORSOrigins: firstNonEmpty("CORS_ORIGINS"),

		TurnstileSecretKey:    firstNonEmpty("TURNSTILE_SECRET_KEY"),
		TurnstileExpectedHost: firstNonEmpty("TURNSTILE_EXPECTED_HOST"),

		RateLimitOTPRequestIPPerMin:   10,
		RateLimitOTPRequestEmailPer15Min: 3,

		// REQUIRED - no default. Must be set via Doppler.
		SessionCookieName: firstNonEmpty("SESSION_COOKIE_NAME"),
	}

	// Validate required configuration at startup so missing secrets surface
	// immediately (e.g. an empty BREVO_API_KEY) instead of as confusing
	// per-request failures such as a 401 from the email provider.
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// validate ensures all required configuration is present. Fields without a
// sensible default must be provided by the operator (injected by Doppler into
// docker-compose at runtime, not checked into .env).
func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":        c.DatabaseURL,
		"BREVO_API_KEY":       c.BrevoAPIKey,
		"EMAIL_HMAC_KEY":      c.EmailHMACKey,
		"SESSION_COOKIE_NAME": c.SessionCookieName,
		"TURNSTILE_SECRET_KEY": c.TurnstileSecretKey,
	}
	var missing []string
	for k, v := range required {
		if v == "" {
			missing = append(missing, k)
		}
	}

	// In production, session cookie MUST use __Host- prefix
	if c.AppEnv == "production" && !strings.HasPrefix(c.SessionCookieName, "__Host-") {
		missing = append(missing, "SESSION_COOKIE_NAME (must start with __Host- in production)")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

// MustLoad loads config and panics on error.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

// SessionIdleTimeout returns the idle timeout duration.
func (c *Config) SessionIdleTimeout() time.Duration {
	return 7 * 24 * time.Hour
}

// SessionAbsoluteTimeout returns the absolute session lifetime.
func (c *Config) SessionAbsoluteTimeout() time.Duration {
	return 30 * 24 * time.Hour
}

// OTPValidity returns the OTP validity window.
func (c *Config) OTPValidity() time.Duration {
	return 10 * time.Minute
}

// OTPMaxAttempts returns the maximum OTP verification attempts.
func (c *Config) OTPMaxAttempts() int {
	return 5
}

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}
