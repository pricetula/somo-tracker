# SPEC-012: Rate Limiting Phase 2 — OTP Hardening & Brevo Budget Protection

Endpoint: `POST /api/v1/auth/otp/request`, `POST /api/v1/auth/otp/verify`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-012 |
| title | Rate Limiting Phase 2 — OTP Hardening & Brevo Budget Protection |
| status | draft |
| depends_on | docs/adr/0002-rate-limiting-strategy.md, specs/011-ratelimit-phase1-core-limiter.md, specs/001-auth-otp-request.md, specs/glossary.md |
| api_endpoint | POST /api/v1/auth/otp/request, POST /api/v1/auth/otp/verify |
| owner | backend |

## Summary

Implement **strict OTP-specific controls** on top of the core limiter: per-challenge attempt caps, account-level failure caps, **Brevo 250/day global circuit breaker with 40% returning-user reservation**, domain velocity limiting, and disposable domain handling. This phase directly protects the email budget and prevents credential cracking (OAT-007).

## Tier: OTP Request (`POST /api/v1/auth/otp/request`)

### Rate Limit Keys & Quotas

| Key Type | Identifier | Quota | Window | Notes |
|----------|------------|-------|--------|-------|
| `email` | HMAC-SHA256(normalized_email) | 3 | 15 min | Per-email cap |
| `IP` | Normalized IP (/32 or /64) | 30 | 15 min | Loose ceiling |
| `global` | Constant `global:otp_send` | 250 | 24 hr | **Hard circuit breaker** |

### Returning-User Reservation (Within 250/day Global Budget)

- **Reservation**: 40% of daily budget = 100 sends/day reserved for returning verified users
- **Returning user marker**: Redis key `rl:returning_user:{email_hmac}` set on successful `/otp/verify`
  - TTL: 30 days (configurable `RATE_LIMIT_RETURNING_USER_TTL_DAYS`)
- **Allocation logic**:
  1. Check if email hash has `returning_user` marker
  2. If yes → draw from reserved 100/day pool
  3. If no → draw from general 150/day pool
  4. If respective pool exhausted → `429 RATE_LIMITED` (no cross-pool borrowing)

### Domain Velocity Limit

- **Threshold**: 20 OTP requests per hour per email domain
- **Key**: `rl:otp_request:domain:{normalized_domain}`
- **Action on exceed**: Require step-up challenge (PoW difficulty increase) on subsequent requests to that domain
- **Normalization**: Extract domain from email, lowercase, IDNA to ASCII

### Disposable Domain Handling

- **Source**: Maintained blocklist (file or Redis set `rl:disposable_domains`)
- **Action**: Requests using disposable domains → step-up challenge or reject on `/otp/request` and `/onboard`
- **Refresh**: Periodic job updates blocklist from upstream source

### Email Normalization (Before HMAC)

1. Trim whitespace
2. Lowercase
3. Unicode NFC normalization
4. Strip `+tag` aliases **only** for known safe providers (Gmail, Outlook, Yahoo, iCloud, Proton)
5. Apply HMAC-SHA256(`RATE_LIMIT_KEY_HMAC_SECRET`, normalized_email)

## Tier: OTP Verify (`POST /api/v1/auth/otp/verify`)

### Rate Limit Keys & Quotas

| Key Type | Identifier | Quota | Window | Notes |
|----------|------------|-------|--------|-------|
| `challengeID` | Challenge UUID | 5 attempts | Per challenge | 6th attempt invalidates challenge |
| `email` | HMAC-SHA256(normalized_email) | 20 failed | 24 hr | Account-level failure cap |
| `IP` | Normalized IP | 50 | 1 min | Loose ceiling |

### Per-Challenge Attempt Cap

- **Max attempts**: 5 per `challengeID` (configurable `OTP_VERIFY_MAX_ATTEMPTS_PER_CHALLENGE`)
- **On 6th attempt**: Challenge immediately invalidated (`used_at = NOW()`)
- **Response**: `401 INVALID_TOKEN` (same as wrong code — no enumeration)

### Account-Level Failure Cap

- **Max failed verifications**: 20 per email per 24 hours
- **On exceed**: Exponential backoff cool-down (1h, 4h, 12h, 24h...)
- **Key**: `rl:otp_verify:email:{email_hmac}:failures`
- **Reset**: Successful verification resets failure count to 0

### Challenge Invalidation Triggers

1. `attempt_count >= max_attempts` (5)
2. `expires_at < NOW()` (10 min TTL)
3. New OTP requested for same email (invalidates previous challenge)
4. Successful verification (marks `used_at`)

## Response Headers (All OTP Endpoints)

```http
RateLimit-Limit: <limit>
RateLimit-Remaining: <remaining>
RateLimit-Reset: <seconds_until_reset>
X-RateLimit-Limit: <limit>
X-RateLimit-Remaining: <remaining>
X-RateLimit-Reset: <unix_timestamp>
Retry-After: <seconds>          # On 429 only
```

### 429 Response Body (RFC 9457 — No Key Disclosure)

```json
{
  "type": "https://api.example.com/problems/rate-limited",
  "title": "Too many requests",
  "status": 429,
  "code": "rate_limited",
  "retry_after_seconds": 42
}
```
> **Never disclose** which key (`email`, `IP`, `challengeID`, `global`) triggered the limit.

## Configuration (Doppler)

```bash
# OTP Request
RATE_LIMIT_OTP_REQUEST_EMAIL=3                    # per 15m
RATE_LIMIT_OTP_REQUEST_IP=30                      # per 15m
RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY=250            # hard ceiling
RATE_LIMIT_OTP_RESERVED_RETURNING_PCT=40          # 100/day reserved
RATE_LIMIT_OTP_DOMAIN_VELOCITY_PER_HOUR=20        # per domain
RATE_LIMIT_RETURNING_USER_TTL_DAYS=30             # returning user marker TTL

# OTP Verify
OTP_VERIFY_MAX_ATTEMPTS_PER_CHALLENGE=5
RATE_LIMIT_OTP_VERIFY_EMAIL_PER_DAY=20
RATE_LIMIT_OTP_VERIFY_IP_PER_MIN=50

# Disposable domains (comma-separated or Redis set)
RATE_LIMIT_DISPOSABLE_DOMAINS=tempmail.com,10minutemail.com,...

# Global
RATE_LIMIT_REDIS_TIMEOUT_MS=50
RATE_LIMIT_MODE=shadow                            # shadow | enforce
```

## Implementation Files

| File | Responsibility |
|------|----------------|
| `internal/auth/handler/handler.go` | Wire limiters into `RequestOTP`, `VerifyOTP`; emit headers; RFC 9457 429 |
| `internal/auth/service/service.go` | Challenge invalidation logic; returning user marker on success; failure count tracking |
| `internal/platform/ratelimit/key_builder.go` | `EmailKey` (HMAC), `DomainKey`, `GlobalKey`, `ChallengeKey` |
| `internal/platform/ratelimit/disposable.go` | Disposable domain blocklist loader/checker |
| `internal/platform/config/config.go` | All OTP-specific Doppler fields + validation |

## Business Rules

### RequestOTP

1. Check global budget first (fail-closed: 503 if Redis down)
2. Check returning-user marker → allocate from reserved/general pool
3. Check email limit (3/15m)
4. Check IP limit (30/15m)
5. Check domain velocity (20/hr) → flag for step-up if exceeded
6. Check disposable domain → flag for step-up/reject
7. All checks in shadow mode unless `RATE_LIMIT_MODE=enforce`
8. On allow: create challenge, queue email, return generic 200

### VerifyOTP

1. Lookup challenge by ID (401 if not found)
2. Check expiry, attempt count, used_at (401 if any fail)
3. Verify HMAC (constant-time)
4. On success:
   - Mark challenge used
   - Set returning-user marker in Redis (TTL 30d)
   - Reset email failure count to 0
   - Create session, return token
5. On failure:
   - Increment challenge attempt_count
   - Increment email failure count (24hr window)
   - If challenge attempts ≥ 5 → invalidate challenge
   - If email failures ≥ 20 → exponential backoff flag
   - Return 401 INVALID_TOKEN (same for all failure types)

## Tests Required

| Test Type | What to Verify |
|-----------|----------------|
| Unit | Email normalization (lowercase, trim, NFC, +tag stripping for safe providers) |
| Unit | HMAC-SHA256 email key derivation |
| Unit | Returning-user reservation logic (reserved vs general pool) |
| Unit | Domain velocity key construction |
| Unit | Disposable domain detection |
| Unit | Challenge invalidation on 6th attempt |
| Unit | Account failure cap + exponential backoff |
| Unit | Global budget allocation: reserved vs general pool math |
| Unit | Challenge invalidation triggers (expiry, new OTP, success, max attempts) |
| Integration | Global budget exhaustion → 429 (general pool) |
| Integration | Global budget exhaustion → returning user still allowed (reserved pool) |
| Integration | Domain velocity exceed → step-up challenge flag |
| Integration | Disposable domain → step-up/reject |
| Integration | Challenge invalidated after new OTP request |
| Integration | Successful verify → returning user marker set, failure count reset |
| Integration | Failed verify → failure count incremented |
| Integration | Fail-closed on Redis down: OTP Request/Verify → 503, fallback activates |
| Integration | Budget pools exhausted independently (no cross-pool borrowing) |
| Contract | 429 response never discloses limiting key |
| Contract | All OTP endpoints emit RateLimit headers |
| Contract | 401 INVALID_TOKEN identical for all failure types (no enumeration) |
| Contract | RFC 9457 problem details on 429 (`code: rate_limited`, `retry_after_seconds`) |
| Load | 250 OTP sends/day — global budget respected under concurrency |
| Load | Burst 100 OTP requests/min — per-email/IP limits enforced |
| Chaos | Redis down during budget check → fail-closed 503 |
| Chaos | Redis failover mid-OTP flow → no limit bypass, no duplicate sends |
| Security | Account enumeration via 429: identical response for exist/non-exist emails |
| Security | Account enumeration via timing: constant-time responses |
| Security | Email never appears in Redis keys/logs (only HMAC-SHA256) |
| Security | Challenge ID unguessable (UUID v4) |

## Acceptance Criteria

Given global OTP send budget = 250/day, reserved = 40% (100)
When 150 new-user OTP requests sent in a day
Then 151st new-user request returns `429 RATE_LIMITED`
And returning user (with marker) can still send up to 100

Given a returning user with valid marker
When global budget exhausted (250 sent)
Then returning user requests still succeed (up to 100 reserved)

Given `challengeID` with 5 failed verify attempts
When 6th verify attempt made
Then challenge invalidated (`used_at` set)
And response `401 INVALID_TOKEN`

Given email with 20 failed verifications in 24h
When 21st verify attempted
Then exponential backoff triggered
And response `401 INVALID_TOKEN` (or `429` with backoff hint)

Given domain `example.com` with 20 OTP requests in 1 hour
When 21st request to `user@example.com` arrives
Then step-up challenge required flag set

Given email `test@tempmail.com` (disposable domain)
When OTP request made
Then step-up challenge required or rejected

## Open Questions

- Disposable domain list source: open source file? Vendor API? Self-hosted?
- Exponential backoff formula: `2^failure_count * base_minutes`?
- Returning user marker TTL: 30 days optimal?
- Cross-pool borrowing: ever allow general→reserved or vice versa?

## Links

- ADR: `docs/adr/0002-rate-limiting-strategy.md` §4, §5, §9
- Glossary: `specs/glossary.md` (entities: `otp_challenge`, `returning_user_marker`, `brevo_budget`, `domain_velocity`, `disposable_domain`)
- Phase 1: `specs/011-ratelimit-phase1-core-limiter.md`
- Auth Specs: `specs/001-auth-otp-request.md`, `specs/002-auth-otp-verify.md`