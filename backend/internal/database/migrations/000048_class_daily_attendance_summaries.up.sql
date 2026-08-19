-- Migration: 000048_class_daily_attendance_summaries
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: class_daily_attendance_summaries

CREATE TABLE IF NOT EXISTS class_daily_attendance_summaries (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID          NOT NULL,
    school_id             UUID          NOT NULL,
    class_id              UUID          NOT NULL,
    academic_term_id      UUID          NOT NULL,
    date                  DATE          NOT NULL,
    total_enrolled        INT           NOT NULL,
    present_count         INT           NOT NULL,
    absent_count          INT           NOT NULL,
    late_count            INT           NOT NULL,
    excused_count         INT           NOT NULL,
    daily_attendance_rate NUMERIC(5,2)  NOT NULL,
    last_refreshed_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_class_daily_attendance UNIQUE (class_id, date),
    CONSTRAINT fk_class_daily_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_daily_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_class_daily_tenant
    ON class_daily_attendance_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_class_daily_school
    ON class_daily_attendance_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_class_daily_class_date
    ON class_daily_attendance_summaries (class_id, date);
CREATE INDEX IF NOT EXISTS idx_class_daily_academic_term
    ON class_daily_attendance_summaries (academic_term_id);

DROP TRIGGER IF EXISTS trg_class_daily_attendance_summaries_updated_at
    ON class_daily_attendance_summaries;
CREATE TRIGGER trg_class_daily_attendance_summaries_updated_at
    BEFORE UPDATE ON class_daily_attendance_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE class_daily_attendance_summaries IS
    'Materialised rollup of attendance records per class per date. Populated
     by incremental background tasks triggered when all attendance for a
     class-date is marked (or on a timeout). "Total enrolled" is derived from
     distinct students who have attendance_records rows that day, not from
     cbc_student_enrollments.status, because enrollment status has no effective
     date within a term — a student suspended on day 50 would otherwise vanish
     from every earlier day too. This is a documented workaround, not a perfect fix.';

COMMENT ON COLUMN class_daily_attendance_summaries.daily_attendance_rate IS
    'Calculated as (present_count / (present_count + absent_count + late_count + excused_count)) * 100,
     stored as a decimal with two fractional digits (e.g. 94.60).';

-- Migration: 000006_create_student_term_subject_summaries
-- Creates the student_term_subject_summaries materialised table and the
-- PostgreSQL function + triggers that keep it in sync when assessment
-- sessions are published.
--
-- Grain: (student_id, academic_term_id, learning_area_id)
--
-- This table is a blended rollup of quantitative scores and rubric outcome
-- grades across all PUBLISHED assessment sessions for a given student,
-- term, and learning area.
--
-- Quantitative scores contribute their calculated_percentage directly.
-- Rubric outcome grades are converted to a percentage using the
-- grading_scale_ranges.default_percentage_mapping for the awarded level.
-- Both sources are then averaged together into average_percentage.
--
-- The has_quantitative_data and has_rubric_data flags let the UI render
-- the result honestly — a blended average from rubric-only data implies
-- false precision that these flags help the report avoid.
