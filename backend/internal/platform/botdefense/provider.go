// Package botdefense provides adaptive bot defense with pluggable challenge providers.
package botdefense

import (
	"context"
	"time"
)

// ChallengeProvider is the pluggable bot defense interface.
type ChallengeProvider interface {
	// Name returns the provider identifier (e.g., "turnstile", "pow", "webauthn").
	Name() string
	// IssueChallenge generates a new challenge for the given action and context.
	IssueChallenge(ctx context.Context, req ChallengeRequest) (ChallengeResponse, error)
	// VerifyChallenge validates a submitted challenge solution.
	VerifyChallenge(ctx context.Context, req VerifyRequest) (VerifyResult, error)
	// RiskScore computes a 0-100 risk score from baseline signals.
	RiskScore(ctx context.Context, signals BaselineSignals) (int, error)
}

// ChallengeRequest represents a request to issue a new challenge.
type ChallengeRequest struct {
	Action        string // "otp_request" | "onboard"
	EmailHash     string // HMAC-SHA256 of normalized email
	ClientSubnet  string // Normalized client IP (/32 IPv4, /64 IPv6)
	RiskScore     int    // Current risk score (0-100)
	ReturningUser bool   // Whether user has returning user marker
}

// ChallengeResponse represents a newly issued challenge.
type ChallengeResponse struct {
	ChallengeID   string
	Type          string // "pow" | "turnstile" | "webauthn"
	Difficulty    int    // For PoW: leading zero bits
	Seed          string // Base64 encoded seed for PoW
	ExpiresAt     time.Time
	SiteKey       string // For Turnstile
	Action        string // For Turnstile/WebAuthn
	RPID          string // For WebAuthn
	AllowCredentials []WebAuthnCredential // For WebAuthn
}

// VerifyRequest represents a challenge verification request.
type VerifyRequest struct {
	ChallengeID string
	Type        string // "pow" | "turnstile" | "webauthn"
	Solution    interface{} // PoW: PowSolution, Turnstile: token, WebAuthn: credential
	EmailHash   string
	ClientSubnet string
	Action      string // "otp_request" | "onboard"
}

// VerifyResult represents the result of challenge verification.
type VerifyResult struct {
	Valid       bool
	Error       string
	NewRiskScore int
}

// PowSolution represents a Proof-of-Work solution.
type PowSolution struct {
	Nonce string
}

// BaselineSignals collected from BFF and request.
type BaselineSignals struct {
	TurnstileSuccess      bool    // Turnstile verification result
	TurnstileAction       string  // "otp_request" | "onboard"
	DwellTimeMs           int64   // From BFF pre-session cookie (X-BFF-Form-Dwell-Ms)
	HoneypotTriggered     bool    // Hidden field non-empty
	PoWSolved             bool    // PoW solution valid
	PoWDifficulty         int     // Difficulty used
	Recent429Count        int     // Recent 429s on IP/64 or email hash
	DisposableDomain      bool    // Email domain in disposable list
	UnseenDomain          bool    // Email domain not in allowlist
	NoPriorSession        bool    // No device session cookie
	IPReputationScore     float64 // Optional: external reputation (0-1)
}

// WebAuthnCredential represents a WebAuthn credential for allow list.
type WebAuthnCredential struct {
	ID        string
	Type      string
	Transports []string
}

// RiskLevel represents the escalation tier.
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"       // < 50
	RiskLevelMedium   RiskLevel = "medium"    // 50-84
	RiskLevelHigh     RiskLevel = "high"      // 85-94
	RiskLevelCritical RiskLevel = "critical"  // >= 95
)

// RiskLevelFromScore returns the risk level for a given score.
func RiskLevelFromScore(score int) RiskLevel {
	switch {
	case score >= 95:
		return RiskLevelCritical
	case score >= 85:
		return RiskLevelHigh
	case score >= 50:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// ChallengeConfig holds configuration for challenge providers.
type ChallengeConfig struct {
	POWSecret              string
	POWBaseDifficulty      int
	POWChallengeTTLSeconds int
	BotRiskStepUpThreshold int
	BotRiskBlockThreshold  int
	BotRiskCriticalThreshold int
	TurnstileSecretKey     string
	TurnstileExpectedHost  string
	BFFDwellCookieName     string
	BFFDwellSubHumanMS     int
	BFFDwellStaleMS        int
	BotHoneypotFieldPrefix string
	RateLimitCriticalBudgetPct int
}