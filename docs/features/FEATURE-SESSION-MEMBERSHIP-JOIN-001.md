---
id: FEATURE-SESSION-MEMBERSHIP-JOIN-001
title: Session Validation with Membership Join Optimisation
status: approved
depends_on:
  - ADR-0002
  - FEATURE-STYTCH-B2B-ONBOARDING-SESSION-001
touches:
  - backend
---

# Feature

## Goal
Reduce per-request DB round-trips for authentication by resolving the opaque session and the user's active tenant membership in a single database query, while keeping membership live and never caching role/school in the session.

## Out of scope
* Caching memberships in Redis or in `app_session`
* Changing cookie format, session expiry policy, or Stytch integration
* Frontend changes – frontend remains cookie-only

## Domain references
* docs/adr/0002-auth-and-roles.md – Go is sole security authority, role comes from live membership
* docs/architecture/session-resolution.md – session resolution contract
* backend/AGENTS.md BE-018, BE-017 – never cache role, load session context once per request

## Acceptance criteria
- AC-1: `service.ValidateSession` returns `SessionContext` plus active membership in a single DB round-trip
- AC-2: No `school_id` or `role` is persisted in `app_session`; session row remains token hash, user/tenant/expiry only
- AC-3: Middleware `RequireAuth` still derives `user_id`, `tenant_id`, `roles`, `school_id` from the combined result, no additional query (only one active membership exists per user due to `memberships_active_user_uniq_idx`)
- AC-4: Revocation and expiry are still enforced live; a revoked session returns `ErrSessionNotFound` even if memberships exist
- AC-5: Tenant isolation is preserved; cross-tenant memberships are never returned
- AC-6: Existing tests for tenant isolation and RBAC continue to pass without modification to expectations
- AC-7: Migration is not required; change is read-path only. If a new view is introduced, follow BE-037 migrator_integration_test.go verification

## API changes
No API changes. OpenAPI unchanged. Internal service interface changes only.

## Backend notes
* Repository method `GetSessionWithMemberships(ctx, tokenHash []byte) (AppSession, Membership)` replaces separate `GetAppSessionByTokenHash` + `GetUserMemberships` calls in the auth middleware path.
* SQL: single query joining `app_session s` → `users u` → `memberships m` filtered by `s.token_hash = $1`, `s.revoked_at IS NULL`, `s.expires_at > now()`, `m.is_active = true`, `u.tenant_id = s.tenant_id`. Returns at most one row per user (enforced by partial unique index `memberships_active_user_uniq_idx`).
* Keep Repository interface small; service never imports sqlc types.
* Ensure error mapping stays per `apperr` contract.
* No Redis involvement. Keep Postgres as source of truth per BE-028.

## Frontend notes
None. Frontend remains cookie-presence only.

## Verification
* Unit: mock repository returns session + membership; middleware builds locals correctly
* Integration: authenticated request hits protected route, logs show single DB query for session+memberships
* Security: revoke session → subsequent request 401 despite cached membership data in app
* Run `make compliance`, `go test ./...`, `go vet`, `golangci-lint`

## Open questions
* Do we need a dedicated SQL view for session+memberships for readability vs inline join?
* Should we also pre-load guardian students / teacher classes in same round-trip for those RBAC checks, or keep lazy?

## Decisions
* Do not store active school id in session. Membership remains live per BE-018.
* Optimisation is read-path only; Postgres remains authoritative.
