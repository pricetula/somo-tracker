-- ================================================================================
-- 💾 FULL_MODELS_ENHANCED.txt — MULTI-TENANT ARCHITECTURE SCHEMA CONFIGURATION
-- ================================================================================

-- =============================================================================
-- UTILITY: auto-updating updated_at trigger
-- Must be created first — referenced by multiple tables below.
-- =============================================================================

CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- SECTION 1: GLOBAL STRUCTURAL FOUNDATION (SHARED MULTI-TENANT CORE)
-- =============================================================================

CREATE TYPE user_role AS ENUM ('SYSTEM_ADMIN', 'SCHOOL_ADMIN', 'TEACHER', 'SUPPORT_STAFF');
CREATE TYPE enrollment_status AS ENUM ('ACTIVE', 'SUSPENDED', 'TRANSFERRED', 'GRADUATED');
CREATE TYPE attendance_status AS ENUM ('PRESENT', 'ABSENT', 'LATE', 'EXCUSED');
CREATE TYPE gender_type AS ENUM ('MALE', 'FEMALE', 'OTHER', 'PREFER_NOT_TO_SAY');
-- CBC: Exceeds Expectation, Meets Expectation, Approaches Expectation, Below Expectation
CREATE TYPE cbc_score_level AS ENUM ('EE', 'ME', 'AE', 'BE');
-- IB MYP: Criteria A–D (one score per criterion per task per student)
CREATE TYPE ib_criterion_type AS ENUM ('A', 'B', 'C', 'D');
CREATE TYPE assessment_type AS ENUM ('CAT', 'MID_TERM', 'END_TERM', 'MOCK', 'OTHER');

-- -----------------------------------------------------------------------------
CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- -----------------------------------------------------------------------------
CREATE TABLE users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email            VARCHAR(255) UNIQUE NOT NULL,
    tenant_id        UUID REFERENCES tenants(id) ON DELETE CASCADE, -- NULL for SYSTEM_ADMIN
    first_name       VARCHAR(100) NOT NULL,
    last_name        VARCHAR(100) NOT NULL,
    is_active        BOOLEAN NOT NULL DEFAULT true,
    external_auth_id VARCHAR(100) UNIQUE NOT NULL, -- Stytch Mapping Key
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- -----------------------------------------------------------------------------
CREATE TABLE education_systems (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(100) NOT NULL,                    -- "Kenya CBC", "Cambridge IGCSE", "IB MYP"
    country_code CHAR(2) NOT NULL
        CONSTRAINT chk_country_code_format CHECK (country_code ~ '^[A-Z]{2}$')
);

-- -----------------------------------------------------------------------------
CREATE TABLE schools (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    education_system_id  UUID NOT NULL REFERENCES education_systems(id),
    name                 VARCHAR(255) NOT NULL,
    is_active            BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX idx_schools_tenant_id ON schools(tenant_id);
CREATE INDEX idx_schools_education_system_id ON schools(education_system_id);

-- -----------------------------------------------------------------------------
CREATE TABLE memberships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role       user_role NOT NULL,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id  UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_user_school_membership UNIQUE (user_id, school_id)
);
CREATE INDEX idx_memberships_user_id ON memberships(user_id);
CREATE INDEX idx_memberships_school_id ON memberships(school_id);

-- -----------------------------------------------------------------------------
CREATE TABLE academic_years (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id  UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name       VARCHAR(50) NOT NULL,
    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_academic_years_school_id ON academic_years(school_id);
CREATE UNIQUE INDEX idx_one_current_year_per_school
    ON academic_years(school_id)
    WHERE is_current = true;

-- -----------------------------------------------------------------------------
CREATE TABLE academic_terms (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    name             VARCHAR(100) NOT NULL,
    start_date       DATE NOT NULL,
    end_date         DATE NOT NULL,
    is_current       BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_academic_terms_year_id ON academic_terms(academic_year_id);
CREATE UNIQUE INDEX idx_one_current_term_per_year
    ON academic_terms(academic_year_id)
    WHERE is_current = true;

-- -----------------------------------------------------------------------------
CREATE TABLE grades (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE CASCADE,
    name                VARCHAR(100) NOT NULL,  -- "Grade 7", "Year 11", "MYP 4"
    sequence_order      INT NOT NULL
);
CREATE INDEX idx_grades_education_system_id ON grades(education_system_id);

-- -----------------------------------------------------------------------------
CREATE TABLE classes (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id        UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    grade_id         UUID NOT NULL REFERENCES grades(id),
    name             VARCHAR(100) NOT NULL,  -- "East", "Alpha"
    is_active        BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX idx_classes_school_id ON classes(school_id);
CREATE INDEX idx_classes_academic_year_id ON classes(academic_year_id);
CREATE INDEX idx_classes_grade_id ON classes(grade_id);

-- -----------------------------------------------------------------------------
CREATE TABLE students (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    first_name  VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    last_name   VARCHAR(100) NOT NULL,
    gender      gender_type NOT NULL,
    date_of_birth DATE NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_students_tenant_id ON students(tenant_id);

-- -----------------------------------------------------------------------------
CREATE TABLE student_enrollments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id       UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id        UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_term_id UUID NOT NULL REFERENCES academic_terms(id) ON DELETE CASCADE,
    class_id         UUID REFERENCES classes(id) ON DELETE SET NULL,
    status           enrollment_status NOT NULL DEFAULT 'ACTIVE',
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_active_term_enrollment UNIQUE (student_id, academic_term_id)
);
CREATE INDEX idx_enrollments_student_id ON student_enrollments(student_id);
CREATE INDEX idx_enrollments_school_id ON student_enrollments(school_id);
CREATE INDEX idx_enrollments_term_id ON student_enrollments(academic_term_id);
CREATE INDEX idx_enrollments_class_id ON student_enrollments(class_id);

-- -----------------------------------------------------------------------------

-- =============================================================================
-- SECTION 2: KENYA COMPETENCY-BASED CURRICULUM MODULE (`cbc_`)
-- =============================================================================

CREATE TABLE cbc_learning_areas (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id  UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    grade_id   UUID NOT NULL REFERENCES grades(id) ON DELETE CASCADE,
    name       VARCHAR(150) NOT NULL,
    code       VARCHAR(50) NOT NULL
);
CREATE INDEX idx_cbc_learning_areas_school_id ON cbc_learning_areas(school_id);
CREATE INDEX idx_cbc_learning_areas_grade_id ON cbc_learning_areas(grade_id);

-- -----------------------------------------------------------------------------
CREATE TABLE cbc_strands (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    learning_area_id UUID NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL
);
CREATE INDEX idx_cbc_strands_learning_area_id ON cbc_strands(learning_area_id);

-- -----------------------------------------------------------------------------
CREATE TABLE cbc_sub_strands (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strand_id UUID NOT NULL REFERENCES cbc_strands(id) ON DELETE CASCADE,
    name      VARCHAR(255) NOT NULL
);
CREATE INDEX idx_cbc_sub_strands_strand_id ON cbc_sub_strands(strand_id);

-- -----------------------------------------------------------------------------
CREATE TABLE cbc_learning_outcomes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sub_strand_id UUID NOT NULL REFERENCES cbc_sub_strands(id) ON DELETE CASCADE,
    description   TEXT NOT NULL
);
CREATE INDEX idx_cbc_learning_outcomes_sub_strand_id ON cbc_learning_outcomes(sub_strand_id);

-- -----------------------------------------------------------------------------
CREATE TABLE cbc_formative_tasks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id            UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    academic_term_id    UUID NOT NULL REFERENCES academic_terms(id) ON DELETE CASCADE,
    learning_outcome_id UUID NOT NULL REFERENCES cbc_learning_outcomes(id) ON DELETE CASCADE,
    title               VARCHAR(255) NOT NULL,
    assessment_type     assessment_type NOT NULL DEFAULT 'OTHER',
    date_administered   DATE NOT NULL,
    created_by          UUID NOT NULL REFERENCES users(id)
);
CREATE INDEX idx_cbc_formative_tasks_class_id ON cbc_formative_tasks(class_id);
CREATE INDEX idx_cbc_formative_tasks_term_id ON cbc_formative_tasks(academic_term_id);
CREATE INDEX idx_cbc_formative_tasks_outcome_id ON cbc_formative_tasks(learning_outcome_id);

-- -----------------------------------------------------------------------------
CREATE TABLE cbc_task_evaluations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    formative_task_id UUID NOT NULL REFERENCES cbc_formative_tasks(id) ON DELETE CASCADE,
    student_id        UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    graded_by_user_id UUID NOT NULL REFERENCES users(id),
    score_level       cbc_score_level NOT NULL,
    teacher_remarks   TEXT,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_cbc_student_task UNIQUE (formative_task_id, student_id)
);
CREATE INDEX idx_cbc_task_evals_task_id ON cbc_task_evaluations(formative_task_id);
CREATE INDEX idx_cbc_task_evals_student_id ON cbc_task_evaluations(student_id);
CREATE TRIGGER trg_cbc_task_evaluations_updated_at
    BEFORE UPDATE ON cbc_task_evaluations
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- -----------------------------------------------------------------------------
CREATE TABLE cbc_class_teachers (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id             UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    learning_area_id     UUID NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    is_primary           BOOLEAN NOT NULL DEFAULT false,
    created_at           TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_cbc_class_teacher UNIQUE (class_id, user_id, learning_area_id)
);
CREATE INDEX idx_cbc_class_teachers_class_id ON cbc_class_teachers(class_id);
CREATE INDEX idx_cbc_class_teachers_user_id ON cbc_class_teachers(user_id);
CREATE INDEX idx_cbc_class_teachers_area_id ON cbc_class_teachers(learning_area_id);
-- One primary teacher per learning area per class
CREATE UNIQUE INDEX idx_cbc_one_primary_per_area
    ON cbc_class_teachers(class_id, learning_area_id)
    WHERE is_primary = true;

-- =============================================================================
-- SECTION 3: CAMBRIDGE INTERNATIONAL CURRICULUM MODULE (`igcse_`)
-- =============================================================================

CREATE TABLE igcse_subjects (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    grade_id  UUID NOT NULL REFERENCES grades(id) ON DELETE CASCADE,
    name      VARCHAR(150) NOT NULL,
    code      VARCHAR(20) NOT NULL  -- e.g. "0620" for Chemistry
);
CREATE INDEX idx_igcse_subjects_school_id ON igcse_subjects(school_id);
CREATE INDEX idx_igcse_subjects_grade_id ON igcse_subjects(grade_id);

-- -----------------------------------------------------------------------------
CREATE TABLE igcse_papers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id        UUID NOT NULL REFERENCES igcse_subjects(id) ON DELETE CASCADE,
    paper_number      VARCHAR(10) NOT NULL,    -- "Paper 2", "Paper 4"
    max_raw_mark      NUMERIC(5,2) NOT NULL,
    weight_percentage NUMERIC(5,2) NOT NULL
);
CREATE INDEX idx_igcse_papers_subject_id ON igcse_papers(subject_id);

-- -----------------------------------------------------------------------------
CREATE TABLE igcse_class_assessments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    paper_id         UUID NOT NULL REFERENCES igcse_papers(id) ON DELETE CASCADE,
    class_id         UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    academic_term_id UUID NOT NULL REFERENCES academic_terms(id) ON DELETE CASCADE,
    title            VARCHAR(255) NOT NULL,
    assessment_type  assessment_type NOT NULL DEFAULT 'OTHER',
    examination_date DATE NOT NULL,
    created_by       UUID NOT NULL REFERENCES users(id)
);
CREATE INDEX idx_igcse_assessments_paper_id ON igcse_class_assessments(paper_id);
CREATE INDEX idx_igcse_assessments_class_id ON igcse_class_assessments(class_id);
CREATE INDEX idx_igcse_assessments_term_id ON igcse_class_assessments(academic_term_id);

-- -----------------------------------------------------------------------------
CREATE TABLE igcse_assessment_marks (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id     UUID NOT NULL REFERENCES igcse_class_assessments(id) ON DELETE CASCADE,
    student_id        UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    graded_by_user_id UUID NOT NULL REFERENCES users(id),
    raw_score_earned  NUMERIC(5,2) NOT NULL,
    teacher_remarks   TEXT,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_igcse_student_assessment UNIQUE (assessment_id, student_id)
);
CREATE INDEX idx_igcse_marks_assessment_id ON igcse_assessment_marks(assessment_id);
CREATE INDEX idx_igcse_marks_student_id ON igcse_assessment_marks(student_id);
CREATE TRIGGER trg_igcse_assessment_marks_updated_at
    BEFORE UPDATE ON igcse_assessment_marks
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- -----------------------------------------------------------------------------
CREATE TABLE igcse_class_teachers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id     UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id   UUID NOT NULL REFERENCES igcse_subjects(id) ON DELETE CASCADE,
    is_primary   BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_igcse_class_teacher UNIQUE (class_id, user_id, subject_id)
);
CREATE INDEX idx_igcse_class_teachers_class_id ON igcse_class_teachers(class_id);
CREATE INDEX idx_igcse_class_teachers_user_id ON igcse_class_teachers(user_id);
CREATE INDEX idx_igcse_class_teachers_subject_id ON igcse_class_teachers(subject_id);
-- One primary teacher per subject per class
CREATE UNIQUE INDEX idx_igcse_one_primary_per_subject
    ON igcse_class_teachers(class_id, subject_id)
    WHERE is_primary = true;

-- =============================================================================
-- SECTION 4: INTERNATIONAL BACCALAUREATE MODULE (`ib_`)
-- =============================================================================

CREATE TABLE ib_subject_groups (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id  UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    group_name VARCHAR(100) NOT NULL  -- "Sciences", "Mathematics", "Language Acquisition"
);
CREATE INDEX idx_ib_subject_groups_school_id ON ib_subject_groups(school_id);

-- -----------------------------------------------------------------------------
CREATE TABLE ib_disciplines (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES ib_subject_groups(id) ON DELETE CASCADE,
    grade_id UUID NOT NULL REFERENCES grades(id) ON DELETE CASCADE,
    name     VARCHAR(150) NOT NULL  -- "Physics", "Chemistry"
);
CREATE INDEX idx_ib_disciplines_group_id ON ib_disciplines(group_id);
CREATE INDEX idx_ib_disciplines_grade_id ON ib_disciplines(grade_id);

-- -----------------------------------------------------------------------------
CREATE TABLE ib_tasks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    discipline_id  UUID NOT NULL REFERENCES ib_disciplines(id) ON DELETE CASCADE,
    class_id       UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    academic_term_id UUID NOT NULL REFERENCES academic_terms(id) ON DELETE CASCADE,
    title          VARCHAR(255) NOT NULL,
    assessment_type  assessment_type NOT NULL DEFAULT 'OTHER',
    execution_date DATE NOT NULL,
    created_by     UUID NOT NULL REFERENCES users(id)
);
CREATE INDEX idx_ib_tasks_discipline_id ON ib_tasks(discipline_id);
CREATE INDEX idx_ib_tasks_class_id ON ib_tasks(class_id);
CREATE INDEX idx_ib_tasks_term_id ON ib_tasks(academic_term_id);

-- -----------------------------------------------------------------------------
CREATE TABLE ib_task_criterion_scores (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id           UUID NOT NULL REFERENCES ib_tasks(id) ON DELETE CASCADE,
    student_id        UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    graded_by_user_id UUID NOT NULL REFERENCES users(id),
    --          now replaced entirely by the ENUM type
    criterion_type    ib_criterion_type NOT NULL,
    achievement_level INT NOT NULL,
    teacher_remarks   TEXT,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_ib_achievement_level_bounds CHECK (achievement_level BETWEEN 0 AND 8),
    CONSTRAINT unique_ib_student_task_criterion UNIQUE (task_id, student_id, criterion_type)
);
CREATE INDEX idx_ib_criterion_scores_task_id ON ib_task_criterion_scores(task_id);
CREATE INDEX idx_ib_criterion_scores_student_id ON ib_task_criterion_scores(student_id);
CREATE TRIGGER trg_ib_task_criterion_scores_updated_at
    BEFORE UPDATE ON ib_task_criterion_scores
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- -----------------------------------------------------------------------------
CREATE TABLE ib_class_teachers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    class_id      UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    discipline_id UUID NOT NULL REFERENCES ib_disciplines(id) ON DELETE CASCADE,
    is_primary    BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_ib_class_teacher UNIQUE (class_id, user_id, discipline_id)
);
CREATE INDEX idx_ib_class_teachers_class_id ON ib_class_teachers(class_id);
CREATE INDEX idx_ib_class_teachers_user_id ON ib_class_teachers(user_id);
CREATE INDEX idx_ib_class_teachers_discipline_id ON ib_class_teachers(discipline_id);
-- One primary teacher per discipline per class
CREATE UNIQUE INDEX idx_ib_one_primary_per_discipline
    ON ib_class_teachers(class_id, discipline_id)
    WHERE is_primary = true;

-- =============================================================================
-- SECTION 4b: ASSESSMENT WEIGHTS
-- =============================================================================
-- School-wide per-grade weighting rules. Weights apply equally across all
-- subjects within a grade — no per-subject overrides.
-- The three weights (CAT + MID_TERM + END_TERM) must sum to 100; this is
-- enforced at the application layer since SQL cannot easily aggregate-check
-- across rows in a single constraint.

CREATE TABLE assessment_weights (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id        UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    grade_id         UUID NOT NULL REFERENCES grades(id) ON DELETE CASCADE,
    academic_term_id UUID NOT NULL REFERENCES academic_terms(id) ON DELETE CASCADE,
    assessment_type  assessment_type NOT NULL,
    weight_percent   NUMERIC(5,2) NOT NULL CHECK (weight_percent > 0 AND weight_percent <= 100),
    CONSTRAINT unique_weight_rule UNIQUE (school_id, grade_id, academic_term_id, assessment_type)
);

CREATE INDEX idx_assessment_weights_school_grade ON assessment_weights(school_id, grade_id);
CREATE INDEX idx_assessment_weights_term_id ON assessment_weights(academic_term_id);

-- =============================================================================
-- SECTION 5: UNIFIED CORE SYSTEMS (ATTENDANCE, HEALTH, FINANCES)
-- =============================================================================

-- ====================================================================
-- UNIFIED ATTENDANCE SYSTEM
-- ====================================================================

CREATE TABLE attendance_periods (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id            UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_term_id     UUID NOT NULL REFERENCES academic_terms(id) ON DELETE CASCADE,
    class_id             UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    cbc_learning_area_id UUID REFERENCES cbc_learning_areas(id) ON DELETE SET NULL,
    igcse_subject_id     UUID REFERENCES igcse_subjects(id) ON DELETE SET NULL,
    ib_discipline_id     UUID REFERENCES ib_disciplines(id) ON DELETE SET NULL,
    date_recorded        DATE NOT NULL,
    CONSTRAINT chk_single_curriculum_subject_link CHECK (
        (cbc_learning_area_id IS NOT NULL AND igcse_subject_id IS NULL AND ib_discipline_id IS NULL) OR
        (cbc_learning_area_id IS NULL AND igcse_subject_id IS NOT NULL AND ib_discipline_id IS NULL) OR
        (cbc_learning_area_id IS NULL AND igcse_subject_id IS NULL AND ib_discipline_id IS NOT NULL)
    )
);
CREATE INDEX idx_att_periods_class_date ON attendance_periods(class_id, date_recorded);
CREATE INDEX idx_att_periods_term_id ON attendance_periods(academic_term_id);
CREATE INDEX idx_att_periods_school_id ON attendance_periods(school_id);
--         COALESCE to a sentinel UUID handles the nullable curriculum columns in the UNIQUE key.
CREATE UNIQUE INDEX idx_unique_attendance_period
    ON attendance_periods (
        class_id,
        date_recorded,
        COALESCE(cbc_learning_area_id, '00000000-0000-0000-0000-000000000000'::uuid),
        COALESCE(igcse_subject_id,     '00000000-0000-0000-0000-000000000000'::uuid),
        COALESCE(ib_discipline_id,     '00000000-0000-0000-0000-000000000000'::uuid)
    );

-- -----------------------------------------------------------------------------
CREATE TABLE attendance_logs (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attendance_period_id UUID NOT NULL REFERENCES attendance_periods(id) ON DELETE CASCADE,
    student_id           UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    status               attendance_status NOT NULL,
    remarks              VARCHAR(255),
    recorded_by          UUID NOT NULL REFERENCES users(id),
    CONSTRAINT unique_student_attendance_period UNIQUE (attendance_period_id, student_id)
);
CREATE INDEX idx_att_logs_period_id ON attendance_logs(attendance_period_id);
CREATE INDEX idx_att_logs_student_id ON attendance_logs(student_id);

-- ====================================================================
-- UNIFIED TIMETABLE SYSTEM
-- ====================================================================

CREATE TABLE timetable_slots (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id            UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    --         timetable history and preventing stale slots carrying into new years
    academic_year_id     UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    class_id             UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    teacher_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cbc_learning_area_id UUID REFERENCES cbc_learning_areas(id) ON DELETE SET NULL,
    igcse_subject_id     UUID REFERENCES igcse_subjects(id) ON DELETE SET NULL,
    ib_discipline_id     UUID REFERENCES ib_disciplines(id) ON DELETE SET NULL,
    room_identifier      VARCHAR(50),
    day_of_week          INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time           TIME NOT NULL,
    end_time             TIME NOT NULL,
    CONSTRAINT chk_timetable_curriculum_link CHECK (
        (cbc_learning_area_id IS NOT NULL AND igcse_subject_id IS NULL AND ib_discipline_id IS NULL) OR
        (cbc_learning_area_id IS NULL AND igcse_subject_id IS NOT NULL AND ib_discipline_id IS NULL) OR
        (cbc_learning_area_id IS NULL AND igcse_subject_id IS NULL AND ib_discipline_id IS NOT NULL) OR
        (cbc_learning_area_id IS NULL AND igcse_subject_id IS NULL AND ib_discipline_id IS NULL)  -- breaks/assemblies
    )
);
CREATE INDEX idx_timetable_school_year ON timetable_slots(school_id, academic_year_id);
CREATE INDEX idx_timetable_class_id ON timetable_slots(class_id);
CREATE INDEX idx_timetable_teacher_id ON timetable_slots(teacher_id);

-- ====================================================================
-- UNIFIED HEALTH SYSTEM
-- ====================================================================

CREATE TABLE student_health_profiles (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id            UUID UNIQUE NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    blood_group           VARCHAR(5),
    allergies             TEXT[],
    chronic_conditions    TEXT[],
    emergency_instructions TEXT
);

-- -----------------------------------------------------------------------------
CREATE TABLE medical_incidents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id          UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    incident_timestamp  TIMESTAMP WITH TIME ZONE NOT NULL,
    symptoms            TEXT NOT NULL,
    action_taken        TEXT NOT NULL,
    logged_by           UUID NOT NULL REFERENCES users(id)
);
CREATE INDEX idx_medical_incidents_student_id ON medical_incidents(student_id);

-- ====================================================================
-- UNIFIED FINANCIAL SYSTEM
-- ====================================================================

CREATE TABLE fee_categories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id    UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    name         VARCHAR(150) NOT NULL,
    is_mandatory BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX idx_fee_categories_school_id ON fee_categories(school_id);

-- -----------------------------------------------------------------------------
CREATE TABLE fee_templates (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id        UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    academic_term_id UUID NOT NULL REFERENCES academic_terms(id) ON DELETE CASCADE,
    grade_id         UUID NOT NULL REFERENCES grades(id) ON DELETE CASCADE,
    fee_category_id  UUID NOT NULL REFERENCES fee_categories(id) ON DELETE CASCADE,
    amount           NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
    CONSTRAINT unique_fee_template_rule UNIQUE (academic_term_id, grade_id, fee_category_id)
);
CREATE INDEX idx_fee_templates_school_term ON fee_templates(school_id, academic_term_id);
CREATE INDEX idx_fee_templates_grade_id ON fee_templates(grade_id);

-- -----------------------------------------------------------------------------
--         audit trail, making it impossible to know when/how payments arrived.
--         Balance is now derived from payments (see view below).
CREATE TABLE student_invoices (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id       UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    academic_term_id UUID NOT NULL REFERENCES academic_terms(id) ON DELETE CASCADE,
    fee_category_id  UUID NOT NULL REFERENCES fee_categories(id) ON DELETE CASCADE,
    amount_due       NUMERIC(12,2) NOT NULL CHECK (amount_due >= 0),
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_student_line_item_billing UNIQUE (student_id, academic_term_id, fee_category_id)
);
CREATE INDEX idx_invoices_student_term ON student_invoices(student_id, academic_term_id);
CREATE INDEX idx_invoices_fee_category_id ON student_invoices(fee_category_id);

-- -----------------------------------------------------------------------------
--         amount_paid on the invoice is now: SUM(payments.amount) per invoice.
CREATE TABLE payments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id     UUID NOT NULL REFERENCES student_invoices(id) ON DELETE CASCADE,
    amount         NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    payment_method VARCHAR(50),          -- 'MPESA', 'BANK_TRANSFER', 'CASH', etc.
    reference_code VARCHAR(100),         -- e.g. M-Pesa transaction code
    recorded_by    UUID NOT NULL REFERENCES users(id),
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_payments_invoice_id ON payments(invoice_id);

-- =============================================================================
-- SECTION 6: CONVENIENCE VIEWS
-- =============================================================================
--         Use this anywhere you previously read invoice.amount_paid.
CREATE VIEW v_invoice_balances AS
SELECT
    i.id                                          AS invoice_id,
    i.student_id,
    i.academic_term_id,
    i.fee_category_id,
    i.amount_due,
    COALESCE(SUM(p.amount), 0.00)                 AS amount_paid,
    i.amount_due - COALESCE(SUM(p.amount), 0.00)  AS balance_outstanding
FROM student_invoices i
LEFT JOIN payments p ON p.invoice_id = i.id
GROUP BY i.id, i.student_id, i.academic_term_id, i.fee_category_id, i.amount_due;
-- Step 1 (latest_igcse_scores CTE): for each student + subject + term + assessment_type,
--   pick the single latest sitting using DISTINCT ON ... ORDER BY examination_date DESC.
--   This handles the case where a teacher ran two CATs in the same term — only the
--   most recent one counts.
-- Step 2: JOIN to assessment_weights and compute the weighted sum.
--   raw_score is normalised to a percentage before weighting so scores from
--   papers with different max marks are comparable.
CREATE VIEW v_igcse_final_term_scores AS
WITH latest_igcse_scores AS (
    SELECT DISTINCT ON (m.student_id, p.subject_id, ca.academic_term_id, ca.assessment_type)
        m.student_id,
        p.subject_id,
        ca.academic_term_id,
        ca.class_id,
        ca.assessment_type,
        m.raw_score_earned,
        p.max_raw_mark
    FROM igcse_assessment_marks m
    JOIN igcse_class_assessments ca ON ca.id = m.assessment_id
    JOIN igcse_papers p             ON p.id  = ca.paper_id
    ORDER BY
        m.student_id,
        p.subject_id,
        ca.academic_term_id,
        ca.assessment_type,
        ca.examination_date DESC
)
SELECT
    ls.student_id,
    ls.subject_id,
    ls.academic_term_id,
    ls.class_id,
    ROUND(
        SUM(
            (ls.raw_score_earned / ls.max_raw_mark * 100)  -- normalise to %
            * (w.weight_percent / 100)                      -- apply weight
        ),
    2) AS final_score_percent
FROM latest_igcse_scores ls
JOIN classes cl        ON cl.id              = ls.class_id
JOIN assessment_weights w   ON  w.school_id        = cl.school_id
                            AND w.grade_id          = cl.grade_id
                            AND w.academic_term_id  = ls.academic_term_id
                            AND w.assessment_type   = ls.assessment_type
GROUP BY ls.student_id, ls.subject_id, ls.academic_term_id, ls.class_id;
-- Same DISTINCT ON pattern as IGCSE to pick the latest task per assessment_type.
-- score_level is mapped to a numeric scale (EE=4, ME=3, AE=2, BE=1) before
-- weighting so it can participate in the weighted average.
-- Result is on a 1.0–4.0 scale matching the CBC achievement level range.
CREATE VIEW v_cbc_final_term_scores AS
WITH latest_cbc_scores AS (
    SELECT DISTINCT ON (e.student_id, t.learning_area_id, t.academic_term_id, t.assessment_type)
        e.student_id,
        t.learning_area_id,
        t.academic_term_id,
        t.class_id,
        t.assessment_type,
        CASE e.score_level
            WHEN 'EE' THEN 4
            WHEN 'ME' THEN 3
            WHEN 'AE' THEN 2
            WHEN 'BE' THEN 1
        END AS numeric_score
    FROM cbc_task_evaluations e
    JOIN cbc_formative_tasks t ON t.id = e.formative_task_id
    ORDER BY
        e.student_id,
        t.learning_area_id,
        t.academic_term_id,
        t.assessment_type,
        t.date_administered DESC
)
SELECT
    ls.student_id,
    ls.learning_area_id,
    ls.academic_term_id,
    ls.class_id,
    ROUND(
        SUM(
            ls.numeric_score
            * (w.weight_percent / 100)
        ),
    2) AS final_score   -- 1.0–4.0 scale
FROM latest_cbc_scores ls
JOIN classes cl        ON cl.id              = ls.class_id
JOIN assessment_weights w   ON  w.school_id        = cl.school_id
                            AND w.grade_id          = cl.grade_id
                            AND w.academic_term_id  = ls.academic_term_id
                            AND w.assessment_type   = ls.assessment_type
GROUP BY ls.student_id, ls.learning_area_id, ls.academic_term_id, ls.class_id;

-- =============================================================================
-- END OF SCHEMA
-- =============================================================================
--
================================================================================