-- Migration: 000039_attendance_records
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: attendance_records

CREATE TABLE IF NOT EXISTS attendance_records (
    id                UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID              NOT NULL,
    school_id         UUID              NOT NULL,
    student_id        UUID              NOT NULL,
    timetable_slot_id UUID              NOT NULL,
    academic_term_id  UUID              NOT NULL,
    date              DATE              NOT NULL,
    status            attendance_status NOT NULL,
    marked_by         UUID              NOT NULL,
    marked_at         TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    note              TEXT              NULL,
    attendance_session_id UUID NULL,
    created_at        TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_attendance_student_slot_date
        UNIQUE (student_id, timetable_slot_id, date),
    CONSTRAINT fk_attendance_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_attendance_timetable_slot
        FOREIGN KEY (tenant_id, timetable_slot_id)
        REFERENCES cbc_timetable_slots(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_attendance_tenant_session
        FOREIGN KEY (tenant_id, attendance_session_id)
        REFERENCES cbc_attendance_sessions(tenant_id, id) ON DELETE SET NULL (attendance_session_id),
    CONSTRAINT fk_attendance_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_attendance_marked_by
        FOREIGN KEY (tenant_id, marked_by)
        REFERENCES users(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_attendance_slot_date
    ON attendance_records (timetable_slot_id, date);
CREATE INDEX IF NOT EXISTS idx_attendance_student_term
    ON attendance_records (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_attendance_tenant
    ON attendance_records (tenant_id);
CREATE INDEX IF NOT EXISTS idx_attendance_school
    ON attendance_records (school_id);

COMMENT ON TABLE attendance_records IS
    'Per-student, per-timetable-slot, per-date attendance records. The unique
     constraint (student_id, timetable_slot_id, date) prevents duplicate marks.
     Only created for slots where timetable_structures.is_break = false.';

COMMENT ON COLUMN attendance_records.note IS
    'Optional short free text (e.g. "left early, picked up by parent").';

-- ============================================================================
-- updated_at TRIGGER + NON-BREAK CONSTRAINT (squashed from 000003)
-- The updated_at column is already in the CREATE TABLE above.
-- ============================================================================

DROP TRIGGER IF EXISTS trg_attendance_records_updated_at ON attendance_records;
CREATE TRIGGER trg_attendance_records_updated_at
    BEFORE UPDATE ON attendance_records
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON COLUMN attendance_records.updated_at IS
    'Tracks when the record was last modified (status or note update).
     Populated automatically by the trg_attendance_records_updated_at trigger.';

-- ---------------------------------------------------------------------------
-- Non-break slot enforcement: attendance can only be marked for instructional
-- periods, not breaks, recess, or assemblies.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION fn_check_non_break_slot()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM cbc_timetable_slots ts
        JOIN timetable_structures tstr ON tstr.id = ts.structure_id
        WHERE ts.id = NEW.timetable_slot_id
          AND tstr.is_break = true
    ) THEN
        RAISE EXCEPTION 'Cannot create attendance record for a break period (timetable_slot_id: %)', NEW.timetable_slot_id
            USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_attendance_check_non_break_slot ON attendance_records;
CREATE TRIGGER trg_attendance_check_non_break_slot
    BEFORE INSERT OR UPDATE ON attendance_records
    FOR EACH ROW
    EXECUTE FUNCTION fn_check_non_break_slot();

COMMENT ON TRIGGER trg_attendance_check_non_break_slot ON attendance_records IS
    'Enforces that attendance records can only reference timetable slots
     whose corresponding timetable_structures row has is_break = false.
     Prevents system or application bugs from creating attendance marks
     for break/assembly/recess periods.';



CREATE INDEX IF NOT EXISTS idx_attendance_records_session
    ON attendance_records (attendance_session_id);

COMMENT ON COLUMN attendance_records.attendance_session_id IS
    'Optional reference to the cbc_attendance_sessions row. Populated when
     session is marked as SKIPPED to link existing records. NULL for normal
     (non-skipped) attendance marks.';
