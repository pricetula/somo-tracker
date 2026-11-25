# SPEC-016: Rate Limiting Configuration Reference (Doppler)

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-016 |
| title | Rate Limiting Configuration Reference |
| status | draft |
| depends_on | docs/adr/0002-rate-limiting-strategy.md, specs/010-ratelimit-phase0-trust-boundary.md, specs/011-ratelimit-phase1-core-limiter.md, specs/012-ratelimit-phase2-otp-hardening.md, specs/013-ratelimit-phase3-bot-defense.md, specs/014-ratelimit-phase4-global-middleware-observability.md |
| owner | backend |

## Summary

Complete reference of all Doppler environment variables for rate limiting, bot defense, and trust boundary. Includes validation rules, defaults, and rollout flags.

---

## Configuration Categories

### 1. Trust Boundary & BFF Authentication (Phase 0)

| Variable | Required | Default | Validation | Description |
|----------|----------|---------|------------|-------------|
| `BFF_SERVICE_SECRET` | **Yes (prod)** | — | Non-empty, ≥32 chars | HMAC secret for BFF→API authentication. API rejects requests without valid `X-BFF-Service-Secret`. |
| `TRUSTED_PROXY_HOPS` | No | `1` | Integer ≥1 | Number of trusted proxy hops from right of `X-Forwarded-For`. **Must verify in staging**. |
| `TRUSTED_BFF_CIDRS` | No | `""` | Comma-separated CIDRs | Optional CIDR allowlist for BFF IPs. If set, API verifies peer IP ∈ CIDR before trusting `X-Client-IP`. |
| `RATE_LIMIT_KEY_HMAC_SECRET` | **Yes** | — | Non-empty, ≥32 chars | Secret for HMAC-SHA256(email) keys. Never logged. Rotate with key versioning. |

### 2. Core Limiter & Rollout (Phase 1)

| Variable | Required | Default | Validation | Description |
|----------|----------|---------|------------|-------------|
| `RATE_LIMIT_MODE` | No | `shadow` | `shadow` \| `enforce` | Global mode. `shadow`: log decisions, never block. `enforce`: return 429. Overridable per-tier. |
| `RATE_LIMIT_REDIS_TIMEOUT_MS` | No | `50` | Integer 10-500 | Lua script execution deadline. Exceed → fail-open/closed per tier. |

#### Per-Tier Mode Overrides (Optional)

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_MODE_OTP_REQUEST` | Inherits global | Override for OTP Request tier |
| `RATE_LIMIT_MODE_OTP_VERIFY` | Inherits global | Override for OTP Verify tier |
| `RATE_LIMIT_MODE_ONBOARD` | Inherits global | Override for Onboard tier |
| `RATE_LIMIT_MODE_SESSION` | Inherits global | Override for Session tier |
| `RATE_LIMIT_MODE_GLOBAL` | Inherits global | Override for Global API tier |

### 3. OTP Request Tier (Phase 2)

| Variable | Required | Default | Validation | Description |
|----------|----------|---------|------------|-------------|
| `RATE_LIMIT_OTP_REQUEST_EMAIL` | No | `3` | Integer 1-100 | Max requests per email (HMAC) per 15 min |
| `RATE_LIMIT_OTP_REQUEST_IP` | No | `30` | Integer 1-1000 | Max requests per IP per 15 min |
| `RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY` | No | `250` | Integer 1-300 | **Hard ceiling** for OTP emails/day. Must be ≤ Brevo free tier (300) minus headroom. |
| `RATE_LIMIT_OTP_RESERVED_RETURNING_PCT` | No | `40` | Integer 0-100 | % of daily budget reserved for returning verified users (40% = 100/day) |
| `RATE_LIMIT_OTP_DOMAIN_VELOCITY_PER_HOUR` | No | `20` | Integer 1-1000 | Max OTP requests per email domain per hour. Exceed → step-up challenge. |
| `RATE_LIMIT_RETURNING_USER_TTL_DAYS` | No | `30` | Integer 1-365 | TTL for returning-user marker in Redis (set on successful verify) |
| `RATE_LIMIT_DISPOSABLE_DOMAINS` | No | `""` | Comma-separated domains | Blocklist of disposable email domains. Requests → step-up/reject. |

### 4. OTP Verify Tier (Phase 2)

| Variable | Required | Default | Validation | Description |
|----------|----------|---------|------------|-------------|
| `OTP_VERIFY_MAX_ATTEMPTS_PER_CHALLENGE` | No | `5` | Integer 1-20 | Max verify attempts per challenge. 6th → challenge invalidated. |
| `RATE_LIMIT_OTP_VERIFY_EMAIL_PER_DAY` | No | `20` | Integer 1-500 | Max failed verifications per email per 24h. Exceed → exponential backoff. |
| `RATE_LIMIT_OTP_VERIFY_IP_PER_MIN` | No | `50` | Integer 1-1000 | Max verify requests per IP per minute. |

### 5. Onboard Tier (Phase 4)

| Variable | Required | Default | Validation | Description |
|----------|----------|---------|------------|-------------|
| `RATE_LIMIT_ONBOARD_USER_PER_MIN` | No | `5` | Integer 1-100 | Max onboard requests per user per minute |
| `RATE_LIMIT_ONBOARD_USER_PER_DAY` | No | `20` | Integer 1-500 | Max onboard requests per user per 24h |
| `RATE_LIMIT_ONBOARD_IP_PER_MIN` | No | `50` | Integer 1-1000 | Max onboard requests per IP per minute |

### 6. Session Tier (Phase 4)

| Variable | Required | Default | Validation | Description |
|----------|----------|---------|------------|-------------|
| `RATE_LIMIT_SESSION_USER_PER_MIN` | No | `30` | Integer 1-500 | Max session requests per user per minute |
| `RATE_LIMIT_SESSION_IP_PER_MIN` | No | `100` | Integer 1-1000 | Max session requests per IP per minute |

### 7. Global API Tier (Phase 4)

| Variable | Required | Default | Validation | Description |
|----------|----------|---------|------------|-------------|
| `RATE_LIMIT_GLOBAL_USER_PER_MIN` | No | `300` | Integer 1-5000 | Max requests per authenticated user per minute |
| `RATE_LIMIT_GLOBAL_IP_PER_MIN` | No | `600` | Integer 1-10000 | Max requests per unauthenticated IP per minute |

### 8. Bot Defense (Phase 3)

| Variable | Required | Default | Validation | Description |
|----------|----------|---------|------------|-------------|
| `POW_SECRET` | **Yes (prod)** | — | Non-empty, ≥32 chars | HMAC secret for PoW challenge signing. |
| `POW_BASE_DIFFICULTY` | No | `4` | Integer 1-20 | Leading zero bits for Low-risk PoW. Target: ~50-200ms solve on mobile. |
| `POW_CHALLENGE_TTL_SECONDS` | No | `120` | Integer 30-600 | PoW challenge validity (2 min default). |
| `BOT_RISK_STEPUP_THRESHOLD` | No | `50` | Integer 0-100 | Risk score ≥ this → Medium tier (increased PoW + interactive). |
| `BOT_RISK_BLOCK_THRESHOLD` | No | `85` | Integer 0-100 | Risk score ≥ this → High tier (WebAuthn/deferred). |
| `BOT_RISK_CRITICAL_THRESHOLD` | No | `95` | Integer 0-100 | Risk score ≥ this → Critical (circuit breaker). |
| `BFF_DWELL_COOKIE_NAME` | No | `__Host-dwell_ts` | String | BFF pre-session timing cookie name. |
| `BFF_DWELL_SUB_HUMAN_MS` | No | `800` | Integer 100-5000 | Dwell time < this → sub-human (high risk). |
| `BFF_DWELL_STALE_MS` | No | `14400000` | Integer 60000-86400000 | Dwell time > this → stale (medium risk, 4h default). |
| `BOT_HONEYPOT_FIELD_PREFIX` | No | `hp_` | String | Prefix for randomized honeypot field names. |
| `RATE_LIMIT_CRITICAL_BUDGET_PCT` | No | `90` | Integer 50-99 | Global OTP budget > this % → auto-Critical risk. |

### 9. Observability (Phase 4)

| Variable | Required | Default | Validation | Description |
|----------|----------|---------|------------|-------------|
| `RATE_LIMIT_METRICS_ENABLED` | No | `true` | `true` \| `false` | Enable Prometheus metrics export. |
| `RATE_LIMIT_ALERTS_ENABLED` | No | `true` | `true` \| `false` | Enable PrometheusRule alerts. |

---

## Validation Rules (Startup)

The application **fails fast** on invalid configuration:

1. **Required in production**: `BFF_SERVICE_SECRET`, `POW_SECRET`, `RATE_LIMIT_KEY_HMAC_SECRET`
2. **Numeric bounds**: All integer variables validated against min/max
3. **Enum values**: `RATE_LIMIT_MODE` and per-tier overrides must be `shadow` or `enforce`
4. **Budget sanity**: `RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY` ≤ 300 (Brevo free tier)
5. **Threshold ordering**: `STEPUP < BLOCK < CRITICAL` (50 < 85 < 95)
6. **Secret strength**: ≥32 chars, high entropy (checked via `crypto/rand` entropy estimate)

---

## Rollout Sequence (Doppler Changes)

| Step | Change | Verification |
|------|--------|--------------|
| 1 | Deploy code with `RATE_LIMIT_MODE=shadow` | Metrics: `ratelimit_decisions_total{result="would_block"}` |
| 2 | `RATE_LIMIT_MODE_OTP_VERIFY=enforce` | 429s on verify, no user impact |
| 3 | `RATE_LIMIT_MODE_OTP_REQUEST=enforce` | 429s on request, monitor Brevo budget |
| 4 | `RATE_LIMIT_MODE_ONBOARD=enforce` | 429s on onboard, monitor conversions |
| 5 | `RATE_LIMIT_MODE_SESSION=enforce` | Tune for SPA polling (shadow first) |
| 6 | `RATE_LIMIT_MODE_GLOBAL=enforce` | Full enforcement |
| 7 | `RATE_LIMIT_MODE=enforce` (global) | All tiers enforced |

---

## Secret Rotation

| Secret | Rotation Frequency | Procedure |
|--------|-------------------|-----------|
| `BFF_SERVICE_SECRET` | 90 days | Dual-secret: add new to Doppler, deploy BFF+API with both, remove old |
| `POW_SECRET` | 90 days | Deploy new secret, existing challenges expire in 2 min |
| `RATE_LIMIT_KEY_HMAC_SECRET` | 180 days | **Key versioning**: prefix keys with `v2:`, migrate gradually |

---

## Environment-Specific Defaults

| Variable | Development | Staging | Production |
|----------|-------------|---------|------------|
| `RATE_LIMIT_MODE` | `shadow` | `shadow` | `enforce` (phased) |
| `TRUSTED_PROXY_HOPS` | `1` | **Verify** | **Verify** |
| `TRUSTED_BFF_CIDRS` | `""` | Render staging CIDRs | Render prod CIDRs |
| `TURNSTILE_SECRET_KEY` | `""` (skip) | Test key | Prod key |
| `POW_BASE_DIFFICULTY` | `1` (trivial) | `4` | `4` |
| `RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY` | `1000` | `250` | `250` |

---

## Links

- ADR: `docs/adr/0002-rate-limiting-strategy.md` §9
- Phase 0: `specs/010-ratelimit-phase0-trust-boundary.md`
- Phase 1: `specs/011-ratelimit-phase1-core-limiter.md`
- Phase 2: `specs/012-ratelimit-phase2-otp-hardening.md`
- Phase 3: `specs/013-ratelimit-phase3-bot-defense.md`
- Phase 4: `specs/014-ratelimit-phase4-global-middleware-observability.md`