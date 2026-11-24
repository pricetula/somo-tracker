// Package metrics provides Prometheus metrics for rate limiting.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BFF Auth Rejected Total tracks requests rejected at the BFF trust boundary.
	BFFAuthRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bff_auth_rejected_total",
			Help: "Total number of requests rejected by BFF auth middleware",
		},
		[]string{"reason"}, // missing_secret, invalid_secret, invalid_cidr, missing_client_ip
	)

	// RateLimitDecisionsTotal tracks all rate limit decisions.
	RateLimitDecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ratelimit_decisions_total",
			Help: "Total number of rate limit decisions",
		},
		[]string{"tier", "result"}, // tier: otp_request, otp_verify, onboard, session, global; result: allowed, denied, would_block
	)

	// RateLimit429Total tracks 429 responses returned.
	RateLimit429Total = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ratelimit_429_total",
			Help: "Total number of 429 responses returned",
		},
		[]string{"tier"},
	)

	// RateLimitCheckDurationSeconds tracks rate limit check latency.
	RateLimitCheckDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ratelimit_check_duration_seconds",
			Help:    "Duration of rate limit check in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"tier"},
	)

	// RateLimitRedisErrorsTotal tracks Redis errors during rate limiting.
	RateLimitRedisErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ratelimit_redis_errors_total",
			Help: "Total number of Redis errors during rate limiting",
		},
		[]string{"tier", "error"}, // error: timeout, connection, script
	)

	// RateLimitFailOpenTotal tracks fail-open events.
	RateLimitFailOpenTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ratelimit_failopen_total",
			Help: "Total number of fail-open events",
		},
		[]string{"tier"},
	)

	// RateLimitFailClosedTotal tracks fail-closed events.
	RateLimitFailClosedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ratelimit_failclosed_total",
			Help: "Total number of fail-closed events (503 returned)",
		},
		[]string{"tier"},
	)

	// OTPSendsTotal tracks OTP emails sent by budget pool.
	OTPSendsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "otp_sends_total",
			Help: "Total number of OTP emails sent",
		},
		[]string{"pool"}, // reserved, general
	)

	// OTPVerifyFailuresTotal tracks OTP verification failures.
	OTPVerifyFailuresTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "otp_verify_failures_total",
			Help: "Total number of OTP verification failures",
		},
		[]string{"reason"}, // expired, used, wrong_code, max_attempts, challenge_invalidated
	)

	// OTPChallengeInvalidatedTotal tracks challenge invalidations.
	OTPChallengeInvalidatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "otp_challenge_invalidated_total",
			Help: "Total number of OTP challenges invalidated",
		},
		[]string{"reason"}, // max_attempts, expiry, new_otp, success
	)

	// BotChallengeTotal tracks bot challenge lifecycle.
	BotChallengeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_challenge_total",
			Help: "Total number of bot challenges",
		},
		[]string{"provider", "result"}, // provider: turnstile, pow, webauthn; result: issued, solved, failed, expired
	)

	// BotRiskScore tracks risk score distribution.
	BotRiskScore = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "bot_risk_score",
			Help:    "Risk score distribution (0-100)",
			Buckets: []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 100},
		},
		[]string{"tier"}, // otp_request, onboard
	)

	// BotStepUpTotal tracks escalation events.
	BotStepUpTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bot_stepup_total",
			Help: "Total number of risk escalation events",
		},
		[]string{"level"}, // medium, high, critical
	)

	// BrevoBudgetUtilization tracks daily Brevo budget usage.
	BrevoBudgetUtilization = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "brevo_budget_utilization",
			Help: "Daily Brevo budget utilization percentage (0-1)",
		},
		[]string{"pool"}, // reserved, general
	)
)

// RecordRateLimitDecision records a rate limit decision.
func RecordRateLimitDecision(tier, result string) {
	RateLimitDecisionsTotal.WithLabelValues(tier, result).Inc()
}

// RecordRateLimit429 records a 429 response.
func RecordRateLimit429(tier string) {
	RateLimit429Total.WithLabelValues(tier).Inc()
}

// RecordRateLimitCheckDuration records the duration of a rate limit check.
func RecordRateLimitCheckDuration(tier string, durationSeconds float64) {
	RateLimitCheckDurationSeconds.WithLabelValues(tier).Observe(durationSeconds)
}

// RecordRateLimitRedisError records a Redis error.
func RecordRateLimitRedisError(tier, errType string) {
	RateLimitRedisErrorsTotal.WithLabelValues(tier, errType).Inc()
}

// RecordRateLimitFailOpen records a fail-open event.
func RecordRateLimitFailOpen(tier string) {
	RateLimitFailOpenTotal.WithLabelValues(tier).Inc()
}

// RecordRateLimitFailClosed records a fail-closed event.
func RecordRateLimitFailClosed(tier string) {
	RateLimitFailClosedTotal.WithLabelValues(tier).Inc()
}

// RecordOTPSend records an OTP email send.
func RecordOTPSend(pool string) {
	OTPSendsTotal.WithLabelValues(pool).Inc()
}

// RecordOTPVerifyFailure records an OTP verification failure.
func RecordOTPVerifyFailure(reason string) {
	OTPVerifyFailuresTotal.WithLabelValues(reason).Inc()
}

// RecordOTPChallengeInvalidated records a challenge invalidation.
func RecordOTPChallengeInvalidated(reason string) {
	OTPChallengeInvalidatedTotal.WithLabelValues(reason).Inc()
}

// RecordBotChallenge records a bot challenge event.
func RecordBotChallenge(provider, result string) {
	BotChallengeTotal.WithLabelValues(provider, result).Inc()
}

// RecordBotRiskScore records a risk score.
func RecordBotRiskScore(tier string, score float64) {
	BotRiskScore.WithLabelValues(tier).Observe(score)
}

// RecordBotStepUp records a step-up escalation.
func RecordBotStepUp(level string) {
	BotStepUpTotal.WithLabelValues(level).Inc()
}

// SetBrevoBudgetUtilization sets the Brevo budget utilization gauge.
func SetBrevoBudgetUtilization(pool string, utilization float64) {
	BrevoBudgetUtilization.WithLabelValues(pool).Set(utilization)
}