# Phase 2 — Domain Knowledge

Entities, relationships, business rules, and workflows. All content below is derived from verified sources: backend migrations (`backend/migrations/`), backend `AGENTS.md` rules, and `backend/internal/app/app.go`.

## Status

draft — bootstrapped from code. No human sign-off yet.

## Entities

Entity list by migration file. Every table has exactly one owning domain; only that domain's store writes to it (`BE-003`).

### `00000000000001_extensions.sql`

- `pgcrypto` extension (enables `uuidv7()`).

### `00000000000002_auth_domain.sql` — owning domain: identity

| Table | Description |
|-------|-------------|
| `pending_onboarding` | Resumable onboarding state per normalized email; unique `idempotency_key`; status enum `pending|provisioning|completed|failed_retryable` |
| `app_session` | Opaque sessions; `token_hash` (SHA-256 of token), `stytch_member_session_id`, `expires_at`, `revoked_at` |
| `onboarding_ticket` | Short-lived onboarding ticket; encrypted IST ciphertext/nonce; `used_at`, `expires_at` |
| `pkce_codes` | PKCE codes for magic-link flow |

### `00000000000003_core_domain.sql` — owning domain: core (tenancy / identity base)

| Table | Description |
|-------|-------------|
| `countries` | Reference table, ISO 3166-1 alpha-2 country codes; `ON DELETE RESTRICT` |
| `education_systems` | Belongs to exactly one country (`ON DELETE RESTRICT`) |
| `class_levels` | Grade levels scoped to an education system |
| `tenants` | Organization boundary for data isolation; `tenant_status` enum `ACTIVE|SUSPENDED`; Stytch organization ID link |
| `users` | Identity per tenant; `email_normalized` unique per tenant; Stytch member ID link |
| `schools` | Institution within a tenant; `code` unique per tenant; links country + education system |
| `memberships` | User ↔ School role binding; `role` enum; `is_active`; one active membership per user (partial unique index) |

Roles: `OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN, FINANCE, PLATFORMOPERATOR`. All roles, including Guardian, are in MVP scope.

### `00000000000004_student_domain.sql` — owning domain: students

| Table | Description |
|-------|-------------|
| `students` | Scoped to a school; `admission_number` unique within a school |
| `guardian_student_links` | Links a guardian membership to a student; `relationship_type`; `is_emergency_contact` (at most one emergency contact per student via partial unique index) |

### `00000000000005_academic_domain.sql` — owning domain: academics

| Table | Description |
|-------|-------------|
| `subjects` | Belongs to education system + class level (`ON DELETE RESTRICT`) |
| `topics` | Belongs to a subject; `sequence_index` |
| `subtopics` | Belongs to a topic; `sequence_index` |
| `streams` | Color/group name scoped to a school |
| `classrooms` | School + class level + stream (unique combination) |
| `academic_years` | Scoped to a school; `is_current` per school (partial unique index) |
| `terms` | 1–3 per academic year; `sequence_index`, `start_date`, `end_date` |
| `enrollments` | One active enrollment per student per term (partial unique index); `enrollment_status` enum `ACTIVE|TRANSFERRED|GRADUATED|WITHDRAWN` |

### `00000000000006_rls_and_security.sql` — cross-cutting security

- `trigger_set_timestamp()` maintains `updated_at` on all domain tables via `BEFORE UPDATE` triggers (`BE-039`).
- `current_tenant_id()` SQL function resolves tenant from session GUC `app.current_tenant_id`.
- RLS enabled + `FORCE ROW LEVEL SECURITY` on all tables; tenant-scoped policies (`BE-016`, `BE-040`).
- Soft delete column `deleted_at` on all tables with partial indexes `WHERE deleted_at IS NULL` (`BE-041`).

## Relationships

Verified from foreign-key constraints in migrations:

- `education_systems.country_id → countries.id` (RESTRICT)
- `class_levels.education_system_id → education_systems.id` (RESTRICT)
- `tenants` ← no FK (root of tenancy)
- `users.tenant_id → tenants.id` (CASCADE)
- `schools.tenant_id → tenants.id` (CASCADE)
- `schools.country_id → countries.id` (RESTRICT)
- `schools.education_system_id → education_systems.id` (RESTRICT)
- `memberships.user_id → users.id` (CASCADE)
- `memberships.school_id → schools.id` (CASCADE)
- `pending_onboarding.tenant_id` (nullable; links provisioned tenant after completion)
- `students.school_id → schools.id` (CASCADE)
- `guardian_student_links.guardian_id → memberships.id` (CASCADE)
- `guardian_student_links.student_id → students.student_id` (CASCADE)
- `subjects.education_system_id → education_systems.id` (RESTRICT)
- `subjects.class_level_id → class_levels.id` (RESTRICT)
- `topics.subject_id → subjects.id` (RESTRICT)
- `subtopics.topic_id → topics.id` (RESTRICT)
- `streams.school_id → schools.id` (RESTRICT)
- `classrooms.school_id → schools.id` (RESTRICT)
- `classrooms.class_level_id → class_levels.id` (RESTRICT)
- `classrooms.stream_id → streams.id` (RESTRICT)
- `academic_years.school_id → schools.id` (RESTRICT)
- `terms.academic_year_id → academic_years.id` (RESTRICT)
- `enrollments.student_id → students.student_id` (RESTRICT)
- `enrollments.classroom_id → classrooms.id` (RESTRICT)
- `enrollments.term_id → terms.id` (RESTRICT)
- `onboarding_ticket.pending_onboarding_id → pending_onboarding.id` (CASCADE)
- `app_session.user_id`, `app_session.tenant_id` (no FK declared in migration 02; tenant-scoped by RLS policy)

**Inference:** subject→topic→subtopic is a 3-level academic hierarchy. Subjects are scoped to `education_system_id` + `class_level_id`, so multiple countries' curricula (Kenya CBC, Tanzania, Uganda later) can be added under their own `countries` → `education_systems` rows. Each `school` references one `education_system_id`, fixing its curriculum. This replaces the older CBC "learning area / strand / sub-strand" terminology; the subject/topic/sub-topic model is the adopted model (see `docs/product/README.md`).

## Business rules

### Tenancy and authorization

- Every query is tenant-scoped and school-scoped for school data, in addition to RLS (`BE-016`).
- The app connects as a non-owner DB role; RLS variables set per transaction from session context (`BE-016`).
- Cross-tenant and nonexistent records return the identical 404 (`BE-016`, `BE-017`).
- The session context (user, tenant, school, role) is loaded once per request from one query and carried in `context.Context` as a typed struct (`BE-017`).
- The role always comes from live membership; never cached in session or Redis (`BE-018`).
- Authorization uses permissions, never role lists (`BE-020`).
- Services obtain a `Scope` from `authz.ScopeFor(ctx, perm)` and pass it to the store (`BE-021`). `ScopeFor` currently returns `"scope:" + resource`; the full scope/permission matrix must be implemented via specification.
- Nothing else reads or writes the active school; never accept a school/tenant ID from the client as scope (`BE-017`).
- Frontend never sends a school identifier or `X-Active-School` header; no `[school]` route segments (`FE-007`, `FE-011`).

### Sessions and identity

- Sessions use an opaque random token (≥256 bits); only SHA-256 hash stored (`BE-019`).
- Cookie `__Secure-session_token`, `HttpOnly; Secure; SameSite=Lax`, `Domain=domain.com`, `Path=/`, per-environment name suffix (`BE-019`).
- Reject unsafe methods whose `Origin` doesn't match; no state changes on GET (`BE-019`).
- Sessions have idle and absolute expiry owned by our DB; absolute expiry never exceeds Stytch session's `expires_at` (`BE-019`).
- Stytch is called at login and logout only, never per request (`BE-019`).
- The magic-link callback is a POST, never a GET (`BE-019`).
- The session stores the Stytch member-session ID for revocation (`BE-019`).
- Stytch is reached only through an interface declared in `identity/service`, implemented by adapter in `platform/stytch` (`BE-019`).

### Data modeling

- UUID primary keys; UUIDv7 defaults for new rows (`BE-013`, `BE-038` — never `gen_random_uuid()`).
- Timestamps `timestamptz` UTC; school-day dates `date` interpreted in Africa/Nairobi (`BE-013`).
- Money is `bigint` minor units (KES), never floats (`BE-013`, `FE-024`).
- `updated_at` maintained via DB trigger, not only app code (`BE-039`).
- Soft delete via `deleted_at` + partial index; prefer hard delete when domain allows (`BE-041`).
- Reference tables (`countries`, `education_systems`) use `ON DELETE RESTRICT`, never CASCADE (`BE-042`).
- Status/check fields use `ENUM`, not `TEXT CHECK` (`BE-043`).
- Composite indexes for frequent lookup pairs (`BE-044`).
- Bulk operations (import, rollover) are atomic with rollback (`BE-015`).

### API contract

- Every API route under `/api/v1/` (webhooks included) (`BE-022`).
- API v1 status: **unreleased** — breaking changes allowed until first production deploy (`BE-022`).
- OpenAPI contract `docs/api/openapi.yaml` is the frontend type source of truth, backend-owned (`BE-023`).
- JSON field names snake_case; query parameter names camelCase; IDs UUID strings; timestamps RFC 3339 UTC `Z`; dates `yyyy-MM-dd`; money integer minor units with `Minor` suffix (`BE-025` naming conventions).
- Enum values SCREAMING_SNAKE strings (`BE-025`).
- URLs plural kebab-case nouns; non-CRUD actions `POST /<resource>/{id}/<verb>` (`BE-025`).
- Single resource returns bare object; lists return `{ items, nextCursor }`; no null values, arrays are `[]` (`BE-025`).
- Updates use `PUT` with full editable body; no generic `PATCH` (`BE-025`).
- Success codes: 200 reads/updates, 201 creates, 202 queued work, 204 deletes (`BE-025`).
- Error body `{ code, message, errors?, request_id }`; `code` is stable SCREAMING_SNAKE enum (`BE-025`).
- 401 `UNAUTHENTICATED`, 403 `FORBIDDEN`, 404 `NOT_FOUND`, 409 `CONFLICT`, 422 `VALIDATION_FAILED`, 429 `RATE_LIMITED` with `Retry-After`, 500 `INTERNAL` generic message only (`BE-025`).

### Jobs, Redis, audit

- Background jobs use asynq; handlers idempotent; payloads carry `tenant_id` and `school_id` (`BE-027`).
- Redis holds ephemeral data only; Postgres is source of truth; queue and cache never share eviction policy (`BE-028`).
- Rate-limit fallback: in-process limiter fail open if Redis down (`BE-028`).
- Audit: create/update/delete on health, behavior, fee records written in same transaction; view events via `audit.Recorder` (queued; direct DB write on enqueue failure; deny view if both fail) (`BE-029`).
- Audit rows hold IDs and changed field names, never sensitive values; fee waivers also record old/new value, proposer, approver (`BE-029`).

## Workflows

### Authentication and onboarding (verified from `docs/api/openapi.yaml` and migrations)

1. User requests a magic link (`POST /api/v1/auth/request-magic-link`); response always success (no account enumeration).
2. User authenticates the Stytch Discovery token (`POST /api/v1/auth/authenticate-token`); response includes `auth_state` (`onboarding_required`, `organization_selection_required`, `authenticated`), ticket token, and onboarding state.
3. Onboarding state is polled/updated (`GET`/`PUT /api/v1/onboarding/current`).
4. Onboarding completes and provisions the tenant (`POST /api/v1/onboarding/complete`); issues session cookie; idempotency-keyed (`idempotency_key`; 409 on conflict).
5. User logs out (`POST /api/v1/auth/logout`); session revoked.

### Session lifecycle (inferred from `BE-019`)

Login → Stytch call → create `app_session` with token hash → set cookie → subsequent requests resolve session from cookie → middleware sets `app.current_tenant_id` GUC for RLS → logout revokes session and clears cookie.

**Assumption:** the exact mapping from Stytch organization to tenant/school selection is not fully spelled out in code; confirm before building organization-selection workflows.
