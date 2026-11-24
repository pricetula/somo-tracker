// Package botdefense provides adaptive bot defense with pluggable challenge providers.
package botdefense

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"somotracker/internal/platform/redis"
	"somotracker/internal/platform/turnstile"
)

// DwellValidator validates BFF pre-session timing cookie.
type DwellValidator struct {
	cookieName    string
	secret        []byte
	subHumanMS    int
	staleMS       int
}

// NewDwellValidator creates a new dwell time validator.
func NewDwellValidator(cookieName, secret string, subHumanMS, staleMS int) *DwellValidator {
	return &DwellValidator{
		cookieName: cookieName,
		secret:     []byte(secret),
		subHumanMS: subHumanMS,
		staleMS:    staleMS,
	}
}

// ValidateCookie parses and validates the dwell cookie.
// Returns (dwellTimeMs, isSubHuman, isStale, error).
func (d *DwellValidator) ValidateCookie(cookieValue string) (int64, bool, bool, error) {
	if cookieValue == "" {
		return 0, false, false, errors.New("dwell cookie missing")
	}

	// Format: timestamp_ms|hmac_signature
	parts := strings.Split(cookieValue, "|")
	if len(parts) != 2 {
		return 0, false, false, errors.New("invalid dwell cookie format")
	}

	timestampStr := parts[0]
	signature := parts[1]

	// Verify HMAC
	expectedSig := d.sign(timestampStr)
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return 0, false, false, errors.New("dwell cookie signature invalid")
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return 0, false, false, errors.New("invalid timestamp")
	}

	now := time.Now().UnixMilli()
	dwellMs := now - timestamp

	isSubHuman := dwellMs < int64(d.subHumanMS)
	isStale := dwellMs > int64(d.staleMS)

	return dwellMs, isSubHuman, isStale, nil
}

// sign creates HMAC signature for a timestamp.
func (d *DwellValidator) sign(timestamp string) string {
	mac := hmac.New(sha256.New, d.secret)
	mac.Write([]byte(timestamp))
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateCookie creates a new dwell cookie value.
func (d *DwellValidator) GenerateCookie() string {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signature := d.sign(timestamp)
	return timestamp + "|" + signature
}

// HoneypotValidator validates honeypot field.
type HoneypotValidator struct {
	fieldPrefix string
}

// NewHoneypotValidator creates a new honeypot validator.
func NewHoneypotValidator(fieldPrefix string) *HoneypotValidator {
	return &HoneypotValidator{fieldPrefix: fieldPrefix}
}

// Validate checks if honeypot field is triggered.
// Returns true if triggered (bot detected).
func (h *HoneypotValidator) Validate(formData map[string]string) bool {
	for key, value := range formData {
		if strings.HasPrefix(key, h.fieldPrefix) && value != "" {
			return true
		}
	}
	return false
}

// GenerateFieldName generates a randomized honeypot field name.
func (h *HoneypotValidator) GenerateFieldName() string {
	// Use timestamp + random for uniqueness
	return fmt.Sprintf("%s%d", h.fieldPrefix, time.Now().UnixNano()%1000000)
}

// RiskScorer computes risk score from baseline signals.
type RiskScorer struct {
	stepUpThreshold  int
	blockThreshold   int
	criticalThreshold int
}

// NewRiskScorer creates a new risk scorer.
func NewRiskScorer(stepUp, block, critical int) *RiskScorer {
	return &RiskScorer{
		stepUpThreshold:   stepUp,
		blockThreshold:    block,
		criticalThreshold: critical,
	}
}

// Score computes risk score (0-100) from baseline signals.
func (r *RiskScorer) Score(signals BaselineSignals) int {
	score := 0

	// Turnstile failure: +40
	if !signals.TurnstileSuccess {
		score += 40
	}

	// Sub-human dwell time: +25
	if signals.DwellTimeMs > 0 && signals.DwellTimeMs < 800 {
		score += 25
	}

	// Stale dwell time: +10
	if signals.DwellTimeMs > 14400000 { // 4 hours
		score += 10
	}

	// Honeypot triggered: +35
	if signals.HoneypotTriggered {
		score += 35
	}

	// PoW unsolved/invalid: +20
	if !signals.PoWSolved {
		score += 20
	}

	// Recent 429s on IP: +5 per (cap 20)
	if signals.Recent429Count > 0 {
		ipScore := signals.Recent429Count * 5
		if ipScore > 20 {
			ipScore = 20
		}
		score += ipScore
	}

	// Recent 429s on email hash: +10 per (cap 30)
	if signals.Recent429Count > 0 {
		emailScore := signals.Recent429Count * 10
		if emailScore > 30 {
			emailScore = 30
		}
		score += emailScore
	}

	// Disposable domain: +15
	if signals.DisposableDomain {
		score += 15
	}

	// Unseen domain: +10
	if signals.UnseenDomain {
		score += 10
	}

	// No prior session/device: +5
	if signals.NoPriorSession {
		score += 5
	}

	// IP reputation (if enabled): 0-20
	if signals.IPReputationScore > 0 {
		score += int(signals.IPReputationScore * 20)
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// EscalationLevel returns the escalation level for a risk score.
func (r *RiskScorer) EscalationLevel(score int) RiskLevel {
	switch {
	case score >= r.criticalThreshold:
		return RiskLevelCritical
	case score >= r.blockThreshold:
		return RiskLevelHigh
	case score >= r.stepUpThreshold:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// BotDefenseManager orchestrates all bot defense providers.
type BotDefenseManager struct {
	providers      map[string]ChallengeProvider
	powProvider    *PoWProvider
	turnstileProv  *TurnstileProvider
	dwellValidator *DwellValidator
	honeypotVal    *HoneypotValidator
	riskScorer     *RiskScorer
	redis          *redis.Client
	criticalBudgetPct int
}

// NewBotDefenseManager creates a new bot defense manager.
func NewBotDefenseManager(
	redisClient *redis.Client,
	powSecret string,
	powBaseDifficulty int,
	powTTLSeconds int,
	turnstileVerifier turnstile.Verifier,
	dwellCookieName string,
	dwellSecret string,
	dwellSubHumanMS int,
	dwellStaleMS int,
	honeypotFieldPrefix string,
	stepUpThreshold int,
	blockThreshold int,
	criticalThreshold int,
	criticalBudgetPct int,
) *BotDefenseManager {
	powProv := NewPoWProvider(redisClient, powSecret, powBaseDifficulty, powTTLSeconds)
	tsProv := NewTurnstileProvider(turnstileVerifier, "")
	dwellVal := NewDwellValidator(dwellCookieName, dwellSecret, dwellSubHumanMS, dwellStaleMS)
	honeypotVal := NewHoneypotValidator(honeypotFieldPrefix)
	riskScorer := NewRiskScorer(stepUpThreshold, blockThreshold, criticalThreshold)

	m := &BotDefenseManager{
		providers: map[string]ChallengeProvider{
			"pow":      powProv,
			"turnstile": tsProv,
		},
		powProvider:       powProv,
		turnstileProv:     tsProv,
		dwellValidator:    dwellVal,
		honeypotVal:       honeypotVal,
		riskScorer:        riskScorer,
		redis:             redisClient,
		criticalBudgetPct: criticalBudgetPct,
	}

	return m
}

// NewBotDefenseManagerWithSiteKey creates a new bot defense manager with Turnstile site key.
func NewBotDefenseManagerWithSiteKey(
	redisClient *redis.Client,
	powSecret string,
	powBaseDifficulty int,
	powTTLSeconds int,
	turnstileVerifier turnstile.Verifier,
	turnstileSiteKey string,
	dwellCookieName string,
	dwellSecret string,
	dwellSubHumanMS int,
	dwellStaleMS int,
	honeypotFieldPrefix string,
	stepUpThreshold int,
	blockThreshold int,
	criticalThreshold int,
	criticalBudgetPct int,
) *BotDefenseManager {
	powProv := NewPoWProvider(redisClient, powSecret, powBaseDifficulty, powTTLSeconds)
	tsProv := NewTurnstileProvider(turnstileVerifier, turnstileSiteKey)
	dwellVal := NewDwellValidator(dwellCookieName, dwellSecret, dwellSubHumanMS, dwellStaleMS)
	honeypotVal := NewHoneypotValidator(honeypotFieldPrefix)
	riskScorer := NewRiskScorer(stepUpThreshold, blockThreshold, criticalThreshold)

	m := &BotDefenseManager{
		providers: map[string]ChallengeProvider{
			"pow":      powProv,
			"turnstile": tsProv,
		},
		powProvider:       powProv,
		turnstileProv:     tsProv,
		dwellValidator:    dwellVal,
		honeypotVal:       honeypotVal,
		riskScorer:        riskScorer,
		redis:             redisClient,
		criticalBudgetPct: criticalBudgetPct,
	}

	return m
}

// AssessRisk computes risk score and escalation level from request signals.
func (m *BotDefenseManager) AssessRisk(ctx context.Context, signals BaselineSignals) (int, RiskLevel, error) {
	score := m.riskScorer.Score(signals)
	level := m.riskScorer.EscalationLevel(score)

	// Check global budget for auto-critical escalation
	if m.criticalBudgetPct > 0 {
		budgetUsed, err := m.getGlobalBudgetUsage(ctx)
		if err == nil && budgetUsed >= float64(m.criticalBudgetPct) {
			if level != RiskLevelCritical {
				level = RiskLevelCritical
			}
		}
	}

	return score, level, nil
}

// getGlobalBudgetUsage returns the percentage of global OTP budget used.
func (m *BotDefenseManager) getGlobalBudgetUsage(ctx context.Context) (float64, error) {
	// This would check the Redis keys for global budget
	// For now, return 0 to avoid dependency on internal structures
	return 0, nil
}

// DwellValidator returns the dwell validator.
func (m *BotDefenseManager) DwellValidator() *DwellValidator {
	return m.dwellValidator
}

// HoneypotVal returns the honeypot validator.
func (m *BotDefenseManager) HoneypotVal() *HoneypotValidator {
	return m.honeypotVal
}

// IssueChallenge issues a challenge appropriate for the risk level.
func (m *BotDefenseManager) IssueChallenge(ctx context.Context, req ChallengeRequest) (ChallengeResponse, error) {
	// Always issue Turnstile as baseline
	if m.turnstileProv != nil && m.turnstileProv.verifier != nil {
		_, err := m.turnstileProv.IssueChallenge(ctx, req)
		if err == nil {
			// Also issue PoW
			powResp, _ := m.powProvider.IssueChallenge(ctx, req)
			// Return combined response (PoW as primary for verification)
			return powResp, nil
		}
	}

	// Fallback to PoW only
	return m.powProvider.IssueChallenge(ctx, req)
}

// VerifyChallenge verifies a submitted challenge solution.
func (m *BotDefenseManager) VerifyChallenge(ctx context.Context, req VerifyRequest) (VerifyResult, error) {
	provider, ok := m.providers[req.Type]
	if !ok {
		return VerifyResult{Valid: false, Error: "unknown challenge type"}, nil
	}
	return provider.VerifyChallenge(ctx, req)
}