package config

import (
	"os"

	"go.uber.org/fx"
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
	FrontendURL       string
	CookieSecret      string // HMAC-SHA256 key for signing somo_role cookie
}

// Load reads configuration from environment variables with safe fallbacks.
func Load() Config {
	return Config{
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://somo_admin:somo_secure_password@postgres:5432/somotracker_dev?sslmode=disable"),
		RedisURL:          getEnv("REDIS_URL", "redis:6379"),
		AppEnv:            getEnv("APP_ENV", "development"),
		Port:              getEnv("PORT", "3030"),
		AllowedOrigins:    getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		CookieDomain:      getEnv("COOKIE_DOMAIN", "localhost"),
		StytchProjectID:   getEnv("STYTCH_PROJECT_ID", ""),
		StytchSecret:      getEnv("STYTCH_SECRET", ""),
		StytchEnv:         getEnv("STYTCH_ENV", "test"),
		StytchRedirectURL: getEnv("STYTCH_REDIRECT_URL", "http://localhost:3030/api/auth/callback"),
		StytchBaseURL:     getEnv("STYTCH_BASE_URL", ""),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:3000"),
		CookieSecret:      getEnv("COOKIE_SECRET", "dev-insecure-change-in-production"),
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
