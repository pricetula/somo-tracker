package config

import (
	"os"

	"go.uber.org/fx"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DatabaseURL    string
	RedisURL       string
	AppEnv         string
	Port           string
	AllowedOrigins string
}

// Load reads configuration from environment variables with safe fallbacks.
func Load() Config {
	return Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://somo_admin:somo_secure_password@postgres:5432/somotracker_dev?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis:6379"),
		AppEnv:         getEnv("APP_ENV", "development"),
		Port:           getEnv("PORT", "3030"),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
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
