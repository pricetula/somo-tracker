---
id: FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001
title: Stytch B2B Magic-Link Authentication, Resumable Onboarding and Opaque Sessions
status: approved
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

## Entities
**Tenant** – organization boundary for data isolation
- id `uuid`
- name `text` unique
- stytch_organization_id `uuid` nullable unique
- status `enum(active, suspended)`
- created_at `timestamptz`, updated_at `timestamptz`

**User** – identity per tenant
- id `uuid`
- tenant_id `uuid` FK
- email_normalized `text` unique per tenant
- stytch_member_id `uuid` nullable unique
- name `text`
- created_at `timestamptz`, updated_at `timestamptz`

**School** – institution within tenant
- id `uuid`
- tenant_id `uuid` FK
- name `text`
- code `text` unique per tenant
- created_at `timestamptz`, updated_at `timestamptz`

**Membership** – User ↔ School role binding
- id `uuid`
- user_id `uuid` FK
- school_id `uuid` FK
- role `enum(tenant_owner, head, admin, teacher, finance, guardian, platform_operator)`
- created_at `timestamptz`

**PendingOnboarding** – resumable onboarding state before tenant provisioning
- verified_email_normalized `text` PK-ish, unique
- status `enum(pending, provisioning, completed, failed_retryable)`
- current_step `text`
- form_data `jsonb`
- stytch_organization_id `uuid` nullable
- stytch_member_id `uuid` nullable
- tenant_id `uuid` nullable
- school_name `text` nullable
- created_at `timestamptz`, updated_at `timestamptz`, last_activity_at `timestamptz`
- idempotency_key `text` nullable, unique

**AppSession** – opaque application session
- id `uuid`
- user_id `uuid`
- tenant_id `uuid`
- token_hash `bytea` unique
- stytch_member_session_id `text` nullable
- created_at `timestamptz`, expires_at `timestamptz`, revoked_at `timestamptz` nullable

**OnboardingTicket** – short-lived binding for IST to pending record
- id `uuid`
- pending_onboarding_id `uuid` FK
- ticket_hash `bytea` unique
- encrypted_ist `bytea` nullable (encrypted at rest)
- used_at `timestamptz` nullable, expires_at `timestamptz`

**PKCECode** – per-auth PKCE verifier
- pending_onboarding_id `uuid` FK
- code_verifier `text` (hashed at rest)
- expires_at `timestamptz`

Notes: No IST, magic-link token or raw session token is stored in PendingOnboarding. Token hashes only. RLS: all queries tenant-scoped where applicable.

## Acceptance criteria
- AC-1: POST /api/v1/auth/request-magic-link requests a Stytch Discovery magic link, applies rate limiting, and returns generic success without revealing account existence
- AC-2: Frontend callback page extracts one-time token, removes it from URL via history.replaceState, sets Referrer-Policy:no-referrer, and POSTs it to Go
- AC-3: Go authenticates Discovery magic link with Stytch, uses verified email only, creates or updates pending onboarding record with status pending and current_step
- AC-4: Pending onboarding record can exist with verified_email_normalized only, with school_name, tenant_id, stytch_organization_id and stytch_member_id all NULL
- AC-5: User can leave onboarding without entering school details; pending record persists
- AC-6: Returning after a month with new magic link re-authenticates email, finds pending record by normalized verified email, and resumes saved step
- AC-7: Identity resume requires fresh Stytch authentication; email normalization is consistent and uniqueness enforced per policy
- AC-8: After Discovery authentication, Go inspects discovered_organizations and returns onboarding_required, organization_selection_required, or authenticated per product policy
- AC-9: Provisioning creates Stytch organization and Member via current Discovery exchange/create endpoint, saves stytch_organization_id and stytch_member_id, then creates local tenant, user and membership in a recoverable workflow
- AC-10: Provisioning is idempotent with application idempotency key, unique constraints on Stytch IDs, and state tracking pending/provisioning/completed/failed_retryable
- AC-11: Opaque application session token is generated with ≥256 bits entropy after tenant is named and Stytch organization/member is created, hash stored in PostgreSQL, raw token issued in __Secure-session_token cookie with Domain=domain.com, Path=/, HttpOnly, Secure, SameSite=Lax, Max-Age aligned to session policy, per-environment cookie name; no session cookie is issued during pre-tenant onboarding steps
- AC-12: Backend validates every protected API request: token hash lookup, expiry/revocation, user and tenant resolution, tenant membership and RBAC per RULE-001
- AC-13: Logout revokes server session, revokes Stytch member session, and clears cookie with same name/Domain/Path; clearing cookie alone is insufficient
- AC-14: No IST, magic-link token or raw session token is logged, stored in pending onboarding, or exposed to browser JavaScript
- AC-15: Cross-system partial failures are recoverable; Stytch success with DB failure is reconciled, DB success with cookie delivery failure allows re-authentication without duplicate tenant

## API changes
Pointer to docs/api/openapi.yaml diff: TODO — define
- POST /api/v1/auth/request-magic-link
- POST /api/v1/auth/authenticate-token
- GET /api/v1/onboarding/current
- PUT /api/v1/onboarding/current
- POST /api/v1/onboarding/complete
- POST /api/v1/auth/logout
Update openapi.yaml and run `make generate` when defined.

## Backend notes
- Pin Stytch Go SDK version and confirm exact Discovery method names before implementation
- IST is short-lived and never stored as durable onboarding state; pending onboarding record is saved after successful Discovery authentication
- Pending onboarding keyed by normalized verified email before Stytch organization/member exists; switch to stytch organization/member IDs once provisioned
- Do not create tenant on magic-link click; create only after user submits required school/tenant details. After tenant details are submitted, backend creates Stytch organization/member, exchanges current IST for a Stytch session, then issues opaque application session cookie. No session cookie exists before tenant naming.
- Onboarding ticket: after Discovery authenticate, issue host-only cookie __Host-onboarding on api.domain.com with HttpOnly, Secure, Path=/, TTL ≤ IST TTL. Server stores ticket hash + encrypted IST against pending record, single-use, deleted on use. Accepted only on tenant-name submit and pending-record read.
- PKCE: Stytch Discovery supports PKCE. Go generates pkce_code_verifier, stores in short-lived HttpOnly cookie, sends pkce_code_challenge to Stytch on magic-link send. Verifier is read from cookie on callback.
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

## Test plan
**AC → Test mapping**
- AC-1: POST /api/v1/auth/request-magic-link returns 200 generic success, rate limited, no account enumeration
- AC-2: Callback extracts one-time token, removes from URL via history.replaceState, Referrer-Policy:no-referrer, POSTs token to backend
- AC-3: Valid Stytch token creates/updates PendingOnboarding with status pending and current_step
- AC-4: Pending record created with verified_email_normalized only, school_name/tenant_id/stytch_* all NULL
- AC-5: Leaving onboarding preserves PendingOnboarding, no tenant created
- AC-6: Re-auth after 30 days finds same PendingOnboarding and resumes step
- AC-7: Email normalized consistently, uniqueness enforced, fresh auth required for resume
- AC-8: Discovered organizations inspected → returns onboarding_required / organization_selection_required / authenticated
- AC-9: Provisioning creates Stytch org/member, saves IDs, creates local tenant/user/membership atomically
- AC-10: Idempotency key prevents duplicate tenant/user on retry
- AC-11: Opaque session ≥256 bits, hash stored, cookie __Secure-session_token with correct attributes, no cookie pre-tenant
- AC-12: Protected API validates token hash, expiry/revocation, tenant membership, RBAC per RULE-001
- AC-13: Logout revokes AppSession and Stytch member session, clears cookie with matching attributes
- AC-14: IST/magic-link token/raw session never logged, stored or exposed to JS
- AC-15: Partial failures recoverable, no duplicate tenant on re-auth

**Unit tests**
- Stytch discovery authenticate mock: valid token → pending record upsert; invalid/expired token → 401
- Token generator: ≥256 bits entropy, hash storage, raw token never persisted
- Idempotency: duplicate provisioning request with same idempotency key returns same tenant/user
- Route policy: cookie presence only, no secret exposure

**Integration tests**
- End-to-end magic-link → authenticate-token → session cookie set with correct attributes
- Resume after 30 days: new magic link finds existing PendingOnboarding, preserves current_step and form_data
- Partial failure recovery: Stytch success + DB failure → reconciliation job creates pending record; DB success + cookie failure → re-auth succeeds without duplicate
- Logout revokes AppSession and Stytch member session, clears cookie
- Tenant isolation: session scoped to tenant, cross-tenant queries return 404

**Security tests**
- Token reuse rejected
- IST never logged, never stored in PendingOnboarding
- CSRF: mutations without Origin header rejected unless Sec-Fetch-Site same-origin
- Rate limiting on request-magic-link

## User acceptance scenarios
- **First-time onboarding**: User enters email → receives magic link → clicks → provides school name → tenant provisioned → opaque session cookie issued → redirected to dashboard. No session existed before tenant naming.
- **Abandoned onboarding**: User starts onboarding, closes browser. After 3 weeks, clicks new magic link → sees saved form data and resumes at same step.
- **Existing tenant login**: Authenticated user with existing tenant returns via magic link → Go inspects discovered_organizations → returns authenticated → session issued immediately, no onboarding screens.
- **Logout**: User clicks logout → session revoked server-side and in Stytch, cookie cleared, next API call returns 401 → redirected to login.

## Decisions
- Session lifetime policy: application-managed opaque session adopted for now. Session Max-Age and expires_at are set to match Stytch member session expires_at; Stytch-bound revalidation can be added later.
- PKCE: deferred for now; decision pending security review.
- Redis miss policy for session lookups: fallback to PostgreSQL adopted for now.
- Cleanup/retention for abandoned pending onboarding records: soft expire after 90 days of no activity, retain for audit, purge after 180 days.
- Email change / account linking: email change requires fresh Stytch verification and admin approval; no automatic merge of pending records.
- CSRF Origin missing behavior: mutations require Origin header. If Origin is missing, allow only when Sec-Fetch-Site is 'same-origin'. Otherwise reject. Harden later with double-submit token.

## Open questions
- Exact Stytch Go SDK Discovery method names and payload fields: use latest Stytch B2B docs; verify against pinned SDK version before implementation.
