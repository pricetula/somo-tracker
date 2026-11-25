// Package botdefense provides adaptive bot defense with pluggable challenge providers.
package botdefense

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"somotracker/internal/platform/redis"
)

// PoWProvider implements Proof-of-Work challenge provider.
type PoWProvider struct {
	redis       *redis.Client
	secret      []byte
	baseDiff    int
	challengeTTL time.Duration
}

// NewPoWProvider creates a new PoW provider.
func NewPoWProvider(redisClient *redis.Client, secret string, baseDifficulty int, ttlSeconds int) *PoWProvider {
	return &PoWProvider{
		redis:        redisClient,
		secret:       []byte(secret),
		baseDiff:     baseDifficulty,
		challengeTTL: time.Duration(ttlSeconds) * time.Second,
	}
}

// Name returns the provider name.
func (p *PoWProvider) Name() string {
	return "pow"
}

// IssueChallenge generates a new PoW challenge.
func (p *PoWProvider) IssueChallenge(ctx context.Context, req ChallengeRequest) (ChallengeResponse, error) {
	challengeID := generateChallengeID()
	seed := generateSeed()

	// Difficulty scales with risk score
	difficulty := p.baseDiff
	if req.RiskScore >= 50 {
		difficulty += (req.RiskScore - 50) / 10 * 2 // +2 per 10 points above 50
	}
	if difficulty > 20 {
		difficulty = 20
	}

	expiresAt := time.Now().Add(p.challengeTTL)

	// Sign the challenge with HMAC to prevent tampering
	challengeData := fmt.Sprintf("%s|%d|%s|%d", challengeID, difficulty, seed, expiresAt.Unix())
	mac := hmac.New(sha256.New, p.secret)
	mac.Write([]byte(challengeData))
	signature := hex.EncodeToString(mac.Sum(nil))

	// Store challenge in Redis with TTL
	challengeKey := fmt.Sprintf("rl:pow:challenge:%s", challengeID)
	challengeValue := fmt.Sprintf("%d|%s|%d|%s", difficulty, seed, expiresAt.Unix(), signature)
	if err := p.redis.SetString(ctx, challengeKey, challengeValue, p.challengeTTL); err != nil {
		return ChallengeResponse{}, fmt.Errorf("store challenge: %w", err)
	}

	return ChallengeResponse{
		ChallengeID: challengeID,
		Type:        "pow",
		Difficulty:  difficulty,
		Seed:        seed,
		ExpiresAt:   expiresAt,
	}, nil
}

// VerifyChallenge validates a PoW solution.
func (p *PoWProvider) VerifyChallenge(ctx context.Context, req VerifyRequest) (VerifyResult, error) {
	solution, ok := req.Solution.(PowSolution)
	if !ok {
		return VerifyResult{Valid: false, Error: "invalid solution format"}, nil
	}

	// Retrieve challenge from Redis
	challengeKey := fmt.Sprintf("rl:pow:challenge:%s", req.ChallengeID)
	challengeValue, err := p.redis.GetString(ctx, challengeKey)
	if err != nil {
		return VerifyResult{Valid: false, Error: "challenge not found or expired"}, nil
	}

	// Parse challenge
	parts := strings.Split(challengeValue, "|")
	if len(parts) != 4 {
		return VerifyResult{Valid: false, Error: "invalid challenge format"}, nil
	}

	difficulty, _ := strconv.Atoi(parts[0])
	seed := parts[1]
	expiresAtUnix, _ := strconv.ParseInt(parts[2], 10, 64)
	expectedSig := parts[3]

	// Check expiry
	if time.Now().Unix() > expiresAtUnix {
		return VerifyResult{Valid: false, Error: "challenge expired"}, nil
	}

	// Verify HMAC signature
	challengeData := fmt.Sprintf("%s|%d|%s|%d", req.ChallengeID, difficulty, seed, expiresAtUnix)
	mac := hmac.New(sha256.New, p.secret)
	mac.Write([]byte(challengeData))
	actualSig := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(actualSig), []byte(expectedSig)) {
		return VerifyResult{Valid: false, Error: "challenge signature invalid"}, nil
	}

	// Check nonce hasn't been used (single-use)
	nonceKey := fmt.Sprintf("rl:pow:nonce:%s:%s", req.ChallengeID, solution.Nonce)
	used, err := p.redis.SetNX(ctx, nonceKey, "1", p.challengeTTL)
	if err != nil {
		return VerifyResult{Valid: false, Error: "nonce check failed"}, nil
	}
	if !used {
		return VerifyResult{Valid: false, Error: "nonce already used"}, nil
	}

	// Verify PoW: SHA256(seed + nonce) has required leading zero bits
	input := seed + solution.Nonce
	hash := sha256.Sum256([]byte(input))
	leadingZeros := countLeadingZeros(hash[:])
	if leadingZeros < difficulty {
		return VerifyResult{Valid: false, Error: "insufficient work"}, nil
	}

	// Clean up challenge after successful verification
	_, _ = p.redis.Del(ctx, challengeKey)

	return VerifyResult{Valid: true}, nil
}

// RiskScore returns base risk score (PoW itself doesn't compute risk).
func (p *PoWProvider) RiskScore(ctx context.Context, signals BaselineSignals) (int, error) {
	return 0, nil
}

// generateChallengeID creates a unique challenge ID.
func generateChallengeID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateSeed creates a random seed for PoW.
func generateSeed() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// countLeadingZeros counts leading zero bits in a byte slice.
func countLeadingZeros(data []byte) int {
	count := 0
	for _, b := range data {
		if b == 0 {
			count += 8
		} else {
			// Count leading zeros in this byte
			for i := 7; i >= 0; i-- {
				if b&(1<<i) != 0 {
					return count
				}
				count++
			}
		}
	}
	return count
}

// VerifyPoWSolution verifies a PoW solution without Redis (for testing).
func VerifyPoWSolution(seed string, nonce string, difficulty int) bool {
	input := seed + nonce
	hash := sha256.Sum256([]byte(input))
	return countLeadingZeros(hash[:]) >= difficulty
}