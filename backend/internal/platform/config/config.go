package config

import (
	"os"
)

// Config holds typed application configuration loaded from environment.
// Secrets are injected via Doppler, never from .env files (BE-033).
type Config struct {
	Env        string
	PostgresDSN string
	RedisAddr   string
	ServerPort  string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	return &Config{
		Env:         getEnv("ENV", "development"),
		PostgresDSN: getEnv("POSTGRES_DSN", ""),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
	}, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
