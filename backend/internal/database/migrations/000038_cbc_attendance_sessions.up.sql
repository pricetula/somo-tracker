-- Migration: 000038_cbc_attendance_sessions
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: cbc_attendance_sessions

CREATE TABLE IF NOT EXISTS cbc_attendance_sessions (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id         UUID         NOT NULL,
    timetable_slot_id UUID         NOT NULL,
    date              DATE         NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'SUBMITTED',
    skip_reason       TEXT         NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_cbc_attendance_session_status
        CHECK (status IN ('SUBMITTED', 'SKIPPED')),
    CONSTRAINT uq_cbc_attendance_sessions_slot_date
        UNIQUE (school_id, timetable_slot_id, date),
    CONSTRAINT fk_cbc_attendance_sessions_tenant_slot
        FOREIGN KEY (tenant_id, timetable_slot_id)
        REFERENCES cbc_timetable_slots(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_attendance_sessions_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_attendance_sessions_slot_date
    ON cbc_attendance_sessions (timetable_slot_id, date);
CREATE INDEX IF NOT EXISTS idx_cbc_attendance_sessions_tenant
    ON cbc_attendance_sessions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_attendance_sessions_school
    ON cbc_attendance_sessions (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_attendance_sessions_status
    ON cbc_attendance_sessions (status);

-- Composite key so attendance_records can reference (tenant_id, attendance_session_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cbc_attendance_sessions_tenant
    ON cbc_attendance_sessions (tenant_id, id);

COMMENT ON TABLE cbc_attendance_sessions IS
    'Tracks actual lesson execution instances per timetable slot and date.
     Teachers flag sessions as SKIPPED when a class did not hold (teacher
     absence, school assembly, sports day, etc.). Skipped sessions exclude
     their attendance records from terminal percentage calculations so
     students are not penalised for cancelled lessons.';

COMMENT ON COLUMN cbc_attendance_sessions.status IS
    'SUBMITTED = lesson held as scheduled (default). SKIPPED = lesson did
     not hold. Only SKIPPED sessions affect terminal attendance calculations
     by reducing the expected denominator.';

COMMENT ON COLUMN cbc_attendance_sessions.skip_reason IS
    'Teacher-provided reason when status is SKIPPED. Examples: School
     Assembly, Public Holiday, Teacher Absence, Sports/Field Event.';



CREATE TRIGGER trg_cbc_attendance_sessions_updated_at
    BEFORE UPDATE ON cbc_attendance_sessions
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_attendance_sessions.updated_at IS
    'Tracks session status changes and skip reason updates.';
