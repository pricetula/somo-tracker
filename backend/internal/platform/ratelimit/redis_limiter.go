// Package ratelimit provides Redis-backed rate limiting using sliding window algorithm.
package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"somotracker/internal/platform/redis"
)

// RedisLimiter implements Limiter using Redis sorted sets with atomic Lua script.
type RedisLimiter struct {
	store Store
	sha   string // cached script SHA
}

// Lua script for atomic sliding window log rate limiting.
// KEYS[1] = rate limit key
// ARGV[1] = limit (int)
// ARGV[2] = window_ms (int)
// ARGV[3] = now_ms (int, from Redis TIME)
// ARGV[4] = member (string, unique request ID)
const slidingWindowScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local member = ARGV[4]

local window_start = now_ms - window_ms

-- 1. Prune expired entries
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- 2. Count current entries
local count = redis.call('ZCARD', key)

-- 3. Check limit
if count >= limit then
  -- Rejected: do NOT add to ZSET (prevents memory exhaustion)
  local ttl_ms = redis.call('PTTL', key)
  if ttl_ms < 0 then ttl_ms = window_ms + 60000 end
  return {0, count, limit, 0, window_ms, window_ms}
end

-- 4. Allowed: add entry and refresh TTL
redis.call('ZADD', key, now_ms, member)
redis.call('PEXPIRE', key, window_ms + 60000)

local remaining = limit - count - 1
local reset_ms = window_ms
local retry_after_ms = 0

return {1, count + 1, limit, remaining, reset_ms, retry_after_ms}
`

// NewRedisLimiter creates a new Redis-backed rate limiter.
func NewRedisLimiter(store Store) *RedisLimiter {
	return &RedisLimiter{store: store}
}

// LoadScript loads the Lua script into Redis and caches the SHA.
func (rl *RedisLimiter) LoadScript(ctx context.Context) error {
	sha, err := rl.store.ScriptLoad(ctx, slidingWindowScript)
	if err != nil {
		return fmt.Errorf("load rate limit script: %w", err)
	}
	rl.sha = sha
	return nil
}

// Allow checks the rate limit using the sliding window log algorithm.
// Returns a Decision with allowed status, current count, limit, remaining, and timing info.
func (rl *RedisLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error) {
	if rl.sha == "" {
		if err := rl.LoadScript(ctx); err != nil {
			return Decision{}, err
		}
	}

	nowMs := time.Now().UnixMilli()
	member := fmt.Sprintf("%d-%d", nowMs, time.Now().UnixNano()%1000000)
	windowMs := window.Milliseconds()

	// Try EVALSHA first (faster), fall back to EVAL on NOSCRIPT
	result, err := rl.store.ScriptEvalSHA(ctx, rl.sha, []string{key},
		limit, windowMs, nowMs, member)
	if err != nil {
		// NOSCRIPT error - script was flushed, reload and retry once
		if err := rl.LoadScript(ctx); err != nil {
			return Decision{}, fmt.Errorf("reload script: %w", err)
		}
		result, err = rl.store.ScriptEvalSHA(ctx, rl.sha, []string{key},
			limit, windowMs, nowMs, member)
		if err != nil {
			return Decision{}, fmt.Errorf("eval script after reload: %w", err)
		}
	}

	return parseDecision(result)
}

// parseDecision converts the Lua script result to a Decision struct.
// Lua returns: {allowed, current, limit, remaining, reset_ms, retry_after_ms}
func parseDecision(result interface{}) (Decision, error) {
	arr, ok := result.([]interface{})
	if !ok || len(arr) != 6 {
		return Decision{}, fmt.Errorf("unexpected script result: %v", result)
	}

	allowed := arr[0].(int64) == 1
	current := arr[1].(int64)
	limit := int(arr[2].(int64))
	remaining := arr[3].(int64)
	resetMs := arr[4].(int64)
	retryAfterMs := arr[5].(int64)

	return Decision{
		Allowed:    allowed,
		Current:    current,
		Limit:      limit,
		Remaining:  remaining,
		ResetAfter: time.Duration(resetMs) * time.Millisecond,
		RetryAfter: time.Duration(retryAfterMs) * time.Millisecond,
	}, nil
}

// RedisStore wraps the redis.Client to implement the Store interface.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new RedisStore.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// EvalScript executes a Lua script directly.
func (rs *RedisStore) EvalScript(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return rs.client.Client.Eval(ctx, script, keys, args...).Result()
}

// ScriptLoad loads a Lua script into Redis and returns its SHA.
func (rs *RedisStore) ScriptLoad(ctx context.Context, script string) (string, error) {
	return rs.client.Client.ScriptLoad(ctx, script).Result()
}

// ScriptEvalSHA executes a loaded Lua script by its SHA.
func (rs *RedisStore) ScriptEvalSHA(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error) {
	return rs.client.Client.EvalSha(ctx, sha, keys, args...).Result()
}

// MarshalDecision serializes a Decision to JSON for logging/debugging.
func MarshalDecision(d Decision) string {
	b, _ := json.Marshal(d)
	return string(b)
}

// UnmarshalDecision deserializes a Decision from JSON.
func UnmarshalDecision(data string) (Decision, error) {
	var d Decision
	err := json.Unmarshal([]byte(data), &d)
	return d, err
}

// DecisionToHeaders converts a Decision to rate limit headers.
func DecisionToHeaders(d Decision) map[string]string {
	resetSec := int(d.ResetAfter.Seconds())
	if resetSec < 0 {
		resetSec = 0
	}

	return map[string]string{
		"RateLimit-Limit":     strconv.Itoa(d.Limit),
		"RateLimit-Remaining": strconv.FormatInt(d.Remaining, 10),
		"RateLimit-Reset":     strconv.Itoa(resetSec),
		"X-RateLimit-Limit":   strconv.Itoa(d.Limit),
		"X-RateLimit-Remaining": strconv.FormatInt(d.Remaining, 10),
		"X-RateLimit-Reset":   strconv.FormatInt(time.Now().Add(d.ResetAfter).Unix(), 10),
	}
}

// DecisionToRetryAfter returns the Retry-After header value in seconds.
func DecisionToRetryAfter(d Decision) string {
	if d.RetryAfter > 0 {
		return strconv.Itoa(int(d.RetryAfter.Seconds()))
	}
	return "0"
}