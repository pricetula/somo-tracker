---
id: FEATURE-ASSESSMENTS-001
title: Assessment, Results Computation, and Report Cards
status: implementation-ready
version: 1.0.0
scope: v1
depends_on:
  - existing tenant, school, user, role, education-system, academic-year, term, classroom, subject, enrollment, guardian-link, and authentication domains
touches:
  - backend
  - database
  - frontend
  - reporting
---

# Assessment, Results Computation, and Report Cards

## Authoritative Technical Specification — v1

## 0. Document authority and implementation rules

This document is the authoritative v1 specification for the assessment module. If an older feature note, diagram, example, or implementation comment conflicts with this document, this document takes precedence.

The implementation must not infer business rules from UI behavior or from sample data. All rules that affect persisted results must be enforced in the service layer and, where practical, by database constraints.

**No migrations are authorized by this document alone.** This is a design specification. Before implementation, inspect the existing schema and map each referenced core entity and role to its actual table, key, enum, and tenant model. Do not recreate existing core entities.

### 0.1 Product goal

Provide configurable assessment and reporting for schools using the supported Kenyan CBE/CBC and Tanzanian NECTA education-system configurations. Teachers enter marks or rubric levels; the system normalizes them onto one school working scale, computes explainable term results using versioned policies, and publishes versioned report cards to authorized guardians.

### 0.2 V1 scope

Included:

- Global grading templates and tenant-owned, versioned grading schemes.
- School assessment settings and one working scale per school.
- Subject offerings and class-teacher assignments, if not already present.
- Formative and summative assessments.
- Percentage and rubric result entry.
- Absent, excused, and not-marked result states.
- Versioned school computation policies and subject overrides.
- Deterministic term-result computation with full input explanations.
- Term-result freezing through report-card publication and controlled revision.
- Report-card review, approval, publication, history, and guardian access.
- Audit events for sensitive changes.
- English and Swahili presentation labels.
- Offline-tolerant bulk result entry through idempotent client request identifiers.

Excluded:

- Attendance, timetable, and school administration unrelated to assessment.
- Official KNEC/NECTA data exchange, uploads, or certification.
- Class ranking/position calculation.
- Strand/sub-strand assessment breakdowns.
- Online quizzes, question banks, and automated quiz marking.
- Free-form formulas or arbitrary user-defined expressions.
- Cross-school analytics.
- Parent result entry and student self-assessment.
- Automatic guardian notifications on report-card revision in v1.

### 0.3 Core design invariants

1. A school belongs to exactly one education system at a time.
2. A school has exactly one assessment working scale in v1: `percentage` or `level_points`.
3. Every assessment, result, term result, policy, scheme, and report card is scoped to exactly one school and tenant.
4. A result can only reference an enrollment belonging to the same school as its assessment's subject offering.
5. Every assessment freezes an exact grading-scheme version at creation.
6. Every term-result computation freezes exact policy and grading-scheme versions.
7. Historical computed results and published report cards are immutable snapshots.
8. A new revision creates a new version; it never overwrites a published version.
9. A category with no usable selected results is handled only by the configured missing-data rule.
10. A result state other than `entered` has no numeric value and is never silently treated as zero.
11. Every calculation can be reproduced from its stored inputs, policy version, scheme version, and algorithm version.
12. Authorization is checked server-side for every read and write; RLS is defense in depth, not a replacement for service authorization.

---

# 1. Terminology

| Term                       | Meaning                                                                           |
| -------------------------- | --------------------------------------------------------------------------------- |
| Education system           | Global configuration such as Kenya CBE/CBC or Tanzania NECTA.                     |
| School assessment settings | School-specific assessment configuration, including the working scale.            |
| Grading scheme family      | Stable identity grouping successive versions of one grading scheme.               |
| Grading scheme version     | Immutable scheme definition used by an assessment or computation.                 |
| Grading band               | A named result category with percentage boundaries and a working-scale value.     |
| Subject offering           | A subject taught to a class/cohort in an academic year at a school.               |
| Assessment                 | A marked activity in a subject offering and term.                                 |
| Assessment result          | One learner's recorded state/value for one assessment.                            |
| Working value              | Numeric value normalized to the school's working scale and used in computation.   |
| Computation policy family  | Stable identity grouping successive policy versions.                              |
| Term result                | Computed outcome for one enrollment, subject offering, and term.                  |
| Term-result input          | An explanation row showing how an assessment result was treated in a computation. |
| Report card                | A versioned publication for one enrollment and term.                              |
| Frozen                     | A result snapshot that cannot be recomputed in place.                             |

---

# 2. Existing-domain integration

The following are assumed to exist in the core system. Confirm actual names and keys before implementation:

- `tenants`
- `schools`
- `users`
- role/membership assignments, including `TEACHER`, `ADMIN`, and `HEADTEACHER`
- `education_systems`
- `countries`
- `academic_years`
- `terms`
- `classrooms`
- `subjects`
- `enrollments`
- guardian/student links, including or extended with `can_view_results`

The assessment module must reference these entities rather than duplicating them.

## 2.1 Required relationship checks

A foreign key to a parent row is not sufficient to prove that two related rows belong to the same tenant or school. Use composite keys/foreign keys or equivalent database constraints for all same-school relationships.

At minimum, enforce:

- `schools(tenant_id, id)` as a candidate key if not already available.
- `subject_offerings(tenant_id, school_id, id)` as a candidate key.
- `enrollments(tenant_id, school_id, id)` as a candidate key.
- Assessment references its subject offering using `(tenant_id, school_id, subject_offering_id)`.
- Assessment references a term that belongs to the same school/academic year.
- Assessment result references an assessment and enrollment with matching tenant and school.
- Term result references an enrollment and subject offering with matching tenant and school.
- Report card references an enrollment with matching tenant and school.
- Report-card lines reference subject offerings from the same school as the report card.

If the existing core schema cannot support these composite constraints without a broad refactor, add equivalent transactional validation in services and record the database limitation explicitly. Do not claim database-enforced integrity when it is only application-enforced.

## 2.2 School assessment settings

Do not put `working_scale` on the global `education_systems` row: different schools using the same national system may need different operational settings, and the working scale is a school-level v1 invariant.

Create `school_assessment_settings`:

| Field                                | Type        | Required | Rules                                                 |
| ------------------------------------ | ----------- | -------: | ----------------------------------------------------- |
| id                                   | UUID        |      yes | PK                                                    |
| tenant_id                            | UUID        |      yes | FK tenant                                             |
| school_id                            | UUID        |      yes | FK school; unique per school                          |
| working_scale                        | enum        |      yes | `percentage` or `level_points`                        |
| default_grading_scheme_family_id     | UUID        |      yes | Must belong to same school and education system       |
| default_computation_policy_family_id | UUID        |      yes | Must belong to same school                            |
| show_rank                            | boolean     |      yes | Display preference only; no ranking calculation in v1 |
| created_at, updated_at               | timestamptz |      yes | Standard timestamps                                   |
| created_by, updated_by               | UUID        |      yes | Actor references                                      |

Unique: `(school_id)`. Add composite uniqueness for `(tenant_id, school_id)`.

A school may change its working scale only before it has any assessment, result, term-result, or report-card data. After first use, changing the working scale is blocked in v1. A future migration/conversion workflow would be required.

---

# 3. Grading model

## 3.1 Global templates

Templates are platform-managed read-only starting points. Schools may copy a template into a tenant-owned scheme family and customize future versions.

### `grading_scheme_templates`

| Field                  | Type         | Required | Rules                                                                |
| ---------------------- | ------------ | -------: | -------------------------------------------------------------------- |
| id                     | UUID         |      yes | PK                                                                   |
| education_system_id    | UUID         |      yes | FK global education system                                           |
| template_key           | varchar(64)  |      yes | Stable machine key                                                   |
| name                   | varchar(128) |      yes | Display name                                                         |
| description            | text         |       no |                                                                      |
| scheme_kind            | enum         |      yes | `percentage_bands` or `level_bands`                                  |
| is_recommended_default | boolean      |      yes | At most one recommended default per education system and scheme kind |
| seed_version           | integer      |      yes | Platform template version                                            |
| created_at             | timestamptz  |      yes |                                                                      |

Unique: `(education_system_id, template_key, seed_version)`.

### `grading_scheme_template_bands`

| Field          | Type         | Required | Rules                                                                           |
| -------------- | ------------ | -------: | ------------------------------------------------------------------------------- |
| id             | UUID         |      yes | PK                                                                              |
| template_id    | UUID         |      yes | FK template                                                                     |
| label          | varchar(16)  |      yes | Stable grade label                                                              |
| min_percentage | numeric(6,3) |      yes | Inclusive lower boundary                                                        |
| max_percentage | numeric(6,3) |      yes | Exclusive upper boundary, except final upper boundary 100 inclusive             |
| point_value    | numeric(6,3) |      yes | Value on a level-points scale; may also store scheme-defined points for display |
| description_en | text         |       no |                                                                                 |
| description_sw | text         |       no |                                                                                 |
| sequence_index | integer      |      yes | Stable display order                                                            |

Unique: `(template_id, label)` and `(template_id, sequence_index)`.

## 3.2 Boundary convention

All percentage bands use half-open intervals `[min_percentage, max_percentage)`, except the final band, which includes 100. This prevents decimal gaps and overlaps.

Example: 0–40, 40–60, 60–80, 80–100 is represented as:

- `[0, 40)`
- `[40, 60)`
- `[60, 80)`
- `[80, 100]`

A validator must ensure:

- First band starts at 0.
- Last band ends at 100.
- Every band has `min < max`.
- Bands are contiguous with no gaps or overlaps.
- There is exactly one band for every possible percentage value.
- `sequence_index` is unique and ordered consistently with the percentage boundaries.

Sublevels such as EE1/EE2 are not overlapping extra bands. They are ordinary distinct bands that partition the relevant percentage interval. For example, EE2 `[80, 90)` and EE1 `[90, 100]`. Do not seed both EE and EE1/EE2 as overlapping bands in the same scheme. If a parent label is needed for display, represent it as a separate `parent_label`/grouping attribute, not as an overlapping grading band.

## 3.3 Tenant-owned grading schemes

Use immutable version rows.

### `grading_scheme_families`

| Field               | Type         | Required | Rules                                |
| ------------------- | ------------ | -------: | ------------------------------------ |
| id                  | UUID         |      yes | PK; stable family identifier         |
| tenant_id           | UUID         |      yes | FK tenant                            |
| school_id           | UUID         |      yes | FK school                            |
| education_system_id | UUID         |      yes | Must equal school's education system |
| name                | varchar(128) |      yes |                                      |
| scheme_kind         | enum         |      yes | `percentage_bands` or `level_bands`  |
| source_template_id  | UUID         |       no | FK template                          |
| created_by          | UUID         |      yes |                                      |
| created_at          | timestamptz  |      yes |                                      |

Unique: `(school_id, name)` unless product requirements explicitly permit duplicate names.

### `grading_schemes`

| Field          | Type        | Required | Rules                                      |
| -------------- | ----------- | -------: | ------------------------------------------ |
| id             | UUID        |      yes | PK; exact version ID                       |
| tenant_id      | UUID        |      yes |                                            |
| school_id      | UUID        |      yes |                                            |
| family_id      | UUID        |      yes | FK family, same school                     |
| version        | integer     |      yes | Starts at 1, increments by 1               |
| status         | enum        |      yes | `draft`, `active`, `superseded`, `retired` |
| change_summary | text        |       no | Required for versions after v1             |
| created_by     | UUID        |      yes |                                            |
| created_at     | timestamptz |      yes |                                            |

Unique: `(family_id, version)`. Only one active version per family. A version becomes immutable when activated or referenced by an assessment. A correction creates a new version; it never edits an activated version.

### `grading_bands`

| Field             | Type         | Required | Rules                                                                                 |
| ----------------- | ------------ | -------: | ------------------------------------------------------------------------------------- |
| id                | UUID         |      yes | PK                                                                                    |
| tenant_id         | UUID         |      yes |                                                                                       |
| school_id         | UUID         |      yes |                                                                                       |
| grading_scheme_id | UUID         |      yes | Exact scheme version                                                                  |
| label             | varchar(16)  |      yes |                                                                                       |
| parent_label      | varchar(16)  |       no | Grouping label for sublevels, not a second band                                       |
| min_percentage    | numeric(6,3) |      yes | Inclusive                                                                             |
| max_percentage    | numeric(6,3) |      yes | Exclusive except final 100                                                            |
| point_value       | numeric(6,3) |      yes | Required for `level_points`; may store configured grade points for percentage schemes |
| description_en    | text         |       no |                                                                                       |
| description_sw    | text         |       no |                                                                                       |
| sequence_index    | integer      |      yes |                                                                                       |

Unique: `(grading_scheme_id, label)` and `(grading_scheme_id, sequence_index)`.

### 3.4 Scheme validation and activation

A scheme can be saved as draft while incomplete. It cannot be activated until:

- bands fully and continuously cover 0–100;
- all labels and sequence indexes are unique;
- the scheme kind is compatible with the school's working scale;
- level-based schemes have unique `point_value` values for each distinct level label used for final mapping;
- all bands have valid localized labels/descriptions where localization is required by the school;
- no referenced assessment would be changed by activation.

For v1, `level_bands` is intended for a school using `level_points`; `percentage_bands` is intended for a school using `percentage`. Do not mix scheme kinds within a school.

### 3.5 Grading scheme resolution

Resolve the exact scheme version in this order:

1. `Assessment.grading_scheme_id` (always populated and immutable after creation).
2. If creating a new assessment, use `SubjectOffering.grading_scheme_family_id` active version if configured.
3. Otherwise use `SchoolAssessmentSettings.default_grading_scheme_family_id` active version.

If no active scheme resolves, assessment creation fails with a configuration error. Never silently select an arbitrary template or the most recently created scheme.

When an assessment is created, persist the exact `grading_scheme_id` version. Assessment results copy that exact scheme version reference. The integer version alone is not sufficient.

---

# 4. Teaching assignments and subject offerings

If equivalent core entities already exist, map to them rather than creating duplicates.

## 4.1 `class_teacher_assignments`

One class teacher per class and academic year in a school.

Fields: `id`, `tenant_id`, `school_id`, `classroom_id`, `academic_year_id`, `teacher_user_id`, `created_at`, `updated_at`, `deleted_at`.

Unique active assignment: `(school_id, classroom_id, academic_year_id)`.

## 4.2 `subject_offerings`

One subject taught to one class for one academic year at one school.

Fields:

- `id`
- `tenant_id`, `school_id`
- `subject_id`
- `classroom_id`
- `academic_year_id`
- `teacher_user_id`
- optional `grading_scheme_family_id` (subject-specific scheme override)
- optional `computation_policy_family_id` (subject-specific policy override)
- `created_at`, `updated_at`, `deleted_at`

Unique active offering: `(school_id, subject_id, classroom_id, academic_year_id)`.

The subject teacher assignment is authoritative for teacher access to assessments/results. If multiple teachers per offering become necessary later, replace the single teacher column with an assignment table; do not simulate multiple teachers by duplicate subject offerings.

---

# 5. Assessments

## 5.1 `assessments`

| Field                  | Type         | Required | Rules                                              |
| ---------------------- | ------------ | -------: | -------------------------------------------------- |
| id                     | UUID         |      yes | PK                                                 |
| tenant_id, school_id   | UUID         |      yes | Same school as offering                            |
| subject_offering_id    | UUID         |      yes | FK offering                                        |
| term_id                | UUID         |      yes | Must belong to offering's academic year and school |
| name                   | varchar(128) |      yes |                                                    |
| instructions           | text         |       no |                                                    |
| category               | enum         |      yes | `formative` or `summative`                         |
| scoring_mode           | enum         |      yes | `percentage` or `rubric`                           |
| max_score              | numeric(8,3) |      yes | > 0; default 100                                   |
| assessment_date        | date         |      yes | Used for ordering                                  |
| sequence_index         | integer      |      yes | Used as deterministic same-date tiebreaker         |
| grading_scheme_id      | UUID         |      yes | Exact immutable scheme version                     |
| status                 | enum         |      yes | `draft`, `open`, `finalized`, `cancelled`          |
| finalized_by           | UUID         |       no |                                                    |
| finalized_at           | timestamptz  |       no |                                                    |
| created_by             | UUID         |      yes |                                                    |
| created_at, updated_at | timestamptz  |      yes |                                                    |
| deleted_at             | timestamptz  |       no | Only soft delete while draft and no results        |

Unique ordering key: `(subject_offering_id, term_id, assessment_date, sequence_index)`. `sequence_index` must be unique within an offering/term/date to make last-N selection deterministic.

`cancelled` is an administrative terminal state for an assessment that should not contribute to results. It is not a substitute for absent/excused learner states.

### Scoring rules

- For `percentage`, `max_score` is the maximum raw mark. A learner's raw mark must be between 0 and `max_score`.
- For `rubric`, the teacher selects a band label from the assessment's frozen grading scheme. `max_score` is ignored and must be 100 or null by API convention; database stores 100 for consistency.
- A rubric label must belong to the exact scheme version on the assessment.
- Assessment date must fall within its term's date range unless an authorized admin explicitly enables an out-of-term assessment exception. V1 default: reject out-of-term dates.

### Lifecycle

`draft → open → finalized`

- Draft: editable metadata; no results can be finalized; teacher can delete if no results exist.
- Open: teacher enters/edits learner results; metadata affecting computation is locked except by admin with reason.
- Finalized: result set is locked; contributes to term computation. Finalization is blocked unless every enrolled learner has a resolved status `entered`, `absent`, or `excused` for this assessment.
- Reopen: admin or head teacher only, with a reason and audit event. Reopening changes status to `open`; it does not delete prior result history.
- Cancel: admin or head teacher only, with reason and audit. A finalized assessment can be cancelled only through a controlled correction action that recomputes affected unfrozen term results or starts a report-card revision.

Teacher finalization is allowed only for the teacher assigned to the subject offering. Admin and head teacher may finalize only where explicitly granted by school policy; v1 default is teacher-only to preserve the original decision.

---

# 6. Assessment results

## 6.1 `assessment_results`

One current result row per enrollment per assessment. Preserve previous values in audit history, not as duplicate active rows.

Fields:

- `id UUID PK`
- `tenant_id UUID NOT NULL`
- `school_id UUID NOT NULL`
- `assessment_id UUID NOT NULL`
- `enrollment_id UUID NOT NULL`
- `status enum NOT NULL`: `not_marked`, `entered`, `absent`, `excused`
- `raw_score numeric(8,3) NULL`
- `raw_band_label varchar(16) NULL`
- `normalized_percentage numeric(7,3) NULL`
- `working_value numeric(8,3) NULL`
- `derived_grade_label varchar(16) NULL`
- `grading_scheme_id UUID NOT NULL` exact scheme version
- `entered_by UUID NULL`
- `entered_at timestamptz NULL`
- `last_changed_by UUID NOT NULL`
- `created_at`, `updated_at`
- `client_request_id UUID NULL` for offline idempotency, scoped to actor and operation
- optional future source fields excluded from active v1 behavior

Unique: `(assessment_id, enrollment_id)`.
Composite FKs enforce same tenant/school for assessment and enrollment.

### State/value constraints

- `not_marked`, `absent`, `excused`: `raw_score`, `raw_band_label`, `normalized_percentage`, `working_value`, and `derived_grade_label` must all be NULL.
- `entered` with percentage scoring: `raw_score` is required; `raw_band_label` is NULL.
- `entered` with rubric scoring: `raw_band_label` is required; `raw_score` is NULL.
- For every `entered` row, `normalized_percentage` and `working_value` are required after successful validation/conversion.
- `entered_by` and `entered_at` are required for `entered`, `absent`, and `excused`; `not_marked` may have them null.
- An `absent` or `excused` result is not zero. Zero is an entered score of 0.

### Conversion

Always preserve the raw entry and store normalized percentage separately from the working value.

1. Percentage assessment:
   `normalized_percentage = raw_score / max_score * 100`.
2. Rubric assessment:
   `normalized_percentage = midpoint of the selected band's percentage interval`.
   For the final interval ending at 100, use the arithmetic midpoint of its min and max.
3. If school working scale is `percentage`:
   `working_value = normalized_percentage`.
4. If school working scale is `level_points`:
   `working_value = selected grading band's point_value`, where the band is found by normalized percentage using the frozen scheme's half-open boundary rules.

Do not use `point_value * 25`. Do not interpolate level points in v1. The grading band is the single source of truth for mapping a percentage to a level point.

`derived_grade_label` is the band label corresponding to normalized percentage at entry time. It is a convenience snapshot; the exact scheme version remains the authoritative definition.

### Enrollment eligibility

A result may be created only for an enrollment:

- belonging to the same school and tenant as the assessment;
- active for at least part of the assessment date;
- belonging to the classroom/cohort of the subject offering at that date.

For a transfer, use enrollment start/end dates. Assessments before the old enrollment's end date belong to the old enrollment; assessments on or after the new enrollment's start date belong to the new enrollment. If dates overlap or leave a gap, block entry and require administrative correction.

### Bulk entry

Bulk endpoint must:

- validate every row independently;
- persist valid rows and return row-level errors for invalid rows;
- use a transaction per row or a bounded batch with explicit partial-success semantics;
- be idempotent using `(tenant_id, actor_user_id, client_request_id)` or a dedicated idempotency table;
- reject reuse of the same idempotency key with a different payload;
- never overwrite a newer server update silently. Require `expected_updated_at` or row version for optimistic concurrency;
- return the persisted canonical values, including conversion output and updated timestamp.

---

# 7. Computation policies

## 7.1 `computation_policy_families`

Fields: `id`, `tenant_id`, `school_id`, `name`, `created_by`, `created_at`.

Unique: `(school_id, name)`.

## 7.2 `computation_policies`

Each policy version is immutable once active or referenced.

Fields:

- `id UUID PK`
- `tenant_id`, `school_id`
- `family_id`
- `version integer`
- `status enum`: `draft`, `active`, `superseded`, `retired`
- `formative_weight numeric(5,2)`
- `summative_weight numeric(5,2)`
- `formative_selection enum`: `all`, `last_n`
- `formative_last_n integer NULL`
- `summative_selection enum`: `all`, `last_n`
- `summative_last_n integer NULL`
- `aggregation enum`: only `average` in v1
- `rounding_rule enum`: `nearest`, `down`
- `missing_data_rule enum`: `skip_reweight`, `zero`, `block`
- `change_summary text`
- `created_by`, `created_at`

Unique: `(family_id, version)`.

### Policy validation

- Weights are non-negative and sum exactly to 100.00.
- A category with weight 0 may still be assessed but does not contribute to the combined value.
- If selection is `last_n`, corresponding N must be >= 1.
- If selection is `all`, corresponding N must be NULL.
- `aggregation` must be `average`.
- Exactly one active default policy family is configured per school in school settings.
- A subject offering override resolves to an active family in the same school.

## 7.3 Policy resolution

For a subject offering:

1. If `computation_policy_family_id` is set, use its active version.
2. Otherwise use the active family referenced by `SchoolAssessmentSettings.default_computation_policy_family_id`.

If no policy resolves, computation fails with a configuration error. Do not invent implicit weights or defaults at runtime.

When a computation runs, persist the exact `computation_policy_id` and `computation_policy_version` used.

---

# 8. Term-result computation

## 8.1 Computation contract

Implement as a pure deterministic domain function with no database, clock, network, or authorization calls.

Inputs:

- school working scale;
- finalized assessments for one subject offering and term;
- the learner's corresponding result rows, including non-entered states;
- exact active computation policy version;
- exact grading scheme version;
- algorithm version identifier.

Output:

- computed/incomplete status;
- unrounded and rounded working values;
- final grade label where computable;
- per-category aggregates and effective weights;
- ordered input explanation rows, including excluded rows and reasons;
- immutable references to policy, scheme, and algorithm versions.

The database service loads and validates inputs, invokes the pure function, then persists the result and explanation rows in one transaction.

## 8.2 Eligibility and deterministic selection

For a given enrollment, subject offering, and term:

1. Consider only assessments belonging to that offering and term.
2. Consider only assessments with status `finalized`.
3. Exclude cancelled, draft, and open assessments.
4. Preserve every learner result state for each eligible assessment.
5. For each category, order assessments by `assessment_date DESC`, then `sequence_index DESC`, then `assessment.id DESC` as a final stable tie-breaker.
6. Apply `all` or `last_n` to the **scheduled finalized assessments**, not merely to entered marks.
7. After selection, classify selected learner result rows:
   - `entered`: usable numeric input.
   - `absent`: unusable, reason `absent`.
   - `excused`: unusable, reason `excused`.
   - `not_marked` or missing row: unusable, reason `not_marked`.
8. A selected absent assessment consumes a last-N slot. The algorithm must not reach further back and silently replace it with an older assessment.

This definition makes last-N reproducible and ensures a learner's absence is not hidden by selecting an older mark.

## 8.3 Category aggregation

For each category:

- Compute arithmetic mean of usable `working_value` inputs selected for that category.
- Do not round category averages before combining.
- Record the count of selected assessments, usable inputs, and excluded inputs.
- If no usable input exists, the category is missing.

## 8.4 Missing-data rules

Apply after selection and category aggregation.

### `skip_reweight`

- Ignore missing categories.
- Redistribute their weight proportionally across categories with usable results.
- If both categories are missing, status is `incomplete`.
- Formula: `effective_weight_i = configured_weight_i / sum(configured_weights_of_present_categories)`.
- A present category with configured weight 0 contributes 0. If all present categories have zero configured weight, status is `incomplete` due to invalid effective weights.

### `zero`

- A missing category contributes a category average of zero and retains its configured weight.
- At least one category must have at least one finalized assessment in the period; if there are no finalized assessments in either category, status is `incomplete` rather than a fabricated zero result.
- Absence/excused/not-marked are treated as missing, not as individual zero marks.

### `block`

- If any category with configured weight > 0 has no usable selected result, status is `incomplete`.
- A category with configured weight 0 does not block.
- No final grade is produced.

## 8.5 Combined value

For categories that contribute:

`combined_value = Σ(category_average × effective_category_weight / 100)`

Use decimal arithmetic, not binary floating point. Retain at least 6 decimal places internally. Persist the unrounded value at scale 6 or greater, and persist the rounded value separately.

## 8.6 Rounding and grade mapping

Rounding must be defined by working scale and policy. No implicit precision is allowed.

### Percentage working scale

- `nearest`: round to the nearest whole percentage point; exact .5 rounds upward.
- `down`: floor to the nearest whole percentage point.

The final percentage must be clamped only for harmless decimal representation drift; values materially outside 0–100 indicate invalid input/computation and must fail.

### Level-points working scale

- `nearest`: map the unrounded value to the nearest configured `point_value`; exact midpoint ties go to the better/higher point value. Persist the mapped point as `rounded_working_value`.
- `down`: map to the greatest configured point value less than or equal to the unrounded value. If none exists, map to the lowest configured point; if above the maximum, map to the maximum.

After rounding:

- For `percentage`, map rounded value to the single band containing it.
- For `level_points`, map rounded point value to the band with that exact `point_value`.
- If mapping returns zero or multiple bands, scheme configuration is invalid and computation fails loudly.

## 8.7 Term-result statuses

Use exactly:

- `computed`: a final working value and grade were produced.
- `incomplete`: missing-data rule blocked computation or there was no usable basis.

Do not use `draft` or `frozen` as computation statuses. Lifecycle/frozen state is separate from computation outcome.

## 8.8 Worked examples (normative)

### Example 1 — Percentage working scale, weighted categories

Formative marks: 72, 68, 75. Summative marks: 60, 85, 78. Formative weight 40%; summative 60%; formative all; summative last 2.

- Formative average = 71.666667.
- Summative selected assessments are 85 and 78; average = 81.5.
- Combined = 71.666667 × 0.4 + 81.5 × 0.6 = 77.566667.
- `nearest` produces 78.
- If the active scheme has B covering 65 to less than 80, final grade is B.

### Example 2 — Level-points working scale

Selected formative point values: 3, 2, 3. Summative point values: 4, 3. Weights 70% and 30%.

- Formative average = 2.666667.
- Summative average = 3.5.
- Combined = 2.916667.
- `nearest` maps to configured point 3.0 because it is closer than 2.0 or 4.0. Therefore final point = 3.0 and the corresponding grade is ME.
- An exact midpoint such as 3.5 is resolved upward to the better configured point (4.0/EE). `down` would map 2.916667 to 2.0/AE.

This example intentionally makes the tie policy explicit. It must be included in tests.

### Example 3 — Missing summative category

Formative average = 2.5, formative weight 70%; summative has no usable selected result, summative weight 30%.

- `skip_reweight`: formative effective weight becomes 100%; combined = 2.5; level-points nearest mapping follows the scheme point rule.
- `zero`: combined = 2.5 × 0.7 + 0 × 0.3 = 1.75.
- `block`: status `incomplete`, no final grade.

### Example 4 — Last-N and absence

There are three finalized summative assessments ordered newest to oldest: C, B, A. Policy is last 2. Learner is absent for C, has 80 for B, and 60 for A.

- Selected assessments are C and B.
- C is recorded as absent and consumes one selected slot.
- Only B contributes a numeric value.
- A is excluded with reason `outside_last_n`.
- Missing-data behavior is applied to the selected category after selection. The algorithm must not substitute A for C.

---

# 9. Term results and explainability

## 9.1 Versioned term-result records

Use a stable logical key and immutable computation versions.

### `term_result_families`

Fields: `id`, `tenant_id`, `school_id`, `enrollment_id`, `subject_offering_id`, `term_id`, `created_at`.

Unique logical key: `(enrollment_id, subject_offering_id, term_id)`.

### `term_results`

| Field                   | Type          | Required | Rules                                         |
| ----------------------- | ------------- | -------: | --------------------------------------------- |
| id                      | UUID          |      yes | PK; one computation version                   |
| tenant_id, school_id    | UUID          |      yes |                                               |
| family_id               | UUID          |      yes | FK logical family                             |
| version                 | integer       |      yes | Starts at 1; increments on recompute/revision |
| status                  | enum          |      yes | `computed`, `incomplete`                      |
| lifecycle_state         | enum          |      yes | `current`, `superseded`, `frozen`             |
| unrounded_working_value | numeric(12,6) |       no |                                               |
| rounded_working_value   | numeric(12,6) |       no |                                               |
| final_grade_label       | varchar(16)   |       no |                                               |
| computation_policy_id   | UUID          |      yes | Exact immutable policy version                |
| grading_scheme_id       | UUID          |      yes | Exact immutable scheme version                |
| algorithm_version       | varchar(32)   |      yes | e.g. `assessment-v1`                          |
| explanation_json        | jsonb         |      yes | Category aggregates/effective weights         |
| computed_by             | UUID          |      yes |                                               |
| computed_at             | timestamptz   |      yes |                                               |
| created_at              | timestamptz   |      yes |                                               |

Unique: `(family_id, version)`. Only one `current` version per family. A frozen version is immutable.

Recompute before publication creates a new version and marks the prior version `superseded`; it does not overwrite the previous computation. If no previous version exists, create version 1.

## 9.2 `term_result_inputs`

One row per assessment result or missing result slot considered by the computation.

Fields:

- `id`
- `tenant_id`, `school_id`
- `term_result_id`
- nullable `assessment_result_id` (null when no result row exists)
- `assessment_id`
- `category`
- `result_status`
- nullable `working_value`
- `selected_for_last_n boolean`
- `included_in_aggregate boolean`
- `exclusion_reason enum/text`: `not_finalized`, `outside_last_n`, `absent`, `excused`, `not_marked`, `missing_result`, `zero_weight`, `category_missing`
- `created_at`

Do not make `working_value` NOT NULL. Missing, absent, and excluded rows legitimately have no numeric value.

Unique: `(term_result_id, assessment_id)`.

The explanation must include:

- every finalized assessment considered for the period;
- non-finalized assessments only if the UI explicitly requests a diagnostic view (they are not computation inputs);
- which assessments were selected by last-N;
- learner state and exclusion reason;
- category average and effective weight;
- final formula and rounding/mapping outcome.

## 9.3 Recompute triggers

Recompute an unfrozen term-result family when:

- an assessment is finalized;
- a finalized assessment is reopened or cancelled;
- a result in a finalized assessment is corrected by an authorized actor;
- an active computation policy version is changed and an authorized user explicitly requests recomputation;
- an authorized grading scheme revision is adopted for a new assessment set (existing assessments retain their scheme).

Do not silently recompute every historical term result when a policy or scheme changes. Recompute must be explicit, logged, and versioned.

If the family is frozen, normal recompute is rejected with a `REVISION_REQUIRED` error.

---

# 10. Report cards

## 10.1 `report_card_families`

Fields: `id`, `tenant_id`, `school_id`, `enrollment_id`, `term_id`, `created_at`.

Unique: `(enrollment_id, term_id)`.

## 10.2 `report_cards`

One row per report-card version.

Fields:

- `id`
- `tenant_id`, `school_id`
- `family_id`
- `version integer`
- `status enum`: `draft`, `reviewed`, `approved`, `published`, `superseded`
- `class_teacher_remark text`
- `head_teacher_remark text`
- `reviewed_by`, `reviewed_at`
- `approved_by`, `approved_at`
- `published_by`, `published_at`
- `revision_reason text`
- `created_at`, `updated_at`

Unique: `(family_id, version)`. Only one published version is current for guardian display; prior published versions remain queryable as historical versions but are not the default guardian view.

## 10.3 `report_card_lines`

Snapshot of each subject result included in the report card.

Fields:

- `id`
- `tenant_id`, `school_id`
- `report_card_id`
- `subject_offering_id`
- `term_result_id` exact computation version
- `subject_name_snapshot`
- `final_grade_label` nullable
- `unrounded_working_value` nullable
- `rounded_working_value` nullable
- `result_status` (`computed` or `incomplete`)
- `subject_remark`
- `grading_labels_snapshot jsonb`
- `created_at`

Unique: `(report_card_id, subject_offering_id)`.

Nullable grade/value fields are required to represent incomplete results. Snapshot all labels and names required to render the historical report card even if the subject or grading scheme is later renamed.

## 10.4 Report-card lifecycle

`draft → reviewed → approved → published`

- Draft: generated from current term-result versions; class teacher may edit class remark and subject remarks.
- Reviewed: class teacher confirms review. Store actor/time.
- Approved: head teacher approves and may add head-teacher remark. Store actor/time.
- Published: admin publishes. Publication is transactional and freezes the exact term-result versions referenced by report-card lines.
- Revision: admin initiates a new report-card version with a mandatory reason. New version starts as draft; old published version remains immutable and is marked superseded only after the new version is successfully published.

A revision must:

1. create a new report-card version;
2. create new term-result computation versions for the affected enrollment/subjects;
3. create new report-card lines referencing those exact versions;
4. repeat review and approval;
5. publish the new version;
6. mark prior report card as superseded for default display, retaining history.

If revision is abandoned, the prior published version remains the current guardian-visible version.

## 10.5 Publication preconditions

Publication fails unless:

- every expected subject offering for the enrollment/class/term has a report-card line or an explicitly recorded omission reason;
- every line references a current computed or incomplete term-result version;
- all required class-teacher and head-teacher review/approval steps are complete;
- no referenced term result is already frozen by a different published report card;
- enrollment and term belong to the same school and academic year;
- the actor has publish permission.

Whether incomplete subjects may be published is a school-level explicit setting. V1 default: block publication while any required subject is incomplete. If product requirements allow incomplete publication, record it as an explicit school setting and visibly label incomplete lines; never silently omit them.

## 10.6 Guardian visibility

Guardians may read only the latest published report card version for a learner when:

- guardian link is active;
- `can_view_results = true`;
- report card belongs to the same school/tenant context as the linked learner.

Guardians cannot read draft, reviewed, approved, superseded, or audit data. Guardian endpoints must scope by the authenticated guardian's linked learner IDs; never accept an arbitrary learner ID as sufficient authorization.

V1 does not send notifications on publication or revision unless a separate notification feature is explicitly added.

---

# 11. Permissions and authorization

Authorization must consider tenant membership, school membership, role, assignment, record state, and action. A role name alone is insufficient.

| Action                                | Teacher           | Admin                    | Head Teacher             | Guardian                             |
| ------------------------------------- | ----------------- | ------------------------ | ------------------------ | ------------------------------------ |
| View active grading schemes           | Own school        | Own school               | Own school               | No                                   |
| Create/edit/activate scheme versions  | No                | Yes                      | No                       | No                                   |
| View computation policies             | Own school        | Own school               | Own school               | No                                   |
| Create policy versions / set defaults | No                | Yes                      | No                       | No                                   |
| Set subject policy override           | No                | Yes                      | No                       | No                                   |
| Create assessment                     | Assigned offering | Yes                      | Yes                      | No                                   |
| Edit draft/open assessment            | Assigned offering | Yes                      | Yes                      | No                                   |
| Enter results while open              | Assigned offering | Yes                      | Yes                      | No                                   |
| Finalize assessment                   | Assigned offering | Admin only if configured | Admin only if configured | No                                   |
| Reopen finalized assessment           | No                | Yes, reason required     | Yes, reason required     | No                                   |
| Correct result after finalization     | No                | Yes, reason required     | Yes, reason required     | No                                   |
| View term results                     | Assigned offering | Own school               | Own school               | No                                   |
| Trigger recompute before freeze       | Assigned offering | Own school               | Own school               | No                                   |
| Initiate report-card revision         | No                | Yes, reason required     | No                       | No                                   |
| Add class-teacher remark/review       | Assigned class    | Yes                      | Yes                      | No                                   |
| Add head-teacher remark/approve       | No                | Yes                      | Yes                      | No                                   |
| Publish report card                   | No                | Yes                      | No                       | No                                   |
| View published report card            | School scope      | School scope             | School scope             | Linked learner, permission flag only |
| View audit events                     | No by default     | Own school               | Own school               | No                                   |

### 11.1 Finalized result corrections

There is a strict distinction:

- Teachers cannot edit results after assessment finalization.
- Admin or head teacher may correct a finalized result only with a non-empty reason.
- Every correction writes an audit event containing actor, timestamp, reason, before/after values, assessment, learner, and affected term-result family.
- If the term result is unfrozen, a new term-result version is computed.
- If frozen, correction requires report-card revision; it must not mutate the published result.

### 11.2 Separation of duties

A class teacher may review a report card but cannot approve it as head teacher unless the school explicitly enables a documented emergency override. V1 should not enable that override. Admin publishes only after head-teacher approval.

---

# 12. Audit and history

Use the existing audit domain if available. Otherwise add an assessment audit table.

Required audit events:

- grading scheme created, activated, superseded, retired;
- computation policy created, activated, superseded, retired;
- school default or subject override changed;
- assessment created, opened, finalized, reopened, cancelled;
- result entered, changed, set absent, set excused, corrected after finalization;
- term result recomputed, superseded, frozen;
- report card reviewed, approved, published, revision initiated, superseded.

Audit fields:
`id`, `tenant_id`, `school_id`, `actor_user_id`, `action`, `entity_type`, `entity_id`, `occurred_at`, `reason`, `before_json`, `after_json`, `request_id`, `ip_metadata` only if permitted by privacy policy.

Audit records are append-only. Application database role may insert and select but not update/delete. Avoid storing unnecessary personal data in audit payloads. Restrict audit access to admin/head teacher.

---

# 13. API contract

Use the existing API versioning and response conventions. All endpoints require authentication and tenant/school context from the authenticated session, not from untrusted body fields.

Suggested routes:

## Grading schemes

- `GET /api/v1/schools/{schoolId}/grading-scheme-families`
- `POST /api/v1/schools/{schoolId}/grading-scheme-families`
- `GET /api/v1/grading-scheme-families/{familyId}/versions`
- `POST /api/v1/grading-scheme-families/{familyId}/versions`
- `POST /api/v1/grading-schemes/{schemeId}/activate`
- `POST /api/v1/grading-schemes/{schemeId}/retire`

## Computation policies

- `GET /api/v1/schools/{schoolId}/computation-policy-families`
- `POST /api/v1/schools/{schoolId}/computation-policy-families`
- `POST /api/v1/computation-policy-families/{familyId}/versions`
- `POST /api/v1/computation-policies/{policyId}/activate`
- `PATCH /api/v1/schools/{schoolId}/assessment-settings`

## Subject offerings and assessments

- `GET /api/v1/subject-offerings/{offeringId}/assessments`
- `POST /api/v1/subject-offerings/{offeringId}/assessments`
- `GET /api/v1/assessments/{assessmentId}`
- `PATCH /api/v1/assessments/{assessmentId}` (draft/open permitted fields only)
- `POST /api/v1/assessments/{assessmentId}/open`
- `POST /api/v1/assessments/{assessmentId}/finalize`
- `POST /api/v1/assessments/{assessmentId}/reopen`
- `POST /api/v1/assessments/{assessmentId}/cancel`

## Results

- `GET /api/v1/assessments/{assessmentId}/results`
- `PUT /api/v1/assessments/{assessmentId}/results/{enrollmentId}`
- `POST /api/v1/assessments/{assessmentId}/results:bulk-upsert`

Bulk request includes `client_request_id`, `expected_updated_at` per row, and result state/value. Response returns `saved[]`, `conflicts[]`, and `errors[]`; each row has a stable enrollment ID and machine-readable error code.

## Term results

- `GET /api/v1/enrollments/{enrollmentId}/terms/{termId}/subject-results`
- `GET /api/v1/term-results/{termResultId}`
- `POST /api/v1/term-result-families/{familyId}/recompute`

Recompute request requires an explicit reason when it changes a previously computed result and includes an idempotency key.

## Report cards

- `GET /api/v1/enrollments/{enrollmentId}/terms/{termId}/report-card`
- `POST /api/v1/enrollments/{enrollmentId}/terms/{termId}/report-card`
- `PATCH /api/v1/report-cards/{reportCardId}`
- `POST /api/v1/report-cards/{reportCardId}/review`
- `POST /api/v1/report-cards/{reportCardId}/approve`
- `POST /api/v1/report-cards/{reportCardId}/publish`
- `POST /api/v1/report-cards/{reportCardId}/revise`
- `GET /api/v1/guardians/me/report-cards`

## API error codes

Use stable machine-readable codes, including:

- `FORBIDDEN`
- `NOT_FOUND`
- `VALIDATION_ERROR`
- `ASSESSMENT_NOT_OPEN`
- `ASSESSMENT_ALREADY_FINALIZED`
- `REVISION_REQUIRED`
- `SCHEME_NOT_ACTIVE`
- `SCHEME_SCALE_MISMATCH`
- `POLICY_NOT_ACTIVE`
- `POLICY_INVALID_WEIGHTS`
- `ENROLLMENT_NOT_ELIGIBLE`
- `RESULT_CONFLICT`
- `IDEMPOTENCY_KEY_REUSED`
- `TERM_RESULT_INCOMPLETE`
- `REPORT_CARD_NOT_READY`
- `REPORT_CARD_ALREADY_PUBLISHED`

Do not expose whether a record exists in another tenant/school; return not found or forbidden according to the platform's established anti-enumeration policy.

---

# 14. Database integrity, indexes, and transaction rules

## 14.1 Required constraints

- Check numeric marks are within `[0, max_score]`.
- Check `max_score > 0`.
- Check policy weights sum to 100.00 (use numeric arithmetic; service validation plus DB trigger/constraint where feasible).
- Check last-N values are present and positive only when selection is `last_n`.
- Unique result `(assessment_id, enrollment_id)`.
- Unique scheme version `(family_id, version)`.
- Unique policy version `(family_id, version)`.
- Unique term-result version `(family_id, version)`.
- Unique report-card version `(family_id, version)`.
- Partial unique index for one active scheme version per family.
- Partial unique index for one active policy version per family.
- Partial unique index for one current term-result version per family.
- Unique active school default policy/scheme setting through the one-row settings table.
- Composite same-school foreign keys for all tenant-owned relationships.
- State/value check constraints on assessment results.
- No hard delete of records referenced by published snapshots.

## 14.2 Recommended indexes

- `assessments (school_id, subject_offering_id, term_id, status, assessment_date, sequence_index)`
- `assessment_results (school_id, assessment_id, enrollment_id)` unique
- `assessment_results (school_id, enrollment_id, assessment_id)`
- `subject_offerings (school_id, classroom_id, academic_year_id)`
- `grading_schemes (school_id, family_id, version)`
- `grading_bands (grading_scheme_id, min_percentage, max_percentage)`
- `computation_policies (school_id, family_id, version)`
- `term_result_families (school_id, enrollment_id, subject_offering_id, term_id)` unique
- `term_results (family_id, version)` unique
- `term_result_inputs (term_result_id, assessment_id)` unique
- `report_card_families (school_id, enrollment_id, term_id)` unique
- `report_cards (family_id, version)` unique
- `report_cards (school_id, status, published_at)`
- `report_card_lines (report_card_id, subject_offering_id)` unique
- `audit_logs (school_id, entity_type, entity_id, occurred_at DESC)`

Add partial indexes for active/non-deleted records where appropriate. Avoid duplicate index declarations.

## 14.3 Transaction boundaries

- Creating an assessment: validate school settings, offering, term, scheme, teacher permission; insert assessment with exact scheme version in one transaction.
- Saving a result: validate assessment state, enrollment eligibility, scoring mode, conversion; upsert result and audit event in one transaction.
- Finalizing an assessment: lock assessment row, validate finalization permission, update state, enqueue/recompute affected term-result families idempotently.
- Computing a term result: load a consistent snapshot; execute pure function; insert new term-result version and all explanation rows atomically.
- Publishing a report card: lock report-card family and referenced term-result rows; validate readiness; insert/finalize snapshots; freeze referenced term results; publish atomically.
- Revising: create new version without changing prior published version; all new result snapshots remain isolated until republished.

Use an outbox/job mechanism for expensive recomputations. Jobs must be idempotent and carry exact entity/version identifiers. Do not run a long batch recomputation while holding broad database locks.

---

# 15. RLS and tenant isolation

Enable RLS on every tenant-owned assessment table. Policies must derive tenant and school access from trusted authenticated session context or membership tables.

Required tests:

- A user in tenant A cannot read or mutate tenant B records.
- A user in school A cannot access school B records unless explicitly assigned.
- A teacher cannot access an offering they do not teach.
- A guardian cannot query arbitrary learners or unpublished report cards.
- Composite relationship constraints prevent cross-school references even if a service bug attempts them.
- Background workers use narrowly scoped service roles and explicitly set tenant/school context; do not bypass RLS globally without controls.

RLS is defense in depth. Service-layer permission checks remain mandatory.

---

# 16. Frontend requirements

Feature structure may follow the existing frontend conventions, for example:
`features/assessments/{types,schemas,api,hooks,components,pages}`.

Required screens:

1. School assessment settings.
2. Grading scheme list, editor, preview, version history.
3. Computation policy list, editor, version history.
4. Subject offering assessment list.
5. Assessment create/edit and lifecycle actions.
6. Result-entry grid with status selection, validation, save state, and conflict handling.
7. Term-result preview with category aggregates and expandable calculation explanation.
8. Report-card draft/review/approval/publish workflow.
9. Guardian published report-card view.
10. Admin/head-teacher audit and revision history.

### Result-entry grid

- Distinguish blank/not marked, entered zero, absent, and excused.
- Do not display absent or excused as zero.
- Show the normalized/working value and grade preview after server validation.
- Support partial saves and row-level errors.
- Show stale-write conflicts rather than silently overwriting.
- Offline queue is optional/deferred; if implemented, use client-generated idempotency keys and explicit conflict resolution.
- Never treat a locally queued unsynced mark as finalized.

### Report-card UI

- Clearly show incomplete subjects.
- Render from immutable report-card snapshots, not live grading scheme labels.
- Guardian view shows only latest published version by default; historical versions are available only if product policy permits guardian history access.
- English/Swahili labels are localized. User-entered remarks are not machine-translated in v1.

---

# 17. Localization and data protection

- Store stable machine labels/codes separately from localized display labels.
- Support English and Swahili for grade descriptions, category names, status labels, and system-generated report-card text.
- Do not store localized labels as the sole canonical value.
- Apply the relevant Kenya and Tanzania privacy requirements based on the school's jurisdiction and the platform's legal review.
- Restrict learner results to staff with a legitimate school role and guardians with an active authorized link.
- Minimize personal data in logs, analytics, exports, and error messages.
- Define retention and deletion behavior with the platform's broader student-record policy; published academic records must not be hard-deleted by ordinary soft-delete flows.
- Log access to especially sensitive exports if required by the platform's audit policy.

This document is technical guidance, not legal advice. The product owner/legal reviewer must confirm country-specific retention, consent, access, and data-subject procedures.

---

# 18. Migration and implementation sequence

Do not blindly apply this order without inspecting existing tables and migration conventions. Create or extend the minimum necessary tables.

1. **Schema reconnaissance**
   - Verify core school/tenant/enrollment/term/role/guardian schemas.
   - Verify existing audit table and RLS helpers.
   - Produce a mapping document from this spec to actual schema names.
2. **School assessment settings**
   - Add one settings row per school and validated working scale.
3. **Grading templates and scheme versions**
   - Add template tables, family/version tables, bands, validation, seed templates.
4. **Audit foundation**
   - Reuse existing audit log or add append-only assessment audit support before result writes.
5. **Teaching assignments and subject offerings**
   - Reuse existing equivalents; add missing assignment entities only.
6. **Assessments and results**
   - Add assessment lifecycle, result state constraints, bulk idempotency and permissions.
7. **Computation policies**
   - Add policy family/version, defaults and subject overrides.
8. **Term-result versioning**
   - Add result family/version, explanation rows, pure computation service.
9. **Report cards**
   - Add report-card family/version, immutable lines, publication transaction and guardian RLS.
10. **Frontend and localization**
11. **End-to-end verification and operational rollout**

Audit support must exist before any operation that requires audit. Do not schedule audit-log creation after result-edit functionality.

---

# 19. Acceptance criteria

## Grading

- GS-01: Admin can copy a platform template into a school-owned scheme family.
- GS-02: Scheme bands cover 0–100 with no gaps or overlaps under the half-open boundary convention.
- GS-03: Sublevels are separate non-overlapping bands; parent labels are metadata only.
- GS-04: Activated scheme versions are immutable.
- GS-05: Assessments retain exact scheme version IDs.
- GS-06: A scheme incompatible with school working scale cannot be activated or assigned.

## Assessments

- AS-01: Assigned teacher can create an assessment for their offering and term.
- AS-02: Assessment captures category, scoring mode, max score, date, sequence, and exact scheme version.
- AS-03: Draft/open/finalized transitions follow authorization rules.
- AS-04: Finalized assessment cannot be edited by teacher.
- AS-05: Reopening/cancelling requires authorized actor, reason, and audit event.

## Results

- AR-01: One result exists per enrollment/assessment.
- AR-02: Percentage score is normalized using `raw_score / max_score * 100`.
- AR-03: Rubric entry uses a valid label from the assessment's exact scheme version.
- AR-04: Absent, excused, not-marked, and entered-zero are distinct.
- AR-05: Non-entered states have null numeric values.
- AR-06: Result enrollment and assessment must belong to same school and tenant.
- AR-07: Bulk writes are idempotent and return row-level partial results.
- AR-08: Stale writes return conflict rather than overwriting newer data.

## Policies and computation

- CP-01: Policy weights sum to 100.
- CP-02: Last-N settings are valid and deterministic.
- CP-03: Policy versions are immutable once active/referenced.
- CP-04: Subject override wins over school default.
- TR-01: Only finalized assessments participate.
- TR-02: Last-N selection is applied to finalized scheduled assessments before learner status filtering.
- TR-03: Absent/excused/not-marked are missing values, never implicit zero.
- TR-04: All three missing-data rules behave exactly as specified.
- TR-05: Computation uses decimal arithmetic and explicit rounding.
- TR-06: Level-points nearest ties go to the better configured point; down selects the lower configured point.
- TR-07: Every computation persists exact policy, scheme, algorithm version and explanation rows.
- TR-09: Recompute creates a new term-result version.
- TR-10: Frozen results reject normal recompute.

## Report cards

- RC-01: One report-card family exists per enrollment/term.
- RC-02: Report-card lines snapshot labels, values, subject names and exact term-result version.
- RC-03: Review, approval, and publication are separate authorized transitions.
- RC-04: Publication freezes the referenced term-result versions atomically.
- RC-05: Revision creates a new version and preserves the old published version.
- RC-06: Abandoned revision leaves prior published report card visible.
- RC-07: Guardian can read only latest published report card for linked learner with `can_view_results=true`.
- RC-08: Incomplete results block publication by default.

## Security and operations

- SEC-01: Tenant/school isolation is tested at service and RLS layers.
- SEC-02: Cross-school foreign-key relationships are rejected.
- SEC-03: Audit records are append-only and include required before/after values and reasons.
- SEC-04: No sensitive endpoint trusts tenant/school IDs supplied by client.
- OPS-01: Migrations pass from an empty database and from the latest supported production schema.
- OPS-02: All migrations have rollback/forward-recovery strategy consistent with project policy.
- OPS-03: Background recomputation jobs are idempotent.
- OPS-04: API contract tests match OpenAPI.
- OPS-05: Existing project compliance checks pass.

---

# 20. Required test matrix

At minimum, automate the following.

### Conversion tests

- 0, exact band lower boundary, just below boundary, exact upper boundary, 100.
- Decimal percentage values at every boundary.
- Non-100 max score normalization.
- Rubric midpoint conversion.
- Invalid rubric label.
- Scheme version changed after assessment creation; old assessment still uses old version.
- Percentage scheme with percentage working scale.
- Level scheme with level-points working scale.
- Mismatched scheme/scale rejected.

### Computation tests

- Both categories present with equal and unequal weights.
- One category missing under each missing-data rule.
- Both categories missing under each missing-data rule.
- Category weight zero.
- Last-N fewer than N assessments.
- Last-N includes absent, excused, and not-marked records.
- Same-date sequence ordering.
- Exact nearest midpoint tie.
- Down rounding at exact and fractional points.
- Scheme cutoff at every percentage boundary.
- Recompute after policy version change.
- Deterministic repeated computation with same inputs.
- Decimal precision and no floating-point drift.

### Lifecycle tests

- Teacher finalizes own assessment.
- Teacher cannot finalize another teacher's assessment.
- Teacher cannot edit finalized results.
- Admin/head teacher correction requires reason.
- Reopen/cancel writes audit.
- Frozen term result cannot be overwritten.
- Revision produces new term-result/report-card versions.
- Failed revision leaves old publication intact.

### Security tests

- Cross-tenant read/write.
- Cross-school read/write.
- Mismatched school foreign keys.
- Teacher offering authorization.
- Guardian linked learner authorization.
- Guardian cannot view unpublished or superseded versions.
- Audit access restrictions.

### Bulk and concurrency tests

- Duplicate idempotency key with same payload returns original outcome.
- Duplicate idempotency key with different payload is rejected.
- Partial valid/invalid batch.
- Concurrent update conflict.
- Retry after network timeout does not duplicate writes.

---

# 21. Decisions fixed for v1

| Decision               | V1 rule                                                     |
| ---------------------- | ----------------------------------------------------------- |
| Working scale          | One per school: percentage or level_points                  |
| Scheme ownership       | Global templates; school-owned versioned copies             |
| Scheme freeze          | Exact scheme version frozen at assessment creation          |
| Policy selection       | Subject override, then school default                       |
| Aggregation            | Arithmetic average only                                     |
| Categories             | Formative and summative                                     |
| Selection              | All or last N finalized scheduled assessments               |
| Last-N and absence     | Absence consumes a selected slot; no backfill               |
| Missing values         | Absent/excused/not-marked are not zero                      |
| Missing-data default   | `skip_reweight`                                             |
| Rounding default       | `nearest`, with scale-specific exact behavior               |
| Level-points tie       | Tie goes to better/higher point                             |
| Finalized result edits | Admin/head teacher only, reason and audit                   |
| Term result            | Versioned computation record                                |
| Freeze point           | Report-card publication                                     |
| Report-card revision   | New version; old published version retained                 |
| Approval               | Class teacher review, head teacher approval, admin publish  |
| Guardian visibility    | Latest published only, active link and permission flag      |
| Ranking                | No ranking computation in v1                                |
| Notifications          | No revision notification in v1                              |
| Offline queue          | Optional/deferred; idempotency contract required if enabled |

---

# 22. Explicit non-goals and future extension points

### Considered but excluded for v1

- `scheme_cutoff` rounding: left undefined and removed from v1. Rounding must be `nearest` or `down`. If direct band mapping without rounding is needed later, it must be introduced as a new algorithm version with explicit band definitions.

### Online quizzes

Future quiz attempts may feed assessment results through a trusted source adapter. The v1 result model should not accept arbitrary external source references or allow clients to claim a quiz result without server-side verification.

### Strand/sub-strand assessment

Future strand breakdowns should use a separate normalized result-detail table rather than adding unvalidated UUID columns to assessments.

### More advanced computation

Weighted averages, formulas, best-of-N, dropping lowest marks, and custom expressions are out of scope. Add them only through a new algorithm version and explicit policy schema changes.

### Official national reporting

The platform's configurable grading templates are not a claim of official KNEC or NECTA certification. National grading defaults must be verified against current authoritative guidance before being presented as official.

---

# 23. Implementation handoff checklist

Before coding:

- [ ] Map all existing core tables, enums, role names, and RLS helpers.
- [ ] Confirm whether class teacher assignment and subject offering already exist.
- [ ] Confirm audit-log reuse strategy.
- [ ] Confirm school-to-education-system relationship.
- [ ] Review seed grading templates with an education-domain expert; do not label placeholders as official.
- [ ] Approve the worked examples and especially the level-points tie behavior.
- [ ] Add the pure computation test suite before wiring persistence.
- [ ] Finalize OpenAPI schemas and stable error codes.
- [ ] Produce migration plan against the real database.

**Implementation gate:** do not begin migrations until schema reconnaissance, grading-template verification, and the normative computation examples have been reviewed and approved.

_End of authoritative v1 specification._
