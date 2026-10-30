---
id: FEATURE-PROD-SEC-BASELINE-001
title: Production Security Baseline for Go Fiber API
status: approved
depends_on: []
touches: []
---

# Feature

## Goal
Document and implement production security baseline for Somotracker's Go Fiber v3 REST API covering Redis-backed rate limiting, multi-tenant isolation tests, HTTPS/proxy handling, security headers, CORS/CSRF, session cookie security, and security logging.

## Out of scope
- PostgreSQL Row-Level Security
- Replacement of Stytch or session architecture
- Frontend security controls only

## Domain references
- docs/architecture/overview.md
- docs/architecture/production-security-baseline.md
- docs/domain/model.md
- ADR-0002 auth and roles

## Acceptance criteria
- AC-1: Environment config distinguishes development/test/production with startup validation
- AC-2: Redis provisioned locally via Docker Compose without public port exposure; production config documented
- AC-3: Magic-link requests rate limited per IP 10/min and per email 3/15min via Redis
- AC-4: Token exchange rate limited per IP 10/min
- AC-5: Email provider hourly/daily caps enforced atomically in Redis
- AC-6: Session cookie Secure, HttpOnly, SameSite=Lax with env-appropriate Secure flag
- AC-7: Proxy trust configured for Render, client IP extraction correct
- AC-8: Security headers set per policy table; HSTS enabled only in production HTTPS
- AC-9: CORS allowlist env-based, no wildcard for credentialed requests
- AC-10: CSRF protection for state-changing cookie-authenticated routes
- AC-11: Multi-tenant isolation integration tests verify 404 for cross-tenant access, 401 unauthenticated
- AC-12: Rate-limit tests verify IP/email limits, atomicity, shared counters
- AC-13: Security events logged without tokens/emails; no secrets in logs
- AC-14: Docker development works with HTTP; production requires HTTPS

## API changes
Pointer to docs/api/openapi.yaml - no breaking changes expected

## Backend notes
Implement rate-limit service backed by Redis with HMAC email keys. Configure Fiber TrustProxy, Helmet, CORS per env. Centralize config validation.

## Frontend notes
Ensure frontend origins match CORS allowlist. No changes required.

## Verification
- `go test ./...` passes
- `make compliance` passes
- Integration tests for tenant isolation pass
- Rate limit tests pass
- `curl` verification of security headers in production mode

## Open questions
- Final production domain for cookie Domain and CSP
- Stytch token lifecycle details for replay protection
- Metrics backend integration
