---
id: ARCH-SEC-BASELINE
title: Production Security Baseline Assessment
status: draft
depends_on: []
touches: []
---

# Production Security Baseline Assessment

## Scope
Go Fiber v3 REST API security baseline covering rate limiting, multi-tenant isolation, HTTPS/proxy, security headers, CORS/CSRF, session cookies, and observability per Somotracker document-led workflow.

## Repository Assessment Summary

### Existing stack
- Backend: Go 1.26, Fiber v3.0.0, pgx/v5, redis/go-redis/v9
- Config: `internal/platform/config/config.go` loads `APP_ENV/ENV`, `DATABASE_URL/POSTGRES_DSN`, `REDIS_ADDR`/`REDIS_URL`, `PORT/SERVER_PORT`, `STYTCH_*`
- App init: `internal/app/app.go` wires DB, migrations, Redis, Fiber server
- Middleware currently applied: `recover`, `helmet` default, `cors` with hardcoded origins `http://localhost:3000`, `logger`
- Auth: Stytch B2B magic-link flow, opaque session cookie `__Secure-session_token` with env suffix, `HttpOnly`, `Secure` always true, `SameSite=Lax`
- Session validation: `internal/identity/middleware/auth.go` with SHA-256 token hash lookup
- Tenant middleware extracts `tenant_id` from session locals, stores in context
- CSRF middleware exists but uses hardcoded allowed origins
- Docker Compose provides `postgres`, `redis` with ports published `5432:5432`, `6379:6379`
- Tests exist for identity handler, middleware, apperr, httpserver, logger

### Identified gaps vs specification

#### Environment & Deployment Security
- [ ] Config does not enforce distinct development/test/production validation; missing required production config startup failure
- [ ] Session cookie `Secure` is hard-coded true → breaks local HTTP dev
- [ ] No explicit HTTPS detection / proxy trust configuration
- [ ] No HSTS configuration
- [ ] Redis port published publicly in docker-compose.yml

#### Redis provisioning
- Redis provided in compose but port exposed. Health check present.
- No Redis authentication/TLS config validation
- No startup connectivity verification beyond Ping

#### Rate limiting
- No rate limiting middleware present
- No Redis-backed IP or email limits for magic-link / token exchange
- No email provider quota caps
- No failure-closed policy documented

#### Token verification & session security
- Cookie `Secure` always true, prevents local HTTP
- Cookie name changes per env, acceptable
- No rate limiting on token verification
- No documented token replay protection beyond existing flow

#### Proxy, HTTPS, transport
- Fiber `TrustProxy` not configured
- No explicit client IP extraction validation
- No HSTS
- No HTTPS enforcement logic

#### Security headers
- Helmet used with defaults, not explicitly configured per policy table
- CSP not configured
- Permissions-Policy not reviewed

#### CORS & CSRF
- CORS origins hardcoded, no env separation
- CSRF middleware uses hardcoded origins
- No `SameSite`/`Domain` review

#### Multi-tenant isolation tests
- No integration tests for cross-tenant isolation found
- No tenant fixtures

#### Observability
- Logging redaction exists but rate-limit, quota exhaustion events not emitted
- No metrics for security events

## Documentation Checklist

1. Repository assessment and configuration ✓ partial
2. Redis provisioning and connection management
3. IP-based and email-based rate limiting
4. Token verification and session security
5. Proxy and environment-specific HTTPS behavior
6. Security headers and CSP
7. CORS and CSRF review
8. Tenant-isolation integration tests
9. CI integration
10. Final verification

## Next steps
- Create spec feature doc with status approved
- Centralize config with env validation
- Make session cookie Secure conditional on env
- Remove public Redis port exposure
- Add Fiber TrustProxy config for Render
- Implement Redis-backed rate limiting for auth endpoints
- Add tenant isolation integration tests
- Update CORS/CSRF to env-based allowlists
- Configure explicit Helmet headers and HSTS for production
- Document Render production variables

## Non-negotiables confirmed
- Preserve Stytch architecture
- No PostgreSQL RLS
- No frontend-only security
- Do not trust client-supplied tenant ID
- Local HTTP development must remain functional
- Production security must not be weakened for dev
