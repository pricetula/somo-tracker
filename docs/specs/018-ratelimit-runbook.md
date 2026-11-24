# SPEC-018: Rate Limiting Runbook

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-018 |
| title | Rate Limiting Runbook — Operations & Incident Response |
| status | draft |
| depends_on | docs/adr/0002-rate-limiting-strategy.md, specs/014-ratelimit-phase4-global-middleware-observability.md, specs/016-ratelimit-config-reference.md, specs/017-ratelimit-test-plan.md |
| owner | backend / on-call |

## Summary

Operational procedures for rate limiting system: alert responses, fail-open/closed handling, secret rotation, capacity planning, and rollback procedures.

---

## Alert Response Procedures

### 🔴 Critical: `RatelimitAuth429RateHigh`
**Condition**: Auth tier 429 rate > 5% over 10 min
**Impact**: Legitimate users blocked from login/onboard
**Runbook**:

1. **Check dashboard**: `ratelimit_429_total` by tier (otp_request, otp_verify, onboard)
2. **Identify tier**: Which tier is spiking?
3. **Check risk score**: `bot_risk_score` histogram — shift to low scores = solver farm
4. **Check Brevo budget**: `brevo_budget_utilization` — near 70% = budget pressure
5. **Immediate actions**:
   - If bot attack: Increase `POW_BASE_DIFFICULTY` by 2 (Doppler, no deploy)
   - If budget exhaustion: Verify `RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY` not exceeded
   - If false positive: Temporarily increase tier limit 2x (Doppler)
6. **Notify**: #on-call, #security
7. **Post-incident**: Analyze attack pattern, update disposable domains, tune risk weights

### 🔴 Critical: `RatelimitRedisErrorsHigh`
**Condition**: Redis error rate > 1% over 5 min
**Impact**: Fail-closed tiers (OTP, onboard) returning 503
**Runbook**:

1. **Check Redis**: `redis-cli INFO` — memory, connections, replication lag
2. **Check limiter logs**: `ratelimit_redis_errors_total` by error type (timeout, connection, script)
3. **Immediate actions**:
   - If timeout: Increase `RATE_LIMIT_REDIS_TIMEOUT_MS` to 100ms (Doppler)
   - If OOM: Check `maxmemory-policy`, consider scaling Redis
   - If connection storm: Check BFF edge shedding (should drop before API)
4. **Failover**: If primary down, verify replica promotion
5. **Notify**: #on-call, #infra

### 🔴 Critical: `RatelimitFailclosedTriggered`
**Condition**: Any fail-closed 503 in last 5 min
**Impact**: OTP/onboard completely unavailable
**Runbook**:

1. **This should be rare** — indicates Redis completely unavailable
2. **Check in-process fallback**: Logs should show `fallback_token_bucket_active`
3. **Immediate actions**:
   - Verify Redis connectivity from API pods
   - If Redis down: Scale Redis, check network policies
   - If fallback overwhelmed: Increase fallback bucket capacity (code change)
4. **Escalate**: #infra, #backend-lead

### 🔴 Critical: `BrevoBudgetUtilizationHigh`
**Condition**: Daily OTP budget > 70% (175/250)
**Impact**: Risk of hitting 250/day hard ceiling
**Runbook**:

1. **Check pools**: `otp_sends_total{pool="general"}` vs `reserved`
2. **Identify cause**: New user surge? Bot attack? Email loop?
3. **Immediate actions**:
   - If bot: Enable Critical risk mode (auto at 90%), increase PoW difficulty
   - If legitimate surge: Request Brevo upgrade (business decision)
   - If email loop: Check `email_outbox` for stuck records
4. **Monitor**: Hourly until < 50%

### 🟡 Warning: `RatelimitFailopenTriggered`
**Condition**: Fail-open event on session/global tier
**Impact**: Rate limiting degraded for authenticated users
**Runbook**:

1. **Check Redis**: Same as `RedisErrorsHigh`
2. **Impact**: Low — fail-open prioritizes availability
3. **Action**: Investigate root cause, fix Redis
4. **Track**: Ticket for follow-up

### 🟡 Warning: `BotRiskScoreShift`
**Condition**: p95 risk score drops > 20% over 1 hour
**Impact**: Solver farm likely adapted (solving PoW faster)
**Runbook**:

1. **Check PoW solve times**: `bot_challenge_total{provider="pow",result="solved"}` latency
2. **Immediate actions**:
   - Increase `POW_BASE_DIFFICULTY` by 2
   - Add new disposable domains to blocklist
   - Consider enabling WebAuthn for High tier
2. **Notify**: #security

### 🟡 Warning: `BotStepupCriticalRate`
**Condition**: Critical escalations > 1% over 10 min
**Impact**: Legitimate users hitting circuit breaker
**Runbook**:

1. **Check global budget**: `brevo_budget_utilization`
2. **Check risk distribution**: `bot_risk_score` by tier
3. **Action**: If budget OK, tune risk weights (reduce IP reputation weight)

---

## Fail Mode Procedures

### Fail-Closed (OTP Request, OTP Verify, Onboard)

**Behavior**: Redis unavailable → `503 Service Unavailable` + in-process token bucket

**Token Bucket Params** (code constants, configurable via Doppler if needed):
- Capacity: 10 requests
- Refill: 1 request/second per IP
- Scope: Per IP (/32 or /64)

**Verification**:
```bash
# Check fallback active
grep "fallback_token_bucket_active" /var/log/api.log | tail -5

# Verify 503 responses
curl -H "X-BFF-Service-Secret: $SECRET" -H "X-Client-IP: 1.2.3.4" \
  https://api.example.com/api/v1/auth/otp/request
# Should return 503 with Retry-After
```

**Recovery**: Automatic when Redis recovers. Fallback deactivates, Lua script resumes.

### Fail-Open (Session, Global API)

**Behavior**: Redis unavailable → allow request, increment `ratelimit_failopen_total`, alert

**Verification**:
```bash
# Check fail-open metrics
curl -s http://api:9090/metrics | grep ratelimit_failopen_total

# Request should succeed
curl -H "Authorization: Bearer $TOKEN" https://api.example.com/api/v1/auth/session
# Returns 200, no RateLimit headers
```

**Recovery**: Automatic. Metrics stop incrementing when Redis recovers.

---

## Secret Rotation Procedures

### BFF_SERVICE_SECRET (Every 90 Days)

**Zero-Downtime Dual-Secret Rotation**:

1. **Generate new secret**: `openssl rand -hex 32`
2. **Add to Doppler** as `BFF_SERVICE_SECRET_V2` (both BFF and API projects)
3. **Deploy API** with support for both secrets:
   ```go
   func verifySecret(provided string) bool {
       return subtle.ConstantTimeCompare([]byte(provided), []byte(os.Getenv("BFF_SERVICE_SECRET"))) == 1 ||
              subtle.ConstantTimeCompare([]byte(provided), []byte(os.Getenv("BFF_SERVICE_SECRET_V2"))) == 1
   }
   ```
4. **Deploy BFF** sending new secret in `X-BFF-Service-Secret`
5. **Verify**: No 403 spikes in `bff_auth_rejected_total`
6. **Remove old**: Delete `BFF_SERVICE_SECRET_V2` from Doppler, rename new to primary
7. **Deploy cleanup**: Remove dual-secret code

### POW_SECRET (Every 90 Days)

1. **Generate new secret**: `openssl rand -hex 32`
2. **Add to Doppler** as `POW_SECRET_V2`
3. **Deploy API** with dual-secret verification (challenges signed with either)
4. **Wait 5 minutes** (max challenge TTL = 2 min, 2x buffer)
5. **Remove old**: Rotate to primary, remove V2

### RATE_LIMIT_KEY_HMAC_SECRET (Every 180 Days)

**Key Versioning Required** (cannot rotate atomically — existing HMAC keys in Redis):

1. **Generate new secret**: `openssl rand -hex 32`
2. **Add to Doppler** as `RATE_LIMIT_KEY_HMAC_SECRET_V2`
3. **Deploy API** with key versioning:
   ```go
   func emailKey(email string) string {
       v1 := hmacSHA256(os.Getenv("RATE_LIMIT_KEY_HMAC_SECRET"), email)
       v2 := hmacSHA256(os.Getenv("RATE_LIMIT_KEY_HMAC_SECRET_V2"), email)
       return "v2:" + v2 + "|v1:" + v1  // Check both, prefer v2
   }
   ```
4. **Migrate Redis keys**: Background job rewrites `v1:` keys to `v2:` over 7 days
5. **Remove v1**: After migration complete, remove v1 secret and code

---

## Capacity Planning

### Redis Memory Estimation

| Tier | Keys (est.) | Members/Key | Bytes/Member | Total |
|------|-------------|-------------|--------------|-------|
| OTP Request (email) | 10,000 | 3 | ~100 | 3 MB |
| OTP Request (IP) | 5,000 | 30 | ~100 | 15 MB |
| OTP Verify (challenge) | 50,000 | 5 | ~100 | 25 MB |
| OTP Verify (email) | 10,000 | 20 | ~100 | 20 MB |
| Onboard (user) | 5,000 | 5 | ~100 | 2.5 MB |
| Session (user) | 50,000 | 30 | ~100 | 150 MB |
| Global (user) | 50,000 | 300 | ~100 | 1.5 GB |
| Global (IP) | 10,000 | 600 | ~100 | 600 MB |
| PoW nonces | 100,000 | 1 | ~100 | 10 MB |
| **Total** | | | | **~2.3 GB** |

**Recommendation**: Redis `maxmemory` ≥ 4 GB, `maxmemory-policy` = `volatile-lru`

### Brevo Budget

| Metric | Value |
|--------|-------|
| Free tier | 300 emails/day |
| OTP budget | 250/day (83%) |
| Reserved (returning) | 100/day (40%) |
| General (new) | 150/day (60%) |
| Headroom | 50/day (17%) |
| **Upgrade trigger** | > 70% utilization sustained (175/day) |

---

## Rollback Procedures

### Full Rollback (All Tiers to Shadow)

```bash
# Via Doppler (instant, no deploy)
doppler secrets set RATE_LIMIT_MODE=shadow
doppler secrets set RATE_LIMIT_MODE_OTP_REQUEST=shadow
doppler secrets set RATE_LIMIT_MODE_OTP_VERIFY=shadow
doppler secrets set RATE_LIMIT_MODE_ONBOARD=shadow
doppler secrets set RATE_LIMIT_MODE_SESSION=shadow
doppler secrets set RATE_LIMIT_MODE_GLOBAL=shadow
```

### Per-Tier Rollback

```bash
# Example: Rollback only OTP Verify
doppler secrets set RATE_LIMIT_MODE_OTP_VERIFY=shadow
```

### Code Rollback

```bash
# Revert to previous Git tag
git revert <commit-hash>  # or
git checkout <previous-tag>
# Deploy via CI/CD
```

---

## Debugging Commands

### Check Current Limits (Redis)

```bash
# List all rate limit keys
redis-cli --scan --pattern "rl:*" | head -20

# Inspect a specific key
redis-cli ZRANGE "rl:otp_request:email:hmac_abc123" 0 -1 WITHSCORES

# Check TTL
redis-cli PTTL "rl:global:user:user_123"

# Count members
redis-cli ZCARD "rl:session:user:user_123"
```

### Check Limiter Decisions (Logs)

```bash
# Recent 429s
grep "rate_limited" /var/log/api.log | tail -20

# Fail-open/closed events
grep "failopen\|failclosed" /var/log/api.log | tail -10

# Bot challenges
grep "challenge_required" /var/log/api.log | tail -10
```

### Verify Headers

```bash
# Global API
curl -I -H "Authorization: Bearer $TOKEN" https://api.example.com/api/v1/schools

# OTP Request
curl -I -H "X-BFF-Service-Secret: $SECRET" -H "X-Client-IP: 1.2.3.4" \
  -X POST -d '{"email":"test@example.com"}' https://api.example.com/api/v1/auth/otp/request
```

---

## On-Call Contacts

| Role | Contact | Escalation |
|------|---------|------------|
| Primary On-Call | @backend-oncall | +30 min |
| Secondary On-Call | @backend-lead | +60 min |
| Security | @security-lead | Immediate for bot attacks |
| Infra | @infra-oncall | For Redis issues |

---

## Links

- ADR: `docs/adr/0002-rate-limiting-strategy.md` §8, §11
- Phase 4: `specs/014-ratelimit-phase4-global-middleware-observability.md`
- Config Reference: `specs/016-ratelimit-config-reference.md`
- Test Plan: `specs/017-ratelimit-test-plan.md`
- Metrics Dashboard: `grafana/dashboards/rate-limiting.json`
- Alert Rules: `monitoring/alerts/ratelimit.yaml`