-- Migration: 000042_attendance_term_summaries
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: attendance_term_summaries

CREATE TABLE IF NOT EXISTS attendance_term_summaries (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID          NOT NULL,
    school_id            UUID          NOT NULL,
    student_id           UUID          NOT NULL,
    academic_term_id     UUID          NOT NULL,
    learning_area_id     UUID          NOT NULL,
    periods_total        INT           NOT NULL,
    periods_present      INT           NOT NULL,
    periods_absent       INT           NOT NULL,
    periods_late         INT           NOT NULL,
    periods_excused      INT           NOT NULL,
    attendance_percentage NUMERIC(5,2) NOT NULL,
    last_refreshed_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    academic_year_id     UUID          NOT NULL,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_summaries_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_summaries_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_summaries_learning_area
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_summary_student_term_area
        UNIQUE (student_id, academic_term_id, learning_area_id),
    CONSTRAINT fk_summaries_tenant_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_att_summaries_student_term
    ON attendance_term_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_att_summaries_tenant
    ON attendance_term_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_att_summaries_school
    ON attendance_term_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_att_summaries_academic_year
    ON attendance_term_summaries (academic_year_id);

COMMENT ON TABLE attendance_term_summaries IS
    'Materialised rollup of attendance records per student per term per learning
     area. Populated by a background task (nightly or on-demand when an admin
     generates a term report). Not authoritative — attendance_records is the
     source of truth for all attendance calculations.';

COMMENT ON COLUMN attendance_term_summaries.attendance_percentage IS
    'Calculated as (periods_present / periods_total) * 100, stored as a
     decimal with two fractional digits (e.g. 92.50).';




CREATE TRIGGER trg_attendance_term_summaries_updated_at
    BEFORE UPDATE ON attendance_term_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN attendance_term_summaries.updated_at IS
    'Tracks materialised summary refresh cycles.';
