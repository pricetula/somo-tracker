---
id: FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001
title: Stytch B2B Magic-Link Authentication, Resumable Onboarding and Opaque Sessions
status: draft
depends_on:
  - ADR-0002
touches:
  - backend
  - frontend
---

# Feature

## Goal
Enable B2B authentication via Stytch Discovery magic links, allow users to start onboarding and resume weeks later without relying on a durable Intermediate Session Token, and issue application-controlled opaque session tokens. Onboarding progress is stored as application data in PostgreSQL keyed by Stytch-verified email; the Go backend remains the security authority for all protected API requests.

## Out of scope
- Frontend route guard UX – covered by FEATURE-AUTHENTICATED-CHECK-SESSION-COOKIE-001
- SMS notifications, M-Pesa gateway, NEMIS sync
- Per-period attendance, senior school Grade 10+

## Domain references
- docs/domain/glossary.md — Tenant, User, Membership
- docs/domain/model.md — Tenant → School → User via Membership, Enrollment
- docs/adr/0002-auth-and-roles.md — Auth and Roles, server-side role checks
- docs/product/brief.md — Role checks enforced server-side on every request, mobile-first intermittent connectivity

## Acceptance criteria
- AC-1: POST /api/v1/auth/magic-link/send requests a Stytch Discovery magic link, applies rate limiting, and returns generic success without revealing account existence
- AC-2: Frontend callback page extracts one-time token, removes it from URL via history.replaceState, sets Referrer-Policy:no-referrer, and POSTs it to Go
- AC-3: Go authenticates Discovery magic link with Stytch, uses verified email only, creates or updates pending onboarding record with status pending and current_step
- AC-4: Pending onboarding record can exist with verified_email_normalized only, with school_name, tenant_id, stytch_organization_id and stytch_member_id all NULL
- AC-5: User can leave onboarding without entering school details; pending record persists
- AC-6: Returning after a month with new magic link re-authenticates email, finds pending record by normalized verified email, and resumes saved step
- AC-7: Identity resume requires fresh Stytch authentication; email normalization is consistent and uniqueness enforced per policy
- AC-8: After Discovery authentication, Go inspects discovered_organizations and returns onboarding_required, organization_selection_required, or authenticated per product policy
- AC-9: Provisioning creates Stytch organization and Member via current Discovery exchange/create endpoint, saves stytch_organization_id and stytch_member_id, then creates local tenant, user and membership in a recoverable workflow
- AC-10: Provisioning is idempotent with application idempotency key, unique constraints on Stytch IDs, and state tracking pending/provisioning/completed/failed_retryable
- AC-11: Opaque application session token is generated with ≥256 bits entropy, hash stored in PostgreSQL, raw token issued in __Secure-session_token cookie with Domain=domain.com, Path=/, HttpOnly, Secure, SameSite=Lax, Max-Age aligned to session policy, per-environment cookie name
- AC-12: Backend validates every protected API request: token hash lookup, expiry/revocation, user and tenant resolution, tenant membership and RBAC per RULE-001
- AC-13: Logout revokes server session and clears cookie with same name/Domain/Path; clearing cookie alone is insufficient
- AC-14: No IST, magic-link token or raw session token is logged, stored in pending onboarding, or exposed to browser JavaScript
- AC-15: Cross-system partial failures are recoverable; Stytch success with DB failure is reconciled, DB success with cookie delivery failure allows re-authentication without duplicate tenant

## API changes
Pointer to docs/api/openapi.yaml diff: TODO — define
- POST /api/v1/auth/magic-link/send
- POST /api/v1/auth/stytch/discovery/authenticate
- GET /api/v1/onboarding/current
- PUT /api/v1/onboarding/current
- POST /api/v1/onboarding/complete
- POST /api/v1/auth/logout
Update openapi.yaml and run `make generate` when defined.

## Backend notes
- Pin Stytch Go SDK version and confirm exact Discovery method names before implementation
- IST is short-lived and never stored as durable onboarding state; pending onboarding record is saved after successful Discovery authentication
- Pending onboarding keyed by normalized verified email before Stytch organization/member exists; switch to stytch organization/member IDs once provisioned
- Do not create tenant on magic-link click; create only after user submits required school/tenant details
- Track provisioning states and idempotency keys; use DB transactions for local writes; log Stytch request IDs and correlation IDs, never secrets
- Session policy: application-managed opaque session; optional Stytch-bound revalidation is a policy choice, not a requirement
- Cookie issuance: Set-Cookie __Secure-session_token=<raw>; Domain=domain.com; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=<seconds>. Use distinct cookie name per environment e.g. __Secure-session_token_stg to avoid parent-domain leakage. __Host- prefix is not usable with Domain.
- Clear cookie on logout with same name, Domain and Path and expired Max-Age
- CORS: exact origin https://app.domain.com, AllowCredentials true, registered before auth middleware
- CSRF protections for mutations via Origin validation and SameSite

## Frontend notes
- Login/signup UI triggers magic-link send; shows generic success
- Callback page handles token extraction and removal, avoids analytics while token in URL
- Onboarding screens load saved form data from GET /api/v1/onboarding/current after fresh authentication
- After provisioning, navigate to dashboard; do not rely on cookie presence alone for data rendering
- Global 401 handler runs logout flow; 403 routes to /unauthorized

## Verification
- Verify AC-3, AC-4, AC-5: create pending record with verified email only, no school details
- Verify AC-6: resume flow after 30 days with new magic link returns same pending record
- Verify AC-9, AC-10: provisioning idempotency and recovery from partial Stytch/DB failure
- Verify AC-11, AC-12: opaque session generation, hash storage, backend validation on protected requests
- Verify AC-14: no secrets logged; token not in URL after callback
- Run `make compliance` and `/review` before status change

## Open questions
- Exact Stytch Go SDK Discovery method names and payload fields for pinned SDK version
- Session lifetime policy: application-managed vs Stytch-bound revalidation interval
- Redis miss policy for session lookups
- Cleanup/retention policy for abandoned pending onboarding records
- Email change handling and account linking rules for existing users
