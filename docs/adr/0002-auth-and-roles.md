---
id: ADR-0002
title: Auth and Roles
status: accepted
depends_on: []
touches: [backend, frontend]
---

# ADR-0002 Auth and Roles

## Context
The platform requires tenant-scoped authentication for PP1-Grade 9 schools under CBC. Users access the system via Stytch B2B Discovery magic links. Frontend is TanStack Start, backend is Go Fiber. Security must be enforceable server-side with mobile-first intermittent connectivity constraints.

## Options
1. Frontend validates session and backend trusts frontend checks. 2. Backend validates every request, frontend guard is UX only. 3. Shared session validation endpoint called on every navigation.

## Decision
Go is the sole security authority. It validates the opaque session token on every protected API request. The frontend guard checks only cookie presence for navigation UX and never calls Go for validation.

Opaque token delivered only via Set-Cookie HttpOnly Secure SameSite=Lax Domain=domain.com Path=/, never in JSON body or URL. Cookie name is __Secure-session_token, per-environment name suffix to avoid cross-env leakage.

Token hygiene: ≥256 bits entropy, store only SHA-256 hash, rotate at login.

Frontend route guard uses pure routePolicy function for middleware and beforeLoad. Data fetching is client-only.

## Consequences
Positive: security boundary is unambiguous, frontend can be bypassed without data exposure, stale cookies are handled gracefully by backend 401.
Negative: frontend route guard cannot prove session validity, may briefly show shell before API 401.
Operational: CORS must be exact origin with credentials, middleware order matters, CSRF protections required for mutations.

See FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 and FEATURE-AUTHENTICATED-CHECK-SESSION-COOKIE-001 for implementation details.
