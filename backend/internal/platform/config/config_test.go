package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSessionCookieName(t *testing.T) {
	tests := []struct {
		env        string
		wantCookie string
	}{
		{"production", "__Host-session_token"},
		{"development", "session_token_development"},
		{"staging", "session_token_staging"},
		{"test", "session_token_test"},
		{"", "__Host-session_token"},
	}

	for _, tc := range tests {
		t.Run(tc.env, func(t *testing.T) {
			cfg := &Config{Env: tc.env}
			got := cfg.SessionCookieName()
			require.Equal(t, tc.wantCookie, got)
		})
	}
}

func TestSessionCookieDomain(t *testing.T) {
	tests := []struct {
		env          string
		cookieDomain string
		wantDomain   string
	}{
		{"production", "example.com", ""},
		{"development", "localhost", "localhost"},
		{"staging", "staging.example.com", "staging.example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.env, func(t *testing.T) {
			cfg := &Config{Env: tc.env, CookieDomain: tc.cookieDomain}
			got := cfg.SessionCookieDomain()
			require.Equal(t, tc.wantDomain, got)
		})
	}
}

func TestValidate_ProductionRequiresISTEncryptionKey(t *testing.T) {
	cfg := &Config{
		Env:            "production",
		PostgresDSN:    "postgres://...",
		RedisAddr:      "redis:6379",
		StytchProjectID: "proj",
		StytchSecret:   "secret",
		EmailHMACKey:   "hmac-key",
		// ISTEncryptionKey missing
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "IST_ENCRYPTION_KEY")
}

func TestValidate_ProductionRequiresManagedRedis(t *testing.T) {
	cfg := &Config{
		Env:                "production",
		PostgresDSN:        "postgres://...",
		RedisAddr:          "localhost:6379",
		StytchProjectID:    "proj",
		StytchSecret:       "secret",
		EmailHMACKey:       "hmac-key",
		ISTEncryptionKey:   "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=",
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "managed Redis endpoint")
}

func TestValidate_ProductionRequiresEmailHMACKey(t *testing.T) {
	cfg := &Config{
		Env:                "production",
		PostgresDSN:        "postgres://...",
		RedisAddr:          "redis:6379",
		StytchProjectID:    "proj",
		StytchSecret:       "secret",
		// EmailHMACKey missing
		ISTEncryptionKey:   "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=",
	}

	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "EMAIL_HMAC_KEY")
}

func TestValidate_DevelopmentAllowsDefaults(t *testing.T) {
	cfg := &Config{
		Env:                "development",
		PostgresDSN:        "postgres://...",
		RedisAddr:          "localhost:6379",
		StytchProjectID:    "proj",
		StytchSecret:       "secret",
		// No EmailHMACKey, ISTEncryptionKey required in dev
	}

	err := cfg.Validate()
	require.NoError(t, err)
}

func TestLoad_FromEnvironment(t *testing.T) {
	// Save original env
	origEnv := os.Getenv("APP_ENV")
	origDSN := os.Getenv("DATABASE_URL")
	origRedis := os.Getenv("REDIS_ADDR")
	origPort := os.Getenv("PORT")
	origStytchEnv := os.Getenv("STYTCH_ENV")
	origProj := os.Getenv("STYTCH_PROJECT_ID")
	origSecret := os.Getenv("STYTCH_SECRET")

	// Set test env
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("DATABASE_URL", "postgres://test")
	_ = os.Setenv("REDIS_ADDR", "redis:6379")
	_ = os.Setenv("PORT", "9090")
	_ = os.Setenv("STYTCH_ENV", "test")
	_ = os.Setenv("STYTCH_PROJECT_ID", "test-proj")
	_ = os.Setenv("STYTCH_SECRET", "test-secret")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "test", cfg.Env)
	require.Equal(t, "postgres://test", cfg.PostgresDSN)
	require.Equal(t, "redis:6379", cfg.RedisAddr)
	require.Equal(t, "9090", cfg.ServerPort)
	require.Equal(t, "test", cfg.StytchEnv)
	require.Equal(t, "test-proj", cfg.StytchProjectID)
	require.Equal(t, "test-secret", cfg.StytchSecret)

	// Restore
	_ = os.Setenv("APP_ENV", origEnv)
	_ = os.Setenv("DATABASE_URL", origDSN)
	_ = os.Setenv("REDIS_ADDR", origRedis)
	_ = os.Setenv("PORT", origPort)
	_ = os.Setenv("STYTCH_ENV", origStytchEnv)
	_ = os.Setenv("STYTCH_PROJECT_ID", origProj)
	_ = os.Setenv("STYTCH_SECRET", origSecret)
}

func TestParseRedisAddr(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"redis://localhost:6379", "localhost:6379"},
		{"redis://user:pass@localhost:6379/0", "localhost:6379"},
		{"redis://:pass@localhost:6379", "localhost:6379"},
		{"localhost:6379", "localhost:6379"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseRedisAddr(tc.input)
			require.Equal(t, tc.expected, got)
		})
	}
}