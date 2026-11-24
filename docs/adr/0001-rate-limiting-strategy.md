# ADR-0002: Rate Limiting and Abuse Resistance for Production API (BFF Architecture)

_Supersedes ADR-0001 ("Rate Limiting and Abuse Resistance for Production API")._

| Field       | Value                                                                                                                                                                                  |
| ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id          | ADR-0002                                                                                                                                                                               |
| title       | Rate Limiting and Abuse Resistance for Production API (BFF Architecture)                                                                                                               |
| status      | accepted                                                                                                                                                                               |
| date        | 2026-11-24                                                                                                                                                                             |
| author      | pi-agent                                                                                                                                                                               |
| deciders    | Team Leads, Security Reviewer                                                                                                                                                          |
| relates\_to | ADR-0001, OWASP API4:2023 (Unrestricted Resource Consumption), OWASP Automated Threats (OAT-007 Credential Cracking, OAT-009 CAPTCHA Defeat, OAT-019 Account Creation), Security Audit |

## Status

Accepted. Ready for implementation; remaining operational verifications (Render proxy hop count in staging and institutional domain allow-listing) are tracked in Follow-up and Rollout.

## Context

### Deployment & Session Topology

- **Public Render URLs with BFF Ingress:** Both the BFF and the Go API run on Render with public URLs. Browser traffic enters through the BFF, which proxies requests to the Go API. Because the Go API retains a public Render URL, direct internet requests could reach the API origin unless blocked at the application boundary.
- **Opaque Session Token Flow:** The BFF stores an opaque session token in a browser cookie and forwards it to the Go API, where the backend resolves the session against Redis/PostgreSQL to identify the authenticated `userID`.
- **Email Capacity Constraint:** Outbound OTP emails run on Brevo's free tier (**300 emails/day** shared across all transactional mail) until revenue supports upgrading to a paid plan.

### Current State

The backend has partial rate limiting, only on `POST /api/v1/auth/otp/request`:

- Per email: 3 requests per 15 minutes (fixed window, `INCR` + `EXPIRE`).
- Per IP: 10 requests per minute.
- A Cloudflare Turnstile token is verified server-side before OTP generation (standalone widget; the API and frontend are not proxied through Cloudflare).

No limits exist on:

- `POST /api/v1/auth/otp/verify`: unlimited verification attempts across challenges.
- `POST /api/v1/auth/onboard`: expensive writes (creates organization, school, and user).
- `GET /api/v1/auth/session`: session enumeration and polling amplification.
- `POST /api/v1/auth/logout`.
- All other `/api/v1/*` endpoints: no global protection.

### Problems to Solve

1. **Public API Origin Exposure:** Because the Go API has a public Render URL, attackers could bypass the BFF's coarse limits and spoof `X-Client-IP` if the API does not enforce cryptographic BFF authentication on every request.
2. **Unrestricted Resource Consumption:** No limits on authentication flows, onboarding writes, or general API endpoints.
3. **OTP Brute Force & Email Budget Exhaustion:** Verification attempts are uncapped per challenge/account, and Brevo's 300/day free ceiling can be exhausted by email bombing.
4. **Institutional IP Sharing:** Schools and organizations place hundreds of users behind a single NAT address, so tight per-IP limits block legitimate users.
5. **No Client/BFF Backoff Contract:** Clients and the BFF lack standardized rate-limit headers and structured `429` responses.
6. **Hardcoded Limits and Fixed-Window Bursts:** Limits cannot be tuned via Doppler, and fixed windows allow boundary bursts.
7. **Automated Abuse:** Modern bots use residential/mobile IP rotation, CAPTCHA-solving services, and LLM-driven headless browsers.

### Threat Model

We assume a capable, funded attacker who can:

- Discover the public Render URL of the Go API and send requests directly, bypassing the BFF.
- Rotate through thousands of residential and mobile IPs, making per-IP limits alone ineffective.
- Solve or outsource browser CAPTCHAs via solver APIs, human farms, or LLM agents.
- Generate unlimited plausible email addresses across real or disposable domains.
- Target a single victim (lockout, inbox flooding, OTP guessing) or spray across many accounts (email bombing, enumeration, mass fake signups).

**Design principle:** Assume any single challenge and any public URL can be probed directly. Reject unauthenticated BFF-bypass traffic immediately at the API edge, bind browser-facing anti-bot artifacts at the BFF, enforce strict per-target and global Brevo budgets in the Go API, and escalate friction when risk signals spike.

### Constraints

- **Stack:** Go 1.26, Fiber v3, Redis (`go-redis/v9`), Doppler for configuration.
- **Dependencies:** No new runtime dependencies for rate limiting; atomic Lua scripts run on the existing Redis instance.
- **Deployment:** Zero-downtime rollout behind feature flags and shadow mode.

## Decision

We will implement a **split-responsibility, defense-in-depth architecture** tailored to a public-URL Render deployment with BFF ingress and backend-resolved opaque sessions:

- **BFF Edge Layer:** Owns client IP extraction, browser cookie lifecycle (opaque session cookie and signed pre-session dwell cookie), coarse per-IP volumetric shedding, and injecting the mandatory BFF service credential and `X-Client-IP` header.
- **Go Core API Layer:** Owns mandatory BFF request authentication (blocking direct public-URL bypasses), backend opaque session resolution for `userID`-keyed limits, atomic Redis Lua sliding-window quotas, strict Brevo 250/day budget protection, OTP attempt/lifetime enforcement, and adaptive bot-challenge verification.

---

### 1. Public-URL Protection, BFF Trust Boundary & Client IP Identity

Because the Go API has a public Render URL, cryptographic verification of the BFF is **mandatory** on all non-exempt routes:

1. **Mandatory Service Credential (`BFF_SERVICE_SECRET`):**
   - Every BFF-to-API request must carry `X-BFF-Service-Secret` (compared using `crypto/subtle.ConstantTimeCompare`).
   - Any request to `/api/v1/*` lacking a valid `X-BFF-Service-Secret` (and not on the explicit webhook/health exemption list) is immediately rejected with `403 Forbidden` before session lookup, rate limiting, or handler execution.
   - If `TRUSTED_BFF_CIDRS` is configured, the API additionally verifies that the connecting peer IP belongs to the trusted BFF range before accepting `X-Client-IP`.
2. **Client IP Extraction at the BFF:**
   - Render's proxy appends the connecting peer IP to `X-Forwarded-For` without stripping client-supplied left-most entries.
   - The BFF extracts the real client IP by counting `TRUSTED_PROXY_HOPS` from the right of `X-Forwarded-For` (defaulting safely to `1` until verified in staging).
   - IPv4 addresses are normalized to `/32`; IPv6 addresses are normalized to the `/64` prefix.
3. **Trusted Header Forwarding & Backend Session Resolution:**
   - The BFF strips any incoming `X-Client-IP` header from external browsers and sets `X-Client-IP` to the verified client IP.
   - The BFF explicitly forwards the client's opaque session token (e.g., mapping the `HttpOnly` session cookie to `Authorization: Bearer <token>` or forwarding the session cookie) to the Go API.
   - **Middleware Ordering in Go API:**
     1. Exempt paths check (`/healthz`, `/readyz`, signature-verified webhooks).
     2. `BFF_SERVICE_SECRET` verification (`403` on failure).
     3. Trusted `X-Client-IP` extraction via single `clientip` resolver.
     4. Optional/required opaque session resolution (Redis/DB lookup) to attach `userID` to request context.
     5. Global & endpoint-specific rate limiters (keying on `userID` when resolved, falling back to `X-Client-IP` when unauthenticated).
4. **Coarse Edge Shedding at the BFF:**
   - The BFF enforces a coarse per-IP in-memory/Redis limiter (e.g., `120 req/min` on auth routes, `900 req/min` general) to drop volumetric floods before they consume BFF-to-API connections or Redis operations.

---

### 2. Hierarchical Rate-Limit Keys

IP is treated as a loose ceiling rather than a primary control, preventing NAT lockouts for schools while stopping single-source floods:

| Priority | Identifier    | Source                                                   | Primary Scope                                                  |
| -------- | ------------- | -------------------------------------------------------- | -------------------------------------------------------------- |
| 1        | `userID`      | Resolved in Go backend from opaque session token         | All authenticated endpoints (`/onboard`, `/session`, `/api/*`) |
| 2        | `challengeID` | Server-issued OTP challenge ID                           | `POST /api/v1/auth/otp/verify` attempt cap                     |
| 3        | `email`       | Normalized request body (`HMAC-SHA256(secret, email)`)   | `POST /otp/request`, `POST /otp/verify`                        |
| 4        | `IP`          | Verified `X-Client-IP` from BFF (`/32` IPv4, `/64` IPv6) | Unauthenticated fallback and loose ceiling                     |
| 5        | `global`      | Constant key                                             | Circuit breaker protecting Brevo's 300/day free email cap      |

**Email Normalization:** Trim whitespace, lowercase, apply Unicode NFC normalization, and strip `+tag` aliases only on known safe providers. Keys and logs store only `HMAC-SHA256(RATE_LIMIT_KEY_HMAC_SECRET, normalized_email)`—raw email addresses never appear in Redis keys or logs.

---

### 3. Atomic Sliding Window Log Algorithm

- **Data Structure:** Redis sorted set (`ZSET`) per rate-limit key, where `score` is the Redis timestamp in milliseconds and `member` is `<timestamp_ms>-<random_hex>` to prevent same-millisecond collisions.
- **Single-Script Execution:** One Lua script executes atomically on Redis:
  1. Read authoritative clock via `TIME` (eliminating app-server clock skew).
  2. Prune expired entries via `ZREMRANGEBYSCORE`.
  3. Count active entries via `ZCARD`.
  4. If `count < limit`, record the request via `ZADD` and refresh TTL via `PEXPIRE` (`window_ms + 60000`).
  5. Return `{allowed, remaining, reset_ms, retry_after_ms}` to populate the Go `Decision` struct.
- **Zero Memory Growth on Attacks:** Rejected requests are **not** added to the `ZSET`, preventing attackers from exhausting Redis memory via blocked traffic.
- **Algorithm Abstraction:** Handlers depend only on the `Limiter` interface. High-throughput global keys may optionally use an O(1)-memory weighted two-window counter behind the same interface.

---

### 4. Tiered Quotas (Doppler-Configurable)

All limits ship in `shadow` mode first and are tuned against real traffic before enforcement:

| Tier            | Endpoint                          | Key & Quota                                                                                | Window                             | Notes                                                                                           |
| --------------- | --------------------------------- | ------------------------------------------------------------------------------------------ | ---------------------------------- | ----------------------------------------------------------------------------------------------- |
| **OTP Request** | `POST /api/v1/auth/otp/request`   | `email`: 3<br>`IP`: 30<br>`global`: 250 (40% = 100 reserved for returning verified emails) | 15 min<br>15 min<br>24 hours       | Hard circuit breaker protecting Brevo's 300/day free tier shared with non-OTP mail              |
| **OTP Verify**  | `POST /api/v1/auth/otp/verify`    | `challengeID`: 5 attempts<br>`email`: 20 failed attempts<br>`IP`: 50                       | Per challenge<br>24 hours<br>1 min | Pre-auth endpoint (no `userID`); 6th attempt invalidates challenge                              |
| **Onboard**     | `POST /api/v1/auth/onboard`       | `userID`: 5<br>`userID`: 20<br>`IP`: 50                                                    | 1 min<br>24 hours<br>1 min         | Requires valid opaque session (`userID` resolved in backend); protects org/school/user creation |
| **Session**     | `GET /api/v1/auth/session`        | `userID`: 30<br>`IP`: 100                                                                  | 1 min<br>1 min                     | Tuned during shadow mode for multi-tab SPA polling                                              |
| **Global API**  | All `/api/v1/*` (incl. `/logout`) | `userID`: 300 (authenticated)<br>`IP`: 600 (unauthenticated)                               | 1 min<br>1 min                     | Authenticated traffic keys on backend-resolved `userID` so NAT schools are unaffected           |

**Exemptions:** Health/readiness endpoints (`/healthz`, `/readyz`), internal allow-listed service traffic, and signature-verified webhooks.

---

### 5. OTP-Specific Controls & Brevo Free-Tier Protection

Because Brevo's free tier is capped at **300 emails/day** across all application mail, OTP issuance is tightly guarded:

- **Global Send Budget & Returning-User Reservation:** `RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY=250` leaves 50 emails/day headroom for non-OTP transactional mail. Within the 250 OTP budget, **40% (100 sends/day)** is reserved exclusively for email hashes marked in Redis as having previously completed a successful `/otp/verify`, ensuring existing school staff/users can still log in even if new-signup traffic spikes (capping new/unverified email sends at 150/day).
- **Per-Challenge Cap:** Maximum **5** verification attempts per `challengeID`. Reaching 5 attempts or requesting a new OTP immediately invalidates the active challenge.
- **Cryptographic Strength & Lifetime:** Codes generated via `crypto/rand`, **6 digits minimum (8 recommended)**, max **10-minute** TTL, single-use, verified using `crypto/subtle.ConstantTimeCompare`.
- **Account-Level Failure Cap:** Maximum **20** failed verifications per email per 24 hours, triggering exponential backoff cool-downs.
- **Enumeration Resistance:** `POST /otp/request` returns identical status codes, response payloads, and normalized latency regardless of whether the email exists.
- **Email-Domain Velocity Rule:** If OTP requests to any single domain exceed **20 per hour**, subsequent requests to that domain require step-up challenges. (Domain allow-listing is deferred until institutional onboarding scales.)
- **Disposable-Domain Handling:** Requests using domains on a maintained disposable-email list trigger step-up challenges or rejection on `/otp/request` and `/onboard`.

---

### 6. Adaptive Bot-Resistance Layer (BFF + API Split)

Because LLM agents and solver services bypass single CAPTCHAs, bot defense is a multi-signal, risk-scored layer behind a pluggable `ChallengeProvider` interface:

**Baseline Signals (required on `POST /otp/request` and `POST /onboard`):**

1. **BFF Pre-Session Timing Cookie:** When the BFF serves the auth or onboarding view, it sets an `HttpOnly`, `Secure`, `SameSite=Lax` HMAC-signed cookie containing the page-load timestamp. On submission, the BFF validates the signature and forwards the elapsed dwell time (`X-BFF-Form-Dwell-Ms`) to the API; sub-human (`<800 ms`) or stale (`>4 hours`) submissions elevate risk.
2. **Honeypot Field:** A hidden input field validated at the BFF/API; non-empty values immediately fail or spike risk.
3. **Turnstile Token:** Verified server-side by the API and bound to the expected action (`otp_request` or `onboard`).
4. **Proof-of-Work (PoW) Challenge:** Self-hosted, HMAC-signed challenge (e.g., Altcha pattern) issued via `POST /api/v1/auth/challenge` (proxied by BFF) and bound to `(action, email_hash, client_subnet, 2m_expiry)` with single-use nonce tracking in Redis. Submitted via `X-PoW-Solution` (or `pow_token` field), difficulty scales dynamically with risk score, imposing direct CPU cost on automated scripts.

**Server-Side Risk Scoring (`0–100` scale) & Escalation Ladder:**

Inputs include missing/failed baseline signals, implausible form dwell time, recent `429` history on the IP `/64` or email hash, disposable/unseen email domains, and absence of prior device session cookies.

| Risk Level   | Score Range                   | Action                                                                                                                         |
| ------------ | ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| **Low**      | `< 50`                        | Baseline Turnstile + base-difficulty PoW (`POW_BASE_DIFFICULTY=4`)                                                             |
| **Medium**   | `50 – 84`                     | Increased PoW difficulty + interactive challenge                                                                               |
| **High**     | `>= 85`                       | WebAuthn/Passkey assertion (if enrolled) or deferred processing with extended cool-down                                        |
| **Critical** | Global budget near exhaustion | Circuit breaker: fail closed for new unauthenticated OTP sends; preserve remaining reserved quota for returning verified users |

---

### 7. HTTP Response Headers & `429` Contract

The API emits standard IETF `RateLimit` headers alongside legacy `X-RateLimit-*` headers; the BFF passes these through transparently to the browser:

```http
RateLimit-Limit: <limit>
RateLimit-Remaining: <remaining>
RateLimit-Reset: <seconds_until_reset>
X-RateLimit-Limit: <limit>
X-RateLimit-Remaining: <remaining>
X-RateLimit-Reset: <unix_timestamp>
Retry-After: <seconds>   # Included on 429 responses per RFC 9110
```

`429` responses use `application/problem+json` (RFC 9457) without disclosing which specific key (`email`, `IP`, `challengeID`) triggered the limit, preventing account enumeration:

```json
{
  "type": "https://api.example.com/problems/rate-limited",
  "title": "Too many requests",
  "status": 429,
  "code": "rate_limited",
  "retry_after_seconds": 42
}
```

Step-up challenge responses return `428 Precondition Required` (or `403`) with `"code": "challenge_required"` and the updated PoW/challenge parameters.

---

### 8. Failure Modes & Resilience

Redis limiter calls enforce a strict **50 ms** context timeout (`RATE_LIMIT_REDIS_TIMEOUT_MS=50`):

| Tier                                 | Redis Unavailable / Timeout Behavior                                                          | Rationale                                                                          |
| ------------------------------------ | --------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| **OTP Request, OTP Verify, Onboard** | **Fail closed** (`503 Service Unavailable`) backed by a coarse in-process per-IP token bucket | Protects Brevo's 300/day free email quota and prevents unthrottled OTP brute force |
| **Session, Global API**              | **Fail open**, increment `ratelimit_failopen_total`, and alert                                | Prioritizes availability for active school/user sessions                           |

---

### 9. Doppler Configuration

Startup validation rejects zero, negative, non-numeric, or out-of-bounds values and aborts boot on invalid config (including requiring `BFF_SERVICE_SECRET` to be non-empty in production):

```bash
# BFF <-> API Trust & Proxy Topology
TRUSTED_PROXY_HOPS=1                      # Verify exact hop count in staging later
TRUSTED_BFF_CIDRS=                        # Optional CIDR restriction in addition to secret
BFF_SERVICE_SECRET=<required_secret>      # Mandatory: API rejects public requests without this
RATE_LIMIT_KEY_HMAC_SECRET=<secret>

# Rollout & Redis
RATE_LIMIT_MODE=shadow                    # shadow | enforce (overridable per tier)
RATE_LIMIT_REDIS_TIMEOUT_MS=50

# OTP Request (Brevo Free Tier = 300/day total)
RATE_LIMIT_OTP_REQUEST_EMAIL=3            # per 15m
RATE_LIMIT_OTP_REQUEST_IP=30              # per 15m
RATE_LIMIT_OTP_SEND_BUDGET_PER_DAY=250    # leaves 50/day headroom for non-OTP mail
RATE_LIMIT_OTP_RESERVED_RETURNING_PCT=40  # 100 sends/day reserved for returning users
RATE_LIMIT_OTP_DOMAIN_VELOCITY_PER_HOUR=20

# OTP Verify
OTP_VERIFY_MAX_ATTEMPTS_PER_CHALLENGE=5
RATE_LIMIT_OTP_VERIFY_EMAIL_PER_DAY=20
RATE_LIMIT_OTP_VERIFY_IP_PER_MIN=50

# Onboard
RATE_LIMIT_ONBOARD_USER_PER_MIN=5
RATE_LIMIT_ONBOARD_USER_PER_DAY=20
RATE_LIMIT_ONBOARD_IP_PER_MIN=50

# Session & Global
RATE_LIMIT_SESSION_USER_PER_MIN=30
RATE_LIMIT_SESSION_IP_PER_MIN=100
RATE_LIMIT_GLOBAL_USER_PER_MIN=300
RATE_LIMIT_GLOBAL_IP_PER_MIN=600

# Bot Defense (Concrete Initial Defaults for Shadow Tuning)
POW_SECRET=<required_secret>
POW_BASE_DIFFICULTY=4
BOT_RISK_STEPUP_THRESHOLD=50
BOT_RISK_BLOCK_THRESHOLD=85
```

---

### 10. Observability & Alerting

- **Prometheus Metrics (never labeled with raw IP or email):**
  - `bff_auth_rejected_total` (tracks direct public-URL probes blocked due to missing/invalid `BFF_SERVICE_SECRET`)
  - `ratelimit_decisions_total{tier,result}`, `ratelimit_429_total{tier}`
  - `ratelimit_check_duration_seconds`, `ratelimit_redis_errors_total`, `ratelimit_failopen_total{tier}`
  - `otp_sends_total`, `otp_verify_failures_total`, `otp_challenge_invalidated_total`
  - `bot_challenge_total{provider,result}`, `bot_risk_score` (histogram), `bot_stepup_total{level}`
- **Alerts:**
  - Auth-tier `429` rate > 5% over 10 minutes.
  - Daily OTP send budget > 70% utilization (175/250 sends).
  - Redis limiter error rate > 1% or any fail-open trigger on a fail-closed tier.
  - Sudden downward shift in PoW solve-time distribution (indicates automated solver farm).

---

### 11. Rollout Plan

1. **Public-URL Lock & Header Plumbing:** Deploy `BFF_SERVICE_SECRET` enforcement on the Go API alongside BFF `X-Client-IP` forwarding and verify direct requests to the API's public Render URL fail with `403`.
2. **Shadow Mode:** Enable API sliding-window limiter and bot risk scoring in `shadow` mode across one full school-week cycle to baseline SPA polling (`GET /session`) and NAT IP patterns.
3. **Staged Enforcement:** Enable enforcement tier by tier via Doppler flags: `OTP Verify` -> `OTP Request` -> `Onboard` -> `Session` -> `Global API`.
4. **Adaptive Bot Escalation:** Enable baseline PoW + Turnstile enforcement, then enable risk-based step-up escalation.

---

### 12. Implementation Scope

| Component / File                | Responsibility                                                                                                                                                                                                                                                |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **BFF Service**                 | Extract client IP via `TRUSTED_PROXY_HOPS`; strip untrusted `X-Client-IP`; attach `BFF_SERVICE_SECRET`, `X-Client-IP`, opaque session token, and signed pre-session dwell cookie metrics; run coarse per-IP edge limiter; pass rate-limit headers to browser. |
| `internal/app/app.go`           | Enforce `BFF_SERVICE_SECRET` middleware on all non-exempt routes; wire global limiter middleware after opaque session resolution.                                                                                                                             |
| `internal/platform/ratelimit/`  | Atomic Redis Lua sliding-window script, `Decision` struct, `KeyBuilder` (HMAC email, `/64` IPv6), shadow-mode toggle, and in-process fallback limiter.                                                                                                        |
| `internal/platform/redis/`      | Lua script loading/eval helpers and 50 ms timeout wrapper.                                                                                                                                                                                                    |
| `internal/platform/botdefense/` | `ChallengeProvider` interface, Turnstile verifier, HMAC PoW issuer/verifier with Redis nonce store, and risk scorer.                                                                                                                                          |
| `internal/auth/handler/`        | Enforce per-challenge/account OTP caps, domain velocity, disposable-domain checks, PoW challenge issuance (`POST /auth/challenge`), returning-user marker, and RFC 9457 `429` / `428` responses.                                                              |
| `internal/platform/config/`     | Doppler environment parsing and fail-fast startup validation.                                                                                                                                                                                                 |

## Alternatives Considered

| Alternative                                                        | Pros                                    | Cons                                                                                                                       | Decision                            |
| ------------------------------------------------------------------ | --------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| **Trusting `X-Forwarded-For` on public API without BFF secret**    | Simpler BFF proxy config                | Any attacker hitting the public Render URL directly can spoof `X-Client-IP` and bypass BFF edge limits                     | **Rejected**                        |
| **Decoding/validating JWTs at the BFF**                            | BFF knows `userID` without backend call | Conflicts with opaque cookie session architecture where session state lives in the backend                                 | **Rejected**                        |
| **Upgrading Brevo or adding secondary email provider immediately** | Removes 300/day email ceiling           | Adds recurring cost before monetization; mitigated via 250/day circuit breaker and 40% returning-user reservation          | **Deferred** (revisit post-revenue) |
| **Fiber built-in limiter**                                         | Minimal custom code                     | Single key per middleware pass; lacks multi-key `Decision` contract and per-challenge invalidation                         | **Rejected**                        |
| **External WAF / Bot SaaS (e.g., Cloudflare Proxy)**               | Managed fingerprinting and edge DDoS    | Requires proxying all traffic through third-party DNS/WAF and adds vendor cost; `ChallengeProvider` leaves door open later | **Deferred**                        |

## Consequences

### Positive

- **Zero public-URL bypass:** Mandatory `BFF_SERVICE_SECRET` ensures the Go API's public Render URL cannot be abused directly.
- **Clean opaque session alignment:** Backend session middleware resolves `userID` once for both authorization and `userID`-keyed rate limits, preventing NAT lockouts for schools.
- **Zero-cost email protection:** Hard 250/day circuit breaker with 40% returning-user reservation protects Brevo's free tier without premature spend.

### Negative & Risks

- **Shared secret rotation hygiene:** `BFF_SERVICE_SECRET` must be kept synchronized across BFF and API Doppler configs; rotating it requires supporting dual-secret validation or coordinated restarts.
- **Brevo 250/day cap under onboarding surges:** Onboarding an entire school of 250+ users in a single day will hit the global OTP circuit breaker until Brevo is upgraded.
- **Deferred proxy hop verification:** Until `TRUSTED_PROXY_HOPS` is verified in staging, left-most `X-Forwarded-For` entries could be spoofed at the BFF if Render adds multiple proxy hops; mitigated by per-email/per-challenge/global quotas in the API.

## Follow-up

- [ ] Enforce `BFF_SERVICE_SECRET` middleware on Go API and configure BFF header forwarding (`X-BFF-Service-Secret`, `X-Client-IP`, opaque session token)
- [ ] Implement Redis Lua sliding-window limiter, `KeyBuilder`, and shadow mode
- [ ] Implement per-challenge OTP attempt cap, account failure cap, and Brevo 250/day returning-user budget split
- [ ] Build `internal/platform/botdefense` (Turnstile, HMAC PoW issuer/verifier, BFF dwell cookie validation, risk scoring)
- [ ] Run shadow mode for one school week, tune limits, and enforce per tier
- [ ] _(Deferred)_ Verify exact `TRUSTED_PROXY_HOPS` on Render with a spoofed `X-Forwarded-For` test in staging
- [ ] _(Deferred)_ Upgrade Brevo plan once revenue begins and configure institutional domain allow-list
