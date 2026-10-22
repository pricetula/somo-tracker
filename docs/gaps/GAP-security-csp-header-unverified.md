# Gap: CSP Header Configuration Unverified / Missing

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The Helmet middleware (security headers) is registered in the Fiber app, but the Content Security Policy (CSP) configuration has not been verified to meet the security standard: `default-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self'`.

## Current Behavior
Helmet middleware exists at `backend/internal/api/middleware/helmet/` but its configuration in `main.go` or `newFiberApp` is not visible in the audited files. The security audit could not confirm:
- CSP policy is set and restrictive
- `frame-ancestors 'none'` prevents clickjacking
- `object-src 'none'` prevents plugin execution
- HSTS, X-Frame-Options, Referrer-Policy are configured

## Expected Behavior
Security headers must include:
```go
helmet.New(helmet.Config{
    ContentSecurityPolicy: "default-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'",
    XSSProtection:         "1; mode=block",
    XFrameOptions:         "DENY",
    ReferrerPolicy:        "strict-origin-when-cross-origin",
    HSTSMaxAge:            31536000, // 1 year
    HSTSIncludeSubdomains: true,
    HSTSPreload:           true,
})
```

## Impact
- **Clickjacking**: Without `frame-ancestors 'none'`, site can be embedded in malicious iframes
- **XSS Risk**: Overly permissive CSP allows inline scripts/styles
- **MIME Sniffing**: Missing `X-Content-Type-Options: nosniff`
- **Compliance**: OWASP API8:2023 (Security Misconfiguration) — missing hardening headers

## Reproduction Steps
1. Deploy backend to staging/production
2. Run `curl -I https://api.staging.somotracker.com/api/health`
3. Check for headers: `Content-Security-Policy`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Strict-Transport-Security`
4. Verify CSP policy restricts sources to `'self'`

## Proposed Solutions
- **Option A (Verify & Harden)**: Locate helmet registration in `main.go`/`newFiberApp`, update config to strict policy
- **Option B (Test-driven)**: Add integration test that asserts security headers on responses
- **Option C (CSP Report-Only)**: Start with `Content-Security-Policy-Report-Only` to detect breakage, then enforce

## Decision
Pending — recommend Option A + Option B.

## Acceptance Criteria
- [ ] Helmet middleware configured with strict CSP in `main.go` or `newFiberApp`
- [ ] `Content-Security-Policy` header present on all API responses
- [ ] CSP includes `default-src 'self'; frame-ancestors 'none'; object-src 'none'`
- [ ] `X-Frame-Options: DENY` header present
- [ ] `X-Content-Type-Options: nosniff` header present
- [ ] `Referrer-Policy: strict-origin-when-cross-origin` header present
- [ ] `Strict-Transport-Security` header present in production (HSTS)
- [ ] Integration test validates all security headers
- [ ] Frontend assets (if served by backend) still load correctly (no CSP violations)

## Links
- Security audit report: `security-report-test.md`
- Helmet middleware: `backend/internal/api/middleware/helmet/`
- OWASP API8:2023 reference: `docs/reference/owasp-api-top10-2023.md#api82023---security-misconfiguration`
- Fiber Helmet docs: https://github.com/gofiber/helmet
- Main app setup: `backend/cmd/main.go` or `backend/internal/api/` app factory