-- Migration: 000059_class_term_attendance_summaries
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: class_term_attendance_summaries

CREATE TABLE IF NOT EXISTS class_term_attendance_summaries (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID          NOT NULL,
    school_id            UUID          NOT NULL,
    class_id             UUID          NOT NULL,
    academic_term_id     UUID          NOT NULL,
    academic_year_id     UUID          NOT NULL,
    days_in_term         INT           NOT NULL DEFAULT 0,
    total_enrolled_avg   NUMERIC(6,2)  NULL,
    present_count        INT           NOT NULL DEFAULT 0,
    absent_count         INT           NOT NULL DEFAULT 0,
    late_count           INT           NOT NULL DEFAULT 0,
    excused_count        INT           NOT NULL DEFAULT 0,
    term_attendance_rate NUMERIC(5,2)  NOT NULL DEFAULT 0.00,
    last_refreshed_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_class_term_attendance UNIQUE (class_id, academic_term_id),
    CONSTRAINT fk_class_term_att_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_term_att_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_term_att_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_class_term_att_tenant
    ON class_term_attendance_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_class_term_att_school_term
    ON class_term_attendance_summaries (school_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_class_term_att_class
    ON class_term_attendance_summaries (class_id);
CREATE INDEX IF NOT EXISTS idx_class_term_att_term
    ON class_term_attendance_summaries (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_class_term_att_year
    ON class_term_attendance_summaries (academic_year_id);

DROP TRIGGER IF EXISTS trg_class_term_attendance_summaries_updated_at
    ON class_term_attendance_summaries;
CREATE TRIGGER trg_class_term_attendance_summaries_updated_at
    BEFORE UPDATE ON class_term_attendance_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE class_term_attendance_summaries IS
    'Term-grain rollup of class_daily_attendance_summaries per
     (class, academic_term). Refreshed exclusively by the Asynq task
     attendance:refresh_class_term_summary (enqueued from inside
     handleClassDailyRefresh so it runs after the daily-grain rollup is
     current). No cascading DB triggers fire on
     class_daily_attendance_summaries writes.';

COMMENT ON COLUMN class_term_attendance_summaries.days_in_term IS
    'Count of class_daily_attendance_summaries rows rolled up for this
     class/term — i.e. school days with recorded attendance, NOT calendar
     days or total days in the term date range.';

COMMENT ON COLUMN class_term_attendance_summaries.total_enrolled_avg IS
    'Average of class_daily_attendance_summaries.total_enrolled across the
     term. Enrollment fluctuates day to day; we inherit the documented
     workaround from the daily table (total_enrolled is derived from
     distinct students with attendance_records that day, not from
     cbc_student_enrollments.status, because enrollment status has no
     effective date within a term). This rollup does NOT attempt to fix
     that limitation — it preserves the same per-day enrollment snapshot
     as the source table.';

COMMENT ON COLUMN class_term_attendance_summaries.term_attendance_rate IS
    'Calculated as (present_count / (present_count + absent_count +
     late_count + excused_count)) * 100, matching the formula used in
     class_daily_attendance_summaries.daily_attendance_rate. Stored as a
     decimal with two fractional digits.';

COMMENT ON COLUMN class_term_attendance_summaries.last_refreshed_at IS
    'Timestamp of the most recent successful Asynq refresh of this row.
     Report generators must surface this value so consumers can flag
     "data as of X" — it is NOT refreshed automatically by DB triggers.';
