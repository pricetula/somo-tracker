-- Migration: 000001_extensions_and_enums
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: Extensions + all ENUM types

-- Migration: 000001_initial_schema
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only, v5)
-- Drops all generic education system abstractions.
-- Rebuilds as a purpose-built, single-system CBC schema.
--
-- NOTE: This file is the SQUASHED fresh-install schema. It folds in the
-- net effect of the former migrations 000003–000017 (review fixes, tenant
-- scoping, summary/rollup tables) directly into the object definitions —
-- no CREATE-then-ALTER sequences, no backfills. The seed data (000002)
-- remains a separate migration.

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- ============================================================================
-- ENUMS
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('SYSTEM_ADMIN', 'SCHOOL_ADMIN', 'TEACHER', 'NURSE', 'FINANCE', 'PARENT');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'expired', 'revoked', 'invite_failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE gender_type AS ENUM ('M', 'F');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_enrollment_status AS ENUM (
        'ACTIVE',            -- Currently enrolled and attending
        'SUSPENDED',         -- Temporarily removed from active learning
        'TRANSFERRED',       -- Moved to another school; record retained
        'COMPLETED_CYCLE'    -- Successfully completed a CBC education cycle
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_grade_level AS ENUM (
        'PP1','PP2',
        'G1','G2','G3',
        'G4','G5','G6',
        'G7','G8','G9',
        'G10','G11','G12'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_education_level AS ENUM (
        'Early_Years',
        'Upper_Primary',
        'Junior_Secondary',
        'Senior_School'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE teacher_role AS ENUM (
        'PRIMARY_CLASS_TEACHER',
        'SUBJECT_TEACHER',
        'SUBSTITUTE_TEACHER'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_school_type AS ENUM (
        'Public',
        'Private',
        'Special_Needs_School'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_learning_pathway AS ENUM (
        'Age_Based',
        'Stage_Based'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE invoice_payment_status AS ENUM (
        'UNPAID',    -- No payments recorded yet
        'PARTIAL',   -- Some payment made, balance remains
        'PAID',      -- Fully settled
        'WAIVED'     -- Debt forgiven by finance admin
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- IMPROVE: import_job_status enum replaces unconstrained TEXT on import_jobs.status
DO $$ BEGIN
    CREATE TYPE import_job_status AS ENUM (
        'pending',
        'processing',
        'completed',
        'failed',
        'cancelled',
        'completed_with_errors',
        'cancelling'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE import_job_type AS ENUM ('STAFF_INVITE', 'STUDENT_IMPORT', 'PARENT_INVITE');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE import_staging_status AS ENUM ('pending', 'succeeded', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE import_chunk_status AS ENUM ('pending', 'processing', 'completed', 'cancelled');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE block_type AS ENUM ('Lesson', 'Break', 'Assembly', 'ExtraCurricular');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE import_failure_type AS ENUM (
        'SCHEMA_VALIDATION',
        'DATABASE_CONSTRAINT',
        'BUSINESS_RULE_VIOLATION',
        'DUPLICATE_EMAIL',
        'INVALID_EMAIL_FORMAT',
        'STYTCH_API_ERROR',
        'INVITATION_INSERT_FAILED'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE payment_method_type AS ENUM (
        'MPESA',
        'CASH',
        'BANK_TRANSFER',
        'CHEQUE',
        'OTHER'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE parent_relationship_type AS ENUM (
        'FATHER',
        'MOTHER',
        'GUARDIAN',
        'OTHER'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE behavior_category_type AS ENUM ('COMMENDATION', 'DISCIPLINARY', 'OTHER');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE attendance_status AS ENUM ('PRESENT', 'ABSENT', 'LATE', 'EXCUSED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE behavior_note_status AS ENUM ('PENDING_REVIEW', 'APPROVED', 'REJECTED', 'INCLUDED_IN_REPORT');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE behavior_severity AS ENUM ('MINOR', 'NEEDS_FOLLOW_UP');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_performance_level AS ENUM ('EE', 'ME', 'AE', 'BE');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON TYPE cbc_performance_level IS
    'CBC rubric performance levels: EE = Exceeding Expectation, ME = Meeting
     Expectation, AE = Approaching Expectation, BE = Below Expectation.';

DO $$ BEGIN
    CREATE TYPE assessment_session_status AS ENUM ('DRAFT', 'PENDING_APPROVAL', 'PUBLISHED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON TYPE assessment_session_status IS
    'Lifecycle state for an assessment session: DRAFT (teacher editing),
     PENDING_APPROVAL (submitted for admin review), PUBLISHED (finalised
     and visible to parents).';

DO $$ BEGIN
    CREATE TYPE assessment_evaluation_method AS ENUM ('QUANTITATIVE', 'RUBRIC');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON TYPE assessment_evaluation_method IS
    'QUANTITATIVE: total-marks grading (raw_score + percentage → performance level).
     RUBRIC: indicator-level grading (direct CBC level per performance_indicator).';
