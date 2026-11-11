---
id: ARCH-AUTH-SESSION-MAP
title: Auth & Session Flow Map
status: approved
depends_on:
  - ADR-0002
  - FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001
  - FEATURE-AUTHENTICATED-CHECK-SESSION-COOKIE-001
  - FEATURE-SESSION-MEMBERSHIP-JOIN-001
touches:
  - backend
  - frontend
---

# Auth & Session Flow Map

## Purpose

One place to orient yourself on the full auth + session lifecycle. It does **not** duplicate spec content — each step links to the owning feature spec or architecture doc, which remains the source of truth (acceptance criteria, tests, decisions).

**Rule:** If you change behavior, edit the owning spec, not this map. Update this map only to add/remove/rename links.

## End-to-end flow

| # | Step | Who acts | Owning spec / doc | Key ACs |
|---|------|----------|-------------------|---------|
| 1 | User enters email; Stytch Discovery magic link sent | Frontend → Stytch | FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 | AC-1 |
| 2 | User clicks magic link; frontend callback extracts one-time token, strips it from URL, sets `Referrer-Policy: no-referrer` | Frontend | FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 | AC-2 |
| 3 | Go verifies token with Stytch; creates/updates `PendingOnboarding` keyed by normalized verified email (status `pending`) | Backend | FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 | AC-3, AC-4 |
| 4 | Returning user re-authenticates with a new magic link; Go resumes the existing `PendingOnboarding` at its saved step | Backend → Stytch | FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 | AC-6, AC-7 |
| 5 | User submits tenant/school details; Go provisions Stytch org + member, then local tenant + user + membership (idempotent) | Backend → Stytch + DB | FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 | AC-8, AC-9, AC-10 |
| 6 | Go issues opaque session token (≥256 bits), stores SHA-256 hash in `app_session`, sets `__Secure-session_token` cookie | Backend | FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 | AC-11 |
| 7 | Frontend guard checks only cookie presence for navigation UX; never reads or validates the session | Frontend | FEATURE-AUTHENTICATED-CHECK-SESSION-COOKIE-001 | AC-1 to AC-10 |
| 8 | Every protected API request: Go hashes cookie token, validates session (expiry/revocation), resolves `user_id`, `tenant_id`, active membership in one DB query | Backend | FEATURE-SESSION-MEMBERSHIP-JOIN-001 | AC-1 to AC-7 |
| 9 | Session resolution contract: fails closed on tenant mismatch, cross-tenant membership yields empty memberships, role/school never cached in session | Backend | docs/architecture/session-resolution.md | — |
| 10 | Logout: revokes `AppSession` and Stytch member session, clears cookie with matching name/Domain/Path | Backend + Frontend | FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 | AC-13 |

## Contracts (reference only — do not edit here)

| Contract | Source |
|----------|--------|
| Go is sole security authority; frontend guard is UX-only | ADR-0002 |
| Cookie: `__Secure-session_token` + per-environment suffix, HttpOnly, Secure, SameSite=Lax, Domain=domain.com, Path=/ | ADR-0002 |
| Token ≥256 bits entropy; only SHA-256 hash stored; rotated at login | ADR-0002 |
| Session resolution flow, token hash lookup, active membership join | docs/architecture/session-resolution.md |
| Rate limiting (magic-link per IP/email, token exchange per IP) | FEATURE-PROD-SEC-BASELINE-001 |

## Domain entities involved

| Entity | Role in flow | Defined in |
|--------|--------------|------------|
| `PendingOnboarding` | Resumable pre-tenant state keyed by verified email | FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 / docs/domain/model.md |
| `AppSession` | Opaque session, hash stored, raw in cookie only | FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001 / docs/domain/model.md |
| `Tenant`, `School`, `User`, `Membership` | Provisioned after tenant naming; membership active per user | docs/domain/model.md |

## Deprecated / removed

- `docs/features/FEATURE-STYTCH-SESSION-001.md` — `status: deprecated`, merged into `FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001`, removed. Do not re-create.

## Open questions

- Final production domain for cookie `Domain` and CSP (FEATURE-PROD-SEC-BASELINE-001 open questions).
- Stytch PKCE enablement for login-CSRF prevention (FEATURE-AUTHENTICATED-CHECK-SESSION-COOKIE-001 open questions).
- Same-origin proxy under `app.domain.com/api/*` for `__Host-` cookie (currently uses `__Secure-session_token` with `Domain=domain.com`).
