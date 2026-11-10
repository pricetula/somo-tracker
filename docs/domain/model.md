---
id: DOMAIN-MODEL
title: Domain Model
status: draft
depends_on: []
touches: []
---

# Domain Model

## Core entities

- **Tenant** -> owns many **School**
- **School** -> has many **User** via **Membership**; owns AcademicYears, ClassLevels, Streams
- **User** -> has one identity per tenant; can have multiple Memberships each with a role scoped to a School
- **Membership** -> User + School + Role. Role is one of Platform Operator, Tenant Owner, Headteacher, School Administrator/Registrar, Teacher, Finance Officer, Guardian
- **AcademicYear** -> contains three **Term**
- **ClassLevel** -> PP1, PP2, Grade 1-9
- **Stream** -> pure color/group name scoped to a School (no ClassLevel); e.g., "Blue", "Red", "Yellow"; table `streams`
- **Classroom** -> logical classroom = School + ClassLevel + Stream; e.g., "Grade 1 Blue"; table `classrooms`; has RLS
- **Student** -> has lifecycle states applicant, active, transferred out, withdrawn, graduated; linked to Guardian
- **Enrollment** -> Student + Classroom + Term. Invariant: one active enrollment per student per term
- **Attendance** -> per-period mark per enrollment tied to `timetable_assignment_id` + `date`: Present, Late, Absent, Excused. Edits after day close require reason and audit
- **Subject** -> tied to EducationSystem + ClassLevel (global reference); contains **Topic**, contains **Subtopic**
- **EducationSystem** -> national education system (Kenya CBC, Tanzania NECTA) with display settings (show_rank); **existing global entity** (table `education_systems`)
- **SchoolAssessmentSettings** -> per-school assessment configuration: working scale `percentage`/`level_points`, default grading scheme family, default computation policy family
- **GradingSchemeTemplate** -> global read-only platform-managed template per education system; marked with `is_recommended_default`
- **GradingSchemeTemplateBand** -> bands belonging to a template, tiling 0-100 without gaps/overlaps
- **GradingScheme** -> tenant-scoped versioned working copy of a scheme, grouped by `family_id` and optionally linked to `template_id`
- **GradingBand** -> row in a tenant scheme with label, min/max percentage, point value, description; powers percentage↔level↔points conversion
- **ClassTeacherAssignment** -> assigns a teacher to a classroom for an academic year
- **SubjectOffering** -> Subject taught in a Classroom for an AcademicYear with assigned Teacher; may override grading scheme family and computation policy family
- **Assessment** -> one marked activity (test, quiz, project, exam) for a SubjectOffering in a Term; category (formative/summative), scoring mode (percentage/rubric), status (draft/open/finalized)
- **AssessmentResult** -> one per Enrollment per Assessment; raw entry (mark or level), converted working value, derived grade, status (entered/absent/excused/not_marked), source ref for future quiz integration
- **ComputationPolicy** -> versioned school-owned policy grouped by `family_id`; category weights, selection (all/last N), aggregation, rounding rule `nearest`/`down`, missing-data rule; default + per-subject override via family reference
- **TermResult** -> computed per Enrollment per SubjectOffering per Term; working value, final grade, exact `computation_policy_id` and `grading_scheme_id` used, status `computed`/`incomplete`; recomputable until report card published
- **TermResultInput** -> explanation rows linking TermResult to each AssessmentResult considered, with inclusion flag and drop reason
- **ReportCard** -> per Enrollment per Term with lifecycle (draft/reviewed/approved/published/revised), remarks, version
- **ReportCardLine** -> snapshot per subject at publication: final grade, working value, subject remark, grading labels as published
- **AuditLog** -> create/update/delete/view events on health, behavior records with actor, timestamp, record ID
- **Invite** -> email invite to staff/guardian with status pending/accepted/expired, expires after 7 days
- **NeedsAttention** -> system-generated tasks e.g., class attendance not marked by 10:00, timetable slot without teacher, invite pending >3 days
- **PendingOnboarding** -> verified email, status, current_step, form_data, stytch_organization_id nullable, stytch_member_id nullable, tenant_id nullable. Created after successful Stytch Discovery authentication, before tenant provisioning. Never stores IST or magic-link token.
- **AppSession** -> token_hash, user_id, tenant_id, created_at, expires_at, revoked_at. Raw opaque token stored only in HttpOnly cookie, hash stored in DB. Session validated by Go on every protected request.

## Relationships & invariants

- Tenant data isolation: no query may return another tenant's records. FR-FND-001
- Membership role scoping: user permissions are per School membership. FR-FND-006
- Tenant Owner minimum one per tenant; cannot remove last owner. FR-FND-007
- Enrollment uniqueness: one active enrollment per student per term enforced DB. FR-ONB-003
- Student lifecycle transitions recorded with date and reason. FR-ONB-004
- Attendance edit after close requires reason and audit. FR-ATT-002
- Health and vaccination data treated as sensitive personal data with audit on every view. FR-WEL-001

See glossary for domain terms: docs/domain/glossary.md
