# ADR-0002: Multi-tenancy Model

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0002 |
| title | Multi-tenancy Model |
| status | proposed |
| date | 2026-11-25 |
| author | engineering |
| deciders | backend |
| relates_to | ADR-0001, ADR-0003 |

## Status

proposed

## Context

Somotracker is a B2B application where customers are organizations (e.g. school districts), each containing one or more schools. Users belong to exactly one organization but may have access to multiple schools within it. The data model must enforce data isolation between organizations so that a user in org A can never see or modify data belonging to org B.

Current schema (`migrations/00000000000002_core_domain.sql`):
- `organizations`: top-level customer account and data-isolation boundary.
- `users`: one global identity per normalized email, scoped to an organization.
- `schools`: institution within an organization.
- `user_schools`: links a user to a school within their organization, with a `role` field.

Key forces:
- Users are created during OTP verification and assigned to an organization at that time.
- New users go through onboarding where the org name and school name are provided.
- The `user_schools` table has composite foreign keys: `(organization_id, user_id)` → `users(organization_id, id)` and `(organization_id, school_id)` → `schools(organization_id, id)` to prevent cross-organization assignment.
- `GetSchoolsByUser` joins `user_schools` with `schools` on both `school_id` and `organization_id`.

## Decision

Adopt a hierarchical multi-tenancy model:

1. **Organization**: Top-level tenant and data-isolation boundary. Every resource (user, school, session) belongs to exactly one organization.
2. **User**: Global identity per normalized email, but scoped to one organization via `users.organization_id`. A user's email is globally unique (`UNIQUE(normalized_email)`).
3. **School**: Institution within an organization. School names are unique within an organization (`UNIQUE(organization_id, name)`).
4. **UserSchool**: Join table linking a user to schools within their organization. Fields: `role` (e.g. `owner`, `admin`, `teacher`, `viewer`), `status` (default `ACTIVE`). Unique constraint on `(user_id, school_id)` prevents duplicate links.
5. **Cross-org prevention**: Composite foreign keys on `user_schools` referencing `users(organization_id, id)` and `schools(organization_id, id)` ensure a user can only be linked to schools in their own organization.
6. **Onboarding ownership**: During onboarding, the user who creates a school is added to `user_schools` with `role = 'owner'` (see ADR-0001, onboarding flow).
7. **Active school selection**: A session stores `active_school_id` (nullable). Auto-selected if exactly one school; NULL if multiple, requiring client-side selection.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Single shared schema, no org isolation | Simpler queries | No data isolation; unsuitable for B2B | Rejected — B2B requires hard tenant boundaries |
| Separate database per organization | Strongest isolation | Operational complexity, cost, migration burden | Overkill for current scale; schema-level isolation sufficient |
| User-to-school via direct `school_id` on users | Simpler, fewer joins | Users can only belong to one school; no role field | Rejected — users need multi-school access with roles |

## Consequences

### Positive

- Hard data isolation at the database level via composite FKs and scoped queries.
- Users can belong to multiple schools within an organization with different roles.
- Organization-scoped queries (`WHERE organization_id = $1`) are simple and indexable.
- Onboarding naturally creates the org → school → user ownership chain.

### Negative

- Every query must include `organization_id` in WHERE/JOIN conditions to avoid cross-org leaks; easy to forget in new code.
- Composite foreign keys add schema complexity.
- Users are globally unique by email but org-scoped; changing a user's organization requires careful data migration.

### Neutral / Risks

- Query review/audit needed to ensure all multi-tenant queries filter by `organization_id`.
- Role-based authorization (owner vs admin vs teacher) is not yet enforced in code — the `role` field is stored but not interpreted (future ADR/spec).
- `active_school_id = NULL` for multi-school users means the client must track selection; no default-persistence mechanism defined yet.

## Links

- Auth ADR: `docs/adr/0001-authentication-architecture.md`
- Session ADR: `docs/adr/0003-session-management.md`
- Migration: `backend/migrations/00000000000002_core_domain.sql`
- Store queries: `backend/internal/auth/store/queries/auth.sql` (`GetSchoolsByUser`)
- Glossary: `docs/specs/glossary.md`
