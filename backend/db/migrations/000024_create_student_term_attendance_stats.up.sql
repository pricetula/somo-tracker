-- Migration: 000024_create_student_term_attendance_stats
-- Purpose: Create summary table for student term attendance stats per ADR-2026-10-15-stats-architecture and docs/planning/stats-summary-tables-plan.md
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

CREATE TABLE student_term_attendance_stats (
    school_id UUID NOT NULL,
    academic_term_id UUID NOT NULL,
    student_id UUID NOT NULL,
    present_count INTEGER NOT NULL DEFAULT 0,
    absent_count INTEGER NOT NULL DEFAULT 0,
    late_count INTEGER NOT NULL DEFAULT 0,
    attendance_rate NUMERIC(5,2),
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (school_id, academic_term_id, student_id),
    FOREIGN KEY (school_id) REFERENCES schools(id) ON DELETE CASCADE,
    FOREIGN KEY (academic_term_id) REFERENCES academic_terms(id) ON DELETE CASCADE,
    FOREIGN KEY (student_id) REFERENCES students(student_id) ON DELETE CASCADE
);

CREATE INDEX student_term_attendance_stats_student_id_idx ON student_term_attendance_stats (student_id);
CREATE INDEX student_term_attendance_stats_school_term_idx ON student_term_attendance_stats (school_id, academic_term_id);

COMMENT ON TABLE student_term_attendance_stats IS 'Summary table for student attendance per term, updated by background workers.';
COMMENT ON COLUMN student_term_attendance_stats.attendance_rate IS 'Percentage present, computed as present / (present+absent+late) * 100';
