package config

import (
	"os"
	"time"
)

// Config holds all configuration loaded from environment variables.
// Secrets are injected by Doppler at runtime; none are read from the repo.
type Config struct {
	AppEnv        string
	LogLevel      string
	ServerPort    string
	PostgresDSN   string
	RedisURL      string

	// Brevo email provider
	BrevoAPIKey    string
	BrevoSenderEmail string
	BrevoSenderName  string

	// OTP security
	EmailHMACKey string

	// Turnstile (Cloudflare bot protection)
	TurnstileEnabled  bool
	TurnstileSecretKey string

	// Migrations
	MigrationsDir string

	// CORS
	CORSOrigins string

	// Rate limits
	RateLimitOTPRequestIPPerMin   int
	RateLimitOTPRequestEmailPer15Min int

	// Feature flags
	SeedReferenceData bool
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
func Load() (*Config, error) {
	return &Config{
		AppEnv:        firstNonEmpty("APP_ENV", "ENV"),
		LogLevel:      firstNonEmpty("LOG_LEVEL", "info"),
		ServerPort:    firstNonEmpty("SERVER_PORT", "8080"),
		PostgresDSN:   firstNonEmpty("POSTGRES_DSN"),
		RedisURL:      firstNonEmpty("REDIS_URL", "redis://localhost:6379"),

		BrevoAPIKey:    firstNonEmpty("BREVO_API_KEY"),
		BrevoSenderEmail: firstNonEmpty("BREVO_SENDER_EMAIL", "noreply@somotracker.com"),
		BrevoSenderName:  firstNonEmpty("BREVO_SENDER_NAME", "Somotracker"),

		EmailHMACKey: firstNonEmpty("EMAIL_HMAC_KEY"),

		TurnstileEnabled:  firstNonEmpty("TURNSTILE_ENABLED") == "true",
		TurnstileSecretKey: firstNonEmpty("TURNSTILE_SECRET_KEY"),

		MigrationsDir: firstNonEmpty("MIGRATIONS_DIR", "migrations"),

		CORSOrigins: firstNonEmpty("CORS_ORIGINS", "*"),

		RateLimitOTPRequestIPPerMin:   10,
		RateLimitOTPRequestEmailPer15Min: 3,

		SeedReferenceData: firstNonEmpty("SEED_REFERENCE_DATA") == "true",
	}, nil
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
