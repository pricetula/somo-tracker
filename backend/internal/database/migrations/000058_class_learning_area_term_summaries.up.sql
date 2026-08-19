-- Migration: 000058_class_learning_area_term_summaries
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: class_learning_area_term_summaries

CREATE TABLE IF NOT EXISTS class_learning_area_term_summaries (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID          NOT NULL,
    school_id             UUID          NOT NULL,
    class_id              UUID          NOT NULL,
    learning_area_id      UUID          NOT NULL,
    academic_term_id      UUID          NOT NULL,
    academic_year_id      UUID          NOT NULL,
    students_included     INT           NOT NULL DEFAULT 0,
    periods_total         INT           NOT NULL DEFAULT 0,
    periods_present       INT           NOT NULL DEFAULT 0,
    periods_absent        INT           NOT NULL DEFAULT 0,
    periods_late          INT           NOT NULL DEFAULT 0,
    periods_excused       INT           NOT NULL DEFAULT 0,
    attendance_percentage NUMERIC(5,2)  NOT NULL DEFAULT 0.00,
    last_refreshed_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_class_learning_area_term
        UNIQUE (class_id, learning_area_id, academic_term_id),
    CONSTRAINT fk_class_la_term_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_la_term_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_la_term_learning_area
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_la_term_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_class_la_term_tenant
    ON class_learning_area_term_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_class_la_term_school
    ON class_learning_area_term_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_class_la_term_class_term
    ON class_learning_area_term_summaries (class_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_class_la_term_area_term
    ON class_learning_area_term_summaries (learning_area_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_class_la_term_year
    ON class_learning_area_term_summaries (academic_year_id);

DROP TRIGGER IF EXISTS trg_class_learning_area_term_summaries_updated_at
    ON class_learning_area_term_summaries;
CREATE TRIGGER trg_class_learning_area_term_summaries_updated_at
    BEFORE UPDATE ON class_learning_area_term_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE class_learning_area_term_summaries IS
    'Class-grain rollup of attendance_term_summaries per (class, learning_area,
     academic_term). Refreshed exclusively by the Asynq task
     attendance:refresh_class_learning_area_term_summary (enqueued from
     inside handleAttendanceTermRefresh so it runs after the student-grain
     rollup is current). No cascading DB triggers fire on
     attendance_term_summaries writes — this keeps the hot attendance
     marking path predictable.';

COMMENT ON COLUMN class_learning_area_term_summaries.students_included IS
    'Count of distinct students whose attendance_term_summaries rows
     contributed to this (class, learning_area, term) aggregate. May be less
     than the total enrolled count for the class because not every enrolled
     student necessarily has an attendance_term_summaries row for every
     subject (e.g. subject not yet taught, no periods in the term).';

COMMENT ON COLUMN class_learning_area_term_summaries.attendance_percentage IS
    'Calculated as (periods_present / periods_total) * 100, stored as a
     decimal with two fractional digits (e.g. 92.50). Matches the formula
     used in attendance_term_summaries.attendance_percentage.';

COMMENT ON COLUMN class_learning_area_term_summaries.last_refreshed_at IS
    'Timestamp of the most recent successful Asynq refresh of this row.
     Report generators must surface this value so consumers can flag
     "data as of X" — it is NOT refreshed automatically by DB triggers.';

-- ============================================================================
-- TABLE 2: class_term_attendance_summaries
-- ============================================================================
