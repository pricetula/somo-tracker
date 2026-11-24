// Package ratelimit provides Redis-backed rate limiting using sliding window algorithm.
package ratelimit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"somotracker/internal/platform/config"
)

// KeyBuilderImpl implements the KeyBuilder interface.
type KeyBuilderImpl struct {
	hmacSecret []byte
}

// NewKeyBuilder creates a new KeyBuilder with the given HMAC secret.
func NewKeyBuilder(cfg *config.Config) *KeyBuilderImpl {
	return &KeyBuilderImpl{
		hmacSecret: []byte(cfg.RateLimitKeyHMACSecret),
	}
}

// TierPrefix returns the tier prefix for key namespacing.
func (kb *KeyBuilderImpl) TierPrefix(tier string) string {
	return "rl:" + tier
}

// UserKey builds a key for authenticated user (highest priority).
// Format: rl:{tier}:user:{userID}
func (kb *KeyBuilderImpl) UserKey(userID string) string {
	return kb.TierPrefix("global") + ":user:" + userID
}

// ChallengeKey builds a key for OTP challenge ID.
// Format: rl:otp_verify:challenge:{challengeID}
func (kb *KeyBuilderImpl) ChallengeKey(challengeID string) string {
	return kb.TierPrefix("otp_verify") + ":challenge:" + challengeID
}

// EmailKey builds a key for normalized email using HMAC-SHA256.
// Format: rl:{tier}:email:{hmac_hex}
// This ensures no raw email addresses appear in Redis keys or logs.
func (kb *KeyBuilderImpl) EmailKey(normalizedEmail string) string {
	// Use HMAC-SHA256 with the configured secret
	mac := hmac.New(sha256.New, kb.hmacSecret)
	mac.Write([]byte(normalizedEmail))
	hmacHex := hex.EncodeToString(mac.Sum(nil))
	return kb.TierPrefix("otp_request") + ":email:" + hmacHex
}

// IPKey builds a key for normalized IP (/32 IPv4, /64 IPv6).
// Format: rl:{tier}:ip:{normalizedIP}
func (kb *KeyBuilderImpl) IPKey(normalizedIP string) string {
	return kb.TierPrefix("global") + ":ip:" + normalizedIP
}

// GlobalKey builds the global circuit breaker key.
// Format: rl:otp_request:global:send_budget
func (kb *KeyBuilderImpl) GlobalKey() string {
	return kb.TierPrefix("otp_request") + ":global:send_budget"
}

// DomainKey builds a key for email domain velocity limiting.
// Format: rl:otp_request:domain:{normalizedDomain}
func (kb *KeyBuilderImpl) DomainKey(domain string) string {
	return kb.TierPrefix("otp_request") + ":domain:" + strings.ToLower(strings.TrimSpace(domain))
}

// ReturningUserKey builds a key for returning user marker.
// Format: rl:returning_user:{email_hmac}
func (kb *KeyBuilderImpl) ReturningUserKey(emailHMAC string) string {
	return "rl:returning_user:" + emailHMAC
}

// FailureCountKey builds a key for account-level failure counting.
// Format: rl:otp_verify:email:{email_hmac}:failures
func (kb *KeyBuilderImpl) FailureCountKey(emailHMAC string) string {
	return kb.TierPrefix("otp_verify") + ":email:" + emailHMAC + ":failures"
}

// PoWNonceKey builds a key for PoW nonce single-use tracking.
// Format: rl:pow:nonce:{challengeID}:{nonce}
func (kb *KeyBuilderImpl) PoWNonceKey(challengeID, nonce string) string {
	return "rl:pow:nonce:" + challengeID + ":" + nonce
}

// NormalizeEmail normalizes an email address for rate limiting.
// 1. Trim whitespace
// 2. Lowercase
// 3. Unicode NFC normalization (handled by Go strings)
// 4. Strip +tag aliases for known safe providers (Gmail, Outlook, Yahoo, iCloud, Proton)
func NormalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)

	// Strip +tag for known providers
	atIdx := strings.Index(email, "@")
	if atIdx > 0 {
		localPart := email[:atIdx]
		domain := email[atIdx+1:]

		// Check if provider supports +tag and strip it
		if isSafePlusTagProvider(domain) {
			if plusIdx := strings.Index(localPart, "+"); plusIdx >= 0 {
				localPart = localPart[:plusIdx]
			}
		}
		email = localPart + "@" + domain
	}

	return email
}

// isSafePlusTagProvider returns true for providers known to support +tag addressing.
func isSafePlusTagProvider(domain string) bool {
	safeProviders := map[string]bool{
		"gmail.com":       true,
		"googlemail.com":  true,
		"outlook.com":     true,
		"hotmail.com":     true,
		"live.com":        true,
		"yahoo.com":       true,
		"ymail.com":       true,
		"icloud.com":      true,
		"me.com":          true,
		"mac.com":         true,
		"protonmail.com":  true,
		"proton.me":       true,
	}
	return safeProviders[domain]
}

// NormalizeIP normalizes an IP address for rate limiting keys.
// IPv4: masked to /32 (single host)
// IPv6: masked to /64 prefix (typical residential prefix)
func NormalizeIP(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ipStr // fallback
	}

	if ip4 := ip.To4(); ip4 != nil {
		// IPv4: /32
		return ip4.String() + "/32"
	}
	// IPv6: /64 prefix
	ip16 := ip.To16()
	if ip16 == nil {
		return ipStr
	}
	// Zero out lower 64 bits
	for i := 8; i < 16; i++ {
		ip16[i] = 0
	}
	return net.IP(ip16).String() + "/64"
}

// BuildKey builds a rate limit key with tier prefix and identifier.
func BuildKey(tier, keyType, identifier string) string {
	return fmt.Sprintf("rl:%s:%s:%s", tier, keyType, identifier)
}