---
id: FEATURE-AUTHENTICATED-CHECK-SESSION-COOKIE-001
title: Authenticated Check Session Cookie
status: draft
depends_on:
  - ADR-0002
  - FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001
touches:
  - frontend
---

# Feature

## Goal
Provide a consistent, tenant-aware authentication guard for the TanStack Start frontend that separates navigation convenience from security authority. Stytch authenticates the user, Go creates and validates the application session, and the browser stores an opaque session token in a shared-domain HttpOnly cookie. The frontend checks only cookie presence for routing; Go remains the sole authority for authentication and authorization on every protected API request.

## Out of scope
- Changes to Stytch integration or session creation logic in Go
- New API endpoints for authentication; existing session endpoints remain
- Backend session store changes (Redis/PostgreSQL)
- SMS notifications, Swahili launch, billing
- Periodic proactive session validation beyond API calls
- Stytch magic-link flow implementation – TODO: separate large feature

## Domain references
- docs/domain/glossary.md — Tenant, User, Membership
- docs/domain/model.md — Tenant → School → User via Membership, session authority in backend
- docs/adr/0002-auth-and-roles.md — Auth and Roles, server-side role checks
- docs/product/brief.md — Role checks enforced server-side on every request, mobile-first intermittent connectivity
- docs/features/FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001.md — Stytch B2B magic-link auth, resumable onboarding and opaque session creation

## Acceptance criteria
- AC-1: Frontend guard checks only cookie presence, makes no calls to Go, Redis, PostgreSQL or Stytch
- AC-2: Missing cookie on protected routes redirects to /login; existing cookie on /login or /signup redirects to /; /auth/callback is allowed without redirect; /onboarding/tenant-name is allowed without cookie, redirects to / if cookie present
- AC-3: Client-side navigation follows same policy without reading HttpOnly cookie
- AC-4: routePolicy is single source of truth for middleware and beforeLoad; both use it and can be unit tested independently
- AC-5: Middleware does not redirect server function calls, static assets, favicon, fonts, images, or framework internals
- AC-6: Public routes, assets and framework internals never enter redirect loops
- AC-7: No flash on full page load or client-side navigation; redirects happen before HTML render or before route commit
- AC-8: TODO Login callback strips Stytch credential from URL via history.replaceState, sets Referrer-Policy:no-referrer, and keeps credential out of logs/analytics – Stytch flow is out of scope for this feature
- AC-9: Stale cookie passes navigation guard intentionally; frontend handles 401 from first protected API call by redirecting to /login without loops
- AC-10: Logout is POST with JSON body via button onClick fetch, credentials:'include', never triggered by TanStack Router preloading

## API changes
Pointer to docs/api/openapi.yaml diff: none for this feature. Existing session endpoints remain unchanged. If session creation or logout paths are modified, update openapi.yaml and regenerate types via `make generate`.

## Backend notes
No backend changes are introduced by this feature. Existing Go session validation, CORS configuration, cookie attributes and logout behavior remain unchanged. The frontend guard assumes the backend remains the sole security authority for authentication and authorization.

## Frontend notes
- Cookie presence only for navigation. Never read HttpOnly cookie in JS.
- routePolicy(pathname, hasCookie) returns allow or redirect. Central pure function used by middleware and beforeLoad.
- Layer A: TanStack Start request middleware for document requests with Accept:text/html and Sec-Fetch-Dest:document. Allowlist paths only; do not redirect server functions, static assets, API requests, /auth/callback, /onboarding/tenant-name, /logout, /unauthorized, health endpoints.
- Layer B: root route beforeLoad calls server function getAuthState that reads Cookie header on TanStack server and returns { hasSession: boolean }. Result placed in router context; child guards use routePolicy.
- Login callback: TODO Stytch integration – read temporary Stytch credential, remove from URL, POST to api.domain.com/auth/session with credentials:'include', then full navigation or router.invalidate().
- Set Referrer-Policy:no-referrer on callback, avoid token in logs/analytics. Stytch flow is TODO.
- Global API client uses credentials:'include' for all protected calls. Global 401 handler triggers logout, debounced.
- No flash: server redirect on full load, beforeLoad before route commit on client navigation. Avoid useEffect guards.

## Verification
- Verify AC-1, AC-4: unit tests for routePolicy covering route categories with cookie present/absent
- Verify AC-2, AC-3, AC-5, AC-6, AC-7: integration tests for middleware redirect decisions and beforeLoad context propagation; ensure server functions, assets and framework internals are not redirected
- Verify AC-8: callback page tests for URL stripping, referrer policy, and credential removal from logs
- Verify AC-9: simulated stale cookie flow results in 401 handling and redirect to /login without loops
- Verify AC-10: logout link cannot be triggered by link preloading; router behavior validated
- Run `make compliance` and `/review` before status change

## Decisions
- CSRF Origin missing behavior: mutations require Origin header. If Origin is missing, allow only when Sec-Fetch-Site is 'same-origin'. Reject otherwise. Can harden later with double-submit token.
- ?redirect= param: supported post-login with allowlist of same-origin relative paths only.
- Local development: use app.local.domain.com over HTTPS or a dev proxy to avoid Domain/Secure issues on localhost.

## Open questions
- TODO Stytch integration is a separate large feature. Is Stytch PKCE enabled to prevent login-CSRF?
- Should same-origin proxy under app.domain.com/api/* be adopted to enable __Host- cookie and reduce CSRF surface? Proposed for future hardening.
