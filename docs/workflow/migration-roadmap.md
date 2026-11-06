# Migration Roadmap – Remaining Domain Tables

Status: planning. Do not implement until spec is approved.

## Principles
* All tables use `uuidv7()` PK, `created_at/updated_at/deleted_at`, soft delete partial index.
* Every multi-tenant table gets RLS + tenant isolation policy.

## 1. attendance_marks
Per-period attendance per enrollment.

Columns:
* id uuidv7 PK
* timetable_assignment_id uuid FK timetable_assignments.id RESTRICT
* enrollment_id uuid FK enrollments.id RESTRICT
* date date NOT NULL
* status enum(PRESENT,LATE,ABSENT,EXCUSED) NOT NULL
* marked_by uuid FK users.id
* marked_at timestamptz NOT NULL DEFAULT NOW()
* reason text
* updated_at timestamptz
* created_at/updated_at/deleted_at
Unique timetable_assignment_id, enrollment_id, date.
RLS via schools.

## 1a. attendance_audit_log
Audit trail for attendance edits after day close.

Columns:
* id uuidv7 PK
* attendance_marks_id uuid FK attendance_marks.id RESTRICT
* old_status text
* new_status text
* changed_by uuid FK users.id
* changed_at timestamptz NOT NULL DEFAULT NOW()
* reason text
Created in same transaction as change. RLS via schools.

## 2. fee_structures
Fee categories per school/class/term.

Columns:
* id uuidv7 PK
* school_id uuid FK schools.id RESTRICT
* class_level_id uuid FK class_levels.id RESTRICT
* term_id uuid FK terms.id RESTRICT
* category varchar(64) NOT NULL
* amount_minor bigint NOT NULL
* description text
* created_at/updated_at/deleted_at
Unique school_id, class_level_id, term_id, category.

## 3. payments
Student payments and receipts.

Columns:
* id uuidv7 PK
* student_id uuid FK students.id RESTRICT
* tenant_id uuid FK tenants.id
* amount_minor bigint NOT NULL
* method varchar(64)
* reference varchar(128)
* paid_at date NOT NULL
* receipt_url text
* created_at/updated_at/deleted_at

## 4. audit_logs
Immutable audit trail.

Columns:
* id uuidv7 PK
* tenant_id uuid
* school_id uuid
* actor_user_id uuid
* action varchar(32) NOT NULL
* table_name varchar(64) NOT NULL
* record_id uuid NOT NULL
* changed_fields jsonb
* occurred_at timestamptz NOT NULL DEFAULT NOW()
No soft delete. App role INSERT/SELECT only.

## 5. invites
Staff/guardian invites.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id
* school_id uuid FK schools.id RESTRICT
* email_normalized text NOT NULL
* role user_role NOT NULL
* status enum(PENDING,ACCEPTED,EXPIRED) NOT NULL
* token_hash bytea NOT NULL
* expires_at timestamptz NOT NULL
* created_at/updated_at
Unique email_normalized, school_id, status=PENDING.

## 6. needs_attention
System generated tasks.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id
* school_id uuid FK schools.id RESTRICT
* type varchar(64) NOT NULL
* entity_type varchar(64) NOT NULL
* entity_id uuid NOT NULL
* message text
* status enum(OPEN,ACKNOWLEDGED,RESOLVED) NOT NULL DEFAULT 'OPEN'
* created_at/updated_at/deleted_at

## 7. education_systems (extend existing)
**Existing table** from migration 20241028000003_core_domain.sql. Add assessment-specific columns via new migration:

Columns to ADD:
* show_rank boolean NOT NULL DEFAULT false

Existing columns (unchanged):
* id uuidv7 PK
* country_id uuid FK countries.id RESTRICT
* system_name varchar(255) NOT NULL
* description text
* created_at/updated_at

Unique: (country_id, system_name) — existing
RLS: Not directly (global table); schools reference via education_system_id.

## 7a. school_assessment_settings
Per-school assessment configuration.

Columns:
* id uuidv7 PK
* school_id uuid FK schools.id RESTRICT
* tenant_id uuid FK tenants.id RESTRICT
* education_system_id uuid FK education_systems.id RESTRICT
* working_scale varchar(16) NOT NULL CHECK (working_scale IN ('percentage','level_points'))
* default_grading_scheme_family_id uuid FK grading_scheme_families.id
* default_computation_policy_family_id uuid FK computation_policy_families.id
* created_at/updated_at/deleted_at
Unique school_id.
RLS via schools.

## 8. grading_scheme_templates
Global read-only templates.

Columns:
* id uuidv7 PK
* education_system_id uuid FK education_systems.id RESTRICT
* template_key varchar(64) NOT NULL
* name varchar(128) NOT NULL
* is_level_based boolean NOT NULL
* description text
* is_recommended_default boolean NOT NULL DEFAULT false
* seed_version integer NOT NULL DEFAULT 1
* created_at
Unique (education_system_id, template_key, seed_version).
Global, not tenant-scoped.

## 8a. grading_scheme_template_bands
Bands for templates.

Columns:
* id uuidv7 PK
* template_id uuid FK grading_scheme_templates.id RESTRICT
* label varchar(16) NOT NULL
* min_percentage decimal(5,2) NOT NULL
* max_percentage decimal(5,2) NOT NULL
* point_value decimal(4,2) NOT NULL
* description text
* sequence_index integer NOT NULL
* created_at
Unique (template_id, label).
Global.

## 9. grading_scheme_families
Tenant-scoped family grouping versions.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id RESTRICT
* school_id uuid FK schools.id RESTRICT
* education_system_id uuid FK education_systems.id RESTRICT
* template_id uuid FK grading_scheme_templates.id
* name varchar(128) NOT NULL
* created_by uuid FK users.id
* created_at/updated_at/deleted_at
Unique (school_id, name).
RLS via schools.

## 10. grading_schemes
Versioned working copies.

Columns:
* id uuidv7 PK
* family_id uuid FK grading_scheme_families.id RESTRICT
* tenant_id uuid FK tenants.id RESTRICT
* education_system_id uuid FK education_systems.id RESTRICT
* name varchar(128) NOT NULL
* version integer NOT NULL DEFAULT 1
* is_level_based boolean NOT NULL
* description text
* is_active boolean NOT NULL DEFAULT true
* created_by uuid FK users.id
* created_at/updated_at/deleted_at
Unique (family_id, version).
RLS via schools.

## 9. grading_bands
Bands within a grading scheme; powers percentage↔level↔points conversion.

Columns:
* id uuidv7 PK
* grading_scheme_id uuid FK grading_schemes.id RESTRICT
* label varchar(16) NOT NULL
* min_percentage decimal(5,2) NOT NULL CHECK (min_percentage >= 0 AND min_percentage <= 100)
* max_percentage decimal(5,2) NOT NULL CHECK (max_percentage >= 0 AND max_percentage <= 100)
* point_value decimal(4,2) NOT NULL
* description text
* sequence_index integer NOT NULL
* created_at/updated_at
Unique (grading_scheme_id, label).
RLS via schools (through grading_scheme → education_system → tenant).

## 10. subject_offerings
Subject taught in a classroom for an academic year with assigned teacher.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id RESTRICT
* school_id uuid FK schools.id RESTRICT
* subject_id uuid FK subjects.id RESTRICT
* classroom_id uuid FK classrooms.id RESTRICT
* academic_year_id uuid FK academic_years.id RESTRICT
* teacher_id uuid FK users.id RESTRICT
* grading_scheme_id uuid FK grading_schemes.id
* computation_policy_id uuid FK computation_policies.id
* created_at/updated_at/deleted_at
Unique (school_id, subject_id, classroom_id, academic_year_id).
RLS via schools.

## 11. assessments
One marked activity (test, quiz, project, exam).

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id RESTRICT
* school_id uuid FK schools.id RESTRICT
* subject_offering_id uuid FK subject_offerings.id RESTRICT
* term_id uuid FK terms.id RESTRICT
* name varchar(128) NOT NULL
* assessment_date date NOT NULL
* sequence_index integer NOT NULL DEFAULT 0
* category varchar(16) NOT NULL CHECK (category IN ('formative','summative'))
* scoring_mode varchar(16) NOT NULL CHECK (scoring_mode IN ('percentage','rubric'))
* max_score decimal(6,2) NOT NULL DEFAULT 100
* grading_scheme_id uuid FK grading_schemes.id RESTRICT
* status varchar(16) NOT NULL CHECK (status IN ('draft','open','finalized')) DEFAULT 'draft'
* strand_id uuid NULL -- future
* sub_strand_id uuid NULL -- future
* created_by uuid FK users.id
* created_at/updated_at/deleted_at
RLS via schools.

## 12. assessment_results
One result per enrollment per assessment.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id RESTRICT
* school_id uuid FK schools.id RESTRICT
* assessment_id uuid FK assessments.id RESTRICT
* enrollment_id uuid FK enrollments.id RESTRICT
* raw_value decimal(6,2)
* raw_level_label varchar(16)
* working_value decimal(6,3)
* derived_grade varchar(16)
* grading_scheme_version integer NOT NULL
* status varchar(16) NOT NULL CHECK (status IN ('entered','absent','excused','not_marked')) DEFAULT 'not_marked'
* entered_by uuid FK users.id
* entered_at timestamptz
* source_type varchar(16) CHECK (source_type IN ('manual','quiz'))
* source_ref_id uuid
* created_at/updated_at
Unique (assessment_id, enrollment_id).
RLS via schools.

## 13. computation_policies
Versioned school-owned computation policies.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id RESTRICT
* school_id uuid FK schools.id RESTRICT
* name varchar(128) NOT NULL
* version integer NOT NULL DEFAULT 1
* is_default boolean NOT NULL DEFAULT false
* formative_weight decimal(5,2) NOT NULL CHECK (formative_weight >= 0 AND formative_weight <= 100)
* summative_weight decimal(5,2) NOT NULL CHECK (summative_weight >= 0 AND summative_weight <= 100)
* formative_selection varchar(16) NOT NULL CHECK (formative_selection IN ('all','last_n')) DEFAULT 'all'
* formative_last_n integer CHECK (formative_last_n > 0)
* summative_selection varchar(16) NOT NULL CHECK (summative_selection IN ('all','last_n')) DEFAULT 'all'
* summative_last_n integer CHECK (summative_last_n > 0)
* aggregation varchar(16) NOT NULL CHECK (aggregation IN ('average')) DEFAULT 'average'
* rounding_rule varchar(16) NOT NULL CHECK (rounding_rule IN ('nearest','down','scheme_cutoff')) DEFAULT 'nearest'
* missing_data_rule varchar(16) NOT NULL CHECK (missing_data_rule IN ('skip_reweight','zero','block')) DEFAULT 'skip_reweight'
* created_by uuid FK users.id
* created_at/updated_at/deleted_at
Unique (school_id, name, version).
RLS via schools.

## 14. term_results
Computed term result per enrollment per subject offering per term.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id RESTRICT
* school_id uuid FK schools.id RESTRICT
* enrollment_id uuid FK enrollments.id RESTRICT
* subject_offering_id uuid FK subject_offerings.id RESTRICT
* term_id uuid FK terms.id RESTRICT
* working_value decimal(6,3)
* final_grade varchar(16)
* computation_policy_version integer NOT NULL
* status varchar(16) NOT NULL CHECK (status IN ('draft','incomplete','frozen')) DEFAULT 'draft'
* computed_at timestamptz
* computed_by uuid FK users.id
* created_at/updated_at/deleted_at
Unique (enrollment_id, subject_offering_id, term_id).
RLS via schools.

## 15. term_result_inputs
Explanation rows for term results.

Columns:
* id uuidv7 PK
* term_result_id uuid FK term_results.id RESTRICT
* assessment_result_id uuid FK assessment_results.id RESTRICT
* category varchar(16) NOT NULL CHECK (category IN ('formative','summative'))
* working_value decimal(6,3) NOT NULL
* included boolean NOT NULL
* drop_reason varchar(64)
* created_at timestamptz NOT NULL DEFAULT NOW()
RLS via schools (through term_result).

## 16. report_cards
Report card per enrollment per term with lifecycle.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id RESTRICT
* school_id uuid FK schools.id RESTRICT
* enrollment_id uuid FK enrollments.id RESTRICT
* term_id uuid FK terms.id RESTRICT
* version integer NOT NULL DEFAULT 1
* status varchar(16) NOT NULL CHECK (status IN ('draft','reviewed','approved','published','revised')) DEFAULT 'draft'
* class_teacher_remark text
* head_teacher_remark text
* reviewed_by uuid FK users.id
* reviewed_at timestamptz
* approved_by uuid FK users.id
* approved_at timestamptz
* published_at timestamptz
* published_by uuid FK users.id
* created_at/updated_at/deleted_at
Unique (enrollment_id, term_id, version).
RLS via schools.

## 17. report_card_lines
Snapshot per subject at publication.

Columns:
* id uuidv7 PK
* report_card_id uuid FK report_cards.id RESTRICT
* subject_offering_id uuid FK subject_offerings.id RESTRICT
* final_grade varchar(16) NOT NULL
* working_value decimal(6,3) NOT NULL
* subject_remark text
* grading_labels_json jsonb NOT NULL
* created_at timestamptz NOT NULL DEFAULT NOW()
Unique (report_card_id, subject_offering_id).
RLS via schools (through report_card).

## 18. audit_logs (assessment domain)
Immutable audit trail for assessment actions.

Columns:
* id uuidv7 PK
* tenant_id uuid FK tenants.id RESTRICT
* school_id uuid FK schools.id RESTRICT
* actor_user_id uuid FK users.id RESTRICT
* action varchar(32) NOT NULL
* table_name varchar(64) NOT NULL
* record_id uuid NOT NULL
* changed_fields jsonb
* reason text
* occurred_at timestamptz NOT NULL DEFAULT NOW()
No soft delete. App role INSERT/SELECT only.
RLS via schools.

## Optional health/behavior tables
To be specced separately:
* health_records, vaccinations, behavior_records with audit on view per FR-WEL-001.

Each table requires approved spec, migration file, integration test, RLS policy, and `make compliance` before merge.
