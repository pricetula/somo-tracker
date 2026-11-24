# SPEC-014: Rate Limiting Phase 4 — Global Middleware, Onboard/Session Tiers & Observability

Endpoint: `ALL /api/v1/*`, `POST /api/v1/auth/onboard`, `GET /api/v1/auth/session`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-014 |
| title | Rate Limiting Phase 4 — Global Middleware, Onboard/Session Tiers & Observability |
| status | draft |
| depends_on | docs/adr/0002-rate-limiting-strategy.md, specs/011-ratelimit-phase1-core-limiter.md, specs/012-ratelimit-phase2-otp-hardening.md, specs/013-ratelimit-phase3-bot-defense.md, specs/glossary.md |
| api_endpoint | ALL /api/v1/*, POST /api/v1/auth/onboard, GET /api/v1/auth/session |
| owner | backend |

## Summary

Wire the **global rate limit middleware** on all `/api/v1/*` routes, implement **Onboard** and **Session** tier limiters, and add **comprehensive Prometheus metrics + alerting**. This phase completes the defense-in-depth layer and provides operational visibility.

---

## 1. Global API Rate Limit Middleware

### Placement in Middleware Chain (after Phase 0)

```
1. ExemptPaths("/healthz", "/readyz", webhooks...)
2. VerifyBFFServiceSecret()
3. (Optional) VerifyTrustedBFFCIDR()
4. ExtractClientIP()
5. ResolveSession() → attaches userID to context
6. **GlobalRateLimitMiddleware()** ← NEW
7. Route-specific limiters (OTP, Onboard, Session...)
```

### Global Tier Quotas

| Key | Quota | Window | Scope |
|-----|-------|--------|-------|
| `userID` | 300 | 1 min | Authenticated requests (keys on backend-resolved `userID`) |
| `IP` | 600 | 1 min | Unauthenticated requests (keys on `X-Client-IP`) |

### Key Resolution Logic

```go
func globalKeyResolver(c fiber.Ctx) string {
    // Priority 1: Authenticated user (from session middleware)
    if userID := c.Locals("userID"); userID != nil {
        return keyBuilder.UserKey(userID.(string))
    }
    // Priority 2: Unauthenticated fallback to IP
    return keyBuilder.IPKey(c.Locals("clientIP").(string))
}
```

### Headers Emitted on All Responses

```http
RateLimit-Limit: 300
RateLimit-Remaining: 287
RateLimit-Reset: 45
X-RateLimit-Limit: 300
X-RateLimit-Remaining: 287
X-RateLimit-Reset: 1699900000
```

---

## 2. Onboard Tier (`POST /api/v1/auth/onboard`)

### Rate Limit Keys & Quotas

| Key | Quota | Window | Notes |
|-----|-------|--------|-------|
| `userID` | 5 | 1 min | Burst protection on expensive writes |
| `userID` | 20 | 24 hr | Daily cap on org/school/user creation |
| `IP` | 50 | 1 min | Loose ceiling |

### Requirements

- Requires valid opaque session → `userID` resolved in backend (Phase 0)
- Bot defense: **Baseline signals required** (Turnstile action=`onboard`, dwell cookie, honeypot, PoW)
- Expensive operations: Creates organization, school, user, upgrades session
- Fail-closed: Redis unavailable → `503 Service Unavailable`

---

## 3. Session Tier (`GET /api/v1/auth/session`)

### Rate Limit Keys & Quotas

| Key | Quota | Window | Notes |
|-----|-------|--------|-------|
| `userID` | 30 | 1 min | Multi-tab SPA polling |
| `IP` | 100 | 1 min | Unauthenticated fallback |

### Requirements

- Lightweight check — session validation already happens in handler
- Tuned during shadow mode for multi-tab SPA polling patterns
- Fail-open: Redis unavailable → allow, log `ratelimit_failopen_total{session}`

---

## 4. Prometheus Metrics

### Naming Convention

- **Prefix**: `ratelimit_`, `otp_`, `bot_`, `bff_`
- **Labels**: Never raw IP or email — use tier, result, provider, level
- **Cardinality**: Keep low (tier: 5 values, result: 2-3, provider: 3-4)

### Metrics List

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `bff_auth_rejected_total` | Counter | `reason` (missing_secret, invalid_secret, invalid_cidr, missing_client_ip) | Direct public-URL probes blocked |
| `ratelimit_decisions_total` | Counter | `tier` (otp_request, otp_verify, onboard, session, global), `result` (allowed, denied, would_block) | All limiter decisions |
| `ratelimit_429_total` | Counter | `tier` | 429 responses returned |
| `ratelimit_check_duration_seconds` | Histogram | `tier` | Lua script execution latency |
| `ratelimit_redis_errors_total` | Counter | `tier`, `error` (timeout, connection, script) | Redis errors by tier |
| `ratelimit_failopen_total` | Counter | `tier` | Fail-open events (session, global) |
| `ratelimit_failclosed_total` | Counter | `tier` | Fail-closed 503s (OTP, onboard) |
| `otp_sends_total` | Counter | `pool` (reserved, general) | OTP emails sent by budget pool |
| `otp_verify_failures_total` | Counter | `reason` (expired, used, wrong_code, max_attempts, challenge_invalidated) | Verification failure breakdown |
| `otp_challenge_invalidated_total` | Counter | `reason` (max_attempts, expiry, new_otp, success) | Challenge invalidation reasons |
| `bot_challenge_total` | Counter | `provider` (turnstile, pow, webauthn), `result` (issued, solved, failed, expired) | Bot challenge lifecycle |
| `bot_risk_score` | Histogram | `tier` (otp_request, onboard) | Risk score distribution (0-100) |
| `bot_stepup_total` | Counter | `level` (medium, high, critical) | Escalation events |
| `brevo_budget_utilization` | Gauge | `pool` (reserved, general) | Daily budget usage % |

### Histogram Buckets

- `ratelimit_check_duration_seconds`: `[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0]`
- `bot_risk_score`: `[10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 100]`

---

## 5. Alerting Rules

### Critical Alerts (Page Immediately)

| Alert | Condition | Severity | Runbook |
|-------|-----------|----------|---------|
| `RatelimitAuth429RateHigh` | `rate(ratelimit_429_total{tier=~"otp_request|otp_verify|onboard"}[10m]) > 0.05` | Critical | Runbook: auth-429-spike |
| `RatelimitRedisErrorsHigh` | `rate(ratelimit_redis_errors_total[5m]) > 0.01` | Critical | Runbook: redis-errors |
| `RatelimitFailclosedTriggered` | `increase(ratelimit_failclosed_total[5m]) > 0` | Critical | Runbook: fail-closed |
| `BrevoBudgetUtilizationHigh` | `brevo_budget_utilization > 0.7` | Critical | Runbook: brevo-budget |

### Warning Alerts (Ticket + Slack)

| Alert | Condition | Severity | Runbook |
|-------|-----------|----------|---------|
| `RatelimitFailopenTriggered` | `increase(ratelimit_failopen_total[5m]) > 0` | Warning | Runbook: fail-open |
| `BotRiskScoreShift` | `histogram_quantile(0.95, bot_risk_score) dropped > 20% over 1h` | Warning | Runbook: bot-solver-detected |
| `BotStepupCriticalRate` | `rate(bot_stepup_total{level="critical"}[10m]) > 0.01` | Warning | Runbook: bot-critical |

---

## 6. BFF Coarse Edge Shedding (Reference)

Implemented in BFF (Next.js), not Go API:

| Route Pattern | Quota | Window | Implementation |
|---------------|-------|--------|----------------|
| `/api/auth/*` | 120 | 1 min | In-memory or Redis token bucket |
| `/api/*` (general) | 900 | 1 min | In-memory or Redis token bucket |

- Drops volumetric floods before BFF→API connections consumed
- Returns `429` with `Retry-After` to browser
- Logs `bff_edge_shed_total{route_pattern}`

---

## Configuration (Doppler)

```bash
# Global API
RATE_LIMIT_GLOBAL_USER_PER_MIN=300
RATE_LIMIT_GLOBAL_IP_PER_MIN=600

# Onboard
RATE_LIMIT_ONBOARD_USER_PER_MIN=5
RATE_LIMIT_ONBOARD_USER_PER_DAY=20
RATE_LIMIT_ONBOARD_IP_PER_MIN=50

# Session
RATE_LIMIT_SESSION_USER_PER_MIN=30
RATE_LIMIT_SESSION_IP_PER_MIN=100

# Observability
RATE_LIMIT_METRICS_ENABLED=true
RATE_LIMIT_ALERTS_ENABLED=true
```

---

## Implementation Files

| File | Responsibility |
|------|----------------|
| `internal/app/app.go` | Global middleware wiring (after session resolution) |
| `internal/platform/ratelimit/global.go` | Global tier limiter with userID/IP key resolver |
| `internal/auth/handler/handler.go` | Onboard/Session tier limiters + bot defense integration |
| `internal/platform/metrics/ratelimit.go` | Prometheus metric definitions + registration |
| `internal/platform/config/config.go` | Global/Onboard/Session Doppler fields |
| `monitoring/alerts/ratelimit.yaml` | PrometheusRule CRDs for alerts |

---

## Tests Required

| Test Type | What to Verify |
|-----------|----------------|
| Unit | Global key resolver: userID > IP priority |
| Unit | Onboard tier: userID 5/min, 20/day, IP 50/min |
| Unit | Session tier: userID 30/min, IP 100/min |
| Unit | Prometheus metric definitions: correct names, labels, types |
| Unit | Alert rule expressions: valid PromQL, correct thresholds |
| Integration | Global middleware emits RateLimit headers on all responses |
| Integration | Authenticated request keys on userID (NAT-safe) |
| Integration | Unauthenticated request keys on IP |
| Integration | Fail-open on session/global: request allowed, metric incremented |
| Integration | Fail-closed on onboard: 503 when Redis down |
| Integration | Metrics emitted with correct labels (no PII) |
| Integration | Alert rules fire on threshold breach |
| Integration | Global middleware position: after session resolution, before route handlers |
| Contract | `RateLimit-*` + `X-RateLimit-*` headers on every `/api/v1/*` response |
| Contract | `Retry-After` only on 429 responses |
| Contract | No rate limit headers on 503 fail-closed responses |
| Load | Global limiter: 10k req/s, p99 < 5ms |
| Load | School simulation: 200 users, 1 IP, 10 req/min each → all succeed |
| Load | Soak: 5k req/s for 1 hour, metrics stable |
| Chaos | Redis restart: metrics recover, no stale limits |
| Chaos | Redis partition during authenticated request → fail-open, metric incremented |
| Chaos | Redis partition during onboard → fail-closed 503, metric incremented |
| Security | NAT school simulation: 200 users, 1 IP, per-user limits independent |
| Security | Unauthenticated IP limit: 600/min enforced, no userID leakage |
| Security | Metrics labels never contain raw IP or email (only tier, result) |

---

## Acceptance Criteria

Given an authenticated user making requests to `/api/v1/schools`
When 300 requests made in 1 minute
Then 301st request returns `429` with `RateLimit-Limit: 300`
And `ratelimit_decisions_total{tier="global",result="denied"}` incremented

Given an unauthenticated request to `/api/v1/public-info`
When 600 requests made in 1 minute from same IP
Then 601st request returns `429` with `RateLimit-Limit: 600`

Given a school with 200 users behind 1 IP
When each user makes 10 requests/minute
Then all succeed (userID-keyed, 300/min each)
And IP-keyed global limit (600) not exceeded

Given `POST /api/v1/auth/onboard` with valid session
When 5 requests in 1 minute
Then 6th returns `429` (userID burst limit)

Given Redis unavailable
When `GET /api/v1/auth/session` called
Then request succeeds (fail-open)
And `ratelimit_failopen_total{tier="session"}` incremented

Given Redis unavailable
When `POST /api/v1/auth/onboard` called
Then `503 Service Unavailable` (fail-closed)
And `ratelimit_failclosed_total{tier="onboard"}` incremented

---

## Open Questions

- Alert notification channels: PagerDuty, Slack, email?
- Metric retention: 30d? 90d?
- Dashboard: Grafana dashboard for rate limit overview?
- BFF edge shedding: in-memory vs Redis for BFF?

---

## Links

- ADR: `docs/adr/0002-rate-limiting-strategy.md` §4, §7, §10, §11
- Glossary: `specs/glossary.md` (entities: `global_rate_limit`, `onboard_tier`, `session_tier`, `prometheus_metrics`, `alert_rules`)
- Phase 1: `specs/011-ratelimit-phase1-core-limiter.md`
- Phase 0: `specs/010-ratelimit-phase0-trust-boundary.md`