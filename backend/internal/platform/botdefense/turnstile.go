// Package botdefense provides adaptive bot defense with pluggable challenge providers.
package botdefense

import (
	"context"
	"errors"
	"fmt"
	"time"

	"somotracker/internal/platform/turnstile"
)

// TurnstileProvider wraps the existing Turnstile verifier.
type TurnstileProvider struct {
	verifier turnstile.Verifier
	siteKey  string
}

// NewTurnstileProvider creates a new Turnstile provider.
func NewTurnstileProvider(verifier turnstile.Verifier, siteKey string) *TurnstileProvider {
	return &TurnstileProvider{
		verifier: verifier,
		siteKey:  siteKey,
	}
}

// Name returns the provider name.
func (t *TurnstileProvider) Name() string {
	return "turnstile"
}

// IssueChallenge returns Turnstile site key and action for client-side widget.
func (t *TurnstileProvider) IssueChallenge(ctx context.Context, req ChallengeRequest) (ChallengeResponse, error) {
	if t.verifier == nil {
		return ChallengeResponse{}, fmt.Errorf("turnstile not configured")
	}

	return ChallengeResponse{
		ChallengeID: "",
		Type:        "turnstile",
		SiteKey:     t.siteKey,
		Action:      req.Action,
		ExpiresAt:   time.Now().Add(10 * time.Minute), // Token typically valid ~10 min
	}, nil
}

// VerifyChallenge validates a Turnstile token.
func (t *TurnstileProvider) VerifyChallenge(ctx context.Context, req VerifyRequest) (VerifyResult, error) {
	if t.verifier == nil {
		return VerifyResult{Valid: false, Error: "turnstile not configured"}, nil
	}

	token, ok := req.Solution.(string)
	if !ok {
		return VerifyResult{Valid: false, Error: "invalid token format"}, nil
	}

	result, err := t.verifier.Verify(ctx, token, req.ClientSubnet)
	if err != nil {
		if errors.Is(err, turnstile.ErrUpstream) {
			return VerifyResult{Valid: false, Error: "turnstile verification unavailable"}, nil
		}
		return VerifyResult{Valid: false, Error: "turnstile verification failed"}, nil
	}

	if !result.Success {
		return VerifyResult{Valid: false, Error: "turnstile verification failed"}, nil
	}

	// Verify action matches expected
	if result.Action != req.Action {
		return VerifyResult{Valid: false, Error: "turnstile action mismatch"}, nil
	}

	return VerifyResult{Valid: true}, nil
}

// RiskScore returns base risk score (Turnstile itself doesn't compute risk).
func (t *TurnstileProvider) RiskScore(ctx context.Context, signals BaselineSignals) (int, error) {
	if signals.TurnstileSuccess {
		return 0, nil
	}
	return 40, nil // Failed Turnstile = high risk
}