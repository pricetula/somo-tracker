-- Migration: 000016_create_class_attendance_rollups
-- Creates two class-grain attendance rollup tables, refreshed exclusively
-- by Asynq background jobs (no cascading DB triggers fire on high-frequency
-- attendance_records writes — the hot write path is untouched):
--
--   1. class_learning_area_term_summaries
--      Per (class_id, learning_area_id, academic_term_id) — rolls up the
--      student-grain attendance_term_summaries into class-grain so admin
--      and teacher reports can answer "which subjects have the worst
--      attendance for this class this term" without aggregating across
--      every individual student row at report-render time.
--
--   2. class_term_attendance_summaries
--      Per (class_id, academic_term_id) — rolls up the day-grain
--      class_daily_attendance_summaries into term-grain so admin reports
--      can answer "what was class X's attendance rate for the whole term"
--      without summing 60-90 daily rows at report-render time.
--
-- Both tables are populated exclusively by Asynq tasks
-- `attendance:refresh_class_learning_area_term_summary` and
-- `attendance:refresh_class_term_summary` (defined in
-- backend/internal/attendance/worker.go). These jobs are enqueued from
-- inside the upstream refresh handlers (handleAttendanceTermRefresh and
-- handleClassDailyRefresh) so the rollup runs *after* its source table is
-- up-to-date, not on an independent timer.

-- ============================================================================
-- TABLE 1: class_learning_area_term_summaries
-- ============================================================================

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
        FOREIGN KEY (learning_area_id)
        REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
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

-- ============================================================================
-- RLS POLICIES
-- ============================================================================

ALTER TABLE IF EXISTS class_learning_area_term_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON class_learning_area_term_summaries;
    CREATE POLICY tenant_isolation_policy ON class_learning_area_term_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE class_learning_area_term_summaries IS
    'Class-grain rollup of attendance_term_summaries per (class, learning_area,
     academic_term). RLS-enabled — tenant-scoped.';

ALTER TABLE IF EXISTS class_term_attendance_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON class_term_attendance_summaries;
    CREATE POLICY tenant_isolation_policy ON class_term_attendance_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE class_term_attendance_summaries IS
    'Term-grain rollup of class_daily_attendance_summaries per
     (class, academic_term). RLS-enabled — tenant-scoped.';
