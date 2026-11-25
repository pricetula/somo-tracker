# ADR-0014: Teacher Qualification ↔ Subject Assignment

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0014 |
| title | Teacher Qualification ↔ Subject Assignment |
| status | proposed |
| date | 2026-11-25 |
| author | backend, product |
| deciders | backend, product |
| relates_to | SPEC-002, ADR-0007, ADR-0008 |

## Status

proposed

## Context

Education regulators across Kenya (TSC), Uganda, Tanzania, and Rwanda require teachers to be registered and assigned only to subjects they are qualified to teach. The current `teachers` table (SPEC-002) stores `employee_number` and `full_name` but has no link to subjects the teacher is authorized to teach.

For MVP, we record qualifications as a data field without enforcing assignment restrictions in code. Enforcement can be added later.

## Decision

Adopt a **lightweight teacher-subject qualification model**:

1. **`teacher_qualifications` table**:
   - `id` UUID PK
   - `teacher_id` UUID → `teachers.id`
   - `subject_code` TEXT (e.g. `MATH`, `ENG`, `SCI`) — references subject catalog
   - `education_level_id` UUID → `education_levels.id` (level the teacher is qualified to teach this subject at)
   - `qualification` TEXT (e.g. `B.Ed Mathematics`, `Diploma in Primary Education`)
   - `registration_number` TEXT (teacher council registration, where applicable)
   - `created_at` TIMESTAMPTZ

2. **No code enforcement in MVP**: The `role` field on `user_schools` (ADR-0002) and qualification data are stored but not interpreted by authorization logic. A future ADR will enforce that teachers can only create assessments for subjects they are qualified to teach.

3. **Multi-tenancy**: Qualifications are scoped to the teacher's school via the `teachers.school_id` foreign key.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Full RBAC enforcement now | Prevents unqualified subject assignment | High complexity, requires role ADR first | Rejected — defer to future |
| Qualification as free text on teacher | Simplest | No queryable subject linkage | Rejected |
| Teacher-subject join table (chosen) | Queryable, extensible, school-scoped | Slightly more tables | Accepted — lightweight |

## Consequences

### Positive

- Teacher qualifications are recorded and queryable.
- Foundation for future subject-assignment enforcement.
- Minimal schema addition, no code behavior change in MVP.

### Negative

- No enforcement that a teacher only teaches subjects they are qualified for (by design for MVP).
- Data quality depends on accurate entry by school admins.

### Neutral / Risks

- Qualification data may go stale (teacher gains new qualification); no expiry tracking in MVP.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Teachers: `docs/specs/002-education-system-configuration.md` (`teachers` table)
- Subjects: `docs/specs/002-education-system-configuration.md` (`school_subjects`, `education_system_subjects`)
- RBAC gap: `docs/adr/0002-multi-tenancy-model.md` (role field not yet interpreted)
