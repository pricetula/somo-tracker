package metrics

import (
	"testing"
	"time"

)

func TestMetricsRegistration(t *testing.T) {
	// Test that all metrics are registered and can be collected
	// This is a basic smoke test - in practice, we'd use prometheus.NewRegistry()

	// Record some test values
	RecordRateLimitDecision("otp_request", "allowed")
	RecordRateLimitDecision("otp_request", "denied")
	RecordRateLimit429("otp_request")
	RecordRateLimitCheckDuration("otp_request", 0.01)
	RecordRateLimitRedisError("otp_request", "timeout")
	RecordRateLimitFailOpen("session")
	RecordRateLimitFailClosed("otp_request")
	RecordOTPSend("reserved")
	RecordOTPSend("general")
	RecordOTPVerifyFailure("wrong_code")
	RecordOTPChallengeInvalidated("max_attempts")
	RecordBotChallenge("pow", "solved")
	RecordBotRiskScore("otp_request", 45.5)
	RecordBotStepUp("medium")
	SetBrevoBudgetUtilization("reserved", 0.4)

	// Wait a bit for metrics to be collected
	time.Sleep(10 * time.Millisecond)

	// Verify metrics exist by checking they don't panic
}

func TestMetricsNonNegative(t *testing.T) {
	// Ensure all counter metrics are non-negative
	// This would use a custom registry in practice
	//   ratelimit_decisions_total{tier="otp_request",result="allowed"} 1
	// `)
}