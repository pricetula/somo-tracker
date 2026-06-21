================================================================================
SOMOTRACKER DATABASE SCHEMA SPECIFICATION: CBC/CBE ACADEMIC TRACKING LAYER (v5)
================================================================================

SYSTEM PROMPT FOR AI CODING AGENT:
You are an expert Principal Database Engineer specializing in PostgreSQL 16+,
Domain-Driven Design (DDD), and Clean Architecture.

Your task is to completely rewrite 000001_initial_schema.up.sql. SomoTracker
is a CBC/CBE-only platform for Kenya's 2-6-3-3 education system. No other
education system exists on this platform — not 8-4-4, not IGCSE, not IB. The
existing schema was scaffolded generically; it must be torn down and rebuilt as
a purpose-built CBC schema. Write the new file from scratch.

---

## CORE ARCHITECTURAL RULES

1. STRICT CBC/CBE COMPLIANCE: No columns for total numeric marks, percentage
   aggregates, or position ranks anywhere in the schema. No 'total_marks',
   'average', 'rank', 'mean_grade', 'points' columns. All assessment evaluations
   are rubric-level records only. The sole permitted numeric assessment field is
   `raw_score` on `learner_rubric_results` — it stores the pre-conversion mark
   only and is never summed or averaged.

2. ONE SYSTEM ONLY: There is no `education_systems` table. There are no
   `curriculum_stages`, `assessment_grade_scales`, or `assessment_types` catalog
   tables. These generic abstractions are deleted. CBC structure is encoded
   directly as CHECK constraints and enums in the relevant tables. No row in
   any table may represent a non-CBC educational context.

3. CLEAN NAMING: Tables that were generic (schools, classes, students,
   student*enrollments) are renamed with the `cbc*`prefix to make the domain
boundary explicit. Tables that already had the`cbc\_` prefix keep it.

4. ENUMS — MINIMAL AND CBC-CORRECT:
   - DROP gender_type (had 'OTHER', 'PREFER_NOT_TO_SAY' — invalid in NEMIS/KNEC).
     Student gender is CHAR(1) with CHECK IN ('M', 'F').
   - DROP enrollment_status (had 'GRADUATED' — 8-4-4 language).
     Redefine with correct CBC values only.
   - KEEP user_role, attendance_status, invitation_status unchanged.

5. MULTI-TENANCY: Every school-scoped table carries tenant_id. The existing
   (tenant_id, id) composite UNIQUE constraint pattern is maintained on all
   tables that participate in cross-table FK chains, enabling tenant-safe
   composite foreign keys.

6. IDENTIFIERS: UUIDv4 via gen_random_uuid() for all PKs.

7. DATA INTEGRITY: Explicit FKs, NOT NULL where applicable, strict CHECK
   constraints for all enum-equivalent columns.

8. INDEXING: Explicit B-Tree indexes on all FK columns. Composite indexes on
   columns that are always filtered together. No reliance on implicit PK indexes
   to satisfy FK join performance.

9. STANDARDIZATION: lowercase snake_case for all identifiers.

---

## KENYAN CBC STRUCTURAL CONTEXT

Kenya's CBC follows the 2-6-3-3 structure. This is hardcoded into the schema
via CHECK constraints — there is no generic tier lookup table.

EARLY_YEARS PP1, PP2, Grade 1–3 (Pre-Primary + Lower Primary)
UPPER_PRIMARY Grade 4–6
JUNIOR_SECONDARY Grade 7–9 (JSS)
SENIOR_SCHOOL Grade 10–12

KNEC GRADE DESIGNATIONS (14 values, used in grade_level CHECK constraints):
PP1, PP2, G1, G2, G3, G4, G5, G6, G7, G8, G9, G10, G11, G12

OFFICIAL RUBRIC LEVELS (4 only — no sub-levels are accepted by KNEC portal):
EE Exceeds Expectations 80–100%
ME Meets Expectations 65–79%
AE Approaching Expectations 50–64%
BE Below Expectations 0–49%

SIX KNEC ASSESSMENT INSTRUMENT TYPES (each has a distinct workflow):
Formative_Classroom Teacher-designed; ongoing; never uploaded to KNEC
KNEC_Written_Assessment WAT papers; Term 3; Grades 3–8; downloaded from KNEC
KNEC_SBA_Project Downloaded from cba.knec.ac.ke; scores uploaded back;
applicable grades: G4, G5, G7, G8, G10, G11
National_KPSEA Grade 6 national transitional exam (end of Upper Primary)
National_KJSEA Grade 9 national transitional exam (end of JSS)
National_KSSEA Grade 12 national exit exam (end of Senior School)

OFFICIAL KNEC WEIGHTING FORMULA:
KPSEA placement: 60% SBA Projects (G4+G5) + 40% KPSEA written (G6)
KJSEA placement: 20% SBA Projects (G7+G8) + 20% KPSEA result + 60% KJSEA written (G9)
KSSEA: Weighting TBD by KNEC (Grade 12 cohort not yet sat as of 2025)

KNEC CBA PORTAL: cba.knec.ac.ke
Schools log in using knec_school_code as username.
Parents access results via cba.knec.ac.ke/Parent using knec_assessment_number.

CBC LEARNING PATHWAYS:
Age_Based Standard mainstream pathway (vast majority of learners)
Stage_Based SNE pathway for learners with severe cognitive/multiple disabilities
Governed by CBAF-FL framework; different assessment instruments apply

---

## TABLES BEING DELETED FROM THE EXISTING SCHEMA

The following tables from the old schema are REMOVED ENTIRELY. Do not recreate
them in any form. Do not reference them. Do not seed them.

education_systems Generic multi-system anchor. No equivalent in CBC-only schema.
curriculum_stages Generic tier catalog. Replaced by education_level CHECK constraint.
assessment_grade_scales Generic rubric catalog. Replaced by rubric_level CHECK constraint.
assessment_types Generic assessment type catalog. Replaced by type CHECK constraint.
grades Generic grade catalog with broken FK chain. Replaced by
grade_level VARCHAR(5) CHECK constraint on cbc_classes.

The following enums are DROPPED and REDEFINED:
DROP TYPE gender_type; -- Had OTHER, PREFER_NOT_TO_SAY. Invalid for NEMIS/KNEC.
DROP TYPE enrollment_status; -- Had GRADUATED. 8-4-4 language. Redefined below.

---

## COMPLETE SCHEMA — ALL 31 TABLES

Write the new 000001_initial_schema.up.sql containing exactly these tables in
the order specified. All DDL must be wrapped in a single BEGIN; ... COMMIT;

================================================================================
LAYER 0 — EXTENSIONS, FUNCTIONS, ENUMS
================================================================================

Extensions:
CREATE EXTENSION IF NOT EXISTS btree_gist;

Functions (keep both existing functions verbatim):
fn_set_updated_at() — sets NEW.updated_at = NOW() on BEFORE UPDATE trigger
fn_timerange(day_of_week, start_time, end_time) → tsrange
— maps day_of_week (1=Mon…7=Sun) onto base week 2024-01-01
for GiST exclusion constraints on cbc_timetable_slots

Enums (only these 3 — gender_type and enrollment_status are gone):

CREATE TYPE user_role AS ENUM (
'SYSTEM_ADMIN', 'SCHOOL_ADMIN', 'TEACHER', 'NURSE', 'FINANCE'
);

CREATE TYPE attendance_status AS ENUM (
'PRESENT', 'ABSENT', 'LATE', 'EXCUSED'
);

CREATE TYPE invitation_status AS ENUM (
'pending', 'accepted', 'expired', 'revoked'
);

-- enrollment_status redefined WITHOUT 'GRADUATED':
CREATE TYPE cbc_enrollment_status AS ENUM (
'ACTIVE', -- Currently enrolled and attending
'SUSPENDED', -- Temporarily removed from active learning
'TRANSFERRED', -- Moved to another school; record retained
'COMPLETED_CYCLE' -- Successfully completed a CBC education cycle
-- (replaces legacy 8-4-4 'GRADUATED')
);

================================================================================
LAYER 1 — PLATFORM INFRASTRUCTURE (unchanged from existing schema)
================================================================================

These 5 tables are kept exactly as they were, with one addition to `users`:

---

## Table: tenants

id UUID PK DEFAULT gen_random_uuid()
name VARCHAR(255) NOT NULL
slug VARCHAR(255) NOT NULL UNIQUE
stytch_org_id VARCHAR(255) NOT NULL UNIQUE
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Indexes: idx_tenants_slug, idx_tenants_stytch_org_id

---

## Table: users

id UUID PK DEFAULT gen_random_uuid()
email VARCHAR(255) NOT NULL
tenant_id UUID FK → tenants(id) ON DELETE CASCADE
first_name VARCHAR(255) NOT NULL DEFAULT ''
last_name VARCHAR(255) NOT NULL DEFAULT ''
is_active BOOLEAN NOT NULL DEFAULT TRUE
external_auth_id VARCHAR(255) UNIQUE
tsc_number VARCHAR(15) NULL
knec_panel_assessor_id VARCHAR(20) NULL
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
UNIQUE (tenant_id, id) -- composite UK for tenant-safe FK referencing
UNIQUE (tsc_number) WHERE tsc_number IS NOT NULL -- partial unique
UNIQUE (knec_panel_assessor_id) WHERE knec_panel_assessor_id IS NOT NULL

Indexes: idx_users_email (UNIQUE), idx_users_tenant
Trigger: trg_users_updated_at → fn_set_updated_at()

COMMENT ON COLUMN users.tsc_number:
'Teachers Service Commission registration number. Populated only for users
with the TEACHER role. Required for TSC portal access and official deployment.'

COMMENT ON COLUMN users.knec_panel_assessor_id:
'Assigned ONLY to teachers formally appointed to KNEC national exam panels
(KPSEA, KJSEA, KSSEA invigilation or marking). NOT required for classroom
SBA delivery — all SBA uploads use the school knec_school_code, not teacher IDs.'

---

## Table: sessions

id UUID PK DEFAULT gen_random_uuid()
token VARCHAR(128) NOT NULL UNIQUE
user_id UUID NOT NULL FK → users(id) ON DELETE CASCADE
tenant_id UUID NOT NULL FK → tenants(id) ON DELETE CASCADE
stytch_member_id VARCHAR(255) NOT NULL
stytch_org_id VARCHAR(255) NOT NULL
stytch_session_token VARCHAR(512) NOT NULL DEFAULT ''
device_fingerprint VARCHAR(128) NOT NULL DEFAULT ''
expires_at TIMESTAMPTZ NOT NULL
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Indexes: idx_sessions_token, idx_sessions_user_id, idx_sessions_tenant_id,
idx_sessions_stytch_session_token

---

## Table: memberships

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL FK → tenants(id) ON DELETE CASCADE
user_id UUID NOT NULL FK → users(id) ON DELETE CASCADE
school_id UUID NOT NULL
role user_role NOT NULL
is_active BOOLEAN NOT NULL DEFAULT true
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
UNIQUE (user_id, school_id)

Indexes: idx_memberships_tenant_id, idx_memberships_user_id,
idx_memberships_school_id

---

## Table: invitations

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL FK → tenants(id) ON DELETE CASCADE
school_id UUID NOT NULL
email VARCHAR(255) NOT NULL
role user_role NOT NULL
status invitation_status NOT NULL DEFAULT 'pending'
invited_by UUID FK → users(id) ON DELETE SET NULL
token TEXT NOT NULL
expires_at TIMESTAMPTZ NOT NULL
accepted_at TIMESTAMPTZ NULL
first_name VARCHAR(255) NULL
last_name VARCHAR(255) NULL
phone VARCHAR(50) NULL
registration_number VARCHAR(100) NULL
stytch_member_id VARCHAR(255) NULL
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE

Indexes: idx_invitations_tenant_id, idx_invitations_school_id,
idx_invitations_email, idx_invitations_status

================================================================================
LAYER 2 — CORE CBC ACTORS
================================================================================

---

Table: cbc_schools
(replaces generic `schools` — education_system_id and is_demo columns removed)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL FK → tenants(id) ON DELETE CASCADE
name VARCHAR(255) NOT NULL
knec_school_code VARCHAR(15) NULL
nemis_institution_code VARCHAR(20) NULL
county VARCHAR(50) NOT NULL
sub_county VARCHAR(50) NOT NULL
ward VARCHAR(50) NULL
school_type VARCHAR(20) NOT NULL
is_active BOOLEAN NOT NULL DEFAULT true
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
UNIQUE (tenant_id, id) -- composite UK for tenant-safe FK referencing
CHECK (school_type IN ('Public', 'Private', 'Special_Needs_School'))
UNIQUE (knec_school_code) WHERE knec_school_code IS NOT NULL
UNIQUE (nemis_institution_code) WHERE nemis_institution_code IS NOT NULL

Indexes: idx_cbc_schools_tenant_id

COMMENT ON COLUMN cbc_schools.knec_school_code:
'Official KNEC center code (8–10 digit numeric string). Used as the school
login username on the CBA portal at cba.knec.ac.ke. Required before any
SBA score uploads can be submitted to KNEC.'

COMMENT ON COLUMN cbc_schools.nemis_institution_code:
'National Education Management Information System institution code.
Assigned by the Ministry of Education. Used for MoE reporting and
NEMIS data synchronisation.'

---

Table: cbc_classes
(replaces generic `classes` — education_system_id and grade_id FK removed;
grade_level is now a direct CHECK constraint, not a FK to a grades table)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
school_id UUID NOT NULL
academic_year_id UUID NOT NULL
name VARCHAR(100) NOT NULL
grade_level VARCHAR(5) NOT NULL
stream VARCHAR(100) NOT NULL DEFAULT ''
is_active BOOLEAN NOT NULL DEFAULT true

Constraints:
UNIQUE (tenant_id, id)
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, academic_year_id) → academic_years(tenant_id, id) ON DELETE CASCADE
CHECK (grade_level IN (
'PP1','PP2',
'G1','G2','G3',
'G4','G5','G6',
'G7','G8','G9',
'G10','G11','G12'
))

Indexes: idx_cbc_classes_tenant_id, idx_cbc_classes_school_id,
idx_cbc_classes_academic_year_id, idx_cbc_classes_grade_level,
idx_cbc_classes_stream

COMMENT ON COLUMN cbc_classes.grade_level:
'Official KNEC grade designation. Determines which assessment instruments,
SBA projects, and KNEC portal upload windows apply to the class. Values
match KNEC CBA portal grade codes: PP1–PP2 (Pre-Primary), G1–G12.'

---

Table: cbc_students
(replaces generic `students` — gender_type enum replaced with CHAR(1) CHECK;
upi_number, knec_assessment_number, learning_pathway added)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL FK → tenants(id) ON DELETE CASCADE
first_name VARCHAR(100) NOT NULL
middle_name VARCHAR(100) NULL
last_name VARCHAR(100) NOT NULL
gender CHAR(1) NOT NULL
date_of_birth DATE NOT NULL
upi_number VARCHAR(20) NULL
knec_assessment_number VARCHAR(15) NULL
learning_pathway VARCHAR(15) NOT NULL DEFAULT 'Age_Based'
is_active BOOLEAN NOT NULL DEFAULT true
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
UNIQUE (tenant_id, id)
CHECK (gender IN ('M', 'F'))
CHECK (learning_pathway IN ('Age_Based', 'Stage_Based'))
UNIQUE (upi_number) WHERE upi_number IS NOT NULL
UNIQUE (knec_assessment_number) WHERE knec_assessment_number IS NOT NULL

Indexes: idx_cbc_students_tenant_id

COMMENT ON COLUMN cbc_students.gender:
'CBC/NEMIS-compliant gender field. M=Male, F=Female only. KNEC registration
and NEMIS records do not support other values.'

COMMENT ON COLUMN cbc_students.upi_number:
'Unique Personal Identifier assigned by NEMIS at school enrollment. Used in
all Ministry of Education reporting and NEMIS data submissions.'

COMMENT ON COLUMN cbc_students.knec_assessment_number:
'Permanent CBC identifier assigned by KNEC from Grade 3 onward. Required for
KPSEA/KJSEA/KSSEA exam registration. Parents use this number to access
learner results at cba.knec.ac.ke/Parent.'

COMMENT ON COLUMN cbc_students.learning_pathway:
'Determines which KNEC assessment framework governs the learner.
Age_Based: standard mainstream CBC curriculum (vast majority).
Stage_Based: SNE pathway for learners with severe cognitive or multiple
disabilities, governed by the CBAF-FL framework.'

---

Table: cbc_student_enrollments
(replaces generic `student_enrollments` — uses cbc_enrollment_status enum;
references cbc_students, cbc_schools, cbc_classes)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
student_id UUID NOT NULL
school_id UUID NOT NULL
academic_term_id UUID NOT NULL
class_id UUID NULL
status cbc_enrollment_status NOT NULL DEFAULT 'ACTIVE'
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
FK (tenant_id, student_id) → cbc_students(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, school_id, academic_term_id)
→ academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
FK (tenant_id, class_id) → cbc_classes(tenant_id, id) ON DELETE SET NULL
UNIQUE (student_id, academic_term_id) -- one enrollment per student per term

Indexes: idx_cbc_enrollments_tenant_id, idx_cbc_enrollments_student_id,
idx_cbc_enrollments_school_id, idx_cbc_enrollments_term_id,
idx_cbc_enrollments_class_id

================================================================================
LAYER 3 — ACADEMIC CALENDAR
================================================================================
(Kept structurally unchanged; only update FK references from schools → cbc_schools)

---

## Table: academic_years

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
school_id UUID NOT NULL
name VARCHAR(50) NOT NULL
start_date DATE NOT NULL
end_date DATE NOT NULL
is_current BOOLEAN NOT NULL DEFAULT false

Constraints:
UNIQUE (tenant_id, id)
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
CHECK (end_date > start_date)

Indexes: idx_academic_years_tenant_id, idx_academic_years_school_id
Partial unique: UNIQUE (school_id) WHERE is_current = true
-- Enforces at most one current year per school

---

## Table: academic_terms

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
school_id UUID NOT NULL
academic_year_id UUID NOT NULL
name VARCHAR(100) NOT NULL
term_number SMALLINT NOT NULL
start_date DATE NOT NULL
end_date DATE NOT NULL
is_current BOOLEAN NOT NULL DEFAULT false
is_final BOOLEAN NOT NULL DEFAULT false

Constraints:
UNIQUE (tenant_id, id)
UNIQUE (tenant_id, school_id, id)
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, academic_year_id) → academic_years(tenant_id, id) ON DELETE CASCADE
CHECK (end_date > start_date)
CHECK (term_number BETWEEN 1 AND 3)

Indexes: idx_academic_terms_tenant_id, idx_academic_terms_school_id,
idx_academic_terms_year_id
Partial unique: UNIQUE (academic_year_id) WHERE is_current = true

COMMENT ON COLUMN academic_terms.term_number:
'Kenya CBC operates a 3-term academic year. term_number enforces this:
1 = Term 1, 2 = Term 2, 3 = Term 3.'

================================================================================
LAYER 4 — HEALTH & FINANCIALS (platform features, no CBC changes)
================================================================================
(Keep all FK references updated to cbc_students/cbc_schools where applicable.
fee_templates: remove education_system_id and grade_id columns — grade context
now lives on cbc_classes. Add grade_level VARCHAR(5) with CHECK instead.)

---

## Table: student_health_profiles

id UUID PK DEFAULT gen_random_uuid()
student_id UUID UNIQUE NOT NULL FK → cbc_students(id) ON DELETE CASCADE
blood_group VARCHAR(5) NULL
allergies TEXT[] NULL
chronic_conditions TEXT[] NULL
emergency_instructions TEXT NULL

---

## Table: medical_incidents

id UUID PK DEFAULT gen_random_uuid()
student_id UUID NOT NULL FK → cbc_students(id) ON DELETE CASCADE
incident_timestamp TIMESTAMPTZ NOT NULL
symptoms TEXT NOT NULL
action_taken TEXT NOT NULL
logged_by UUID NOT NULL FK → users(id)

Indexes: idx_medical_incidents_student_id

---

## Table: fee_categories

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
school_id UUID NOT NULL
name VARCHAR(150) NOT NULL
is_mandatory BOOLEAN NOT NULL DEFAULT true

Constraints:
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE

Indexes: idx_fee_categories_tenant, idx_fee_categories_school_id

---

Table: fee_templates
(education_system_id and grade_id FK removed; grade_level CHECK added)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
school_id UUID NOT NULL
academic_term_id UUID NOT NULL
grade_level VARCHAR(5) NOT NULL
fee_category_id UUID NOT NULL FK → fee_categories(id) ON DELETE CASCADE
amount NUMERIC(12,2) NOT NULL CHECK (amount >= 0)

Constraints:
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, school_id, academic_term_id)
→ academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
CHECK (grade_level IN (
'PP1','PP2','G1','G2','G3','G4','G5','G6',
'G7','G8','G9','G10','G11','G12'
))
UNIQUE (academic_term_id, grade_level, fee_category_id)

Indexes: idx_fee_templates_tenant, idx_fee_templates_school_term,
idx_fee_templates_grade_level

---

## Table: invoices

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
student_id UUID NOT NULL
school_id UUID NOT NULL
academic_term_id UUID NOT NULL
invoice_label VARCHAR(255) NULL
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
UNIQUE (tenant_id, id)
FK (tenant_id, student_id) → cbc_students(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, school_id, academic_term_id)
→ academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
UNIQUE (student_id, academic_term_id)

Indexes: idx_invoices_tenant, idx_invoices_student_term

---

## Table: invoice_items

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
invoice_id UUID NOT NULL
fee_category_id UUID NOT NULL FK → fee_categories(id) ON DELETE CASCADE
description VARCHAR(255) NULL
amount NUMERIC(12,2) NOT NULL CHECK (amount >= 0)
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
FK (tenant_id, invoice_id) → invoices(tenant_id, id) ON DELETE CASCADE

Indexes: idx_invoice_items_tenant, idx_invoice_items_invoice_id,
idx_invoice_items_fee_category

---

## Table: payments

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
invoice_id UUID NOT NULL
amount NUMERIC(12,2) NOT NULL CHECK (amount > 0)
payment_method VARCHAR(50) NULL
reference_code VARCHAR(100) NULL
recorded_by UUID NOT NULL FK → users(id)
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
FK (tenant_id, invoice_id) → invoices(tenant_id, id) ON DELETE CASCADE

Indexes: idx_payments_tenant, idx_payments_invoice_id

================================================================================
LAYER 5 — CBC CURRICULUM STRUCTURE
================================================================================

---

Table: cbc_learning_areas
(education_system_id and grade_id FKs removed; education_level CHECK added)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
school_id UUID NOT NULL
name VARCHAR(150) NOT NULL
code VARCHAR(50) NOT NULL
education_level VARCHAR(20) NOT NULL

Constraints:
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
CHECK (education_level IN (
'Early_Years', -- PP1, PP2, Grade 1–3
'Upper_Primary', -- Grade 4–6
'Junior_Secondary', -- Grade 7–9
'Senior_School' -- Grade 10–12
))
UNIQUE (tenant_id, school_id, code) -- code unique within a school's curriculum

Indexes: idx_cbc_learning_areas_tenant, idx_cbc_learning_areas_school_id,
idx_cbc_learning_areas_education_level

COMMENT ON COLUMN cbc_learning_areas.education_level:
'The CBC tier this learning area belongs to, per KICD curriculum structure.
Determines applicable KNEC assessment instruments and portal upload eligibility.'

COMMENT ON COLUMN cbc_learning_areas.code:
'Short KICD-defined code for this learning area, e.g. MATH, ENG, KISW,
INT_SCI, PRE_TECH, SOC_STD. Unique within a school''s curriculum.'

---

Table: cbc_strands
(unchanged structurally; FK updated to rebuilt cbc_learning_areas)

---

id UUID PK DEFAULT gen_random_uuid()
learning_area_id UUID NOT NULL FK → cbc_learning_areas(id) ON DELETE CASCADE
name VARCHAR(255) NOT NULL

Indexes: idx_cbc_strands_learning_area_id

---

Table: cbc_sub_strands
(unchanged)

---

id UUID PK DEFAULT gen_random_uuid()
strand_id UUID NOT NULL FK → cbc_strands(id) ON DELETE CASCADE
name VARCHAR(255) NOT NULL

Indexes: idx_cbc_sub_strands_strand_id

---

## Table: performance_indicators ← NEW

Responsibility: Atomic, assessable learning outcome within a sub-strand.
The leaf node of the curriculum hierarchy. Teachers assess learners against
these indicators using the EE/ME/AE/BE rubric.

id UUID PK DEFAULT gen_random_uuid()
sub_strand_id UUID NOT NULL FK → cbc_sub_strands(id) ON DELETE CASCADE
description TEXT NOT NULL
sequence_order SMALLINT NOT NULL DEFAULT 1

Indexes: idx_performance_indicators_sub_strand
-- Primary access: "give me all indicators for this sub-strand in order"

COMMENT ON TABLE performance_indicators:
'Atomic CBC learning outcomes within a sub-strand, as defined in KICD
curriculum designs. Leaf nodes of the hierarchy:
Learning Area → Strand → Sub-Strand → Performance Indicator.'

================================================================================
LAYER 6 — TEACHER ASSIGNMENTS, ATTENDANCE, TIMETABLE
================================================================================

---

Table: cbc_class_teachers
(FK references updated to cbc_classes and rebuilt cbc_learning_areas)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
class_id UUID NOT NULL
user_id UUID NOT NULL FK → users(id) ON DELETE CASCADE
learning_area_id UUID NOT NULL FK → cbc_learning_areas(id) ON DELETE CASCADE
is_primary BOOLEAN NOT NULL DEFAULT false
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
FK (tenant_id, class_id) → cbc_classes(tenant_id, id) ON DELETE CASCADE
UNIQUE (class_id, user_id, learning_area_id)

Indexes: idx_cbc_class_teachers_tenant, idx_cbc_class_teachers_class_id,
idx_cbc_class_teachers_user_id, idx_cbc_class_teachers_area_id
Partial unique: UNIQUE (class_id, learning_area_id) WHERE is_primary = true
-- One primary teacher per learning area per class

---

Table: cbc_attendance_periods
(FK references updated to cbc_schools, cbc_classes)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
school_id UUID NOT NULL
academic_term_id UUID NOT NULL
class_id UUID NOT NULL
cbc_learning_area_id UUID NOT NULL FK → cbc_learning_areas(id) ON DELETE CASCADE
date_recorded DATE NOT NULL
recorded_by UUID NOT NULL FK → users(id)
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
UNIQUE (tenant_id, id)
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, school_id, academic_term_id)
→ academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
FK (tenant_id, class_id) → cbc_classes(tenant_id, id) ON DELETE CASCADE

Indexes: idx_cbc_att_periods_tenant, idx_cbc_att_periods_class_date
Partial unique: UNIQUE (class_id, date_recorded, cbc_learning_area_id)

---

Table: cbc_attendance_logs
(FK updated to cbc_students)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
cbc_attendance_period_id UUID NOT NULL
student_id UUID NOT NULL
status attendance_status NOT NULL
remarks VARCHAR(255) NULL
recorded_by UUID NOT NULL FK → users(id)

Constraints:
FK (tenant_id, cbc_attendance_period_id)
→ cbc_attendance_periods(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, student_id) → cbc_students(tenant_id, id) ON DELETE CASCADE
UNIQUE (cbc_attendance_period_id, student_id)

Indexes: idx_cbc_att_logs_tenant, idx_cbc_att_logs_period,
idx_cbc_att_logs_student

---

Table: cbc_timetable_slots
(FK references updated to cbc_schools, cbc_classes; GiST constraints kept verbatim)

---

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
school_id UUID NOT NULL
academic_year_id UUID NOT NULL
class_id UUID NOT NULL
teacher_id UUID NOT NULL
cbc_learning_area_id UUID NULL FK → cbc_learning_areas(id) ON DELETE SET NULL
room_identifier VARCHAR(50) NULL
day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7)
start_time TIME NOT NULL
end_time TIME NOT NULL

Constraints:
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, academic_year_id) → academic_years(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, class_id) → cbc_classes(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, teacher_id) → users(tenant_id, id) ON DELETE CASCADE

Indexes: idx_cbc_timetable_tenant, idx_cbc_timetable_school_year,
idx_cbc_timetable_class, idx_cbc_timetable_teacher

GiST Exclusion Constraints (keep verbatim using fn_timerange):
EXCLUDE USING gist (
teacher_id WITH =, academic_year_id WITH =,
fn_timerange(day_of_week, start_time, end_time) WITH &&
) -- prevents double-booking a teacher in overlapping slots

    EXCLUDE USING gist (
      room_identifier WITH =, academic_year_id WITH =,
      fn_timerange(day_of_week, start_time, end_time) WITH &&
    ) -- prevents double-booking a room in overlapping slots

================================================================================
LAYER 7 — CBC ASSESSMENT ARCHITECTURE
================================================================================

---

## Table: assessment_weight_configs ← NEW

Responsibility: Official KNEC-defined contribution weights per grade per
assessment type toward each national placement exam. Replaces the defunct
assessment_types.weight_contribution column. Must be seeded before any
cbc_term_competency_summaries can be correctly computed for KNEC submission.

id UUID PK DEFAULT gen_random_uuid()
grade_level VARCHAR(5) NOT NULL
assessment_type_code VARCHAR(30) NOT NULL
target_exam VARCHAR(20) NOT NULL
weight_percent NUMERIC(5,2) NOT NULL
effective_from SMALLINT NOT NULL
notes TEXT NULL

Constraints:
CHECK (grade_level IN (
'PP1','PP2','G1','G2','G3','G4','G5','G6',
'G7','G8','G9','G10','G11','G12'
))
CHECK (assessment_type_code IN (
'Formative_Classroom', 'KNEC_Written_Assessment', 'KNEC_SBA_Project',
'National_KPSEA', 'National_KJSEA', 'National_KSSEA'
))
CHECK (target_exam IN ('KPSEA', 'KJSEA', 'KSSEA', 'None'))
CHECK (weight_percent BETWEEN 0.00 AND 100.00)
CHECK (effective_from >= 2017)
UNIQUE (grade_level, assessment_type_code, target_exam, effective_from)

Indexes: idx_awc_grade_exam ON (grade_level, target_exam)
-- Aggregation layer always queries: "weights for grade X toward exam Y"

COMMENT ON TABLE assessment_weight_configs:
'Official KNEC weighting formula per grade per assessment type. Seeded with
the published KNEC formula. KPSEA: 60% SBA (G4+G5) + 40% KPSEA written (G6).
KJSEA: 20% SBA (G7+G8) + 20% KPSEA result + 60% KJSEA written (G9).'

---

## Table: assessment_blueprints ← NEW

Responsibility: Defines a specific assessment instrument — its type, target
grade, term, and academic year. Formative_Classroom blueprints are
school-created. KNEC instrument blueprints are seeded system-wide. The `type`
column is the single source of truth for assessment taxonomy — there is no
longer a separate assessment_types lookup table.

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
school_id UUID NOT NULL
title VARCHAR(255) NOT NULL
type VARCHAR(30) NOT NULL
grade_level VARCHAR(5) NOT NULL
academic_year SMALLINT NOT NULL
term SMALLINT NOT NULL

Constraints:
FK (tenant_id, school_id) → cbc_schools(tenant_id, id) ON DELETE CASCADE
CHECK (type IN (
'Formative_Classroom',
'KNEC_Written_Assessment',
'KNEC_SBA_Project',
'National_KPSEA',
'National_KJSEA',
'National_KSSEA'
))
CHECK (grade_level IN (
'PP1','PP2','G1','G2','G3','G4','G5','G6',
'G7','G8','G9','G10','G11','G12'
))
CHECK (term BETWEEN 1 AND 3)
CHECK (academic_year >= 2017)

Indexes: idx_blueprints_tenant, idx_blueprints_school,
idx_blueprints_grade_year ON (grade_level, academic_year, type)
-- Scheduling queries: "which blueprints apply to G5 in 2025 Term 2?"

---

## Table: assessment_blueprint_indicators ← NEW

Responsibility: Junction — links a blueprint to the KICD performance indicators
it covers.

blueprint_id UUID NOT NULL FK → assessment_blueprints(id) ON DELETE CASCADE
indicator_id UUID NOT NULL FK → performance_indicators(id) ON DELETE CASCADE

Constraints:
PRIMARY KEY (blueprint_id, indicator_id)

Indexes: idx_blueprint_indicators_indicator ON (indicator_id)
-- Reverse lookup: "which blueprints cover this indicator?"

================================================================================
LAYER 8 — CBC ASSESSMENT EXECUTION & RESULTS
================================================================================

---

## Table: assessment_sessions ← NEW

Responsibility: Records one execution of a blueprint by a teacher against a
class on a specific date. `date_administered` is DATE (not TIMESTAMPTZ) with
NO DEFAULT — CBC sessions are often recorded retroactively; the date must be
entered deliberately. `knec_upload_reference` is populated after a successful
SBA score upload to cba.knec.ac.ke; it is NULL for Formative_Classroom sessions.

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
blueprint_id UUID NOT NULL FK → assessment_blueprints(id) ON DELETE RESTRICT
class_id UUID NOT NULL
assessed_by_user_id UUID NOT NULL FK → users(id) ON DELETE RESTRICT
date_administered DATE NOT NULL -- NO DEFAULT. Must be entered explicitly.
knec_upload_reference VARCHAR(50) NULL
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
FK (tenant_id, class_id) → cbc_classes(tenant_id, id) ON DELETE RESTRICT

Indexes: idx_asessions_tenant, idx_asessions_blueprint, idx_asessions_class,
idx_asessions_teacher, idx_asessions_class_date ON (class_id, date_administered)
-- Teacher dashboard: sessions for my class ordered by date

COMMENT ON COLUMN assessment_sessions.date_administered:
'The calendar date on which this assessment was administered. DATE type
(not TIMESTAMPTZ) because CBC records reference dates, not timestamps.
No DEFAULT: must be set explicitly. Retroactive entry is common in CBC
as teachers often batch-enter assessments at end of week.'

COMMENT ON COLUMN assessment_sessions.knec_upload_reference:
'Reference token returned by cba.knec.ac.ke after a successful SBA score
upload. NULL for Formative_Classroom type sessions, which are never
uploaded to KNEC.'

---

## Table: learner_rubric_results ← NEW

Responsibility: Atomic CBC assessment record. One row = one learner's rubric
outcome for one performance indicator in one session. `rubric_level` is
constrained to the official KNEC 4-level rubric only — no sub-levels anywhere.
`raw_score` is only present when score_type = 'Numeric_Raw'; it holds the
pre-conversion mark and is NEVER summed, averaged, or used as a standalone grade.

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
session_id UUID NOT NULL FK → assessment_sessions(id) ON DELETE CASCADE
student_id UUID NOT NULL
indicator_id UUID NOT NULL FK → performance_indicators(id) ON DELETE RESTRICT
score_type VARCHAR(20) NOT NULL
raw_score NUMERIC(5,2) NULL
rubric_level VARCHAR(2) NOT NULL
teacher_observation_notes TEXT NULL

Constraints:
FK (tenant_id, student_id) → cbc_students(tenant_id, id) ON DELETE CASCADE
CHECK (score_type IN ('Numeric_Raw', 'Rubric_Direct'))
CHECK (rubric_level IN ('EE', 'ME', 'AE', 'BE'))
UNIQUE (session_id, student_id, indicator_id)
-- One result per learner per indicator per session

Indexes:
idx_lrr_tenant
idx_lrr_session
idx_lrr_student_indicator ON (student_id, indicator_id) ← HIGH PRIORITY
-- Accelerates the most critical query: single-student competency trace
-- across all sessions — "all results for student X across all indicators"
idx_lrr_indicator

COMMENT ON COLUMN learner_rubric_results.rubric_level:
'Official KNEC 4-level rubric outcome. EE/ME/AE/BE only. No sub-levels
(EE1, ME2 etc.) are permitted here. Sub-levels may exist in internal
school tooling but are not valid in KNEC portal submissions.'

COMMENT ON COLUMN learner_rubric_results.raw_score:
'Pre-conversion numeric mark. Only populated when score_type = Numeric_Raw.
Represents the raw score before it is mapped to a rubric level. NEVER
summed or averaged across indicators — doing so would constitute a CBC
compliance violation.'

---

## Table: learner_portfolios ← NEW

Responsibility: Evidence artifacts for a learner's competency demonstration.
Maintaining learner portfolios is a formal KICD/KNEC assessment requirement.
Covers physical files, digital uploads, video recordings (Creative Arts, PE,
oral presentations), audio logs, and observation checklists (SNE learners).

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
student_id UUID NOT NULL
sub_strand_id UUID NOT NULL FK → cbc_sub_strands(id) ON DELETE RESTRICT
evidence_type VARCHAR(30) NOT NULL
storage_pointer TEXT NOT NULL
linked_result_id UUID NULL FK → learner_rubric_results(id) ON DELETE SET NULL
date_collected DATE NULL
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

Constraints:
FK (tenant_id, student_id) → cbc_students(tenant_id, id) ON DELETE CASCADE
CHECK (evidence_type IN (
'Physical_File_Reference', -- Binder, notebook, artwork held at school
'Digital_Artifact_URL', -- Uploaded document, image, or project file
'Video_Recording', -- Performance evidence: Creative Arts, PE, oral work
'Audio_Log', -- Recorded oral reading, speech, or group discussion
'Observation_Checklist' -- Teacher-completed form; especially for Stage-Based SNE
))

Indexes: idx_portfolios_tenant, idx_portfolios_student, idx_portfolios_sub_strand,
idx_portfolios_result

COMMENT ON COLUMN learner_portfolios.storage_pointer:
'For Digital_Artifact_URL and Video_Recording: full URL to stored file.
For Physical_File_Reference: descriptive location string
(e.g. "Portfolio Binder 2B, page 14, Teacher: J. Mwangi").'

================================================================================
LAYER 9 — CBC AGGREGATION & REPORTING
================================================================================

---

## Table: cbc_term_competency_summaries ← NEW

Responsibility: Definitive per-term competency record per learner per learning
area. Feeds the KNEC CBA portal via knec_sync_status.

CRITICAL COLUMN RULES:
calculated_level — computed by the system from weighted rubric results.
May use school-internal sub-levels (EE1, ME2 etc.) if the school tracks
finer granularity. This is an INTERNAL value.
override_level — entered by class teacher or HOD to override the system
calculation. Also may use sub-levels for internal tracking.
final_level — the value submitted to cba.knec.ac.ke. MUST be one of
the standard 4 KNEC levels (EE/ME/AE/BE) ONLY. KNEC portal REJECTS
sub-levels. Violation causes sync failures. This constraint is STRICT.

id UUID PK DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL
student_id UUID NOT NULL
learning_area_id UUID NOT NULL FK → cbc_learning_areas(id) ON DELETE RESTRICT
class_id UUID NOT NULL
academic_year SMALLINT NOT NULL
term SMALLINT NOT NULL
calculated_level VARCHAR(3) NOT NULL
override_level VARCHAR(3) NULL
final_level VARCHAR(2) NOT NULL
knec_sync_status VARCHAR(20) NOT NULL DEFAULT 'Pending'
knec_synced_at TIMESTAMPTZ NULL

Constraints:
FK (tenant_id, student_id) → cbc_students(tenant_id, id) ON DELETE CASCADE
FK (tenant_id, class_id) → cbc_classes(tenant_id, id) ON DELETE RESTRICT
CHECK (calculated_level IN (
'EE','ME','AE','BE',
'EE1','EE2','ME1','ME2','AE1','AE2','BE1','BE2'
))
CHECK (override_level IN (
'EE','ME','AE','BE',
'EE1','EE2','ME1','ME2','AE1','AE2','BE1','BE2'
))
CHECK (final_level IN ('EE','ME','AE','BE'))
-- STRICT: KNEC portal accepts ONLY these 4 values. No exceptions.
CHECK (knec_sync_status IN ('Pending','Synced','Failed'))
CHECK (term BETWEEN 1 AND 3)
CHECK (academic_year >= 2017)
UNIQUE (student_id, learning_area_id, academic_year, term)
-- One definitive record per learner per subject per term

Indexes:
idx_summaries_tenant
idx_summaries_sync_batch ON (academic_year, term, knec_sync_status)
-- KNEC sync worker: finds all Pending/Failed records for a given term
-- without full table scans
idx_summaries_student_year ON (student_id, academic_year)
-- Student progress report: all areas for a student across a year
idx_summaries_class
-- Teacher class view: all students' competency levels for a class

COMMENT ON TABLE cbc_term_competency_summaries:
'Definitive per-term competency record per learner per learning area.
final_level is the KNEC portal submission value — must always be one of
EE/ME/AE/BE. Sub-levels (EE1 etc.) are only valid for the internal
calculated_level and override_level fields. knec_synced_at is NULL until
the first successful upload to cba.knec.ac.ke.'

---

## MIGRATION FILE INSTRUCTIONS

---

## 000001_initial_schema.up.sql

Structure:

1. Header comment block:
   -- Migration: 000001_initial_schema
   -- SomoTracker — Kenya CBC/CBE academic platform (CBC-only, v5)
   -- Drops all generic education system abstractions.
   -- Rebuilds as a purpose-built, single-system CBC schema.

2. Wrap everything in BEGIN; ... COMMIT;

3. Extensions (btree_gist)

4. Functions (fn_set_updated_at, fn_timerange — keep verbatim)

5. Drop legacy enums before creating replacements:
   DROP TYPE IF EXISTS gender_type CASCADE;
   DROP TYPE IF EXISTS enrollment_status CASCADE;

6. Create enums (user_role, attendance_status, invitation_status unchanged;
   cbc_enrollment_status as new replacement)

7. Tables in strict FK dependency order as specified in Layers 0–9 above.

8. COMMENT ON TABLE for every CBC-domain table.

9. COMMENT ON COLUMN for all government identifier fields.

---

## 000001_initial_schema.down.sql

Drop in strict reverse FK dependency order:

1. Reporting: cbc_term_competency_summaries
2. Results: learner_portfolios, learner_rubric_results, assessment_sessions
3. Assessment: assessment_blueprint_indicators, assessment_blueprints,
   assessment_weight_configs
4. Curriculum: performance_indicators, cbc_sub_strands, cbc_strands,
   cbc_learning_areas
5. Operations: cbc_timetable_slots, cbc_attendance_logs,
   cbc_attendance_periods, cbc_class_teachers
6. Finance: payments, invoice_items, invoices, fee_templates, fee_categories
7. Health: medical_incidents, student_health_profiles
8. Calendar: academic_terms, academic_years
9. Actors: cbc_student_enrollments, cbc_students, cbc_classes, cbc_schools
10. Platform: memberships, invitations, sessions, users, tenants
11. Drop enums: cbc_enrollment_status, invitation_status, attendance_status, user_role
12. Drop functions: fn_timerange, fn_set_updated_at
13. Drop extension: btree_gist

Wrap in BEGIN; ... COMMIT;

---

## 000002_seed.up.sql — FULL REPLACEMENT

The existing seed file must be COMPLETELY REPLACED. Remove everything.

REQUIRED seed data:

A) assessment_weight_configs — 7 rows (official KNEC formula):
('G4', 'KNEC_SBA_Project', 'KPSEA', 60.00, 2023, 'G4 SBA → 60% of KPSEA placement')
('G5', 'KNEC_SBA_Project', 'KPSEA', 60.00, 2023, 'G5 SBA → 60% of KPSEA placement')
('G6', 'National_KPSEA', 'KPSEA', 40.00, 2023, 'KPSEA written → 40% of KPSEA placement')
('G7', 'KNEC_SBA_Project', 'KJSEA', 20.00, 2024, 'G7 SBA → 20% of KJSEA placement')
('G8', 'KNEC_SBA_Project', 'KJSEA', 20.00, 2024, 'G8 SBA → 20% of KJSEA placement')
('G6', 'National_KPSEA', 'KJSEA', 20.00, 2024, 'KPSEA result → 20% of KJSEA placement')
('G9', 'National_KJSEA', 'KJSEA', 60.00, 2024, 'KJSEA written → 60% of KJSEA placement')

B) One sample tenant row (for dev/demo environment)

C) One sample cbc_schools row:
knec_school_code = '12345678' (8-digit numeric format)
county = 'Nairobi', sub_county = 'Westlands'
school_type = 'Public'

No other seed data is required in this file. Learning areas, strands, sub-strands,
and performance indicators are application-managed data, not migration seeds.

---

## 000002_seed.down.sql

DELETE FROM assessment_weight_configs;
DELETE FROM cbc_schools;
DELETE FROM tenants;
(in reverse insert order, wrapped in BEGIN; ... COMMIT;)

---

## FINAL CHECKLIST

DELETED TABLES (must not appear anywhere in output):
[ ] education_systems — gone
[ ] curriculum_stages — gone
[ ] assessment_grade_scales — gone
[ ] assessment_types — gone
[ ] grades — gone
[ ] schools — gone (replaced by cbc_schools)
[ ] classes — gone (replaced by cbc_classes)
[ ] students — gone (replaced by cbc_students)
[ ] student_enrollments — gone (replaced by cbc_student_enrollments)

DELETED ENUMS (must not appear anywhere in output):
[ ] gender_type — gone
[ ] enrollment_status — gone

CBC COMPLIANCE:
[ ] No 'total_marks', 'average', 'percentage', 'rank', 'points', 'mean_grade' columns
[ ] No KCPE, KCSE, IGCSE, IB, or 8-4-4 references anywhere
[ ] gender is CHAR(1) CHECK IN ('M','F') — no enum
[ ] cbc_enrollment_status has ACTIVE/SUSPENDED/TRANSFERRED/COMPLETED_CYCLE only
[ ] rubric_level CHECK IN ('EE','ME','AE','BE') — no sub-levels on results
[ ] final_level CHECK IN ('EE','ME','AE','BE') — strict, KNEC portal constraint
[ ] calculated_level and override_level permit sub-levels (EE1 etc.) for internal use
[ ] assessment_blueprints.type uses only the 6-value CBC enum
[ ] grade_level CHECK covers PP1,PP2,G1–G12 on every table that carries it
[ ] education_level CHECK covers Early_Years/Upper_Primary/Junior_Secondary/Senior_School
[ ] academic_year CHECK (>= 2017) on every table that carries it
[ ] term CHECK BETWEEN 1 AND 3 on every table that carries it
[ ] assessment_sessions.date_administered is DATE, NOT NULL, NO DEFAULT

MULTI-TENANCY:
[ ] Every school-scoped table carries tenant_id NOT NULL
[ ] Every table in the FK chain has UNIQUE (tenant_id, id) composite UK
[ ] Composite FKs (tenant_id, x_id) used consistently throughout
[ ] memberships and invitations FK to cbc_schools (not the deleted schools table)
[ ] health, finance, attendance tables FK to cbc_students (not deleted students table)

GOVERNMENT IDENTIFIERS (COMMENT ON COLUMN required):
[ ] cbc_schools.knec_school_code
[ ] cbc_schools.nemis_institution_code
[ ] cbc_students.upi_number
[ ] cbc_students.knec_assessment_number
[ ] cbc_students.learning_pathway
[ ] cbc_students.gender
[ ] users.tsc_number
[ ] users.knec_panel_assessor_id

INDEXING:
[ ] All FK columns have explicit B-Tree indexes
[ ] Composite indexes on columns always filtered together
[ ] Partial unique indexes used for sparse unique constraints (knec_school_code,
upi_number, knec_assessment_number, tsc_number, knec_panel_assessor_id)
[ ] GiST exclusion constraints on cbc_timetable_slots kept verbatim

SEED:
[ ] assessment_weight_configs seeded with all 7 KNEC formula rows
[ ] No legacy subjects, numeric grades, mean grades, or KCPE/KCSE records

================================================================================
END OF SPECIFICATION
================================================================================
