# Assessment Management Strategy

**Status:** Draft
**Owner:** Product / Platform
**Created:** 2026-10-15
**Related Gaps:** docs/gaps/GAP-assessment-and-report-card-management.md

## Problem Statement
Somotracker has tenancy, schools, academic years/terms, class rooms, students, subjects and events. Assessment capture, grading conversion, report card generation and publishing are missing. Schools cannot record academic performance or issue report cards.

## Scope Principles
* Document-led, data capture focused. No live exam room management now.
* Multi-tenant RLS enforced. All data scoped to school/tenant.
* Error contract enforced. Audit all mark changes.
* No dumb back buttons. Pages are content panes.

## Locked Decisions
* Domain named **Assessments**. Types: Formative / Summative. Sub-types: CAT, Midterm, End Term, Mock etc.
* Marks entry is **percentage-first**. Teacher enters percentage or raw marks. Backend derives grade/CBE descriptor.
* **Conversion granularity = School-wide.** One grade scale per school, snapshotted at assessment creation. Never re-grade.
* **Assessment weighting** configurable per school, snapshotted with assessment.
* **Topic/Sub-topic optional.** Assessments created at Subject level. Optional linkage to Topic/Sub-topic for CBC expansion.
* **Publishing is batchable.** Backend accepts list of student IDs. Frontend decides scope: one student, class, stream, selection.
* **PDF server side** generated.
* **Corrections** after approval/publish require reason, audit trail, re-approval before re-publish.
* No room/invigilation/seating management in MVP.

## User Journeys

### Teacher marks entry
Teacher selects Class → Subject → Term → Assessment type. System loads cohort from `student_class_enrollments`. Teacher enters % per student. System shows derived grade from frozen scale. Save as Draft.

### Review & Approval
HOD reviews entries. Principal approves assessment for class. Approval locks assessment.

### Report Card Generation
At term close, system aggregates approved assessments using snapshotted weighting + attendance from `timetable_attendance`. Report card generated in DRAFT.

### Publish
Admin/Principal selects scope of students and publishes. System locks records, generates PDFs, notifies linked guardians via `guardian_student_links`.

## Data Model Sketch

**assessments**
school_id, academic_year_id, academic_term_id, class_room_id, subject_id, assessment_type, sub_type, title, max_marks/is_percentage, grade_scale_version_id, weighting_config_version_id, status, created_by

**assessment_results**
assessment_id, student_id, percentage, grade, cbe_descriptor, remarks, entered_by, entered_at, last_modified_at, audit trail

**school_grade_scales**
school_id, version, percentage ranges → grade/CBE descriptor

**school_weighting_configs**
school_id, version, assessment_type weights

**report_cards**
school_id, student_id, academic_year_id, academic_term_id, status, generated_at

**report_card_publications**
report_card_id, published_by, published_at, student_ids[]

All tables tenant scoped with RLS.

## Integration Points
* `student_class_enrollments` for cohort validation
* `subjects`, `topics`, `sub_topics` for curriculum linkage
* `school_events` optional for assessment window calendar
* `bulk_jobs` for bulk marks import
* `guardian_student_links` for publish notifications
* Existing error contract and RLS

## Non-Functional Requirements
* Audit immutability for grades after snapshot
* Batch operations must be idempotent
* PDF generation async, stored and retrievable
* Performance for class-level aggregation

## Open Questions Resolved
* Conversion granularity → School-wide
* Re-grading → Forbidden, scale frozen
* Topic linkage → Optional
* Corrections → Reason + re-approval only

## Next Steps
* Create feature specs:
  * FEATURE-assessment-creation-and-marks-entry
  * FEATURE-grade-scale-and-conversion
  * FEATURE-report-card-generation
  * FEATURE-batch-publishing-and-pdf
* Update `docs/reference/backend_schema.md` after spec approval
