# SPEC-017: Rate Limiting Test Plan

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-017 |
| title | Rate Limiting Test Plan |
| status | draft |
| depends_on | docs/adr/0002-rate-limiting-strategy.md, specs/010-ratelimit-phase0-trust-boundary.md, specs/011-ratelimit-phase1-core-limiter.md, specs/012-ratelimit-phase2-otp-hardening.md, specs/013-ratelimit-phase3-bot-defense.md, specs/014-ratelimit-phase4-global-middleware-observability.md, specs/015-ratelimit-api-contract.md, specs/016-ratelimit-config-reference.md |
| owner | backend |

## Summary

Comprehensive test strategy covering **unit**, **integration**, **contract**, **load**, **chaos**, and **security** tests for all rate limiting phases. Tests are organized by phase and tier.

---

## Test Pyramid

```
                    ┌─────────────┐
                    │   Chaos     │  ← 5 tests (Redis fail, network partition, clock skew)
                   ├─────────────┤
                  │    Load     │  ← 8 tests (sustained, burst, concurrent, soak)
                 ├─────────────┤
                │  Contract   │  ← 12 tests (headers, error shapes, OpenAPI)
               ├─────────────┤
              │ Integration │  ← 25 tests (Redis, tiers, fail modes, bot defense)
             ├─────────────┤
            │    Unit     │  ← 40 tests (algorithms, key builders, risk scorer, Lua)
           └─────────────┘
```

---

## Phase 0: Trust Boundary (specs/010)

### Unit Tests

| ID | Test | Expected |
|----|------|----------|
| UT-001 | `VerifyBFFServiceSecret` uses `ConstantTimeCompare` | Timing-safe comparison |
| UT-002 | `ExtractClientIP` with `TRUSTED_PROXY_HOPS=1` | Rightmost IP from `X-Forwarded-For` |
| UT-003 | `ExtractClientIP` with `TRUSTED_PROXY_HOPS=2` | 2nd from right |
| UT-004 | `ExtractClientIP` with `TRUSTED_PROXY_HOPS=3` | 3rd from right |
| UT-005 | IPv4 normalization → `/32` | `1.2.3.4` → `1.2.3.4/32` |
| UT-006 | IPv6 normalization → `/64` | `2001:db8::1` → `2001:db8::/64` |
| UT-007 | Missing `X-Client-IP` after secret verify | Returns error |

### Integration Tests

| ID | Test | Expected |
|----|------|----------|
| IT-001 | Request to `/api/v1/auth/otp/request` without `X-BFF-Service-Secret` | `403 FORBIDDEN` |
| IT-002 | Request with invalid `X-BFF-Service-Secret` | `403 FORBIDDEN` |
| IT-003 | Request with valid secret but no `X-Client-IP` | `400 INVALID_REQUEST` |
| IT-004 | `/healthz` bypasses secret check | `200 OK` |
| IT-004 | `/readyz` bypasses secret check | `200 OK` |
| IT-005 | `TRUSTED_BFF_CIDRS` blocks non-matching IP | `403 FORBIDDEN` |
| IT-006 | Valid secret + valid IP → request proceeds | Middleware chain continues |

---

## Phase 1: Core Limiter (specs/011)

### Unit Tests

| ID | Test | Expected |
|----|------|----------|
| UT-010 | Lua script: 10 allowed, 11th denied | `count=10` allowed, 11th `allowed=false` |
| UT-011 | Lua script: window rollover at 61s | 6th request at 61s allowed |
| UT-012 | Rejected requests NOT added to ZSET | `ZCARD` unchanged on reject |
| UT-013 | TTL refreshed on allowed requests | `PTTL` = window + 60s |
| UT-014 | `KeyBuilder.EmailKey` → HMAC-SHA256 | Deterministic, same input = same output |
| UT-015 | `KeyBuilder.IPKey` IPv4 → `/32` | `1.2.3.4` → `1.2.3.4/32` |
| UT-016 | `KeyBuilder.IPKey` IPv6 → `/64` | `2001:db8::1` → `2001:db8::/64` |
| UT-017 | Shadow mode: logs `would_block`, allows | `ratelimit_decisions_total{result="would_block"}` |
| UT-018 | 50ms timeout enforced on Lua script | Context deadline exceeded → error |

### Integration Tests

| ID | Test | Expected |
|----|------|----------|
| IT-010 | Concurrent 100 goroutines, limit=10 | Exactly 10 allowed, 90 denied |
| IT-011 | Redis restart mid-load | Limiter recovers, no stale limits |
| IT-012 | Fail-closed tier (OTP): Redis down → 503 | `503 SERVICE_UNAVAILABLE` in <50ms |
| IT-013 | Fail-open tier (Session): Redis down → allow | Request allowed, `failopen_total` incremented |
| IT-014 | In-process fallback activates on fail-closed | Token bucket allows coarse limiting |

### Load Tests

| ID | Test | Target |
|----|------|--------|
| LT-001 | Sustained 10k req/s for 5 min | p99 < 5ms, 0 errors |
| LT-002 | Burst 50k req/s for 30s | p99 < 20ms, no crashes |
| LT-003 | Soak 5k req/s for 1 hour | Memory stable, no leaks |

---

## Phase 2: OTP Hardening (specs/012)

### Unit Tests

| ID | Test | Expected |
|----|------|----------|
| UT-020 | Email normalization: lowercase, trim, NFC | `  User@Example.COM  ` → `user@example.com` |
| UT-021 | Email normalization: +tag stripping (Gmail) | `user+tag@gmail.com` → `user@gmail.com` |
| UT-022 | Email normalization: +tag preserved (unknown) | `user+tag@custom.com` → `user+tag@custom.com` |
| UT-023 | Returning-user reservation: 40% pool logic | 150 general + 100 reserved = 250 |
| UT-024 | Domain velocity key: `example.com` | `rl:otp_request:domain:example.com` |
| UT-025 | Disposable domain detection | `tempmail.com` → flagged |

### Integration Tests

| ID | Test | Expected |
|----|------|----------|
| IT-020 | Global budget exhaustion (general pool) | 151st new-user → 429 |
| IT-021 | Global budget exhaustion (reserved pool) | Returning user allowed up to 100 |
| IT-022 | Domain velocity exceed → step-up flag | 21st request to domain → challenge_required |
| IT-023 | Disposable domain → step-up/reject | `user@tempmail.com` → 428 or 403 |
| IT-024 | Challenge invalidation on 6th attempt | `used_at` set, subsequent verify → 401 |
| IT-025 | Challenge invalidation on new OTP request | Previous challenge `used_at` set |
| IT-026 | Successful verify → returning user marker set | Redis `rl:returning_user:{hash}` TTL 30d |
| IT-027 | Successful verify → failure count reset | `rl:otp_verify:email:{hash}:failures` = 0 |
| IT-028 | Failed verify → failure count incremented | Count increases, 20 → backoff |
| IT-029 | 429 never discloses limiting key | Response body no `email`, `ip`, `challenge` |

### Contract Tests

| ID | Test | Expected |
|----|------|----------|
| CT-001 | `POST /otp/request` 200 → RateLimit headers | All 6 headers present |
| CT-002 | `POST /otp/request` 429 → RFC 9457 body | `code: rate_limited`, `retry_after_seconds` |
| CT-003 | `POST /otp/verify` 200 → session_token | `session_token` in response |
| CT-004 | `POST /otp/verify` 401 → INVALID_TOKEN | No enumeration |

---

## Phase 3: Bot Defense (specs/013)

### Unit Tests

| ID | Test | Expected |
|----|------|----------|
| UT-030 | PoW challenge: valid seed, difficulty, expiry, HMAC | All fields present, signature valid |
| UT-031 | PoW verification: correct nonce solves | `SHA256(seed+nonce)` has `difficulty` leading zeros |
| UT-032 | PoW verification: wrong nonce rejects | `allowed=false` |
| UT-033 | PoW verification: replay rejects | Nonce single-use |
| UT-034 | Risk scorer: each signal adds correct weight | Turnstile fail +40, dwell<800 +25, etc. |
| UT-035 | Risk scorer: thresholds map to tiers | <50 Low, 50-84 Med, 85-94 High, ≥95 Critical |
| UT-036 | Dwell cookie: valid HMAC, sub-human, stale | Correct detection |
| UT-037 | Honeypot: non-empty triggers | Risk +35 |

### Integration Tests

| ID | Test | Expected |
|----|------|----------|
| IT-030 | Low risk (<50) → base PoW difficulty 4 | Challenge issued at diff 4 |
| IT-031 | Medium risk (50-84) → PoW diff 6 + interactive | Challenge at diff 6 |
| IT-032 | High risk (85-94) → WebAuthn or deferred | 428 with webauthn challenge |
| IT-033 | Critical risk (≥95) → circuit breaker | 429, reserved quota preserved |
| IT-034 | Global budget >90% → auto-Critical | Even low-risk requests blocked |
| IT-035 | PoW nonce single-use | Replay → 400/428 |
| IT-036 | BFF dwell cookie forwarded and validated | `X-BFF-Form-Dwell-Ms` parsed |

### Load Tests

| ID | Test | Target |
|----|------|--------|
| LT-010 | PoW solve time at diff 4 | p50 50-200ms mobile |
| LT-011 | PoW solve time at diff 6 | p50 200-800ms mobile |
| LT-012 | PoW solve time at diff 8 | p50 800-3000ms mobile |

---

## Phase 4: Global Middleware & Observability (specs/014)

### Unit Tests

| ID | Test | Expected |
|----|------|----------|
| UT-040 | Global key resolver: userID > IP | Authenticated → userID key |
| UT-041 | Onboard tier: userID 5/min, 20/day, IP 50/min | Correct limits applied |
| UT-042 | Session tier: userID 30/min, IP 100/min | Correct limits applied |

### Integration Tests

| ID | Test | Expected |
|----|------|----------|
| IT-040 | Global middleware emits RateLimit headers | All responses have headers |
| IT-041 | Authenticated request keys on userID | NAT-safe (200 users, 1 IP) |
| IT-042 | Unauthenticated request keys on IP | IP-keyed limit |
| IT-043 | Fail-open on session/global | Allowed, metric incremented |
| IT-044 | Fail-closed on onboard | 503, metric incremented |
| IT-045 | Metrics emitted with correct labels | No PII in labels |
| IT-046 | Alert rules fire on threshold | PrometheusRule evaluation |

### Load Tests

| ID | Test | Target |
|----|------|--------|
| LT-020 | Global limiter: 10k req/s | p99 < 5ms |
| LT-021 | School simulation: 200 users, 1 IP, 10 req/min each | All succeed |

### Chaos Tests

| ID | Test | Expected |
|----|------|----------|
| CH-001 | Redis primary failover | <5s recovery, no limit bypass |
| CH-002 | Network partition (Redis unreachable) | Fail-closed/closed per tier |
| CH-003 | Clock skew (app server +5min vs Redis) | Lua uses Redis TIME, accurate |
| CH-004 | Redis OOM (maxmemory) | Eviction policy, limiter degrades gracefully |
| CH-005 | BFF secret rotation mid-load | Dual-secret works, no 403 spikes |

---

## Security Tests

| ID | Test | Expected |
|----|------|----------|
| ST-001 | Account enumeration via 429 | Identical response for exist/non-exist |
| ST-002 | Account enumeration via timing | Constant-time responses |
| ST-003 | BFF bypass direct to API | `403 FORBIDDEN` without secret |
| ST-004 | IP spoofing via X-Forwarded-For | `TRUSTED_PROXY_HOPS` prevents |
| ST-005 | PoW replay attack | Nonce single-use |
| ST-006 | Challenge ID enumeration | UUID v4, unguessable |
| ST-007 | Email in Redis keys/logs | Only HMAC-SHA256(email) |
| ST-008 | Rate limit headers on 503 | No headers on fail-closed |

---

## Test Execution

### CI Pipeline (Every PR)

```yaml
# .github/workflows/test.yml
jobs:
  unit:
    runs-on: ubuntu-latest
    steps:
      - go test ./internal/platform/ratelimit/... -v -race
      - go test ./internal/platform/botdefense/... -v -race
      - go test ./internal/auth/handler/... -v -race -run Unit

  integration:
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:7-alpine
      postgres:
        image: postgres:18-alpine
    steps:
      - go test -tags=integration ./... -v

  contract:
    runs-on: ubuntu-latest
    steps:
      - go test ./... -v -run Contract
      - spectral lint openapi.yaml

  load:
    runs-on: [self-hosted, load-test]
    if: github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'
    steps:
      - k6 run load/ratelimit.js

  chaos:
    runs-on: [self-hosted, chaos]
    if: github.event_name == 'workflow_dispatch'
    steps:
      - chaosiq run chaos/ratelimit.yaml
```

### Local Development

```bash
# Unit tests
cd backend && go test ./internal/platform/ratelimit/... -v -race
cd backend && go test ./internal/platform/botdefense/... -v -race

# Integration tests (requires Docker)
cd backend && go test -tags=integration ./... -v

# Load test (k6)
k6 run load/ratelimit.js

# Contract test
go test ./... -v -run Contract
```

---

## Test Data Management

| Data Type | Strategy |
|-----------|----------|
| Redis keys | Unique prefix per test (`rl:test:{test_id}:...`) |
| Database | Transaction rollback per test |
| Disposable domains | Embedded test fixture |
| PoW challenges | Generated per test, auto-expire |

---

## Acceptance Gates

| Gate | Criteria |
|------|----------|
| PR Merge | All unit + integration + contract tests pass |
| Staging Deploy | Load test LT-001, LT-002 pass; chaos CH-001, CH-002 pass |
| Production Rollout | Shadow mode metrics: `would_block` < 1% of traffic |
| Full Enforce | Zero critical alerts for 24h at each tier |

---

## Links

- ADR: `docs/adr/0002-rate-limiting-strategy.md`
- All Phase Specs: `specs/010` through `specs/016`
- Load test scripts: `load/ratelimit.js`
- Chaos experiments: `chaos/ratelimit.yaml`