package handler

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"somotracker/internal/auth/service"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/botdefense"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
	"somotracker/internal/platform/middleware"
	"somotracker/internal/platform/ratelimit"
	"somotracker/internal/platform/turnstile"
)

// Handler is the HTTP handler for the auth domain.
type Handler struct {
	svc                  *service.Service
	cfg                  *config.Config
	log                  *logger.Logger
	turnstile            turnstile.Verifier
	rateLimiter          ratelimit.Limiter
	keyBuilder           ratelimit.KeyBuilder
	botDefense           *botdefense.BotDefenseManager
	sessionCookieName    string
	onboardingCookieName string
}

// New creates an auth handler.
func New(svc *service.Service, cfg *config.Config, log *logger.Logger, rateLimiter ratelimit.Limiter, keyBuilder ratelimit.KeyBuilder) *Handler {
	sessionCookieName := cfg.SessionCookieName
	// Derive onboarding cookie name from session cookie name
	onboardingCookieName := deriveOnboardingCookieName(sessionCookieName)

	// Turnstile verifier for OTP request protection
	var ts turnstile.Verifier
	if cfg.TurnstileSecretKey != "" {
		ts = turnstile.New(turnstile.Config{
			SecretKey:    cfg.TurnstileSecretKey,
			SiteKey:      cfg.TurnstileSiteKey,
			ExpectedHost: cfg.TurnstileExpectedHost,
		})
	}

	// Bot defense manager (redisClient will be nil initially, set via app if needed)
	botDefense := botdefense.NewBotDefenseManagerWithSiteKey(
		nil, // redisClient - set by app if needed
		cfg.POWSecret,
		cfg.POWBaseDifficulty,
		cfg.POWChallengeTTLSeconds,
		ts,
		cfg.TurnstileSiteKey,
		cfg.BFFDwellCookieName,
		cfg.RateLimitKeyHMACSecret, // reuse HMAC secret for dwell cookie
		cfg.BFFDwellSubHumanMS,
		cfg.BFFDwellStaleMS,
		cfg.BotHoneypotFieldPrefix,
		cfg.BotRiskStepUpThreshold,
		cfg.BotRiskBlockThreshold,
		cfg.BotRiskCriticalThreshold,
		cfg.RateLimitCriticalBudgetPct,
	)

	log.Info(context.Background(), "auth handler initialized",
		"session_cookie", sessionCookieName,
		"onboarding_cookie", onboardingCookieName,
		"app_env", cfg.AppEnv,
		"is_production", cfg.IsProduction(),
		"turnstile_enabled", ts != nil,
		"rate_limiter_enabled", rateLimiter != nil,
		"bot_defense_enabled", true,
	)

	return &Handler{
		svc:                  svc,
		cfg:                  cfg,
		log:                  log,
		turnstile:            ts,
		rateLimiter:          rateLimiter,
		keyBuilder:           keyBuilder,
		botDefense:           botDefense,
		sessionCookieName:    sessionCookieName,
		onboardingCookieName: onboardingCookieName,
	}
}

// SetRedis sets the Redis client for bot defense (called from app initialization).
func (h *Handler) SetRedis(redisClient interface{}) {
	_ = redisClient
}

// deriveOnboardingCookieName derives the onboarding cookie name from session cookie name.
// If session cookie starts with "__Host-", onboarding cookie must also start with "__Host-".
func deriveOnboardingCookieName(sessionCookieName string) string {
	if strings.HasPrefix(sessionCookieName, "__Host-") {
		return "__Host-needs_onboarding"
	}
	return "needs_onboarding"
}

// RequestOTPRequest is the request body for POST /api/v1/auth/otp/request.
type RequestOTPRequest struct {
	Email          string `json:"email" validate:"required,email"`
	TurnstileToken string `json:"turnstile_token" validate:"required"`
	PowSolution    *PowSolution `json:"pow_solution,omitempty"`
	HoneypotField  string `json:"-"` // Hidden field, not in JSON
}

// PowSolution represents a Proof-of-Work solution.
type PowSolution struct {
	ChallengeID string `json:"challenge_id"`
	Nonce       string `json:"nonce"`
}

// RequestOTPResponse is the response body.
type RequestOTPResponse struct {
	OK           bool   `json:"ok"`
	Message      string `json:"message"`
	ChallengeID  string `json:"challenge_id"`
}

// VerifyOTPRequest is the request body for POST /api/v1/auth/otp/verify.
type VerifyOTPRequest struct {
	ChallengeID string `json:"challenge_id" validate:"required,uuid"`
	Code        string `json:"code" validate:"required,len=6"`
}

// VerifyOTPResponse is the response body.
// SessionToken is NOT returned to the browser - it's only set via HttpOnly cookie.
type VerifyOTPResponse struct {
	OK              bool   `json:"ok"`
	Message         string `json:"message"`
	NeedsOnboarding bool   `json:"needs_onboarding"`
}

// ChallengeRequest is the request body for POST /api/v1/auth/challenge.
type ChallengeRequest struct {
	Action       string `json:"action" validate:"required,oneof=otp_request onboard"`
	EmailHash    string `json:"email_hash" validate:"required"`
	ClientSubnet string `json:"client_subnet" validate:"required"`
}

// ChallengeResponse is the response body for challenge issuance.
type ChallengeResponse struct {
	ChallengeID string `json:"challenge_id"`
	Type        string `json:"type"` // "pow" | "turnstile"
	Difficulty  int    `json:"difficulty,omitempty"`
	Seed        string `json:"seed,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	SiteKey     string `json:"site_key,omitempty"`
	Action      string `json:"action,omitempty"`
}

// OnboardRequest is the request body for POST /api/v1/auth/onboard.
type OnboardRequest struct {
	OrganizationName   string         `json:"organization_name" validate:"required"`
	SchoolName         string         `json:"school_name" validate:"required"`
	UserName           string         `json:"user_name" validate:"required"`
	CountryID          string       `json:"country_id,omitempty" validate:"omitempty,uuid"`
	EducationSystemID  string       `json:"education_system_id,omitempty" validate:"omitempty,uuid"`
	PowSolution        *PowSolution   `json:"pow_solution,omitempty"`
	HoneypotField      string         `json:"-"`
}

// SessionResponse is the response body for GET /api/v1/auth/session.
type SessionResponse struct {
	OK              bool   `json:"ok"`
	Authenticated   bool   `json:"authenticated"`
	SessionType     string `json:"session_type,omitempty"`
	NeedsOnboarding bool   `json:"needs_onboarding,omitempty"`
}

// @Summary      Request OTP
// @Description  Request a verification code via email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RequestOTPRequest true "OTP request"
// @Success      200 {object} RequestOTPResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      429 {object} map[string]interface{}
// @Router       /api/v1/auth/otp/request [post]
func (h *Handler) RequestOTP(c fiber.Ctx) error {
	var req RequestOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	normalizedEmail := ratelimit.NormalizeEmail(req.Email)

	// Rate limit per email (before any OTP generation or DB access).
	if h.rateLimiter != nil {
		emailKey := h.keyBuilder.EmailKey(normalizedEmail)
		decision, err := h.rateLimiter.Allow(c.Context(), emailKey, h.cfg.RateLimitOTPRequestEmail, 15*time.Minute)
		if err != nil {
			h.log.Error(c.Context(), err, "email rate limit check failed", "email", normalizedEmail)
			return apperr.New(apperr.INTERNAL_ERROR, "rate limit check failed")
		}
		if !decision.Allowed {
			h.setRateLimitHeaders(c, decision)
			return apperr.New(apperr.RATE_LIMITED, "too many requests for this email")
		}
		h.setRateLimitHeaders(c, decision)
	}

	// Rate limit per IP (from BFF via X-Client-IP header).
	if h.rateLimiter != nil {
		clientIP := middleware.ClientIPFromContext(c)
		if clientIP == "" {
			// Fallback for direct requests (should not happen with BFF auth)
			remoteIP := c.Get("CF-Connecting-IP")
			if remoteIP == "" {
				remoteIP = c.IP()
			}
			clientIP = ratelimit.NormalizeIP(remoteIP)
		}
		ipKey := h.keyBuilder.IPKey(clientIP)
		decision, err := h.rateLimiter.Allow(c.Context(), ipKey, h.cfg.RateLimitOTPRequestIP, 15*time.Minute)
		if err != nil {
			h.log.Error(c.Context(), err, "IP rate limit check failed", "ip", clientIP)
			return apperr.New(apperr.INTERNAL_ERROR, "rate limit check failed")
		}
		if !decision.Allowed {
			h.setRateLimitHeaders(c, decision)
			return apperr.New(apperr.RATE_LIMITED, "too many requests from this IP")
		}
		h.setRateLimitHeaders(c, decision)
	}

	// Global OTP send budget check (250/day with 40% reserved for returning users)
	if h.rateLimiter != nil {
		globalKey := h.keyBuilder.GlobalKey()
		decision, err := h.rateLimiter.Allow(c.Context(), globalKey, h.cfg.RateLimitOTPSendBudgetPerDay, 24*time.Hour)
		if err != nil {
			h.log.Error(c.Context(), err, "global OTP budget check failed")
			return apperr.New(apperr.INTERNAL_ERROR, "rate limit check failed")
		}
		if !decision.Allowed {
			h.setRateLimitHeaders(c, decision)
			return apperr.New(apperr.RATE_LIMITED, "daily OTP limit exceeded")
		}
		h.setRateLimitHeaders(c, decision)
	}

	// Domain velocity check (20 requests per hour per domain)
	if h.rateLimiter != nil {
		atIdx := strings.Index(normalizedEmail, "@")
		if atIdx > 0 {
			domain := normalizedEmail[atIdx+1:]
			domainKey := h.keyBuilder.DomainKey(domain)
			decision, err := h.rateLimiter.Allow(c.Context(), domainKey, h.cfg.RateLimitOTPDomainVelocityPerHour, time.Hour)
			if err != nil {
				h.log.Error(c.Context(), err, "domain velocity check failed", "domain", domain)
			} else if !decision.Allowed {
				h.setRateLimitHeaders(c, decision)
				// Don't block, just flag for step-up challenge in bot defense
				c.Locals("domainVelocityExceeded", true)
			} else {
				h.setRateLimitHeaders(c, decision)
			}
		}
	}

	// Disposable domain check
	if h.cfg.RateLimitDisposableDomains != "" {
		atIdx := strings.Index(normalizedEmail, "@")
		if atIdx > 0 {
			domain := normalizedEmail[atIdx+1:]
			disposable := strings.Split(h.cfg.RateLimitDisposableDomains, ",")
			for _, d := range disposable {
				d = strings.TrimSpace(strings.ToLower(d))
				if d == domain {
					c.Locals("disposableDomain", true)
					break
				}
			}
		}
	}

	// Honeypot validation
	formData := make(map[string]string)
	if err := c.Bind().Body(&formData); err == nil { // Parse form data for honeypot check
		if h.botDefense != nil && h.botDefense.HoneypotVal() != nil {
			if h.botDefense.HoneypotVal().Validate(formData) {
				h.log.Warn("honeypot triggered", "email", normalizedEmail)
				c.Locals("honeypotTriggered", true)
			}
		}
	}

	// Dwell time validation (from BFF cookie)
	dwellTimeMs := int64(0)
	if h.botDefense != nil && h.botDefense.DwellValidator() != nil {
		cookieName := h.cfg.BFFDwellCookieName
		cookieVal := c.Cookies(cookieName)
		if cookieVal != "" {
			if d, sub, stale, err := h.botDefense.DwellValidator().ValidateCookie(cookieVal); err == nil {
				dwellTimeMs = d
				_ = sub
				_ = stale
			}
		}
	}

	// Require a Turnstile token before doing anything else.
	if req.TurnstileToken == "" {
		return apperr.New(apperr.INVALID_REQUEST, "turnstile token required")
	}

	// Verify the Turnstile token with Cloudflare before generating an OTP,
	// touching the DB, or sending any email.
	if h.turnstile != nil {
		// Use CF-Connecting-IP when behind Cloudflare, otherwise fall back to
		// the connection's remote IP.
		remoteIP := c.Get("CF-Connecting-IP")
		if remoteIP == "" {
			remoteIP = c.IP()
		}

		result, err := h.turnstile.Verify(c.Context(), req.TurnstileToken, remoteIP)
		if err != nil {
			if errors.Is(err, turnstile.ErrUpstream) {
				return apperr.New(apperr.BAD_GATEWAY, "turnstile verification unavailable")
			}
			// Verification failed (success: false or hostname mismatch).
			// Log error codes for debugging but do not return them to the client.
			h.log.Error(c.Context(), err, "turnstile verification failed",
				"error_codes", result.ErrorCodes,
				"hostname", result.Hostname,
			)
			return apperr.New(apperr.FORBIDDEN, "turnstile verification failed")
		}
	}

	// PoW verification if provided
	powSolved := false
	if req.PowSolution != nil && h.botDefense != nil {
		verifyReq := botdefense.VerifyRequest{
			ChallengeID:  req.PowSolution.ChallengeID,
			Type:         "pow",
			Solution:     botdefense.PowSolution{Nonce: req.PowSolution.Nonce},
			EmailHash:    h.keyBuilder.EmailKey(normalizedEmail),
			ClientSubnet: middleware.ClientIPFromContext(c),
		}
		powResult, _ := h.botDefense.VerifyChallenge(c.Context(), verifyReq)
		powSolved = powResult.Valid
		if !powResult.Valid && req.PowSolution != nil {
			h.log.Warn("pow verification failed", "error", powResult.Error)
		}
	}

	// Build baseline signals for risk scoring
	signals := botdefense.BaselineSignals{
		TurnstileSuccess:  h.turnstile != nil, // Simplified - actual result would be passed
		DwellTimeMs:       dwellTimeMs,
		HoneypotTriggered: c.Locals("honeypotTriggered") != nil,
		PoWSolved:         powSolved,
		DisposableDomain:  c.Locals("disposableDomain") != nil,
		UnseenDomain:      false, // Would need domain allowlist
		NoPriorSession:    false, // Would need session check
	}

	// Assess risk if bot defense is available
	if h.botDefense != nil {
		riskScore, riskLevel, _ := h.botDefense.AssessRisk(c.Context(), signals)

		// Handle escalation based on risk level
		switch riskLevel {
		case botdefense.RiskLevelCritical:
			h.log.Warn("critical risk level - blocking request", "score", riskScore, "email", normalizedEmail)
			return apperr.New(apperr.RATE_LIMITED, "request blocked due to high risk")
		case botdefense.RiskLevelHigh:
			// Require WebAuthn or defer - for now return challenge required
			h.log.Warn("high risk level - step-up required", "score", riskScore)
			return c.Status(http.StatusPreconditionRequired).JSON(fiber.Map{
				"code": "challenge_required",
				"message": "Additional verification required",
				"challenge": fiber.Map{
					"type": "webauthn",
					"message": "Please use your passkey to continue",
				},
			})
		case botdefense.RiskLevelMedium:
			// Increase PoW difficulty or require interactive challenge
			h.log.Info(c.Context(), "medium risk level - increased PoW", "score", riskScore)
			c.Locals("increasedPoW", true)
		}

		c.Locals("riskScore", riskScore)
		c.Locals("riskLevel", string(riskLevel))
	}

	resp, err := h.svc.RequestOTP(c.Context(), service.RequestOTPRequest{
		Email: req.Email,
	})
	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// @Summary      Issue challenge
// @Description  Issue PoW/Turnstile challenge
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ChallengeRequest true "Challenge request"
// @Success      200 {object} ChallengeResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /api/v1/auth/challenge [post]
func (h *Handler) IssueChallenge(c fiber.Ctx) error {
	var req ChallengeRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	if h.botDefense == nil {
		return apperr.New(apperr.INTERNAL_ERROR, "bot defense not configured")
	}

	challengeReq := botdefense.ChallengeRequest{
		Action:       req.Action,
		EmailHash:    req.EmailHash,
		ClientSubnet: req.ClientSubnet,
		RiskScore:    0, // Would be computed from signals
		ReturningUser: false,
	}

	resp, err := h.botDefense.IssueChallenge(c.Context(), challengeReq)
	if err != nil {
		h.log.Error(c.Context(), err, "issue challenge failed")
		return apperr.New(apperr.INTERNAL_ERROR, "failed to issue challenge")
	}

	return c.Status(http.StatusOK).JSON(ChallengeResponse{
		ChallengeID: resp.ChallengeID,
		Type:        resp.Type,
		Difficulty:  resp.Difficulty,
		Seed:        resp.Seed,
		ExpiresAt:   resp.ExpiresAt.Format(time.RFC3339),
		SiteKey:     resp.SiteKey,
		Action:      resp.Action,
	})
}

// @Summary      Verify OTP
// @Description  Verify a 6-digit OTP code and create a session
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body VerifyOTPRequest true "OTP verification"
// @Success      200 {object} VerifyOTPResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      429 {object} map[string]interface{}
// @Router       /api/v1/auth/otp/verify [post]
func (h *Handler) VerifyOTP(c fiber.Ctx) error {
	var req VerifyOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid challenge ID")
	}

	// Rate limit per challenge (max 5 attempts per challenge)
	if h.rateLimiter != nil {
		challengeKey := h.keyBuilder.ChallengeKey(req.ChallengeID)
		decision, err := h.rateLimiter.Allow(c.Context(), challengeKey, h.cfg.OTPVerifyMaxAttemptsPerChallenge, 10*time.Minute)
		if err != nil {
			h.log.Error(c.Context(), err, "challenge rate limit check failed", "challenge_id", req.ChallengeID)
			return apperr.New(apperr.INTERNAL_ERROR, "rate limit check failed")
		}
		if !decision.Allowed {
			h.setRateLimitHeaders(c, decision)
			return apperr.New(apperr.RATE_LIMITED, "max attempts exceeded for this challenge")
		}
		h.setRateLimitHeaders(c, decision)
	}

	resp, err := h.svc.VerifyOTP(c.Context(), service.VerifyOTPRequest{
		ChallengeID: challengeID,
		Code:        req.Code,
	})
	if err != nil {
		return err
	}

	// If verification succeeded and a session token was returned, set the cookie
	if resp.OK && resp.SessionToken != "" {
		sessionTTL := h.cfg.SessionAbsoluteTimeout()
		if resp.NeedsOnboarding {
			sessionTTL = 30 * time.Minute
			// Set onboarding flag cookie so frontend proxy can route without
			// calling the backend. HttpOnly so JS can't read it, but proxy.ts
			// (server-side) can.
			h.setOnboardingCookie(c, "true", int(sessionTTL.Seconds()))
		}
		h.setSessionCookie(c, resp.SessionToken, time.Now().Add(sessionTTL))
	}

	// Return response WITHOUT session_token - it's in the HttpOnly cookie
	return c.Status(http.StatusOK).JSON(VerifyOTPResponse{
		OK:              resp.OK,
		Message:         resp.Message,
		NeedsOnboarding: resp.NeedsOnboarding,
	})
}

// @Summary      Onboard user
// @Description  Complete organization onboarding and upgrade session
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body OnboardRequest true "Onboard request"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      429 {object} map[string]interface{}
// @Router       /api/v1/auth/onboard [post]
func (h *Handler) Onboard(c fiber.Ctx) error {
	var req OnboardRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Read session token from cookie
	sessionToken := c.Cookies(h.sessionCookieName)
	if sessionToken == "" {
		return apperr.New(apperr.UNAUTHENTICATED, "session required")
	}

	// Hash the token to look up the session
	tokenHash := service.HashToken(sessionToken)

	session, err := h.svc.Store().GetSessionByTokenHash(c.Context(), tokenHash)
	if err != nil {
		return apperr.New(apperr.UNAUTHENTICATED, "invalid session")
	}

	if session.SessionType != "temporary" {
		return apperr.New(apperr.INVALID_REQUEST, "session is not temporary")
	}

	// Rate limit per user (onboard is expensive)
	if h.rateLimiter != nil {
		userKey := h.keyBuilder.UserKey(session.UserID.String())
		decision, err := h.rateLimiter.Allow(c.Context(), userKey, h.cfg.RateLimitOnboardUserPerMin, time.Minute)
		if err != nil {
			h.log.Error(c.Context(), err, "onboard rate limit check failed", "user_id", session.UserID)
			return apperr.New(apperr.INTERNAL_ERROR, "rate limit check failed")
		}
		if !decision.Allowed {
			h.setRateLimitHeaders(c, decision)
			return apperr.New(apperr.RATE_LIMITED, "too many onboarding attempts")
		}
		h.setRateLimitHeaders(c, decision)
	}

	// Complete onboarding
	var countryID, educationSystemID uuid.NullUUID
	if req.CountryID != "" {
		if parsed, err := uuid.Parse(req.CountryID); err == nil {
			countryID = uuid.NullUUID{UUID: parsed, Valid: true}
		}
	}
	if req.EducationSystemID != "" {
		if parsed, err := uuid.Parse(req.EducationSystemID); err == nil {
			educationSystemID = uuid.NullUUID{UUID: parsed, Valid: true}
		}
	}
	if err := h.svc.Onboard(c.Context(), session.UserID, session.OrganizationID, req.OrganizationName, req.SchoolName, req.UserName, countryID, educationSystemID); err != nil {
		return err
	}

	// Clear onboarding flag cookie
	h.clearOnboardingCookie(c)

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"ok":               true,
		"message":          "Onboarding complete",
		"needs_onboarding": false,
	})
}

// @Summary      Get session
// @Description  Get current session info
// @Tags         auth
// @Produce      json
// @Success      200 {object} SessionResponse
// @Failure      429 {object} map[string]interface{}
// @Router       /api/v1/auth/session [get]
func (h *Handler) Session(c fiber.Ctx) error {
	// Rate limit per user
	if h.rateLimiter != nil {
		clientIP := middleware.ClientIPFromContext(c)
		if clientIP == "" {
			clientIP = ratelimit.NormalizeIP(c.IP())
		}

		var decision ratelimit.Decision
		var err error

		// Try userID first (from session), fallback to IP
		sessionToken := c.Cookies(h.sessionCookieName)
		if sessionToken != "" {
			tokenHash := service.HashToken(sessionToken)
			if sess, err := h.svc.Store().GetSessionByTokenHash(c.Context(), tokenHash); err == nil {
				userKey := h.keyBuilder.UserKey(sess.UserID.String())
				_, _ = h.rateLimiter.Allow(c.Context(), userKey, h.cfg.RateLimitSessionUserPerMin, time.Minute)
			}
		}

		// Fallback to IP
		if decision.Limit == 0 {
			ipKey := h.keyBuilder.IPKey(clientIP)
			decision, err = h.rateLimiter.Allow(c.Context(), ipKey, h.cfg.RateLimitSessionIPPerMin, time.Minute)
		}

		if err != nil {
			h.log.Error(c.Context(), err, "session rate limit check failed")
			return apperr.New(apperr.INTERNAL_ERROR, "rate limit check failed")
		}
		if !decision.Allowed {
			h.setRateLimitHeaders(c, decision)
			return apperr.New(apperr.RATE_LIMITED, "too many session requests")
		}
		h.setRateLimitHeaders(c, decision)
	}

	sessionToken := c.Cookies(h.sessionCookieName)
	if sessionToken == "" {
		return c.Status(http.StatusOK).JSON(SessionResponse{
			OK:           true,
			Authenticated: false,
		})
	}

	tokenHash := service.HashToken(sessionToken)

	session, err := h.svc.Store().GetSessionByTokenHash(c.Context(), tokenHash)
	if err != nil {
		return c.Status(http.StatusOK).JSON(SessionResponse{
			OK:           true,
			Authenticated: false,
		})
	}

	// Determine if onboarding is needed based on session type
	needsOnboarding := session.SessionType == "temporary"

	return c.Status(http.StatusOK).JSON(SessionResponse{
		OK:               true,
		Authenticated:    true,
		SessionType:      session.SessionType,
		NeedsOnboarding:  needsOnboarding,
	})
}

// @Summary      Logout
// @Description  Log out and clear session cookies
// @Tags         auth
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /api/v1/auth/logout [post]
func (h *Handler) Logout(c fiber.Ctx) error {
	sessionToken := c.Cookies(h.sessionCookieName)
	if sessionToken != "" {
		tokenHash := sha256.Sum256([]byte(sessionToken))
		_ = tokenHash // TODO: revoke session in service
	}
	h.clearSessionCookie(c)
	h.clearOnboardingCookie(c)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"ok":      true,
		"message": "logged out",
	})
}

// RegisterRoutes registers auth routes under /api/v1/auth.
func (h *Handler) RegisterRoutes(auth fiber.Router) {
	auth.Post("/otp/request", h.RequestOTP)
	auth.Post("/otp/verify", h.VerifyOTP)
	auth.Post("/onboard", h.Onboard)
	auth.Get("/session", h.Session)
	auth.Post("/logout", h.Logout)
	auth.Post("/challenge", h.IssueChallenge) // New: PoW/Turnstile challenge endpoint
}

// secureCookie returns true if the cookie should be Secure.
// In production (APP_ENV=production), always true because TLS is terminated at load balancer.
// In dev, false for HTTP localhost.
func (h *Handler) secureCookie() bool {
	return h.cfg.IsProduction()
}

// setSessionCookie sets the session cookie on the response.
func (h *Handler) setSessionCookie(c fiber.Ctx, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.Cookie(&fiber.Cookie{
		Name:     h.sessionCookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secureCookie(),
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   maxAge,
		// NO Domain attribute - __Host- cookies forbid it
	})
}

// clearSessionCookie clears the session cookie.
func (h *Handler) clearSessionCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     h.sessionCookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secureCookie(),
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   -1,
		// NO Domain attribute
	})
}

// setOnboardingCookie sets the needs_onboarding flag cookie.
func (h *Handler) setOnboardingCookie(c fiber.Ctx, value string, maxAgeSeconds int) {
	c.Cookie(&fiber.Cookie{
		Name:     h.onboardingCookieName,
		Value:    value,
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secureCookie(),
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   maxAgeSeconds,
		// NO Domain attribute
	})
}

// clearOnboardingCookie removes the needs_onboarding flag cookie.
func (h *Handler) clearOnboardingCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     h.onboardingCookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secureCookie(),
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   -1,
		// NO Domain attribute
	})
}

// setRateLimitHeaders sets standard rate limit headers on the response.
func (h *Handler) setRateLimitHeaders(c fiber.Ctx, d ratelimit.Decision) {
	c.Set("RateLimit-Limit", fmt.Sprintf("%d", d.Limit))
	c.Set("RateLimit-Remaining", fmt.Sprintf("%d", d.Remaining))
	c.Set("RateLimit-Reset", fmt.Sprintf("%d", int(d.ResetAfter.Seconds())))
	c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", d.Limit))
	c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", d.Remaining))
	c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(d.ResetAfter).Unix()))
	if d.RetryAfter > 0 {
		c.Set("Retry-After", fmt.Sprintf("%d", int(d.RetryAfter.Seconds())))
	}
}