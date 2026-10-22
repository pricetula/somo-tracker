# Gap: Magic-Link IP Rate Limit Too Permissive (Allows Distributed Attacks)

**Area:** backend
**Severity:** High
**Status:** Done
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The magic-link send endpoint (`POST /api/auth/magic-link/send`) has dual rate limiting:
- **IP-based**: 10 requests/minute (600/hour from single IP)
- **Email-based**: 3 requests/hour (strong, prevents enumeration/bombing)

The IP limit of 10/min is too permissive — a distributed botnet with 100 IPs could send 60,000 magic links/hour. CAPTCHA middleware is configured but disabled by default (`CAPTCHAEnabled=false` in dev).

## Current Behavior
In `backend/internal/api/router.go:26-39`:
```go
var (
    authRateIP     = redis_rate.PerMinute(10) // 10 req/min per IP
    authRateEmail  = redis_rate.PerHour(3)    // 3 req/hour per email
    bulkInviteRate = redis_rate.PerHour(5)    // 5 bulk invites per hour per tenant
)

app.Post("/api/auth/magic-link/send",
    ratelimit.NewRateLimitMiddleware(r.limiter, authRateIP, "api:auth:magic-link:ip"),
    ratelimit.NewRateLimitMiddleware(r.limiter, authRateEmail, "api:auth:magic-link:email"),
    captchaMW,  // Only active if CAPTCHAEnabled=true
    r.Auth.sendMagicLink,
)
```

CAPTCHA config in `config/config.go` (not shown) defaults to disabled for local development.

## Expected Behavior
- IP rate limit lowered to 5 requests/minute (300/hour per IP) as baseline
- CAPTCHA **required** (not optional) for magic-link send in production
- Consider additional signals: device fingerprint, email domain reputation

## Impact
- **Abuse Risk**: Distributed attackers can bypass IP limit using proxy/VPN networks
- **Email Reputation**: High volume of magic links hurts domain sending reputation
- **Cost**: Stytch/email provider charges per magic link sent
- **Compliance**: OWASP API6:2023 (Unrestricted Access to Sensitive Business Flows) — automatable flow without adequate friction

## Reproduction Steps
1. Configure `CAPTCHAEnabled=false` (default dev)
2. From single IP, send 10 magic-link requests in 1 minute → 11th blocked (OK)
3. From 50 different IPs (simulated), each send 10 requests/minute → 500 requests/minute total (not blocked)
4. Enable CAPTCHA → verify challenge presented

## Proposed Solutions
- **Option A (Quick)**: Lower `authRateIP` from `PerMinute(10)` to `PerMinute(5)`
- **Option B (CAPTCHA enforcement)**: Set `CAPTCHAEnabled=true` in production config; make CAPTCHA mandatory for magic-link send
- **Option C (Adaptive)**: Implement progressive rate limiting — stricter limits after N failures, or per-subnet limits
- **Option D (Email domain allowlist)**: Restrict magic links to known email domains (enterprise feature)

## Decision
Pending — recommend Option A + Option B for immediate hardening.

## Acceptance Criteria
- [ ] `authRateIP` changed to `redis_rate.PerMinute(5)` in `router.go`
- [ ] `CAPTCHAEnabled=true` in production environment config
- [ ] Magic-link send requires valid CAPTCHA token in production
- [ ] Integration test: 6 requests from same IP in 1 minute → 429 on 6th
- [ ] Integration test: invalid CAPTCHA → 400 with appropriate error code
- [ ] Document rate limit rationale in security docs

## Links
- Security audit report: `security-report-test.md`
- OWASP API6:2023 reference: `docs/reference/owasp-api-top10-2023.md#api62023---unrestricted-access-to-sensitive-business-flows`
- Rate limit middleware: `backend/internal/api/middleware/ratelimit/`
- CAPTCHA middleware: `backend/internal/api/middleware/captcha/`
- Config: `backend/internal/config/config.go`