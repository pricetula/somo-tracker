# ADR-0013: Guardian Management

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0013 |
| title | Guardian Management |
| status | proposed |
| date | 2026-11-25 |
| author | backend, product |
| deciders | backend, product |
| relates_to | SPEC-002, ADR-0017, ADR-0001 |

## Status

proposed

## Context

The current user model covers school staff only (teachers, admins, owners). In East African school administration, guardians (parents or legal caregivers) are central: they pay fees, give consent for activities and data processing, receive report cards, and are emergency contacts. The system needs a guardian entity linked to students.

Guardians are not system users (no login, no OTP) in the MVP. They are contact/consent records associated with one or more students.

## Decision

Adopt a **guardian entity with a many-to-many link to students**:

1. **`guardians` table**:
   - `id` UUID PK
   - `school_id` UUID → `schools.id` (guardians belong to a school, multi-tenant isolation)
   - `full_name` TEXT
   - `phone` TEXT (primary contact number)
   - `email` TEXT (optional)
   - `relationship` TEXT (e.g. `father`, `mother`, `guardian`, `aunt`, `uncle`)
   - `address` TEXT
   - `is_emergency_contact` BOOL (default false)
   - `created_at`, `updated_at` TIMESTAMPTZ

2. **`student_guardians` join table**:
   - `id` UUID PK
   - `student_id` UUID → `students.id`
   - `guardian_id` UUID → `guardians.id`
   - `is_primary` BOOL (default false — one primary guardian per student)
   - `created_at` TIMESTAMPTZ
   - Unique on `(student_id, guardian_id)`

3. **Guardians are not users**: No authentication, no session, no OTP. Guardian contact data is used for report card delivery (see ADR-0016) and consent (see ADR-0017).

4. **Multi-tenancy**: Guardians are scoped to a school via `school_id`; a guardian linked to students at school A is not visible at school B.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Guardian as user (login/OTP) | Guardian portal, self-service | High complexity, not MVP | Rejected — defer to future |
| Single guardian per student | Simpler schema | Does not match reality (two parents, extended family) | Rejected |
| Guardian ↔ student join table (chosen) | Multiple guardians, primary flag, school-scoped | Slightly more complex | Accepted |

## Consequences

### Positive

- Multiple guardians per student (mother, father, extended family).
- Emergency contact flag supports urgent outreach.
- School-scoped guardians maintain multi-tenant isolation.
- Foundation for fee contact, report card delivery, and consent management.

### Negative

- Guardians cannot log in or view data directly in MVP.
- No guardian-to-student validation beyond the join table.

### Neutral / Risks

- Guardian data may duplicate across schools if a guardian has children at multiple schools (accepted — no cross-school guardian deduplication in MVP).

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Students: `docs/specs/002-education-system-configuration.md` (`students` table)
- Data protection: `docs/adr/0017-data-protection-child-privacy.md`
- Report cards: `docs/adr/0016-simple-report-cards.md`
