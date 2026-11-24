package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"somotracker/internal/platform/config"
)

func TestKeyBuilder_EmailKey(t *testing.T) {
	cfg := &config.Config{
		RateLimitKeyHMACSecret: "test-secret-32-bytes-long-for-hmac!!",
	}
	kb := NewKeyBuilder(cfg)

	key := kb.EmailKey("user@example.com")
	assert.Contains(t, key, "rl:otp_request:email:")
	// Key should be HMAC, not raw email
	assert.NotContains(t, key, "user@example.com")
	assert.NotContains(t, key, "@")
}

func TestKeyBuilder_EmailKey_Deterministic(t *testing.T) {
	cfg := &config.Config{
		RateLimitKeyHMACSecret: "test-secret-32-bytes-long-for-hmac!!",
	}
	kb := NewKeyBuilder(cfg)

	key1 := kb.EmailKey("user@example.com")
	key2 := kb.EmailKey("user@example.com")
	assert.Equal(t, key1, key2, "same email should produce same key")
}

func TestKeyBuilder_EmailKey_DifferentEmails(t *testing.T) {
	cfg := &config.Config{
		RateLimitKeyHMACSecret: "test-secret-32-bytes-long-for-hmac!!",
	}
	kb := NewKeyBuilder(cfg)

	key1 := kb.EmailKey("user1@example.com")
	key2 := kb.EmailKey("user2@example.com")
	assert.NotEqual(t, key1, key2, "different emails should produce different keys")
}

func TestKeyBuilder_UserKey(t *testing.T) {
	cfg := &config.Config{
		RateLimitKeyHMACSecret: "test-secret",
	}
	kb := NewKeyBuilder(cfg)

	key := kb.UserKey("user-123")
	assert.Equal(t, "rl:global:user:user-123", key)
}

func TestKeyBuilder_ChallengeKey(t *testing.T) {
	cfg := &config.Config{
		RateLimitKeyHMACSecret: "test-secret",
	}
	kb := NewKeyBuilder(cfg)

	key := kb.ChallengeKey("challenge-456")
	assert.Equal(t, "rl:otp_verify:challenge:challenge-456", key)
}

func TestKeyBuilder_IPKey(t *testing.T) {
	cfg := &config.Config{
		RateLimitKeyHMACSecret: "test-secret",
	}
	kb := NewKeyBuilder(cfg)

	key := kb.IPKey("1.2.3.4/32")
	assert.Equal(t, "rl:global:ip:1.2.3.4/32", key)
}

func TestKeyBuilder_GlobalKey(t *testing.T) {
	cfg := &config.Config{
		RateLimitKeyHMACSecret: "test-secret",
	}
	kb := NewKeyBuilder(cfg)

	key := kb.GlobalKey()
	assert.Equal(t, "rl:otp_request:global:send_budget", key)
}

func TestKeyBuilder_DomainKey(t *testing.T) {
	cfg := &config.Config{
		RateLimitKeyHMACSecret: "test-secret",
	}
	kb := NewKeyBuilder(cfg)

	key := kb.DomainKey("Example.COM")
	assert.Equal(t, "rl:otp_request:domain:example.com", key)
}

func TestKeyBuilder_TierPrefix(t *testing.T) {
	cfg := &config.Config{
		RateLimitKeyHMACSecret: "test-secret",
	}
	kb := NewKeyBuilder(cfg)

	assert.Equal(t, "rl:otp_request", kb.TierPrefix("otp_request"))
	assert.Equal(t, "rl:otp_verify", kb.TierPrefix("otp_verify"))
	assert.Equal(t, "rl:onboard", kb.TierPrefix("onboard"))
}

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  User@Example.COM  ", "user@example.com"},
		{"user+tag@gmail.com", "user@gmail.com"},
		{"user+tag@GMAIL.COM", "user@gmail.com"},
		{"user+tag@googlemail.com", "user@googlemail.com"},
		{"user+tag@outlook.com", "user@outlook.com"},
		{"user+tag@hotmail.com", "user@hotmail.com"},
		{"user+tag@live.com", "user@live.com"},
		{"user+tag@yahoo.com", "user@yahoo.com"},
		{"user+tag@ymail.com", "user@ymail.com"},
		{"user+tag@icloud.com", "user@icloud.com"},
		{"user+tag@me.com", "user@me.com"},
		{"user+tag@mac.com", "user@mac.com"},
		{"user+tag@protonmail.com", "user@protonmail.com"},
		{"user+tag@proton.me", "user@proton.me"},
		{"user+tag@custom.com", "user+tag@custom.com"}, // unknown provider, keep +tag
		{"USER@EXAMPLE.COM", "user@example.com"},
		{"user@example.com", "user@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.2.3.4", "1.2.3.4/32"},
		{"192.168.1.1", "192.168.1.1/32"},
		{"2001:db8::1", "2001:db8::/64"},
		{"::1", "::/64"},
		{"10.0.0.1", "10.0.0.1/32"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeIP(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeIP_Invalid(t *testing.T) {
	result := NormalizeIP("not-an-ip")
	assert.Equal(t, "not-an-ip", result) // fallback to original
}

func TestBuildKey(t *testing.T) {
	key := BuildKey("otp_request", "email", "hmac123")
	assert.Equal(t, "rl:otp_request:email:hmac123", key)
}