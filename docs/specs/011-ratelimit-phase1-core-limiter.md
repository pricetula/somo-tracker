# SPEC-011: Rate Limiting Phase 1 — Core Sliding Window Limiter (Redis Lua)

Endpoint: Internal library (no HTTP endpoint)

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-011 |
| title | Rate Limiting Phase 1 — Core Sliding Window Limiter (Redis Lua) |
| status | draft |
| depends_on | docs/adr/0002-rate-limiting-strategy.md, specs/010-ratelimit-phase0-trust-boundary.md, specs/glossary.md |
| api_endpoint | N/A (library) |
| owner | backend |

## Summary

Implement the **atomic Redis Lua sliding window log** algorithm as a reusable `Limiter` interface. This is the core rate limiting engine used by all tiers (OTP Request, OTP Verify, Onboard, Session, Global API). It provides accurate rolling-window counting, zero memory growth on rejected requests, and a `Decision` struct for standardized headers.

## Algorithm: Atomic Sliding Window Log (Redis Sorted Set + Lua)

### Data Structure
- **Redis Key**: `rl:{tier}:{key_type}:{identifier}` (e.g., `rl:otp_request:email:hmac_abc123`)
- **Value**: Sorted Set (`ZSET`)
  - **Score**: Redis server timestamp in milliseconds (`TIME` command)
  - **Member**: `{timestamp_ms}-{random_hex_6}` (prevents same-millisecond collisions)

### Lua Script (Single Atomic Execution)

```lua
-- KEYS[1] = rate limit key
-- ARGV[1] = limit (integer)
-- ARGV[2] = window_ms (integer)
-- ARGV[3] = now_ms (integer, from Redis TIME)
-- ARGV[4] = member (string, unique request ID)

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
```

### Return Values → Go `Decision` Struct

```go
type Decision struct {
    Allowed       bool
    Current       int64
    Limit         int
    Remaining     int64
    ResetAfter    time.Duration  // reset_ms
    RetryAfter    time.Duration  // retry_after_ms (0 if allowed)
}
```

### Key Properties

| Property | Implementation |
|----------|----------------|
| **Clock source** | Redis `TIME` (authoritative, eliminates app-server clock skew) |
| **TTL** | `window_ms + 60000` (1 min buffer for cleanup) |
| **Rejected requests** | NOT added to ZSET — zero memory growth under attack |
| **Collision avoidance** | Member = `{timestamp_ms}-{random_hex_6}` |
| **Timeout** | 50 ms context deadline (`RATE_LIMIT_REDIS_TIMEOUT_MS`) |

## Interfaces

```go
// Limiter is the core rate limiting interface.
type Limiter interface {
    // Allow checks the rate limit for a key and returns a Decision.
    // The key should be pre-built by KeyBuilder (HMAC email, normalized IP, etc.)
    Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error)
}

// KeyBuilder constructs hierarchical rate limit keys.
type KeyBuilder interface {
    // UserKey builds a key for authenticated user (highest priority).
    UserKey(userID string) string
    // ChallengeKey builds a key for OTP challenge ID.
    ChallengeKey(challengeID string) string
    // EmailKey builds a key for normalized email (HMAC-SHA256).
    EmailKey(normalizedEmail string) string
    // IPKey builds a key for normalized IP (/32 IPv4, /64 IPv6).
    IPKey(normalizedIP string) string
    // GlobalKey builds the global circuit breaker key.
    GlobalKey() string
    // TierPrefix returns the tier prefix for key namespacing.
    TierPrefix(tier string) string
}
```

## Shadow Mode

- **Config**: `RATE_LIMIT_MODE=shadow|enforce` (Doppler, overridable per tier)
- **Shadow behavior**: Execute Lua script, log `Decision`, but **never return 429**
- **Metrics**: Emit `ratelimit_decisions_total{tier,result="allowed|would_block"}` in shadow mode
- **Transition**: Flip to `enforce` per tier via Doppler after validation

## Failure Modes

| Tier | Redis Timeout/Error | Behavior |
|------|---------------------|----------|
| OTP Request, OTP Verify, Onboard | **Fail closed** | Return `503 Service Unavailable`; backup in-process token bucket (coarse per-IP) |
| Session, Global API | **Fail open** | Log `ratelimit_failopen_total{tier}`, allow request, alert |

## Configuration (Doppler)

```bash
RATE_LIMIT_MODE=shadow                          # shadow | enforce
RATE_LIMIT_REDIS_TIMEOUT_MS=50                  # Lua script execution deadline
```

## Implementation Files

| File | Responsibility |
|------|----------------|
| `internal/platform/ratelimit/limiter.go` | `Limiter` interface, `Decision` struct |
| `internal/platform/ratelimit/redis_limiter.go` | Redis Lua script loading, `Allow` implementation |
| `internal/platform/ratelimit/key_builder.go` | `KeyBuilder` implementation (HMAC email, IP normalization) |
| `internal/platform/ratelimit/fallback.go` | In-process token bucket for fail-closed tiers |
| `internal/platform/redis/lua.go` | Lua script registration/eval helpers |
| `internal/platform/config/config.go` | `RATE_LIMIT_MODE`, `RATE_LIMIT_REDIS_TIMEOUT_MS` parsing |

## Tests Required

| Test Type | What to Verify |
|-----------|----------------|
| Unit | Lua script returns correct Decision for allowed/rejected |
| Unit | Rejected requests NOT added to ZSET (ZCARD unchanged) |
| Unit | TTL refreshed on allowed requests |
| Unit | `KeyBuilder.EmailKey` returns HMAC-SHA256(normalized_email) |
| Unit | `KeyBuilder.IPKey` normalizes IPv4→/32, IPv6→/64 |
| Unit | Shadow mode logs but never blocks |
| Unit | `Decision` struct maps correctly to RateLimit headers |
| Unit | Fail-closed tier: in-process fallback token bucket activates |
| Unit | Fail-open tier: request allowed, metric incremented |
| Integration | Redis Lua script executes atomically under concurrent load |
| Integration | Fail-closed tier returns 503 when Redis unavailable |
| Integration | Fail-open tier allows request when Redis unavailable |
| Integration | 50 ms timeout enforced on Lua script |
| Integration | `Decision` → headers emitted on all responses (2xx, 4xx, 429) |
| Load | 10k req/s sustained with <5 ms p99 latency |
| Load | Burst 50k req/s for 30s, p99 < 20ms |
| Load | Soak 5k req/s for 1 hour, memory stable |
| Chaos | Redis primary failover → <5s recovery, no limit bypass |
| Chaos | Network partition (Redis unreachable) → fail-closed/closed per tier |
| Chaos | Clock skew (app server +5min vs Redis) → Lua uses Redis TIME, accurate |
| Chaos | Redis OOM (maxmemory) → eviction policy, limiter degrades gracefully |
| Security | Rate limit headers absent on 503 fail-closed responses |
| Security | Rejected requests not added to ZSET (no memory exhaustion via blocked traffic) |
| Security | Concurrent requests to same key: exactly `limit` allowed, no race conditions |

## Acceptance Criteria

Given a key with limit=10, window=1 minute
When 10 requests arrive within 30 seconds
Then all 10 return `Allowed=true`, `Remaining=9..0`
And the 11th request returns `Allowed=false`, `Remaining=0`, `RetryAfter>0`

Given a key with limit=5, window=1 minute
When 5 requests arrive at t=0, then 1 request at t=61s
Then the 6th request at t=61s returns `Allowed=true` (window rolled)

Given Redis is unavailable
When `Allow` is called on OTP Request tier
Then `503 Service Unavailable` returned within 50 ms
And in-process fallback token bucket activates

Given `RATE_LIMIT_MODE=shadow`
When limit exceeded
Then request allowed (no 429)
And `ratelimit_decisions_total{tier,result="would_block"}` incremented

Given concurrent requests to same key
When 100 goroutines call `Allow` simultaneously
Then exactly `limit` requests allowed, rest rejected
And no race conditions in ZSET count

## Open Questions

- Lua script caching strategy (`SCRIPT LOAD` + `EVALSHA` vs `EVAL`)?
- In-process fallback: token bucket params (capacity, refill rate)?
- Per-tier `RATE_LIMIT_MODE` override (e.g., `RATE_LIMIT_MODE_OTP_REQUEST=enforce`)?

## Links

- ADR: `docs/adr/0002-rate-limiting-strategy.md` §3, §8, §9
- Glossary: `specs/glossary.md` (entities: `rate_limit_key`, `sliding_window_log`, `decision`, `shadow_mode`)
- Phase 0: `specs/010-ratelimit-phase0-trust-boundary.md`