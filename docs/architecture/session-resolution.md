# Session Resolution from Opaque Token

## Context
Authentication is opaque token based per ADR-0002. The frontend never receives or sends `tenant_id`/`school_id`. All scope is derived server-side from the `__Secure-session_token` cookie.

## Flow
1. Request arrives with cookie `__Secure-session_token=<raw>`.
2. `RequireAuth` middleware hashes raw token with SHA-256.
3. `service.ValidateSessionWithMemberships` performs one query: `app_session s` joined to `users u` (tenant match, fails closed) with a LEFT JOIN to active `memberships m` filtered to the session tenant (school tenant must match; enforced in the join predicate).
4. On success, `SessionContext` contains `user_id`, `tenant_id`, `session_id`.
5. Active memberships are returned inline in the same round-trip. No role/school is cached in the session; memberships remain live per request.
6. Active roles and `school_id` (singular) are derived from the single active membership (enforced by `memberships_active_user_uniq_idx` where only one membership per user can be active).

## Resolved identifiers
* `user_id` – from `app_session.user_id`
* `tenant_id` – from `app_session.tenant_id`
* `active school id` – from live memberships; no school id is accepted from the client. For object access the specific school id is validated against the session’s active school via `RequireSchoolAccess`.

## Cookie contract
* Name: `__Secure-session_token` + per-environment suffix
* Attributes: HttpOnly, Secure, SameSite=Lax, Domain=domain.com, Path=/
* Raw token never in JSON/URL. Only SHA-256 hash stored in `app_session.token_hash`.

## Data changes
If new resolution fields are required:
* Add/alter columns via goose migration under `backend/migrations/`.
* Follow BE-037: every migration must be verified by `migrator_integration_test.go` – tables, indexes, FKs created and insert test succeeds.
* Run `make compliance` and `/review` before status change.

## Testing
* Unit: token hash generation, `ValidateSession` returns ErrSessionNotFound on missing/expired/revoked.
* Integration: session creation → cookie set → protected route returns 200 with resolved `user_id`, `tenant_id`, `school_ids`.
* Tenant isolation: session for tenant A cannot access tenant B resources – returns 404.
* Revocation: `RevokeSession` makes subsequent requests 401.

## References
* ADR-0002 Auth and Roles
* backend/internal/identity/middleware/auth.go
* backend/internal/identity/service/service.go
