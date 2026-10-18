-- Migration: 000025_create_guardian_link_stats
-- Purpose: Create summary table for guardian links per stats architecture
CREATE TABLE guardian_link_stats (
    school_id UUID NOT NULL,
    student_id UUID NOT NULL,
    guardian_count INTEGER NOT NULL DEFAULT 0,
    is_linked BOOLEAN NOT NULL DEFAULT FALSE,
    primary_guardian_id UUID,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (school_id, student_id),
    FOREIGN KEY (school_id) REFERENCES schools(id) ON DELETE CASCADE,
    FOREIGN KEY (student_id) REFERENCES students(student_id) ON DELETE CASCADE
);

CREATE INDEX guardian_link_stats_student_id_idx ON guardian_link_stats (student_id);
COMMENT ON TABLE guardian_link_stats IS 'Summary table for guardian links per student, updated by background workers.';
