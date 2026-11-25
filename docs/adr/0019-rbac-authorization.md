# ADR-0019: Role-Based Access Control (RBAC)

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0019 |
| title | Role-Based Access Control (RBAC) |
| status | proposed |
| date | 2026-11-25 |
| author | backend |
| deciders | backend, product |
| relates_to | ADR-0002, ADR-0007 |

## Status

proposed

## Context

ADR-0002 (Multi-tenancy Model) stores a `role` field on `user_schools` (e.g. `owner`, `admin`, `teacher`, `viewer`) but explicitly defers enforcement: "Role-based authorization is not yet enforced in code — the `role` field is stored but not interpreted (future ADR/spec)."

Without RBAC enforcement, any authenticated user in a school can perform any action within that school's scope. The `role` column is effectively decorative until authorization logic is built.

## Decision

Adopt a **school-scoped RBAC model** with deferred enforcement:

1. **Roles**: `owner`, `admin`, `teacher`, `viewer`. Stored on `user_schools.role`. Role semantics:
   - `owner`: Full control over the school (created during onboarding).
   - `admin`: School management (users, subjects, grading, terms) but cannot transfer ownership.
   - `teacher`: Create assessments, enter results, mark attendance for assigned streams.
   - `viewer`: Read-only access to school data.

2. **Enforcement deferred**: In MVP, the `role` field is stored and displayed but **not enforced** by middleware. All authenticated users with a valid session and active school assignment can perform all actions within their active school. Enforcement will be added in a future release via middleware that checks `user_schools.role` against required role per endpoint.

3. **Scope**: RBAC is scoped to a school within an organization. A user may have different roles in different schools (via `user_schools`).

4. **Active school**: Authorization checks use the session's `active_school_id`. A user with no active school (multi-school user who hasn't selected one) cannot perform school-scoped actions.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Enforce RBAC now | Prevents unauthorized actions immediately | Requires middleware, endpoint-role mapping, UI gating — delays MVP | Rejected — defer to post-MVP |
| Organization-level roles only | Simpler (one role per user) | Cannot express different roles per school within an org | Rejected — multi-school users need per-school roles |
| Store + defer enforcement (chosen) | Schema ready, no behavior change in MVP | Roles are decorative until enforcement built | Accepted — unblocks MVP |

## Consequences

### Positive

- Schema is ready for RBAC; no migration needed when enforcement is added.
- Role semantics are documented, so future enforcement matches stored values.
- MVP ships without authorization complexity.

### Negative

- No actual access control in MVP — any authenticated school user can do anything within their school.
- `role` field may be set incorrectly with no runtime validation.

### Neutral / Risks

- When enforcement is added, existing users with `role` values that don't match their actual responsibilities will need re-classification.
- Endpoint-to-role mapping must be designed carefully to avoid over-restricting teachers.

## Links

- Multi-tenancy: `docs/adr/0002-multi-tenancy-model.md`
- User school: `docs/specs/glossary.md` (`user_school` entity)
- Education config: `docs/adr/0007-table-driven-education-configuration.md`
