package config

import (
	"net/url"
	"os"
	"strings"
)

// Config holds typed application configuration loaded from environment.
// Secrets are injected via Doppler, never from .env files (BE-033).
type Config struct {
	Env           string
	PostgresDSN   string
	RedisAddr     string
	ServerPort    string
	LogLevel      string
	StytchEnv     string
	StytchProjectID string
	StytchSecret  string
}

// Load reads configuration from environment variables with Doppler aliases.
func Load() (*Config, error) {
	env := firstNonEmpty("APP_ENV", "ENV", "development")

	postgresDSN := firstNonEmpty("DATABASE_URL", "POSTGRES_DSN", "")

	redisAddr := firstNonEmpty("REDIS_ADDR", "")
	if redisAddr == "" {
		if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
			redisAddr = parseRedisAddr(redisURL)
		}
	}
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	serverPort := firstNonEmpty("PORT", "SERVER_PORT", "8080")
	logLevel := firstNonEmpty("LOG_LEVEL", "info")
	stytchEnv := firstNonEmpty("STYTCH_ENV", "test")
	stytchProjectID := firstNonEmpty("STYTCH_PROJECT_ID", "")
	stytchSecret := firstNonEmpty("STYTCH_SECRET", "")

	return &Config{
		Env:            env,
		PostgresDSN:    postgresDSN,
		RedisAddr:      redisAddr,
		ServerPort:     serverPort,
		LogLevel:       logLevel,
		StytchEnv:      stytchEnv,
		StytchProjectID: stytchProjectID,
		StytchSecret:   stytchSecret,
	}, nil
}

func firstNonEmpty(keys ...string) string {
	for i, k := range keys {
		v := os.Getenv(k)
		if v != "" {
			return v
		}
		// last key may be default
		if i == len(keys)-1 {
			return v
		}
	}
	return ""
}

func parseRedisAddr(raw string) string {
	// Accept redis://host:port or redis://user:pass@host:port/db
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := u.Host
	if host == "" {
		return raw
	}
	// Strip possible userinfo if present without scheme parsing
	if strings.Contains(host, "@") {
		parts := strings.SplitN(host, "@", 2)
		host = parts[1]
	}
	return host
}
