package ratelimit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter provides Redis-backed rate limiting primitives.
type Limiter struct {
	client   *redis.Client
	hmacKey  []byte
}

// New creates a limiter.
func New(client *redis.Client, hmacKey string) *Limiter {
	var key []byte
	if hmacKey != "" {
		key = []byte(hmacKey)
	}
	return &Limiter{client: client, hmacKey: key}
}

// AllowIP checks and increments an IP-based counter.
// Returns true if request is allowed, false if limit exceeded.
// limit is max requests per window.
func (l *Limiter) AllowIP(ctx context.Context, prefix, ip string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("somotracker:ratelimit:%s:ip:%s", prefix, ip)
	return l.check(ctx, key, limit, window)
}

// AllowEmail checks and increments an email-based counter using HMAC keyed hash.
func (l *Limiter) AllowEmail(ctx context.Context, prefix, email string, limit int, window time.Duration) (bool, error) {
	if l.hmacKey == nil {
		return false, fmt.Errorf("hmac key not configured")
	}
	hash := hmacEmail(email, l.hmacKey)
	key := fmt.Sprintf("somotracker:ratelimit:%s:email:%s", prefix, hash)
	return l.check(ctx, key, limit, window)
}

// AllowGlobal checks a global counter, e.g., email provider quota.
func (l *Limiter) AllowGlobal(ctx context.Context, name string, limit int, window time.Duration) (bool, error) {
	key := fmt.Sprintf("somotracker:ratelimit:global:%s", name)
	return l.check(ctx, key, limit, window)
}

// check performs atomic INCR + EXPIRE.
func (l *Limiter) check(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// Use Lua for atomicity
	script := `
	local current = redis.call("INCR", KEYS[1])
	if tonumber(current) == 1 then
		redis.call("EXPIRE", KEYS[1], ARGV[1])
	end
	if tonumber(current) > tonumber(ARGV[2]) then
		return 0
	else
		return 1
	end
	`
	res, err := l.client.Eval(ctx, script, []string{key}, int(window.Seconds()), limit).Result()
	if err != nil {
		return false, err
	}
	allowed, ok := res.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected redis result")
	}
	return allowed == 1, nil
}

func hmacEmail(email string, key []byte) string {
	// Normalize email to lower case
	norm := ""
	for _, c := range email {
		if c >= 'A' && c <= 'Z' {
			norm += string(c + 32)
		} else {
			norm += string(c)
		}
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(norm))
	return hex.EncodeToString(mac.Sum(nil))
}
