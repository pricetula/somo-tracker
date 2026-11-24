# SPEC-013: Rate Limiting Phase 3 — Adaptive Bot Defense Layer

Endpoint: `POST /api/v1/auth/otp/request`, `POST /api/v1/auth/onboard`, `POST /api/v1/auth/challenge`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-013 |
| title | Rate Limiting Phase 3 — Adaptive Bot Defense Layer |
| status | draft |
| depends_on | docs/adr/0002-rate-limiting-strategy.md, specs/011-ratelimit-phase1-core-limiter.md, specs/012-ratelimit-phase2-otp-hardening.md, specs/glossary.md |
| api_endpoint | POST /api/v1/auth/otp/request, POST /api/v1/auth/onboard, POST /api/v1/auth/challenge |
| owner | backend |

## Summary

Implement the **pluggable `ChallengeProvider` interface** with baseline signals (Turnstile, HMAC PoW, BFF dwell cookie, honeypot), server-side risk scoring (0–100), and escalation ladder (Low → Medium → High → Critical). This phase addresses automated threats (OAT-007, OAT-009, OAT-019) from LLM agents, solver farms, and residential IP rotation.

## Architecture: ChallengeProvider Interface

```go
// ChallengeProvider is the pluggable bot defense interface.
type ChallengeProvider interface {
    // Name returns the provider identifier (e.g., "turnstile", "pow", "webauthn").
    Name() string
    // IssueChallenge generates a new challenge for the given action and context.
    IssueChallenge(ctx context.Context, req ChallengeRequest) (ChallengeResponse, error)
    // VerifyChallenge validates a submitted challenge solution.
    VerifyChallenge(ctx context.Context, req VerifyRequest) (VerifyResult, error)
    // RiskScore computes a 0-100 risk score from baseline signals.
    RiskScore(ctx context.Context, signals BaselineSignals) (int, error)
}

// BaselineSignals collected from BFF and request.
type BaselineSignals struct {
    TurnstileSuccess      bool    // Turnstile verification result
    TurnstileAction       string  // "otp_request" | "onboard"
    DwellTimeMs           int64   // From BFF pre-session cookie (X-BFF-Form-Dwell-Ms)
    HoneypotTriggered     bool    // Hidden field non-empty
    PoWSolved             bool    // PoW solution valid
    PoWDifficulty         int     // Difficulty used
    Recent429Count        int     // Recent 429s on IP/64 or email hash
    DisposableDomain      bool    // Email domain in disposable list
    UnseenDomain          bool    // Email domain not in allowlist
    NoPriorSession        bool    // No device session cookie
    IPReputationScore     float64 // Optional: external reputation (0-1)
}
```

## Baseline Signals (Required on `/otp/request` and `/onboard`)

### 1. BFF Pre-Session Timing Cookie
- **BFF sets**: On auth/onboarding page load, `HttpOnly`, `Secure`, `SameSite=Lax` cookie with HMAC-signed timestamp
- **Cookie name**: `__Host-dwell_ts` (or similar)
- **Payload**: `timestamp_ms|hmac_signature`
- **BFF validates**: On form submit, verify HMAC, compute `now - timestamp = dwell_ms`
- **Forwarded**: `X-BFF-Form-Dwell-Ms` header to Go API
- **Risk thresholds**:
  - `< 800 ms` → sub-human (high risk)
  - `> 4 hours` → stale (medium risk)

### 2. Honeypot Field
- **Field name**: Randomized per session (e.g., `hp_<random>`)
- **Placement**: Hidden via CSS (`display:none`, `visibility:hidden`, `opacity:0`)
- **Validation**: Non-empty value → immediate fail or risk spike (+30)
- **BFF/API**: Both can validate

### 3. Turnstile Token
- **Action binding**: `otp_request` or `onboard` (verified in Turnstile response)
- **Verification**: Server-side via Cloudflare Siteverify API
- **Failure**: `403 FORBIDDEN` (baseline requirement)

### 4. Proof-of-Work (PoW) — HMAC-Signed Challenge
- **Issue endpoint**: `POST /api/v1/auth/challenge` (proxied by BFF)
- **Request**: `{ "action": "otp_request", "email_hash": "hmac...", "client_subnet": "1.2.3.0/24" }`
- **Response**: `{ "challenge_id": "uuid", "difficulty": 4, "expires_at": "timestamp", "seed": "base64" }`
- **Algorithm**: Find `nonce` such that `SHA256(seed + nonce)` has `difficulty` leading zero bits
- **Submit**: `X-PoW-Solution` header or `pow_token` field with `{challenge_id, nonce}`
- **Verification**: Single-use nonce tracking in Redis (`rl:pow:nonce:{challenge_id}:{nonce}`)
- **Difficulty scaling**: Base 4 → +2 per risk tier (Low=4, Medium=6, High=8, Critical=10+)
- **Expiry**: 2 minutes

## Risk Scoring (0–100) & Escalation Ladder

### Risk Factors & Weights

| Signal | Weight | Condition |
|--------|--------|-----------|
| Turnstile failed | +40 | `TurnstileSuccess == false` |
| Sub-human dwell (<800ms) | +25 | `DwellTimeMs < 800` |
| Stale dwell (>4h) | +10 | `DwellTimeMs > 14400000` |
| Honeypot triggered | +35 | `HoneypotTriggered == true` |
| PoW unsolved/invalid | +20 | `PoWSolved == false` |
| Recent 429s (IP/64) | +5 per | `Recent429Count * 5` (cap 20) |
| Recent 429s (email hash) | +10 per | `Recent429Count * 10` (cap 30) |
| Disposable domain | +15 | `DisposableDomain == true` |
| Unseen domain | +10 | `UnseenDomain == true` |
| No prior session/device | +5 | `NoPriorSession == true` |
| IP reputation (if enabled) | 0–20 | `IPReputationScore * 20` |

### Escalation Ladder

| Risk Level | Score Range | Action |
|------------|-------------|--------|
| **Low** | `< 50` | Baseline: Turnstile + PoW base difficulty (4) |
| **Medium** | `50 – 84` | Increased PoW difficulty (+2) + interactive challenge |
| **High** | `85 – 94` | WebAuthn/Passkey assertion (if enrolled) OR deferred processing + extended cool-down |
| **Critical** | `≥ 95` | **Circuit breaker**: Fail closed for new unauthenticated OTP sends; preserve reserved quota for returning verified users |

> **Global budget near exhaustion** (e.g., >90% of 250/day used) → automatically escalate to **Critical** regardless of signals.

## Endpoints

### `POST /api/v1/auth/challenge` — Issue PoW Challenge

**Request:**
```json
{
  "action": "otp_request",
  "email_hash": "hmac_sha256_hex",
  "client_subnet": "1.2.3.0/24"
}
```

**Response (200):**
```json
{
  "challenge_id": "uuid",
  "difficulty": 4,
  "expires_at": "2026-11-24T12:00:00Z",
  "seed": "base64_encoded"
}
```

### `POST /api/v1/auth/otp/request` — Extended with Bot Signals

**Request (additional fields):**
```json
{
  "email": "user@example.com",
  "turnstile_token": "cf_token",
  "pow_solution": { "challenge_id": "uuid", "nonce": "hex" },
  "honeypot_field": ""
}
```

**Headers from BFF:**
- `X-BFF-Form-Dwell-Ms: 1250`
- `X-Client-IP: 1.2.3.4`

### Step-Up Challenge Response (428 Precondition Required)

```json
{
  "type": "https://api.example.com/problems/challenge-required",
  "title": "Additional verification required",
  "status": 428,
  "code": "challenge_required",
  "challenge": {
    "type": "pow",
    "difficulty": 6,
    "seed": "base64",
    "expires_in_seconds": 120
  }
}
```

## Configuration (Doppler)

```bash
# PoW
POW_SECRET=<32+ char secret>                    # HMAC key for challenge signing
POW_BASE_DIFFICULTY=4                           # Leading zero bits (Low tier)
POW_CHALLENGE_TTL_SECONDS=120                   # 2 minutes

# Risk Thresholds
BOT_RISK_STEPUP_THRESHOLD=50                    # Low → Medium
BOT_RISK_BLOCK_THRESHOLD=85                     # Medium → High/Critical
BOT_RISK_CRITICAL_THRESHOLD=95                  # High → Critical (circuit breaker)

# Dwell Cookie
BFF_DWELL_COOKIE_NAME=__Host-dwell_ts
BFF_DWELL_SUB_HUMAN_MS=800
BFF_DWELL_STALE_MS=14400000

# Honeypot
BOT_HONEYPOT_FIELD_PREFIX=hp_

# Turnstile (already in config)
TURNSTILE_SECRET_KEY=<secret>
TURNSTILE_EXPECTED_HOST=<host>

# Disposable domains
RATE_LIMIT_DISPOSABLE_DOMAINS=tempmail.com,10minutemail.com,...

# Global budget critical threshold
RATE_LIMIT_CRITICAL_BUDGET_PCT=90               # >90% of 250/day = Critical
```

## Implementation Files

| File | Responsibility |
|------|----------------|
| `internal/platform/botdefense/provider.go` | `ChallengeProvider` interface |
| `internal/platform/botdefense/turnstile.go` | Turnstile verifier (wraps existing) |
| `internal/platform/botdefense/pow.go` | HMAC PoW issuer/verifier + Redis nonce store |
| `internal/platform/botdefense/dwell.go` | BFF dwell cookie validator |
| `internal/platform/botdefense/honeypot.go` | Honeypot field validator |
| `internal/platform/botdefense/risk.go` | Risk scorer (0-100) + escalation logic |
| `internal/platform/botdefense/webauthn.go` | WebAuthn/Passkey provider (stub for High tier) |
| `internal/auth/handler/handler.go` | Wire bot defense into `RequestOTP`, `Onboard`, new `Challenge` endpoint |
| `internal/platform/config/config.go` | All bot defense Doppler fields |

## Tests Required

| Test Type | What to Verify |
|-----------|----------------|
| Unit | PoW challenge generation: valid seed, difficulty, expiry, HMAC signature |
| Unit | PoW verification: correct nonce solves, wrong nonce rejects, replay rejects |
| Unit | Risk scorer: each signal adds correct weight |
| Unit | Risk scorer: escalation thresholds map to correct tiers |
| Unit | Dwell cookie: valid HMAC, sub-human detection, stale detection |
| Unit | Honeypot: non-empty triggers, empty passes |
| Unit | Turnstile: success/failure/actions binding |
| Unit | PoW challenge HMAC signature verification (constant-time) |
| Unit | Risk score bounds: 0-100, no overflow/underflow |
| Integration | Low risk (<50) → base PoW difficulty 4 |
| Integration | Medium risk (50-84) → PoW difficulty 6 + interactive |
| Integration | High risk (85-94) → WebAuthn or deferred |
| Integration | Critical risk (≥95) → circuit breaker, reserved quota preserved |
| Integration | Global budget >90% → auto-Critical |
| Integration | PoW nonce single-use (replay rejected) |
| Integration | BFF dwell cookie forwarded and validated |
| Integration | Turnstile action binding: `otp_request` vs `onboard` |
| Integration | Escalation ladder: Low→Medium→High→Critical transitions |
| Contract | `POST /auth/challenge` 200 → challenge_id, difficulty, seed, expires_at |
| Contract | `428` step-up response → RFC 9457 with `challenge` object |
| Contract | Challenge types: `pow`, `turnstile`, `webauthn` schemas valid |
| Contract | All bot defense endpoints emit RateLimit headers |
| Load | PoW solve time distribution at difficulty 4, 6, 8, 10 |
| Load | 1000 concurrent PoW challenges — Redis nonce store stable |
| Chaos | PoW Redis nonce store unavailable → fail-open for PoW, fail-closed for OTP |
| Chaos | Turnstile API timeout → risk score increases, graceful degradation |
| Security | PoW replay attack: nonce single-use enforced |
| Security | Challenge ID enumeration: UUID v4, unguessable |
| Security | HMAC timing attack on PoW verification → constant-time resistant |
| Security | Risk score manipulation: signals validated server-side, not client-trusted |
| Security | Dwell cookie HMAC: cannot forge without secret |

## Acceptance Criteria

Given a request with valid Turnstile, dwell=1200ms, empty honeypot, PoW solved at difficulty 4
When risk scored
Then score < 50 (Low)
And baseline challenge sufficient

Given a request with failed Turnstile, dwell=500ms, honeypot triggered
When risk scored
Then score ≥ 85 (High)
And WebAuthn/deferred required

Given global OTP send budget at 230/250 (92%)
When new OTP request arrives
Then risk auto-escalated to Critical
And request fails closed (429) unless returning user with reserved quota

Given PoW challenge issued at difficulty 6
When valid nonce submitted within 2 minutes
Then verification succeeds
And nonce invalidated (cannot replay)

Given PoW challenge issued
When 3 minutes pass
Then challenge expired, verification fails

Given request with disposable domain and no prior session
When risk scored
Then score ≥ 50 (Medium at minimum)

## Open Questions

- WebAuthn enrollment trigger: post-onboard? On first High-risk event?
- External IP reputation integration (AbuseIPDB, CrowdSec) — deferred?
- PoW difficulty calibration: target solve time on mobile (difficulty 4 ≈ ?ms)?
- Interactive challenge UX for Medium tier (slider, click, etc.)?

## Links

- ADR: `docs/adr/0002-rate-limiting-strategy.md` §6, §9
- Glossary: `specs/glossary.md` (entities: `challenge_provider`, `pow_challenge`, `risk_score`, `escalation_ladder`, `dwell_cookie`, `honeypot`)
- Phase 1: `specs/011-ratelimit-phase1-core-limiter.md`
- Phase 2: `specs/012-ratelimit-phase2-otp-hardening.md`