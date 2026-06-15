-- ================================================================================
-- 💾 models.sql — MULTI-TENANT ARCHITECTURE SCHEMA CONFIGURATION
-- ================================================================================
-- Refactored 2026-06-15:
--   • Multi-tenant composite foreign keys across all child tables
--   • Education-system cross-contamination prevention (education_system_id
--     threaded into classes and curriculum tables)
--   • Financial system split into invoices (header) + invoice_items (line items)
--   • Timetable double-booking prevention via GiST exclusion constraints
--   • Dynamic re-scaling in final-term-score views (missing assessments no
--     longer artificially drag down averages)
--   • Data-sanity CHECK constraints (end_date > start_date)
--   • external_auth_id made nullable for pre-invited / placeholder accounts
-- ================================================================================

-- =============================================================================
-- EXTENSION: btree_gist (required for GiST exclusion constraints with = stops)
-- =============================================================================
CREATE EXTENSION IF NOT EXISTS btree_gist;

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
-- UTILITY: immutable function to build a tsrange from day-of-week + time pair
-- Used by GiST exclusion constraints on timetable tables.
--
-- Maps day_of_week (1=Monday … 7=Sunday) to an arbitrary base week so that
-- ranges on different days never overlap.  Only same-day entries are checked
-- for scheduling conflicts.
-- =============================================================================

CREATE OR REPLACE FUNCTION fn_timerange(day_of_week INT, start_time TIME, end_time TIME)
RETURNS tsrange
AS $$
    SELECT tsrange(
        ('2024-01-01'::DATE + (day_of_week - 1)) + start_time,
        ('2024-01-01'::DATE + (day_of_week - 1)) + end_time,
        '[)'
    );
$$ LANGUAGE sql IMMUTABLE;

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
    external_auth_id VARCHAR(100) UNIQUE, -- Stytch Mapping Key; NULLABLE for pre-invited/placeholder accounts
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
-- PARENT COMPOSITE UNIQUE: (tenant_id, id) enables composite FKs from child tables
-- -----------------------------------------------------------------------------
CREATE TABLE schools (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    education_system_id  UUID NOT NULL REFERENCES education_systems(id),
    name                 VARCHAR(255) NOT NULL,
    is_active            BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT uq_schools_tenant UNIQUE (tenant_id, id)
);
CREATE INDEX idx_schools_tenant_id ON schools(tenant_id);
CREATE INDEX idx_schools_education_system_id ON schools(education_system_id);

-- -----------------------------------------------------------------------------
CREATE TABLE memberships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL,
    role       user_role NOT NULL,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id  UUID NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_memberships_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_user_school_membership UNIQUE (user_id, school_id)
);
CREATE INDEX idx_memberships_tenant_id ON memberships(tenant_id);
CREATE INDEX idx_memberships_user_id ON memberships(user_id);
CREATE INDEX idx_memberships_school_id ON memberships(school_id);

-- -----------------------------------------------------------------------------
CREATE TABLE academic_years (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL,
    school_id  UUID NOT NULL,
    name       VARCHAR(50) NOT NULL,
    start_date DATE NOT NULL,
    end_date   DATE NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT chk_academic_year_dates CHECK (end_date > start_date),
    CONSTRAINT uq_academic_years_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_academic_years_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_academic_years_tenant_id ON academic_years(tenant_id);
CREATE INDEX idx_academic_years_school_id ON academic_years(school_id);
CREATE UNIQUE INDEX idx_one_current_year_per_school
    ON academic_years(school_id)
    WHERE is_current = true;

-- -----------------------------------------------------------------------------
CREATE TABLE academic_terms (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    academic_year_id UUID NOT NULL,
    name             VARCHAR(100) NOT NULL,
    start_date       DATE NOT NULL,
    end_date         DATE NOT NULL,
    is_current       BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT chk_academic_term_dates CHECK (end_date > start_date),
    CONSTRAINT uq_academic_terms_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_academic_terms_tenant_year
        FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_academic_terms_tenant_id ON academic_terms(tenant_id);
CREATE INDEX idx_academic_terms_year_id ON academic_terms(academic_year_id);
CREATE UNIQUE INDEX idx_one_current_term_per_year
    ON academic_terms(academic_year_id)
    WHERE is_current = true;

-- -----------------------------------------------------------------------------
CREATE TABLE grades (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    education_system_id UUID NOT NULL REFERENCES education_systems(id) ON DELETE CASCADE,
    name                VARCHAR(100) NOT NULL,  -- "Grade 7", "Year 11", "MYP 4"
    sequence_order      INT NOT NULL,
    CONSTRAINT uq_grades_education_system UNIQUE (education_system_id, id)
);
CREATE INDEX idx_grades_education_system_id ON grades(education_system_id);

-- -----------------------------------------------------------------------------
CREATE TABLE classes (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    school_id          UUID NOT NULL,
    academic_year_id   UUID NOT NULL,
    education_system_id UUID NOT NULL,     -- denormalised from school; guards cross-system grade_id
    grade_id           UUID NOT NULL,
    name               VARCHAR(100) NOT NULL,  -- "East", "Alpha"
    is_active          BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT uq_classes_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_classes_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_classes_tenant_academic_year
        FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_classes_education_system_grade
        FOREIGN KEY (education_system_id, grade_id) REFERENCES grades(education_system_id, id)
);
CREATE INDEX idx_classes_tenant_id ON classes(tenant_id);
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
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_students_tenant UNIQUE (tenant_id, id)
);
CREATE INDEX idx_students_tenant_id ON students(tenant_id);

-- -----------------------------------------------------------------------------
CREATE TABLE student_enrollments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    student_id       UUID NOT NULL,
    school_id        UUID NOT NULL,
    academic_term_id UUID NOT NULL,
    class_id         UUID,
    status           enrollment_status NOT NULL DEFAULT 'ACTIVE',
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_enrollments_tenant_student
        FOREIGN KEY (tenant_id, student_id) REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE SET NULL,
    CONSTRAINT unique_active_term_enrollment UNIQUE (student_id, academic_term_id)
);
CREATE INDEX idx_enrollments_tenant_id ON student_enrollments(tenant_id);
CREATE INDEX idx_enrollments_student_id ON student_enrollments(student_id);
CREATE INDEX idx_enrollments_school_id ON student_enrollments(school_id);
CREATE INDEX idx_enrollments_term_id ON student_enrollments(academic_term_id);
CREATE INDEX idx_enrollments_class_id ON student_enrollments(class_id);

-- =============================================================================
-- SECTION 2: KENYA COMPETENCY-BASED CURRICULUM MODULE (`cbc_`)
-- =============================================================================

CREATE TABLE cbc_learning_areas (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    school_id          UUID NOT NULL,
    education_system_id UUID NOT NULL,   -- guards cross-system grade_id
    grade_id           UUID NOT NULL,
    name               VARCHAR(150) NOT NULL,
    code               VARCHAR(50) NOT NULL,
    CONSTRAINT fk_cbc_learning_areas_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_learning_areas_education_system_grade
        FOREIGN KEY (education_system_id, grade_id) REFERENCES grades(education_system_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_cbc_learning_areas_tenant ON cbc_learning_areas(tenant_id);
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
    tenant_id           UUID NOT NULL,
    class_id            UUID NOT NULL,
    academic_term_id    UUID NOT NULL,
    learning_outcome_id UUID NOT NULL REFERENCES cbc_learning_outcomes(id) ON DELETE CASCADE,
    title               VARCHAR(255) NOT NULL,
    assessment_type     assessment_type NOT NULL DEFAULT 'OTHER',
    date_administered   DATE NOT NULL,
    created_by          UUID NOT NULL REFERENCES users(id),
    CONSTRAINT fk_cbc_formative_tasks_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_formative_tasks_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_cbc_formative_tasks_tenant ON cbc_formative_tasks(tenant_id);
CREATE INDEX idx_cbc_formative_tasks_class_id ON cbc_formative_tasks(class_id);
CREATE INDEX idx_cbc_formative_tasks_term_id ON cbc_formative_tasks(academic_term_id);
CREATE INDEX idx_cbc_formative_tasks_outcome_id ON cbc_formative_tasks(learning_outcome_id);

-- -----------------------------------------------------------------------------
CREATE TABLE cbc_task_evaluations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    formative_task_id UUID NOT NULL,
    student_id        UUID NOT NULL,
    graded_by_user_id UUID NOT NULL REFERENCES users(id),
    score_level       cbc_score_level NOT NULL,
    teacher_remarks   TEXT,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_cbc_task_evals_tenant_task
        FOREIGN KEY (tenant_id, formative_task_id) REFERENCES cbc_formative_tasks(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_task_evals_tenant_student
        FOREIGN KEY (tenant_id, student_id) REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_cbc_student_task UNIQUE (formative_task_id, student_id)
);
CREATE INDEX idx_cbc_task_evals_tenant ON cbc_task_evaluations(tenant_id);
CREATE INDEX idx_cbc_task_evals_task_id ON cbc_task_evaluations(formative_task_id);
CREATE INDEX idx_cbc_task_evals_student_id ON cbc_task_evaluations(student_id);
CREATE TRIGGER trg_cbc_task_evaluations_updated_at
    BEFORE UPDATE ON cbc_task_evaluations
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- -----------------------------------------------------------------------------
CREATE TABLE cbc_class_teachers (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    class_id             UUID NOT NULL,
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    learning_area_id     UUID NOT NULL,
    is_primary           BOOLEAN NOT NULL DEFAULT false,
    created_at           TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_cbc_class_teachers_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_cbc_class_teacher UNIQUE (class_id, user_id, learning_area_id)
);
CREATE INDEX idx_cbc_class_teachers_tenant ON cbc_class_teachers(tenant_id);
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
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    school_id          UUID NOT NULL,
    education_system_id UUID NOT NULL,   -- guards cross-system grade_id
    grade_id           UUID NOT NULL,
    name               VARCHAR(150) NOT NULL,
    code               VARCHAR(20) NOT NULL,  -- e.g. "0620" for Chemistry
    CONSTRAINT fk_igcse_subjects_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_igcse_subjects_education_system_grade
        FOREIGN KEY (education_system_id, grade_id) REFERENCES grades(education_system_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_igcse_subjects_tenant ON igcse_subjects(tenant_id);
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
    tenant_id        UUID NOT NULL,
    paper_id         UUID NOT NULL REFERENCES igcse_papers(id) ON DELETE CASCADE,
    class_id         UUID NOT NULL,
    academic_term_id UUID NOT NULL,
    title            VARCHAR(255) NOT NULL,
    assessment_type  assessment_type NOT NULL DEFAULT 'OTHER',
    examination_date DATE NOT NULL,
    created_by       UUID NOT NULL REFERENCES users(id),
    CONSTRAINT fk_igcse_assessments_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_igcse_assessments_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_igcse_assessments_tenant ON igcse_class_assessments(tenant_id);
CREATE INDEX idx_igcse_assessments_paper_id ON igcse_class_assessments(paper_id);
CREATE INDEX idx_igcse_assessments_class_id ON igcse_class_assessments(class_id);
CREATE INDEX idx_igcse_assessments_term_id ON igcse_class_assessments(academic_term_id);

-- -----------------------------------------------------------------------------
CREATE TABLE igcse_assessment_marks (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    assessment_id     UUID NOT NULL,
    student_id        UUID NOT NULL,
    graded_by_user_id UUID NOT NULL REFERENCES users(id),
    raw_score_earned  NUMERIC(5,2) NOT NULL,
    teacher_remarks   TEXT,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_igcse_marks_tenant_assessment
        FOREIGN KEY (tenant_id, assessment_id) REFERENCES igcse_class_assessments(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_igcse_marks_tenant_student
        FOREIGN KEY (tenant_id, student_id) REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_igcse_student_assessment UNIQUE (assessment_id, student_id)
);
CREATE INDEX idx_igcse_marks_tenant ON igcse_assessment_marks(tenant_id);
CREATE INDEX idx_igcse_marks_assessment_id ON igcse_assessment_marks(assessment_id);
CREATE INDEX idx_igcse_marks_student_id ON igcse_assessment_marks(student_id);
CREATE TRIGGER trg_igcse_assessment_marks_updated_at
    BEFORE UPDATE ON igcse_assessment_marks
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- -----------------------------------------------------------------------------
CREATE TABLE igcse_class_teachers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    class_id     UUID NOT NULL,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject_id   UUID NOT NULL REFERENCES igcse_subjects(id) ON DELETE CASCADE,
    is_primary   BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_igcse_class_teachers_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_igcse_class_teacher UNIQUE (class_id, user_id, subject_id)
);
CREATE INDEX idx_igcse_class_teachers_tenant ON igcse_class_teachers(tenant_id);
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
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    school_id   UUID NOT NULL,
    group_name  VARCHAR(100) NOT NULL,  -- "Sciences", "Mathematics", "Language Acquisition"
    CONSTRAINT fk_ib_subject_groups_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_ib_subject_groups_tenant ON ib_subject_groups(tenant_id);
CREATE INDEX idx_ib_subject_groups_school_id ON ib_subject_groups(school_id);

-- -----------------------------------------------------------------------------
CREATE TABLE ib_disciplines (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    group_id           UUID NOT NULL,
    education_system_id UUID NOT NULL,  -- guards cross-system grade_id
    grade_id           UUID NOT NULL,
    name               VARCHAR(150) NOT NULL,  -- "Physics", "Chemistry"
    CONSTRAINT uq_ib_disciplines_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_ib_disciplines_tenant_group
        FOREIGN KEY (tenant_id, group_id) REFERENCES ib_subject_groups(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_disciplines_education_system_grade
        FOREIGN KEY (education_system_id, grade_id) REFERENCES grades(education_system_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_ib_disciplines_tenant ON ib_disciplines(tenant_id);
CREATE INDEX idx_ib_disciplines_group_id ON ib_disciplines(group_id);
CREATE INDEX idx_ib_disciplines_grade_id ON ib_disciplines(grade_id);

-- -----------------------------------------------------------------------------
CREATE TABLE ib_tasks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    discipline_id  UUID NOT NULL,
    class_id       UUID NOT NULL,
    academic_term_id UUID NOT NULL,
    title          VARCHAR(255) NOT NULL,
    assessment_type  assessment_type NOT NULL DEFAULT 'OTHER',
    execution_date DATE NOT NULL,
    created_by     UUID NOT NULL REFERENCES users(id),
    CONSTRAINT fk_ib_tasks_tenant_discipline
        FOREIGN KEY (tenant_id, discipline_id) REFERENCES ib_disciplines(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_tasks_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_tasks_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_ib_tasks_tenant ON ib_tasks(tenant_id);
CREATE INDEX idx_ib_tasks_discipline_id ON ib_tasks(discipline_id);
CREATE INDEX idx_ib_tasks_class_id ON ib_tasks(class_id);
CREATE INDEX idx_ib_tasks_term_id ON ib_tasks(academic_term_id);

-- -----------------------------------------------------------------------------
CREATE TABLE ib_task_criterion_scores (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    task_id           UUID NOT NULL,
    student_id        UUID NOT NULL,
    graded_by_user_id UUID NOT NULL REFERENCES users(id),
    criterion_type    ib_criterion_type NOT NULL,
    achievement_level INT NOT NULL,
    teacher_remarks   TEXT,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_ib_achievement_level_bounds CHECK (achievement_level BETWEEN 0 AND 8),
    CONSTRAINT fk_ib_criterion_scores_tenant_task
        FOREIGN KEY (tenant_id, task_id) REFERENCES ib_tasks(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_criterion_scores_tenant_student
        FOREIGN KEY (tenant_id, student_id) REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_ib_student_task_criterion UNIQUE (task_id, student_id, criterion_type)
);
CREATE INDEX idx_ib_criterion_scores_tenant ON ib_task_criterion_scores(tenant_id);
CREATE INDEX idx_ib_criterion_scores_task_id ON ib_task_criterion_scores(task_id);
CREATE INDEX idx_ib_criterion_scores_student_id ON ib_task_criterion_scores(student_id);
CREATE TRIGGER trg_ib_task_criterion_scores_updated_at
    BEFORE UPDATE ON ib_task_criterion_scores
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- -----------------------------------------------------------------------------
CREATE TABLE ib_class_teachers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    class_id      UUID NOT NULL,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    discipline_id UUID NOT NULL,
    is_primary    BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_ib_class_teachers_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_ib_class_teacher UNIQUE (class_id, user_id, discipline_id)
);
CREATE INDEX idx_ib_class_teachers_tenant ON ib_class_teachers(tenant_id);
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

CREATE TABLE assessment_weights (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    school_id        UUID NOT NULL,
    education_system_id UUID NOT NULL, -- guards cross-system grade_id
    grade_id         UUID NOT NULL,
    academic_term_id UUID NOT NULL,
    assessment_type  assessment_type NOT NULL,
    weight_percent   NUMERIC(5,2) NOT NULL CHECK (weight_percent > 0 AND weight_percent <= 100),
    CONSTRAINT fk_assessment_weights_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_assessment_weights_education_system_grade
        FOREIGN KEY (education_system_id, grade_id) REFERENCES grades(education_system_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_assessment_weights_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_weight_rule UNIQUE (school_id, grade_id, academic_term_id, assessment_type)
);
CREATE INDEX idx_assessment_weights_tenant ON assessment_weights(tenant_id);
CREATE INDEX idx_assessment_weights_school_grade ON assessment_weights(school_id, grade_id);
CREATE INDEX idx_assessment_weights_term_id ON assessment_weights(academic_term_id);

-- =============================================================================
-- SECTION 5: UNIFIED CORE SYSTEMS (ATTENDANCE, HEALTH, FINANCES)
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. CBC ATTENDANCE
-- -----------------------------------------------------------------------------
CREATE TABLE cbc_attendance_periods (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    school_id            UUID NOT NULL,
    academic_term_id     UUID NOT NULL,
    class_id             UUID NOT NULL,
    cbc_learning_area_id UUID NOT NULL,
    date_recorded        DATE NOT NULL,
    CONSTRAINT fk_cbc_att_periods_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_att_periods_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_att_periods_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_cbc_att_tenant ON cbc_attendance_periods(tenant_id);
CREATE INDEX idx_cbc_att_periods_class_date ON cbc_attendance_periods(class_id, date_recorded);
CREATE UNIQUE INDEX idx_cbc_unique_attendance_period 
    ON cbc_attendance_periods(class_id, date_recorded, cbc_learning_area_id);

CREATE TABLE cbc_attendance_logs (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    cbc_attendance_period_id UUID NOT NULL,
    student_id               UUID NOT NULL,
    status                   attendance_status NOT NULL,
    remarks                  VARCHAR(255),
    recorded_by              UUID NOT NULL REFERENCES users(id),
    CONSTRAINT fk_cbc_att_logs_tenant_period
        FOREIGN KEY (tenant_id, cbc_attendance_period_id)
            REFERENCES cbc_attendance_periods(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_att_logs_tenant_student
        FOREIGN KEY (tenant_id, student_id) REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_cbc_student_attendance_period UNIQUE (cbc_attendance_period_id, student_id)
);
CREATE INDEX idx_cbc_att_logs_tenant ON cbc_attendance_logs(tenant_id);
CREATE INDEX idx_cbc_att_logs_period ON cbc_attendance_logs(cbc_attendance_period_id);
CREATE INDEX idx_cbc_att_logs_student ON cbc_attendance_logs(student_id);


-- -----------------------------------------------------------------------------
-- 2. IGCSE ATTENDANCE
-- -----------------------------------------------------------------------------
CREATE TABLE igcse_attendance_periods (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    school_id            UUID NOT NULL,
    academic_term_id     UUID NOT NULL,
    class_id             UUID NOT NULL,
    igcse_subject_id     UUID NOT NULL,
    date_recorded        DATE NOT NULL,
    CONSTRAINT fk_igcse_att_periods_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_igcse_att_periods_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_igcse_att_periods_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_igcse_att_tenant ON igcse_attendance_periods(tenant_id);
CREATE INDEX idx_igcse_att_periods_class_date ON igcse_attendance_periods(class_id, date_recorded);
CREATE UNIQUE INDEX idx_igcse_unique_attendance_period 
    ON igcse_attendance_periods(class_id, date_recorded, igcse_subject_id);

CREATE TABLE igcse_attendance_logs (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                  UUID NOT NULL,
    igcse_attendance_period_id UUID NOT NULL,
    student_id                 UUID NOT NULL,
    status                     attendance_status NOT NULL,
    remarks                    VARCHAR(255),
    recorded_by                UUID NOT NULL REFERENCES users(id),
    CONSTRAINT fk_igcse_att_logs_tenant_period
        FOREIGN KEY (tenant_id, igcse_attendance_period_id)
            REFERENCES igcse_attendance_periods(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_igcse_att_logs_tenant_student
        FOREIGN KEY (tenant_id, student_id) REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_igcse_student_attendance_period UNIQUE (igcse_attendance_period_id, student_id)
);
CREATE INDEX idx_igcse_att_logs_tenant ON igcse_attendance_logs(tenant_id);
CREATE INDEX idx_igcse_att_logs_period ON igcse_attendance_logs(igcse_attendance_period_id);
CREATE INDEX idx_igcse_att_logs_student ON igcse_attendance_logs(student_id);


-- -----------------------------------------------------------------------------
-- 3. IB MYP ATTENDANCE
-- -----------------------------------------------------------------------------
CREATE TABLE ib_attendance_periods (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    school_id            UUID NOT NULL,
    academic_term_id     UUID NOT NULL,
    class_id             UUID NOT NULL,
    ib_discipline_id     UUID NOT NULL,
    date_recorded        DATE NOT NULL,
    CONSTRAINT fk_ib_att_periods_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_att_periods_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_att_periods_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_ib_att_tenant ON ib_attendance_periods(tenant_id);
CREATE INDEX idx_ib_att_periods_class_date ON ib_attendance_periods(class_id, date_recorded);
CREATE UNIQUE INDEX idx_ib_unique_attendance_period 
    ON ib_attendance_periods(class_id, date_recorded, ib_discipline_id);

CREATE TABLE ib_attendance_logs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL,
    ib_attendance_period_id UUID NOT NULL,
    student_id              UUID NOT NULL,
    status                  attendance_status NOT NULL,
    remarks                 VARCHAR(255),
    recorded_by             UUID NOT NULL REFERENCES users(id),
    CONSTRAINT fk_ib_att_logs_tenant_period
        FOREIGN KEY (tenant_id, ib_attendance_period_id)
            REFERENCES ib_attendance_periods(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_att_logs_tenant_student
        FOREIGN KEY (tenant_id, student_id) REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_ib_student_attendance_period UNIQUE (ib_attendance_period_id, student_id)
);
CREATE INDEX idx_ib_att_logs_tenant ON ib_attendance_logs(tenant_id);
CREATE INDEX idx_ib_att_logs_period ON ib_attendance_logs(ib_attendance_period_id);
CREATE INDEX idx_ib_att_logs_student ON ib_attendance_logs(student_id);

-- ====================================================================
-- CURRICULUM-SPECIFIC TIMETABLE SYSTEM (SPLIT ARCHITECTURE)
-- ====================================================================

-- 1. CBC TIMETABLE
CREATE TABLE cbc_timetable_slots (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    school_id            UUID NOT NULL,
    academic_year_id     UUID NOT NULL,
    class_id             UUID NOT NULL,
    teacher_id           UUID NOT NULL,
    cbc_learning_area_id UUID,
    room_identifier      VARCHAR(50),
    day_of_week          INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time           TIME NOT NULL,
    end_time             TIME NOT NULL,
    CONSTRAINT fk_cbc_timetable_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_year
        FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_teacher
        FOREIGN KEY (tenant_id, teacher_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    -- Prevent teacher double-booking: same teacher, same academic year, overlapping time
    CONSTRAINT excl_cbc_timetable_teacher
        EXCLUDE USING gist (
            teacher_id WITH =,
            academic_year_id WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        ),
    -- Prevent room double-booking: same room (if set), same academic year, overlapping time
    CONSTRAINT excl_cbc_timetable_room
        EXCLUDE USING gist (
            room_identifier WITH =,
            academic_year_id WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        )
);
CREATE INDEX idx_cbc_timetable_tenant ON cbc_timetable_slots(tenant_id);
CREATE INDEX idx_cbc_timetable_school_year ON cbc_timetable_slots(school_id, academic_year_id);
CREATE INDEX idx_cbc_timetable_class ON cbc_timetable_slots(class_id);
CREATE INDEX idx_cbc_timetable_teacher ON cbc_timetable_slots(teacher_id);

-- 2. IGCSE TIMETABLE
CREATE TABLE igcse_timetable_slots (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    school_id            UUID NOT NULL,
    academic_year_id     UUID NOT NULL,
    class_id             UUID NOT NULL,
    teacher_id           UUID NOT NULL,
    igcse_subject_id     UUID REFERENCES igcse_subjects(id) ON DELETE SET NULL, -- Nullable for breaks/assemblies
    room_identifier      VARCHAR(50),
    day_of_week          INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time           TIME NOT NULL,
    end_time             TIME NOT NULL,
    CONSTRAINT fk_igcse_timetable_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_igcse_timetable_tenant_year
        FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_igcse_timetable_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_igcse_timetable_tenant_teacher
        FOREIGN KEY (tenant_id, teacher_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    -- Prevent teacher double-booking
    CONSTRAINT excl_igcse_timetable_teacher
        EXCLUDE USING gist (
            teacher_id WITH =,
            academic_year_id WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        ),
    -- Prevent room double-booking
    CONSTRAINT excl_igcse_timetable_room
        EXCLUDE USING gist (
            room_identifier WITH =,
            academic_year_id WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        )
);
CREATE INDEX idx_igcse_timetable_tenant ON igcse_timetable_slots(tenant_id);
CREATE INDEX idx_igcse_timetable_school_year ON igcse_timetable_slots(school_id, academic_year_id);
CREATE INDEX idx_igcse_timetable_class ON igcse_timetable_slots(class_id);
CREATE INDEX idx_igcse_timetable_teacher ON igcse_timetable_slots(teacher_id);

-- 3. IB TIMETABLE
CREATE TABLE ib_timetable_slots (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    school_id            UUID NOT NULL,
    academic_year_id     UUID NOT NULL,
    class_id             UUID NOT NULL,
    teacher_id           UUID NOT NULL,
    ib_discipline_id     UUID REFERENCES ib_disciplines(id) ON DELETE SET NULL, -- Nullable for breaks/assemblies
    room_identifier      VARCHAR(50),
    day_of_week          INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time           TIME NOT NULL,
    end_time             TIME NOT NULL,
    CONSTRAINT fk_ib_timetable_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_timetable_tenant_year
        FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_timetable_tenant_class
        FOREIGN KEY (tenant_id, class_id) REFERENCES classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_ib_timetable_tenant_teacher
        FOREIGN KEY (tenant_id, teacher_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    -- Prevent teacher double-booking
    CONSTRAINT excl_ib_timetable_teacher
        EXCLUDE USING gist (
            teacher_id WITH =,
            academic_year_id WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        ),
    -- Prevent room double-booking
    CONSTRAINT excl_ib_timetable_room
        EXCLUDE USING gist (
            room_identifier WITH =,
            academic_year_id WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        )
);
CREATE INDEX idx_ib_timetable_tenant ON ib_timetable_slots(tenant_id);
CREATE INDEX idx_ib_timetable_school_year ON ib_timetable_slots(school_id, academic_year_id);
CREATE INDEX idx_ib_timetable_class ON ib_timetable_slots(class_id);
CREATE INDEX idx_ib_timetable_teacher ON ib_timetable_slots(teacher_id);

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

-- -----------------------------------------------------------------------------
-- fee_categories: school-level fee definitions
-- -----------------------------------------------------------------------------
CREATE TABLE fee_categories (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    school_id    UUID NOT NULL,
    name         VARCHAR(150) NOT NULL,
    is_mandatory BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT fk_fee_categories_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_fee_categories_tenant ON fee_categories(tenant_id);
CREATE INDEX idx_fee_categories_school_id ON fee_categories(school_id);

-- -----------------------------------------------------------------------------
-- fee_templates: term-level fee amounts per grade + category
-- -----------------------------------------------------------------------------
CREATE TABLE fee_templates (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    school_id        UUID NOT NULL,
    education_system_id UUID NOT NULL,
    academic_term_id UUID NOT NULL,
    grade_id         UUID NOT NULL,
    fee_category_id  UUID NOT NULL REFERENCES fee_categories(id) ON DELETE CASCADE,
    amount           NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
    CONSTRAINT fk_fee_templates_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_fee_templates_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_fee_templates_education_system_grade
        FOREIGN KEY (education_system_id, grade_id) REFERENCES grades(education_system_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_fee_template_rule UNIQUE (academic_term_id, grade_id, fee_category_id)
);
CREATE INDEX idx_fee_templates_tenant ON fee_templates(tenant_id);
CREATE INDEX idx_fee_templates_school_term ON fee_templates(school_id, academic_term_id);
CREATE INDEX idx_fee_templates_grade_id ON fee_templates(grade_id);

-- -----------------------------------------------------------------------------
-- invoices: per-student, per-term billing header
-- -----------------------------------------------------------------------------
CREATE TABLE invoices (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL,
    student_id       UUID NOT NULL,
    school_id        UUID NOT NULL,
    academic_term_id UUID NOT NULL,
    invoice_label    VARCHAR(255),                       -- e.g. "Term 1 2025 — Final Bill"
    created_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_invoices_tenant_student
        FOREIGN KEY (tenant_id, student_id) REFERENCES students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invoices_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invoices_tenant_term
        FOREIGN KEY (tenant_id, academic_term_id) REFERENCES academic_terms(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_invoice_per_student_term UNIQUE (student_id, academic_term_id)
);
CREATE INDEX idx_invoices_tenant ON invoices(tenant_id);
CREATE INDEX idx_invoices_student_term ON invoices(student_id, academic_term_id);

-- -----------------------------------------------------------------------------
-- invoice_items: individual line items per invoice (flexible, repeatable)
-- -----------------------------------------------------------------------------
CREATE TABLE invoice_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    invoice_id      UUID NOT NULL,
    fee_category_id UUID NOT NULL REFERENCES fee_categories(id) ON DELETE CASCADE,
    description     VARCHAR(255),
    amount          NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_invoice_items_tenant_invoice
        FOREIGN KEY (tenant_id, invoice_id) REFERENCES invoices(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_invoice_items_tenant ON invoice_items(tenant_id);
CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);
CREATE INDEX idx_invoice_items_fee_category ON invoice_items(fee_category_id);

-- -----------------------------------------------------------------------------
-- payments: flexible — tied to an invoice header (not a single item)
-- so a lump sum can settle multiple line items
-- -----------------------------------------------------------------------------
CREATE TABLE payments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    invoice_id     UUID NOT NULL,
    amount         NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    payment_method VARCHAR(50),
    reference_code VARCHAR(100),
    recorded_by    UUID NOT NULL REFERENCES users(id),
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_payments_tenant_invoice
        FOREIGN KEY (tenant_id, invoice_id) REFERENCES invoices(tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_payments_tenant ON payments(tenant_id);
CREATE INDEX idx_payments_invoice_id ON payments(invoice_id);

-- =============================================================================
-- SECTION 6: CONVENIENCE VIEWS
-- =============================================================================

-- -----------------------------------------------------------------------------
-- v_invoice_balances: aggregated invoice header + line-item totals + payments
-- -----------------------------------------------------------------------------
CREATE VIEW v_invoice_balances AS
WITH line_item_totals AS (
    SELECT
        ii.invoice_id,
        SUM(ii.amount) AS total_invoiced
    FROM invoice_items ii
    GROUP BY ii.invoice_id
),
payment_totals AS (
    SELECT
        p.invoice_id,
        COALESCE(SUM(p.amount), 0.00) AS total_paid
    FROM payments p
    GROUP BY p.invoice_id
)
SELECT
    i.id                       AS invoice_id,
    i.student_id,
    i.academic_term_id,
    i.invoice_label,
    COALESCE(lit.total_invoiced, 0.00)                AS total_invoiced,
    COALESCE(pt.total_paid, 0.00)                     AS total_paid,
    (COALESCE(lit.total_invoiced, 0.00)
     - COALESCE(pt.total_paid, 0.00))                 AS balance_due
FROM invoices i
LEFT JOIN line_item_totals lit ON lit.invoice_id = i.id
LEFT JOIN payment_totals   pt  ON pt.invoice_id  = i.id;

-- -----------------------------------------------------------------------------
-- v_igcse_final_term_scores
--
-- For each student / subject / term, picks the latest sitting per
-- assessment_type (DISTINCT ON), normalises raw_score to a percentage,
-- then computes a dynamically re-scaled weighted average.
--
-- FIX (dynamic re-scale): The divisor is the sum of weights for assessment
-- types that *actually have recorded scores* for this student/subject/term.
-- If END_TERM (60%) hasn't been graded yet, only CAT (40%) contributes and
-- the result is re-scaled to 100%, avoiding artificial score depression.
-- -----------------------------------------------------------------------------
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
    CASE
        WHEN SUM(w.weight_percent) > 0
        THEN ROUND(
                 SUM(
                     (ls.raw_score_earned / ls.max_raw_mark * 100)  -- normalise to %
                     * (w.weight_percent / 100)                      -- apply weight
                 )
                 / (SUM(w.weight_percent) / 100),                   -- re-scale to % of recorded only
                 2)
        ELSE NULL
    END AS final_score_percent
FROM latest_igcse_scores ls
JOIN classes cl              ON cl.id              = ls.class_id
JOIN assessment_weights w    ON  w.school_id        = cl.school_id
                             AND w.grade_id          = cl.grade_id
                             AND w.academic_term_id  = ls.academic_term_id
                             AND w.assessment_type   = ls.assessment_type
GROUP BY ls.student_id, ls.subject_id, ls.academic_term_id, ls.class_id;

-- -----------------------------------------------------------------------------
-- v_cbc_final_term_scores
--
-- Same dynamic re-scaling logic as IGCSE view above.
-- score_level mapped to numeric: EE=4, ME=3, AE=2, BE=1.
-- Result stays on a 1.0–4.0 scale (re-scaled to weight of recorded items).
-- -----------------------------------------------------------------------------
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
    CASE
        WHEN SUM(w.weight_percent) > 0
        THEN ROUND(
                 SUM(
                     ls.numeric_score
                     * (w.weight_percent / 100)
                 )
                 / (SUM(w.weight_percent) / 100),  -- re-scale to weight of recorded items
                 2)
        ELSE NULL
    END AS final_score   -- 1.0–4.0 scale
FROM latest_cbc_scores ls
JOIN classes cl              ON cl.id              = ls.class_id
JOIN assessment_weights w    ON  w.school_id        = cl.school_id
                             AND w.grade_id          = cl.grade_id
                             AND w.academic_term_id  = ls.academic_term_id
                             AND w.assessment_type   = ls.assessment_type
GROUP BY ls.student_id, ls.learning_area_id, ls.academic_term_id, ls.class_id;

-- =============================================================================
-- END OF SCHEMA
-- =============================================================================
