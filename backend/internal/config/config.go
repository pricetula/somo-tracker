package config

import (
	"log/slog"
	"os"
	"path/filepath"
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
	CookieSecret      string // HMAC-SHA256 key for signing somo_role cookie
	RateLimitIPMax    string // Tier 1: Max requests per window per IP (e.g. 300)
	RateLimitUserMax  string // Tier 2: Max requests per window per User ID (e.g. 60)
	RateLimitWindow   string // Time window for rate limit (e.g. 1 * time.Minute)
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
				slog.Warn("config: logger sync failed", slog.String("error", syncErr.Error()))
			}
		}
	}

	return Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://somo_admin:somo_secure_password@somotracker_postgres:5432/somotracker_dev?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis:6379"),
		AppEnv:            getEnv("APP_ENV", "development"),
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
		CookieSecret:      getEnv("COOKIE_SECRET", "dev-insecure-change-in-production"),
		// Rate Limiting Configuration
		RateLimitIPMax:   getEnv("RATE_LIMIT_IP_MAX", "300"),                  // Tier 1: Max requests per window per IP (e.g. 300)
		RateLimitUserMax: getEnv("COOKIE_SECRET", "60"),                       // Tier 2: Max requests per window per User ID (e.g. 60)
		RateLimitWindow:  getEnv("COOKIE_SECRET", (time.Minute * 1).String()), // Time window for rate limit (e.g. 1 * time.Minute)
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

// Module is an fx-compatible provider for Config.
var Module = fx.Provide(Load)
