package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DatabaseURL       string
	RedisURL          string
	AppEnv            string
	Port              string
	AllowedOrigins    string
	CookieDomain      string
	StytchProjectID   string
	StytchSecret      string
	StytchEnv         string
	StytchRedirectURL string
	StytchBaseURL     string // optional: override Stytch API base URL (for testing)
	BackendURL        string
	FrontendURL       string
	CookieSecret      string        // HMAC-SHA256 key for signing somo_role cookie
	RateLimitIPMax    int64         // Tier 1: Max requests per window per IP (e.g. 300)
	RateLimitUserMax  int64         // Tier 2: Max requests per window per User ID (e.g. 60)
	RateLimitWindow   time.Duration // Rate limit window (e.g. 1m)
	SessionTTL        time.Duration // Session token TTL, default 30 days

	// EnforceDeviceFingerprint turns the C5 device-bound session check into a
	// hard 401 in production. Defaults to false: mismatches are logged but the
	// request is allowed, so IP/UA churn behind proxies/NAT/mobile networks
	// cannot log users out. Enable only when strict device binding is required.
	EnforceDeviceFingerprint bool
}

// Load reads configuration from environment variables with safe fallbacks.
// It also attempts to load a .env file adjacent to the binary or in the
// working directory as a fallback for local development.
func Load() Config {
	// Attempt to load .env file — this is a no-op if the file doesn't exist.
	// Docker/CI should set all vars via the OS environment; .env is a
	// convenience for local development without docker-compose.
	if err := godotenv.Load(); err == nil {
		// Only log if we actually found and loaded a file
		logger, _ := zap.NewProduction()
		if logger != nil {
			cwd, _ := os.Getwd()
			envPath := filepath.Join(cwd, ".env")
			if _, statErr := os.Stat(envPath); statErr == nil {
				logger.Info("config: loaded .env file", zap.String("path", envPath))
			}
			if syncErr := logger.Sync(); syncErr != nil {
				logger.Warn("config: logger sync failed", zap.Error(syncErr))
			}
		}
	}

	appEnv := getEnv("APP_ENV", "development")
	cookieSecret := getEnv("COOKIE_SECRET", "")
	if cookieSecret == "" {
		if appEnv == "development" {
			cookieSecret = "dev-insecure-change-in-production"
		}
	}
	if appEnv != "development" && cookieSecret == "" {
		panic("COOKIE_SECRET must be set in non-development environments")
	}
	if appEnv != "development" && cookieSecret == "dev-insecure-change-in-production" {
		panic("COOKIE_SECRET must not be the development default in non-development environments")
	}
	enforceDeviceFingerprint := envBool("ENFORCE_DEVICE_FINGERPRINT", false)
	if appEnv != "development" && !enforceDeviceFingerprint {
		panic("ENFORCE_DEVICE_FINGERPRINT must be true in non-development environments")
	}

	return Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://somo_admin:somo_secure_password@somotracker_postgres:5432/somotracker_dev?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis:6379"),
		AppEnv:            appEnv,
		Port:              getEnv("PORT", "3030"),
		AllowedOrigins:    getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		CookieDomain:      getEnv("COOKIE_DOMAIN", ""),
		StytchProjectID:   getEnv("STYTCH_PROJECT_ID", ""),
		StytchSecret:      getEnv("STYTCH_SECRET", ""),
		StytchEnv:         getEnv("STYTCH_ENV", "test"),
		StytchRedirectURL: getEnv("STYTCH_REDIRECT_URL", "http://localhost:3030/api/auth/callback"),
		StytchBaseURL:     getEnv("STYTCH_BASE_URL", ""),
		BackendURL:        getEnv("BACKEND_URL", "http://localhost:3030"),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:3000"),
		CookieSecret:      cookieSecret,
		// Rate Limiting Configuration
		RateLimitIPMax:   envInt("RATE_LIMIT_IP_MAX", 300),
		RateLimitUserMax: envInt("RATE_LIMIT_USER_MAX", 60),
		RateLimitWindow:  envDuration("RATE_LIMIT_WINDOW", time.Minute),
		SessionTTL:       envDuration("SESSION_TTL", 30*24*time.Hour),

		// C5 device-bound session enforcement (opt-in; see field docs)
		EnforceDeviceFingerprint: enforceDeviceFingerprint,
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

// envBool parses key as a boolean, falling back when unset or invalid.
func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

// envInt parses key as a positive int64, falling back when unset or invalid.
func envInt(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

// envDuration parses key as a Go duration (e.g. "1m"), falling back when
// unset or invalid.
func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return fallback
}

// Module is an fx-compatible provider for Config.
var Module = fx.Provide(Load)
