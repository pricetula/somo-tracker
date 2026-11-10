---
id: FEATURE-STYTCH-SESSION-001
title: Stytch Magic Link Session Creation
status: deprecated
depends_on:
  - ADR-0002
touches:
  - backend
  - frontend
---

# Deprecated

Merged into FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001. Do not implement.

# Feature

## Goal
TODO: Define end-to-end Stytch magic-link authentication flow that creates an opaque application session managed by Go. Stytch authenticates the user; Go verifies the credential, creates a session token, stores it securely, and issues a shared-domain HttpOnly cookie for app.domain.com and api.domain.com.

## Out of scope
- Frontend route guard implementation – covered by FEATURE-AUTHENTICATED-CHECK-SESSION-COOKIE-001
- Session validation on API requests – separate feature
- Logout & revocation – separate feature
- CORS/CSRF hardening – separate feature

## Domain references
- docs/domain/glossary.md — Tenant, User, Membership
- docs/domain/model.md — User identity per tenant
- docs/adr/0002-auth-and-roles.md — Auth and Roles
- docs/product/brief.md — Role checks enforced server-side

## Acceptance criteria
- AC-1: TODO Magic-link email is sent via Stytch and contains single-use, time-limited credential
- AC-2: TODO Callback page reads credential from URL, removes it via history.replaceState, sets Referrer-Policy:no-referrer
- AC-3: TODO Frontend POSTs credential to api.domain.com/auth/session with credentials:'include'
- AC-4: TODO Go verifies credential with Stytch API and rejects invalid/expired/reused credentials
- AC-5: TODO On success, Go generates opaque session token with ≥128 bits entropy, stores hash in Redis/PostgreSQL, links to user and tenant
- AC-6: TODO Go responds with Set-Cookie __Secure-session_token with Domain=domain.com, Path=/, HttpOnly, Secure, SameSite=Lax, per-environment name, Max-Age aligned to policy
- AC-7: TODO Token is never returned in JSON body or URL
- AC-8: TODO Session creation fails gracefully with user-friendly error, does not create partial session
- AC-9: TODO PKCE is enabled to prevent login-CSRF, or explicit risk acceptance documented

## API changes
Pointer to docs/api/openapi.yaml diff: TODO — define POST /auth/session request/response schema, error shapes per error-contract

## Backend notes
TODO
- Stytch verification implementation
- Token generation, hashing at rest, rotation on login
- Session store strategy: Redis primary with PostgreSQL fallback
- Per-environment cookie naming

## Frontend notes
TODO
- Callback page handling of Stytch redirect
- Credential stripping from URL and logs
- Error handling for failed session creation

## Verification
- TODO unit tests for Stytch verification mock
- TODO integration tests for session creation endpoint and cookie issuance
- TODO tests for credential reuse and expiry
- Run `make compliance` and `/review` before status change

## Open questions
- TODO Is Stytch PKCE enabled?
- TODO Should ?redirect= be supported post-login with allowlist?
- TODO Session idle vs absolute timeout values?
- TODO Redis miss policy: fallback to PostgreSQL or reject?
