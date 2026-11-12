package config

import (
	"errors"
	"fmt"
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
	// Security
	EmailHMACKey  string
	ISTEncryptionKey string // 32-byte base64 key for AES-256-GCM encryption of IST
	CORSOrigins   string
	CookieDomain  string
	CSPReportOnlyEnabled bool
	CSPReportOnlyURI   string
	CSPReportOnlyPolicy string
	// Email provider quotas
	EmailProviderHourlyCap int
	EmailProviderDailyCap  int
	// Rate limits - configurable via env, defaults per spec
	RateLimitMagicLinkIPPerMin      int
	RateLimitMagicLinkEmailPer15Min int
	RateLimitTokenExchangeIPPerMin  int

	// Reference data seeding
	SeedReferenceData bool
	// Tenant-scoped rate limits
	RateLimitTenantMagicLinkIPPerMin      int
	RateLimitTenantMagicLinkEmailPer15Min int
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

	emailHMACKey := firstNonEmpty("EMAIL_HMAC_KEY", "")
	istEncryptionKey := firstNonEmpty("IST_ENCRYPTION_KEY", "")
	corsOrigins := firstNonEmpty("CORS_ORIGINS", "PUBLIC_APP_URL", "")
	cookieDomain := firstNonEmpty("COOKIE_DOMAIN", "")
	cspReportOnlyEnabled := firstNonEmpty("CSP_REPORT_ONLY_ENABLED", "false")
	cspReportOnlyURI := firstNonEmpty("CSP_REPORT_ONLY_URI", "")
	cspReportOnlyPolicy := firstNonEmpty("CSP_REPORT_ONLY_POLICY", "default-src 'self'")
	// Email provider quotas
	hourlyCap := parseIntEnv("EMAIL_PROVIDER_HOURLY_CAP", 1000)
	dailyCap := parseIntEnv("EMAIL_PROVIDER_DAILY_CAP", 10000)
	// Rate limits with defaults per spec
	ipPerMin := parseIntEnv("RATE_LIMIT_MAGIC_LINK_IP_PER_MIN", 10)
	emailPer15Min := parseIntEnv("RATE_LIMIT_MAGIC_LINK_EMAIL_PER_15_MIN", 3)
	tokenIPPerMin := parseIntEnv("RATE_LIMIT_TOKEN_EXCHANGE_IP_PER_MIN", 10)
	tenantIPPerMin := parseIntEnv("RATE_LIMIT_TENANT_MAGIC_LINK_IP_PER_MIN", 5)
	tenantEmailPer15Min := parseIntEnv("RATE_LIMIT_TENANT_MAGIC_LINK_EMAIL_PER_15_MIN", 2)

	// Reference data seeding - opt-in in production, on in dev/test
	seedRefData := firstNonEmpty("SEED_REFERENCE_DATA", "")
	seedReferenceData := false
	if seedRefData != "" {
		seedReferenceData = seedRefData == "true" || seedRefData == "1"
	} else if env == "development" || env == "test" {
		seedReferenceData = true
	}

	cfg := &Config{
		Env:            env,
		PostgresDSN:    postgresDSN,
		RedisAddr:      redisAddr,
		ServerPort:     serverPort,
		LogLevel:       logLevel,
		StytchEnv:      stytchEnv,
		StytchProjectID: stytchProjectID,
		StytchSecret:   stytchSecret,
		EmailHMACKey:   emailHMACKey,
		ISTEncryptionKey: istEncryptionKey,
		CORSOrigins:    corsOrigins,
		CookieDomain:   cookieDomain,
		CSPReportOnlyEnabled: cspReportOnlyEnabled == "true" || cspReportOnlyEnabled == "1",
		CSPReportOnlyURI: cspReportOnlyURI,
		CSPReportOnlyPolicy: cspReportOnlyPolicy,
		EmailProviderHourlyCap: hourlyCap,
		EmailProviderDailyCap:  dailyCap,
		RateLimitMagicLinkIPPerMin:      ipPerMin,
		RateLimitMagicLinkEmailPer15Min: emailPer15Min,
		RateLimitTokenExchangeIPPerMin:  tokenIPPerMin,
		RateLimitTenantMagicLinkIPPerMin:      tenantIPPerMin,
		RateLimitTenantMagicLinkEmailPer15Min: tenantEmailPer15Min,
		SeedReferenceData:                     seedReferenceData,
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks required configuration for the current environment.
func (c *Config) Validate() error {
	if c.Env != "development" && c.Env != "test" && c.Env != "production" {
		return errors.New("APP_ENV must be development, test, or production")
	}
	if c.PostgresDSN == "" {
		return errors.New("DATABASE_URL/POSTGRES_DSN is required")
	}
	if c.RedisAddr == "" {
		return errors.New("REDIS_ADDR/REDIS_URL is required")
	}
	if c.StytchProjectID == "" {
		return errors.New("STYTCH_PROJECT_ID is required")
	}
	if c.StytchSecret == "" {
		return errors.New("STYTCH_SECRET is required")
	}
	if c.Env == "production" {
		// Production must not use insecure defaults
		if c.RedisAddr == "localhost:6379" {
			return errors.New("REDIS_ADDR must be set to a managed Redis endpoint in production")
		}
		if c.EmailHMACKey == "" {
			return errors.New("EMAIL_HMAC_KEY is required in production")
		}
		if c.ISTEncryptionKey == "" {
			return errors.New("IST_ENCRYPTION_KEY is required in production (32-byte base64)")
		}
		// Cookie domain should not be set for __Host- prefix cookies
		// __Host- prefix ignores Domain attribute; warn in logs if set
		_ = c.CookieDomain // explicitly unused
	}
	return nil
}

// SessionCookieName returns the cookie name for the current environment.
// In production, uses __Host- prefix (requires Secure, Path=/, no Domain).
// In development/test, uses session_token_<env>.
func (c *Config) SessionCookieName() string {
	base := "__Host-session_token"
	if c.Env != "" && c.Env != "production" {
		return "session_token_" + c.Env
	}
	return base
}

// SessionCookieDomain returns the cookie domain.
// For __Host- prefix cookies, this must be empty (browser ignores Domain).
func (c *Config) SessionCookieDomain() string {
	if c.Env == "production" {
		return "" // __Host- prefix requires no Domain attribute
	}
	return c.CookieDomain
}

func parseIntEnv(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	var i int
	_, err := fmt.Sscanf(v, "%d", &i)
	if err != nil || i <= 0 {
		return defaultVal
	}
	return i
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