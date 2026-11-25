package config

import (
	"fmt"
	"os"
	"strconv"
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
	TurnstileSiteKey       string
	TurnstileExpectedHost  string

	// ============ PHASE 0: Trust Boundary & BFF Auth ============
	// BFF Service Secret (REQUIRED in production)
	BFFServiceSecret string
	// Trusted proxy hops for X-Forwarded-For parsing
	TrustedProxyHops int
	// Optional CIDR allowlist for BFF IPs
	TrustedBFFCIDRs string
	// HMAC secret for email keys in rate limiting
	RateLimitKeyHMACSecret string

	// ============ PHASE 1: Core Limiter ============
	RateLimitMode               string // shadow | enforce
	RateLimitRedisTimeoutMS     int    // Lua script timeout
	RateLimitModeOTPRequest     string // override per tier
	RateLimitModeOTPVerify      string
	RateLimitModeOnboard        string
	RateLimitModeSession        string
	RateLimitModeGlobal         string

	// ============ PHASE 2: OTP Hardening & Brevo Budget ============
	RateLimitOTPRequestEmail        int // per 15m
	RateLimitOTPRequestIP           int // per 15m
	RateLimitOTPSendBudgetPerDay    int // hard ceiling (250)
	RateLimitOTPReservedReturningPct int // % reserved for returning users (40)
	RateLimitOTPDomainVelocityPerHour int // per domain per hour (20)
	RateLimitReturningUserTTLDays   int // returning user marker TTL (30)
	RateLimitDisposableDomains      string // comma-separated

	OTPVerifyMaxAttemptsPerChallenge int // 5
	RateLimitOTPVerifyEmailPerDay    int // 20
	RateLimitOTPVerifyIPPerMin       int // 50

	// ============ PHASE 3: Bot Defense ============
	POWSecret                 string
	POWBaseDifficulty         int
	POWChallengeTTLSeconds    int
	BotRiskStepUpThreshold    int
	BotRiskBlockThreshold     int
	BotRiskCriticalThreshold  int
	BFFDwellCookieName        string
	BFFDwellSubHumanMS        int
	BFFDwellStaleMS           int
	BotHoneypotFieldPrefix    string
	RateLimitCriticalBudgetPct int

	// ============ PHASE 4: Global Middleware & Observability ============
	RateLimitOnboardUserPerMin  int
	RateLimitOnboardUserPerDay  int
	RateLimitOnboardIPPerMin    int
	RateLimitSessionUserPerMin  int
	RateLimitSessionIPPerMin    int
	RateLimitGlobalUserPerMin   int
	RateLimitGlobalIPPerMin     int
	RateLimitMetricsEnabled     bool
	RateLimitAlertsEnabled      bool

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

// firstNonEmptyInt returns the first non-empty environment variable parsed as int,
// or the default value if none are set or parsing fails.

// firstNonEmptyBool returns the first non-empty environment variable parsed as bool,
// or the default value if none are set or parsing fails.

// firstNonEmptyIntDev returns the first non-empty environment variable parsed as int,
// or the default value ONLY in non-production environments.
// In production, returns 0 (validation will catch missing required values).
func firstNonEmptyIntDev(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	if os.Getenv("APP_ENV") == "production" {
		return 0
	}
	return defaultVal
}

// firstNonEmptyStringDev returns the first non-empty environment variable,
// or the default value ONLY in non-production environments.
// In production, returns empty string (validation will catch missing required values).
func firstNonEmptyStringDev(key string, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if os.Getenv("APP_ENV") == "production" {
		return ""
	}
	return defaultVal
}

// firstNonEmptyBoolDev returns the first non-empty environment variable parsed as bool,
// or the default value ONLY in non-production environments.
// In production, returns false (validation will catch missing required values).
func firstNonEmptyBoolDev(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "1" || v == "true" || v == "TRUE"
	}
	if os.Getenv("APP_ENV") == "production" {
		return false
	}
	return defaultVal
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
	TurnstileSiteKey:       firstNonEmpty("TURNSTILE_SITE_KEY"),
		TurnstileExpectedHost: firstNonEmpty("TURNSTILE_EXPECTED_HOST"),

		// Phase 0
		BFFServiceSecret:       firstNonEmptyStringDev("BFF_SERVICE_SECRET", ""),
		TrustedProxyHops:       firstNonEmptyIntDev("TRUSTED_PROXY_HOPS", 1),
		TrustedBFFCIDRs:        firstNonEmptyStringDev("TRUSTED_BFF_CIDRS", ""),
		RateLimitKeyHMACSecret: firstNonEmptyStringDev("RATE_LIMIT_KEY_HMAC_SECRET", ""),

		// Phase 1
		RateLimitMode:           firstNonEmptyStringDev("RATE_LIMIT_MODE", "shadow"), // shadow | enforce
		RateLimitRedisTimeoutMS: firstNonEmptyIntDev("RATE_LIMIT_REDIS_TIMEOUT_MS", 50),
		RateLimitModeOTPRequest: firstNonEmptyStringDev("RATE_LIMIT_MODE_OTP_REQUEST", "shadow"),
		RateLimitModeOTPVerify:  firstNonEmptyStringDev("RATE_LIMIT_MODE_OTP_VERIFY", "shadow"),
		RateLimitModeOnboard:    firstNonEmptyStringDev("RATE_LIMIT_MODE_ONBOARD", "shadow"),
		RateLimitModeSession:    firstNonEmptyStringDev("RATE_LIMIT_MODE_SESSION", "shadow"),
		RateLimitModeGlobal:     firstNonEmptyStringDev("RATE_LIMIT_MODE_GLOBAL", "shadow"),

		// Phase 2
		RateLimitOTPRequestEmail:          firstNonEmptyIntDev("RATE_LIMIT_OTP_REQUEST_EMAIL", 3),
		RateLimitOTPRequestIP:             firstNonEmptyIntDev("RATE_LIMIT_OTP_REQUEST_IP", 30),
		RateLimitOTPSendBudgetPerDay:      firstNonEmptyIntDev("RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY", 250),
		RateLimitOTPReservedReturningPct:  firstNonEmptyIntDev("RATE_LIMIT_OTP_RESERVED_RETURNING_PCT", 40),
		RateLimitOTPDomainVelocityPerHour: firstNonEmptyIntDev("RATE_LIMIT_OTP_DOMAIN_VELOCITY_PER_HOUR", 20),
		RateLimitReturningUserTTLDays:     firstNonEmptyIntDev("RATE_LIMIT_RETURNING_USER_TTL_DAYS", 30),
		RateLimitDisposableDomains:        firstNonEmptyStringDev("RATE_LIMIT_DISPOSABLE_DOMAINS", ""),

		OTPVerifyMaxAttemptsPerChallenge: firstNonEmptyIntDev("OTP_VERIFY_MAX_ATTEMPTS_PER_CHALLENGE", 5),
		RateLimitOTPVerifyEmailPerDay:    firstNonEmptyIntDev("RATE_LIMIT_OTP_VERIFY_EMAIL_PER_DAY", 20),
		RateLimitOTPVerifyIPPerMin:       firstNonEmptyIntDev("RATE_LIMIT_OTP_VERIFY_IP_PER_MIN", 50),

		// Phase 3
		POWSecret:                 firstNonEmptyStringDev("POW_SECRET", ""),
		POWBaseDifficulty:         firstNonEmptyIntDev("POW_BASE_DIFFICULTY", 4),
		POWChallengeTTLSeconds:    firstNonEmptyIntDev("POW_CHALLENGE_TTL_SECONDS", 120),
		BotRiskStepUpThreshold:    firstNonEmptyIntDev("BOT_RISK_STEPUP_THRESHOLD", 50),
		BotRiskBlockThreshold:     firstNonEmptyIntDev("BOT_RISK_BLOCK_THRESHOLD", 85),
		BotRiskCriticalThreshold:  firstNonEmptyIntDev("BOT_RISK_CRITICAL_THRESHOLD", 95),
		BFFDwellCookieName:        firstNonEmptyStringDev("BFF_DWELL_COOKIE_NAME", "__Host-dwell_ts"),
		BFFDwellSubHumanMS:        firstNonEmptyIntDev("BFF_DWELL_SUB_HUMAN_MS", 800),
		BFFDwellStaleMS:           firstNonEmptyIntDev("BFF_DWELL_STALE_MS", 14400000),
		BotHoneypotFieldPrefix:    firstNonEmptyStringDev("BOT_HONEYPOT_FIELD_PREFIX", "hp_"),
		RateLimitCriticalBudgetPct: firstNonEmptyIntDev("RATE_LIMIT_CRITICAL_BUDGET_PCT", 90),

		// Phase 4
		RateLimitOnboardUserPerMin: firstNonEmptyIntDev("RATE_LIMIT_ONBOARD_USER_PER_MIN", 5),
		RateLimitOnboardUserPerDay: firstNonEmptyIntDev("RATE_LIMIT_ONBOARD_USER_PER_DAY", 20),
		RateLimitOnboardIPPerMin:   firstNonEmptyIntDev("RATE_LIMIT_ONBOARD_IP_PER_MIN", 50),
		RateLimitSessionUserPerMin: firstNonEmptyIntDev("RATE_LIMIT_SESSION_USER_PER_MIN", 30),
		RateLimitSessionIPPerMin:   firstNonEmptyIntDev("RATE_LIMIT_SESSION_IP_PER_MIN", 100),
		RateLimitGlobalUserPerMin:  firstNonEmptyIntDev("RATE_LIMIT_GLOBAL_USER_PER_MIN", 300),
		RateLimitGlobalIPPerMin:    firstNonEmptyIntDev("RATE_LIMIT_GLOBAL_IP_PER_MIN", 600),
		RateLimitMetricsEnabled:    firstNonEmptyBoolDev("RATE_LIMIT_METRICS_ENABLED", true),
		RateLimitAlertsEnabled:     firstNonEmptyBoolDev("RATE_LIMIT_ALERTS_ENABLED", true),

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

	// Phase 0: Required in production
	if c.AppEnv == "production" {
		if c.BFFServiceSecret == "" {
			missing = append(missing, "BFF_SERVICE_SECRET (required in production)")
		}
		if c.POWSecret == "" {
			missing = append(missing, "POW_SECRET (required in production)")
		}
		if c.RateLimitKeyHMACSecret == "" {
			missing = append(missing, "RATE_LIMIT_KEY_HMAC_SECRET (required in production)")
		}
		if c.TrustedProxyHops == 0 {
			missing = append(missing, "TRUSTED_PROXY_HOPS (required in production)")
		}
		if c.TurnstileSiteKey == "" {
			missing = append(missing, "TURNSTILE_SITE_KEY (required in production)")
		}
		if c.TurnstileExpectedHost == "" {
			missing = append(missing, "TURNSTILE_EXPECTED_HOST (required in production)")
		}
		if c.TurnstileSecretKey == "" {
			missing = append(missing, "TURNSTILE_SECRET_KEY (required in production)")
		}
	}

	// Phase 1: Required in production
	if c.AppEnv == "production" {
		if c.RateLimitRedisTimeoutMS == 0 {
			missing = append(missing, "RATE_LIMIT_REDIS_TIMEOUT_MS (required in production)")
		}
	}

	// Phase 2: Required in production
	if c.AppEnv == "production" {
		if c.RateLimitOTPRequestEmail == 0 {
			missing = append(missing, "RATE_LIMIT_OTP_REQUEST_EMAIL (required in production)")
		}
		if c.RateLimitOTPRequestIP == 0 {
			missing = append(missing, "RATE_LIMIT_OTP_REQUEST_IP (required in production)")
		}
		if c.RateLimitOTPSendBudgetPerDay == 0 {
			missing = append(missing, "RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY (required in production)")
		}
		if c.RateLimitOTPReservedReturningPct == 0 {
			missing = append(missing, "RATE_LIMIT_OTP_RESERVED_RETURNING_PCT (required in production)")
		}
		if c.RateLimitOTPDomainVelocityPerHour == 0 {
			missing = append(missing, "RATE_LIMIT_OTP_DOMAIN_VELOCITY_PER_HOUR (required in production)")
		}
		if c.RateLimitReturningUserTTLDays == 0 {
			missing = append(missing, "RATE_LIMIT_RETURNING_USER_TTL_DAYS (required in production)")
		}
		if c.OTPVerifyMaxAttemptsPerChallenge == 0 {
			missing = append(missing, "OTP_VERIFY_MAX_ATTEMPTS_PER_CHALLENGE (required in production)")
		}
		if c.RateLimitOTPVerifyEmailPerDay == 0 {
			missing = append(missing, "RATE_LIMIT_OTP_VERIFY_EMAIL_PER_DAY (required in production)")
		}
		if c.RateLimitOTPVerifyIPPerMin == 0 {
			missing = append(missing, "RATE_LIMIT_OTP_VERIFY_IP_PER_MIN (required in production)")
		}
	}

	// Phase 3: Required in production
	if c.AppEnv == "production" {
		if c.POWBaseDifficulty == 0 {
			missing = append(missing, "POW_BASE_DIFFICULTY (required in production)")
		}
		if c.POWChallengeTTLSeconds == 0 {
			missing = append(missing, "POW_CHALLENGE_TTL_SECONDS (required in production)")
		}
		if c.BotRiskStepUpThreshold == 0 {
			missing = append(missing, "BOT_RISK_STEPUP_THRESHOLD (required in production)")
		}
		if c.BotRiskBlockThreshold == 0 {
			missing = append(missing, "BOT_RISK_BLOCK_THRESHOLD (required in production)")
		}
		if c.BotRiskCriticalThreshold == 0 {
			missing = append(missing, "BOT_RISK_CRITICAL_THRESHOLD (required in production)")
		}
		if c.BFFDwellCookieName == "" {
			missing = append(missing, "BFF_DWELL_COOKIE_NAME (required in production)")
		}
		if c.BFFDwellSubHumanMS == 0 {
			missing = append(missing, "BFF_DWELL_SUB_HUMAN_MS (required in production)")
		}
		if c.BFFDwellStaleMS == 0 {
			missing = append(missing, "BFF_DWELL_STALE_MS (required in production)")
		}
		if c.BotHoneypotFieldPrefix == "" {
			missing = append(missing, "BOT_HONEYPOT_FIELD_PREFIX (required in production)")
		}
		if c.RateLimitCriticalBudgetPct == 0 {
			missing = append(missing, "RATE_LIMIT_CRITICAL_BUDGET_PCT (required in production)")
		}
	}

	// Phase 4: Required in production
	if c.AppEnv == "production" {
		if c.RateLimitOnboardUserPerMin == 0 {
			missing = append(missing, "RATE_LIMIT_ONBOARD_USER_PER_MIN (required in production)")
		}
		if c.RateLimitOnboardUserPerDay == 0 {
			missing = append(missing, "RATE_LIMIT_ONBOARD_USER_PER_DAY (required in production)")
		}
		if c.RateLimitOnboardIPPerMin == 0 {
			missing = append(missing, "RATE_LIMIT_ONBOARD_IP_PER_MIN (required in production)")
		}
		if c.RateLimitSessionUserPerMin == 0 {
			missing = append(missing, "RATE_LIMIT_SESSION_USER_PER_MIN (required in production)")
		}
		if c.RateLimitSessionIPPerMin == 0 {
			missing = append(missing, "RATE_LIMIT_SESSION_IP_PER_MIN (required in production)")
		}
		if c.RateLimitGlobalUserPerMin == 0 {
			missing = append(missing, "RATE_LIMIT_GLOBAL_USER_PER_MIN (required in production)")
		}
		if c.RateLimitGlobalIPPerMin == 0 {
			missing = append(missing, "RATE_LIMIT_GLOBAL_IP_PER_MIN (required in production)")
		}
	}

	// In production, session cookie MUST use __Host- prefix
	if c.AppEnv == "production" && !strings.HasPrefix(c.SessionCookieName, "__Host-") {
		missing = append(missing, "SESSION_COOKIE_NAME (must start with __Host- in production)")
	}

	// Validate numeric bounds
	if c.RateLimitOTPSendBudgetPerDay > 300 {
		missing = append(missing, "RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY must be <= 300 (Brevo free tier)")
	}
	if c.BotRiskStepUpThreshold >= c.BotRiskBlockThreshold || c.BotRiskBlockThreshold >= c.BotRiskCriticalThreshold {
		missing = append(missing, "Risk thresholds must be: STEPUP < BLOCK < CRITICAL")
	}
	if c.RateLimitRedisTimeoutMS < 10 || c.RateLimitRedisTimeoutMS > 500 {
		missing = append(missing, "RATE_LIMIT_REDIS_TIMEOUT_MS must be 10-500")
	}

	// Validate mode values
	validModes := map[string]bool{"": true, "shadow": true, "enforce": true}
	for _, m := range []string{c.RateLimitMode, c.RateLimitModeOTPRequest, c.RateLimitModeOTPVerify, c.RateLimitModeOnboard, c.RateLimitModeSession, c.RateLimitModeGlobal} {
		if !validModes[m] {
			missing = append(missing, fmt.Sprintf("Invalid rate limit mode: %s (must be shadow|enforce)", m))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing or invalid configuration: %s", strings.Join(missing, ", "))
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
