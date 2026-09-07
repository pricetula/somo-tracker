-- Migration: 000009_create_attendance_tracking
-- Purpose: Primary and event attendance tracking layer.
--   timetable_attendance links to specific timetable slot instances.
--   event_attendance tracks presence at school-wide activities.
--
-- Dependencies:
--   - 000002_create_tenants_and_users (schools, students, school_memberships).
--   - 000006_create_academic_calendar (school_events).
--   - 000008_create_timetable_scheduling (class_timetable_slots).

-- ============================================================================
-- Section 1: Attendance status enums
-- ============================================================================

CREATE TYPE timetable_attendance_status AS ENUM ('PRESENT', 'ABSENT', 'LATE', 'EXCUSED');
CREATE TYPE event_attendance_status AS ENUM ('PRESENT', 'ABSENT', 'EXCUSED');

-- ============================================================================
-- Section 2: timetable_attendance
-- ============================================================================

CREATE TABLE timetable_attendance (
    id                          UUID        NOT NULL    DEFAULT gen_random_uuid()
                                                         PRIMARY KEY,
    school_id                   UUID        NOT NULL    REFERENCES schools(id)
                                                         ON DELETE CASCADE,
    student_id                  UUID        NOT NULL    REFERENCES students(student_id)
                                                         ON DELETE CASCADE,
    class_timetable_slot_id     UUID        NOT NULL    REFERENCES class_timetable_slots(id)
                                                         ON DELETE CASCADE,
    attendance_date             DATE        NOT NULL,
    status                      timetable_attendance_status NOT NULL,
    remarks                     TEXT,
    recorded_by_membership_id   UUID        NOT NULL    REFERENCES school_memberships(id)
                                                         ON DELETE CASCADE,
    created_at                  TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: timetable_attendance_school_id_idx supports school-scoped attendance queries.
CREATE INDEX timetable_attendance_school_id_idx ON timetable_attendance (school_id);

-- Index: timetable_attendance_student_id_idx supports student attendance history.
CREATE INDEX timetable_attendance_student_id_idx ON timetable_attendance (student_id);

-- Index: timetable_attendance_slot_id_idx supports timetable slot attendance lookups.
CREATE INDEX timetable_attendance_slot_id_idx ON timetable_attendance (class_timetable_slot_id);

-- Index: timetable_attendance_date_idx supports calendar-date-based reporting.
CREATE INDEX timetable_attendance_date_idx ON timetable_attendance (attendance_date);

-- Index: timetable_attendance_recorded_by_idx supports audit trails by recording teacher.
CREATE INDEX timetable_attendance_recorded_by_idx ON timetable_attendance (recorded_by_membership_id);

-- Composite index: timetable_attendance_slot_date_idx covers attendance by slot and date.
CREATE INDEX timetable_attendance_slot_date_idx ON timetable_attendance (class_timetable_slot_id, attendance_date);

-- Composite index: timetable_attendance_student_date_idx supports student-day attendance listings.
CREATE INDEX timetable_attendance_student_date_idx ON timetable_attendance (student_id, attendance_date);

-- Constraint: one attendance entry per student per timetable slot per calendar date.
ALTER TABLE timetable_attendance ADD CONSTRAINT timetable_attendance_uniq_student_slot_date
    UNIQUE (student_id, class_timetable_slot_id, attendance_date);

-- ============================================================================
-- Section 3: event_attendance
-- ============================================================================

CREATE TABLE event_attendance (
    id                  UUID        NOT NULL    DEFAULT gen_random_uuid()
                                     PRIMARY KEY,
    school_id           UUID        NOT NULL    REFERENCES schools(id)
                                     ON DELETE CASCADE,
    student_id          UUID        NOT NULL    REFERENCES students(student_id)
                                     ON DELETE CASCADE,
    school_event_id     UUID        NOT NULL    REFERENCES school_events(id)
                                     ON DELETE CASCADE,
    attendance_date     DATE        NOT NULL,
    status              event_attendance_status NOT NULL,
    remarks             TEXT,
    created_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: event_attendance_school_id_idx supports school-scoped event attendance queries.
CREATE INDEX event_attendance_school_id_idx ON event_attendance (school_id);

-- Index: event_attendance_student_id_idx supports student event attendance history.
CREATE INDEX event_attendance_student_id_idx ON event_attendance (student_id);

-- Index: event_attendance_event_id_idx supports event attendance listings.
CREATE INDEX event_attendance_event_id_idx ON event_attendance (school_event_id);

-- Index: event_attendance_date_idx supports date-based event reporting.
CREATE INDEX event_attendance_date_idx ON event_attendance (attendance_date);

-- Composite index: event_attendance_student_event_idx covers student + event lookups.
CREATE INDEX event_attendance_student_event_idx ON event_attendance (student_id, school_event_id);

-- Constraint: one attendance entry per student per event per calendar date.
ALTER TABLE event_attendance ADD CONSTRAINT event_attendance_uniq_student_event_date
    UNIQUE (student_id, school_event_id, attendance_date);

-- ============================================================================
-- Section 4: updated_at triggers
-- ============================================================================

CREATE TRIGGER timetable_attendance_updated_at_trg
    BEFORE UPDATE ON timetable_attendance
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER event_attendance_updated_at_trg
    BEFORE UPDATE ON event_attendance
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- Section 5: Row-Level Security (RLS)
-- ============================================================================

ALTER TABLE timetable_attendance ENABLE ROW LEVEL SECURITY;
ALTER TABLE timetable_attendance FORCE ROW LEVEL SECURITY;

ALTER TABLE event_attendance ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_attendance FORCE ROW LEVEL SECURITY;

CREATE POLICY timetable_attendance_tenant_isolation
    ON timetable_attendance
    FOR ALL TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = timetable_attendance.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = timetable_attendance.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    );

CREATE POLICY event_attendance_tenant_isolation
    ON event_attendance
    FOR ALL TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = event_attendance.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = event_attendance.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    );

-- ============================================================================
-- Section 6: Comments
-- ============================================================================

COMMENT ON TYPE timetable_attendance_status IS 'Attendance state for timetable-linked lessons: PRESENT, ABSENT, LATE, or EXCUSED.';
COMMENT ON TYPE event_attendance_status IS 'Attendance state for special school events: PRESENT, ABSENT, or EXCUSED.';

COMMENT ON TABLE timetable_attendance IS 'Primary attendance tracking table linking to specific timetable slot instances on calendar dates, allowing subject teachers to record presence during instructional periods.';
COMMENT ON COLUMN timetable_attendance.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN timetable_attendance.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN timetable_attendance.student_id IS 'FK to students(student_id). Cascades on student delete.';
COMMENT ON COLUMN timetable_attendance.class_timetable_slot_id IS 'FK to class_timetable_slots(id). Cascades on slot delete.';
COMMENT ON COLUMN timetable_attendance.attendance_date IS 'The specific calendar date of the lesson instance.';
COMMENT ON COLUMN timetable_attendance.status IS 'Attendance state: PRESENT, ABSENT, LATE, or EXCUSED.';
COMMENT ON COLUMN timetable_attendance.remarks IS 'Optional notes (e.g., "Left early due to illness").';
COMMENT ON COLUMN timetable_attendance.recorded_by_membership_id IS 'FK to school_memberships(id) for the teacher who took the register. Cascades on membership delete.';
COMMENT ON COLUMN timetable_attendance.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN timetable_attendance.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE event_attendance IS 'Attendance tracking for special school-wide activities (sports days, symposia) where the regular timetable is suspended.';
COMMENT ON COLUMN event_attendance.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN event_attendance.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN event_attendance.student_id IS 'FK to students(student_id). Cascades on student delete.';
COMMENT ON COLUMN event_attendance.school_event_id IS 'FK to school_events(id). Cascades on event delete.';
COMMENT ON COLUMN event_attendance.attendance_date IS 'The calendar date of the event.';
COMMENT ON COLUMN event_attendance.status IS 'Attendance state: PRESENT, ABSENT, or EXCUSED.';
COMMENT ON COLUMN event_attendance.remarks IS 'Optional details about event attendance.';
COMMENT ON COLUMN event_attendance.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN event_attendance.updated_at IS 'UTC timestamp of last modification.';
