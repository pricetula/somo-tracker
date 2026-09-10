-- Migration: 000008_create_timetable_scheduling
-- Purpose: Create the timetable & scheduling schema:
--   timetable_templates (bell schedule configurations), time_slots (periods/breaks),
--   rooms (physical facilities), class_timetable_slots (weekly recurring assignments),
--   timetable_substitutions (absence coverage overrides).
--
-- Dependencies:
--   - 000001_init_extensions (pgcrypto for gen_random_uuid()).
--   - 000004_create_sis_schema (provides schools, school_memberships tables).
--   - 000005_create_subject_hierarchy (provides subjects table).
--   - 000006_create_academic_calendar (provides academic_terms table).
--   - 000007_create_class_rooms_and_enrollments (provides class_rooms table).
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- ============================================================================
-- Section 1: substitution_status enum
-- ============================================================================

CREATE TYPE substitution_status AS ENUM ('PENDING', 'ASSIGNED', 'COMPLETED', 'CANCELLED');

-- ============================================================================
-- Section 2: room_type enum
-- ============================================================================

CREATE TYPE room_type AS ENUM ('STANDARD', 'SCIENCE_LAB', 'COMPUTER_LAB', 'GYM');

-- ============================================================================
-- Section 3: timetable_templates
-- ============================================================================

-- Parent container for a distinct bell schedule configuration.
-- Examples: "Standard 6-Period Day", "Morning Shift 3-Lesson", "Half-Day Schedule".
CREATE TABLE timetable_templates (
    id          UUID        NOT NULL    DEFAULT gen_random_uuid()
                                     PRIMARY KEY,
    school_id   UUID        NOT NULL    REFERENCES schools(id)
                                     ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: timetable_templates_school_id_idx supports school-scoped template listings.
CREATE INDEX timetable_templates_school_id_idx ON timetable_templates (school_id);

-- Index: timetable_templates_name_idx supports name-based lookups.
CREATE INDEX timetable_templates_name_idx ON timetable_templates (name);

-- Constraint: timetable_templates_school_name_uniq ensures a school cannot have
-- duplicate template names.
ALTER TABLE timetable_templates ADD CONSTRAINT timetable_templates_school_name_uniq
    UNIQUE (school_id, name);

-- ============================================================================
-- Section 4: time_slots
-- ============================================================================

-- Individual periods or breaks belonging to a specific template.
-- Examples: "Period 1" (08:00–08:40), "Morning Break" (10:00–10:15).
CREATE TABLE time_slots (
    id                    UUID        NOT NULL    DEFAULT gen_random_uuid()
                                             PRIMARY KEY,
    school_id             UUID        NOT NULL    REFERENCES schools(id)
                                             ON DELETE CASCADE,
    timetable_template_id UUID        NOT NULL    REFERENCES timetable_templates(id)
                                             ON DELETE CASCADE,
    name                  VARCHAR(255) NOT NULL,
    start_time            TIME        NOT NULL,
    end_time              TIME        NOT NULL,
    sequence_index        INTEGER      NOT NULL,
    is_instructional      BOOLEAN      NOT NULL    DEFAULT TRUE,
    created_at            TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: time_slots_school_id_idx supports school-scoped queries.
CREATE INDEX time_slots_school_id_idx ON time_slots (school_id);

-- Index: time_slots_timetable_template_id_idx supports template-scoped slot listings.
CREATE INDEX time_slots_timetable_template_id_idx ON time_slots (timetable_template_id);

-- Index: time_slots_sequence_idx covers ordered slot retrieval (template ordered by sequence_index).
CREATE INDEX time_slots_sequence_idx ON time_slots (timetable_template_id, sequence_index);

-- Constraint: time_slots_template_seq_uniq ensures sequence_index is unique within a template.
ALTER TABLE time_slots ADD CONSTRAINT time_slots_template_seq_uniq
    UNIQUE (timetable_template_id, sequence_index);

-- Constraint: time_slots_time_order ensures end_time is after start_time.
ALTER TABLE time_slots ADD CONSTRAINT time_slots_time_order
    CHECK (end_time > start_time);

-- ============================================================================
-- Section 5: rooms
-- ============================================================================

-- Physical facilities and campus locations used to prevent room overbooking.
-- Distinct from class_rooms (which are operational containers per year/stream).
-- Examples: "Lab A", "Room 204", "Gymnasium".
CREATE TABLE rooms (
    id         UUID        NOT NULL    DEFAULT gen_random_uuid()
                                  PRIMARY KEY,
    school_id  UUID        NOT NULL    REFERENCES schools(id)
                                  ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    capacity   INTEGER,
    room_type  room_type    NOT NULL    DEFAULT 'STANDARD',
    created_at TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: rooms_school_id_idx supports school-scoped room listings.
CREATE INDEX rooms_school_id_idx ON rooms (school_id);

-- Index: rooms_room_type_idx supports filtering by room category.
CREATE INDEX rooms_room_type_idx ON rooms (room_type);

-- Index: rooms_capacity_idx supports capacity-based room lookups.
CREATE INDEX rooms_capacity_idx ON rooms (capacity);

-- Constraint: rooms_school_name_uniq ensures a room name is unique per school.
ALTER TABLE rooms ADD CONSTRAINT rooms_school_name_uniq
    UNIQUE (school_id, name);

-- ============================================================================
-- Section 6: class_timetable_slots
-- ============================================================================

-- Maps a class_room to an academic term, assigning subjects, teachers, and rooms
-- to a template's time slots for a specific day of the week.
CREATE TABLE class_timetable_slots (
    id                      UUID        NOT NULL    DEFAULT gen_random_uuid()
                                               PRIMARY KEY,
    school_id               UUID        NOT NULL    REFERENCES schools(id)
                                               ON DELETE CASCADE,
    class_room_id           UUID        NOT NULL    REFERENCES class_rooms(id)
                                               ON DELETE CASCADE,
    academic_term_id        UUID        NOT NULL    REFERENCES academic_terms(id)
                                               ON DELETE CASCADE,
    day_of_week             INTEGER      NOT NULL    CHECK (day_of_week BETWEEN 1 AND 7),
    time_slot_id            UUID        NOT NULL    REFERENCES time_slots(id)
                                               ON DELETE CASCADE,
    subject_id              UUID        NOT NULL    REFERENCES subjects(id)
                                               ON DELETE CASCADE,
    teacher_membership_id   UUID        NOT NULL    REFERENCES school_memberships(id)
                                               ON DELETE CASCADE,
    room_id                 UUID        REFERENCES rooms(id)
                                               ON DELETE SET NULL,
    created_at              TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: class_timetable_slots_school_id_idx supports school-scoped queries.
CREATE INDEX class_timetable_slots_school_id_idx ON class_timetable_slots (school_id);

-- Index: class_timetable_slots_class_room_id_idx supports class_room timetable queries.
CREATE INDEX class_timetable_slots_class_room_id_idx ON class_timetable_slots (class_room_id);

-- Index: class_timetable_slots_academic_term_id_idx supports term-scoped timetable queries.
CREATE INDEX class_timetable_slots_academic_term_id_idx ON class_timetable_slots (academic_term_id);

-- Index: class_timetable_slots_teacher_membership_id_idx supports teacher schedule lookups.
CREATE INDEX class_timetable_slots_teacher_membership_id_idx ON class_timetable_slots (teacher_membership_id);

-- Index: class_timetable_slots_room_id_idx supports room booking queries.
CREATE INDEX class_timetable_slots_room_id_idx ON class_timetable_slots (room_id);

-- Index: class_timetable_slots_day_time_idx covers day+time ordered retrieval.
CREATE INDEX class_timetable_slots_day_time_idx ON class_timetable_slots (class_room_id, day_of_week, time_slot_id);

-- Anti-clashing constraint: a teacher cannot be double-booked in the exact same
-- time slot across different classes or templates during the same term.
ALTER TABLE class_timetable_slots ADD CONSTRAINT class_timetable_slots_teacher_no_clash
    UNIQUE (teacher_membership_id, academic_term_id, day_of_week, time_slot_id);

-- Constraint: class_timetable_slots_class_day_time_uniq ensures a class_room
-- has at most one subject per slot per day.
ALTER TABLE class_timetable_slots ADD CONSTRAINT class_timetable_slots_class_day_time_uniq
    UNIQUE (class_room_id, academic_term_id, day_of_week, time_slot_id);

-- ============================================================================
-- Section 7: timetable_substitutions
-- ============================================================================

-- Handles emergency or planned teacher absences on specific calendar dates
-- without modifying the master weekly recurring timetable.
CREATE TABLE timetable_substitutions (
    id                            UUID        NOT NULL    DEFAULT gen_random_uuid()
                                                       PRIMARY KEY,
    school_id                     UUID        NOT NULL    REFERENCES schools(id)
                                                       ON DELETE CASCADE,
    class_timetable_slot_id       UUID        NOT NULL    REFERENCES class_timetable_slots(id)
                                                       ON DELETE CASCADE,
    substitution_date             DATE        NOT NULL,
    original_teacher_membership_id UUID      NOT NULL    REFERENCES school_memberships(id)
                                                       ON DELETE CASCADE,
    substitute_teacher_membership_id UUID     REFERENCES school_memberships(id)
                                                       ON DELETE SET NULL,
    status                        substitution_status NOT NULL DEFAULT 'PENDING',
    reason                        TEXT,
    created_at                    TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at                    TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: timetable_substitutions_school_id_idx supports school-scoped queries.
CREATE INDEX timetable_substitutions_school_id_idx ON timetable_substitutions (school_id);

-- Index: timetable_substitutions_class_timetable_slot_id_idx supports slot override lookups.
CREATE INDEX timetable_substitutions_class_timetable_slot_id_idx ON timetable_substitutions (class_timetable_slot_id);

-- Index: timetable_substitutions_substitution_date_idx supports date-based queries.
CREATE INDEX timetable_substitutions_substitution_date_idx ON timetable_substitutions (substitution_date);

-- Index: timetable_substitutions_original_teacher_idx supports absence tracking by teacher.
CREATE INDEX timetable_substitutions_original_teacher_idx ON timetable_substitutions (original_teacher_membership_id);

-- Index: timetable_substitutions_substitute_teacher_idx supports coverage tracking.
CREATE INDEX timetable_substitutions_substitute_teacher_idx ON timetable_substitutions (substitute_teacher_membership_id);

-- Index: timetable_substitutions_status_idx supports status filtering.
CREATE INDEX timetable_substitutions_status_idx ON timetable_substitutions (status);

-- Composite index: timetable_substitutions_date_status_idx covers pending
-- substitutions by date for daily coverage processing.
CREATE INDEX timetable_substitutions_date_status_idx ON timetable_substitutions (substitution_date, status);

-- Constraint: timetable_substitutions_slot_date_uniq ensures only one substitution
-- record per slot per date (prevents duplicate substitution requests).
ALTER TABLE timetable_substitutions ADD CONSTRAINT timetable_substitutions_slot_date_uniq
    UNIQUE (class_timetable_slot_id, substitution_date);

-- ============================================================================
-- Section 8: updated_at triggers
-- ============================================================================

CREATE TRIGGER timetable_templates_updated_at_trg
    BEFORE UPDATE ON timetable_templates
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER time_slots_updated_at_trg
    BEFORE UPDATE ON time_slots
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER rooms_updated_at_trg
    BEFORE UPDATE ON rooms
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER class_timetable_slots_updated_at_trg
    BEFORE UPDATE ON class_timetable_slots
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER timetable_substitutions_updated_at_trg
    BEFORE UPDATE ON timetable_substitutions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- Section 9: Row-Level Security (RLS)
-- ============================================================================

-- RLS is enabled on all timetable tables to enforce multi-tenant isolation.
-- The same pattern as student_class_enrollments: tenant isolation via
-- schools.tenant_id compared against session variable app.current_tenant_id.

ALTER TABLE timetable_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE timetable_templates FORCE ROW LEVEL SECURITY;

ALTER TABLE time_slots ENABLE ROW LEVEL SECURITY;
ALTER TABLE time_slots FORCE ROW LEVEL SECURITY;

ALTER TABLE rooms ENABLE ROW LEVEL SECURITY;
ALTER TABLE rooms FORCE ROW LEVEL SECURITY;

ALTER TABLE class_timetable_slots ENABLE ROW LEVEL SECURITY;
ALTER TABLE class_timetable_slots FORCE ROW LEVEL SECURITY;

ALTER TABLE timetable_substitutions ENABLE ROW LEVEL SECURITY;
ALTER TABLE timetable_substitutions FORCE ROW LEVEL SECURITY;

-- Policy template: <table>_tenant_isolation
-- Every row-level operation is gated by a comparison ensuring the row's
-- school_id belongs to a school operated by the current tenant.

CREATE POLICY timetable_templates_tenant_isolation
    ON timetable_templates
    FOR ALL TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = timetable_templates.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = timetable_templates.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    );

CREATE POLICY time_slots_tenant_isolation
    ON time_slots
    FOR ALL TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = time_slots.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = time_slots.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    );

CREATE POLICY rooms_tenant_isolation
    ON rooms
    FOR ALL TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = rooms.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = rooms.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    );

CREATE POLICY class_timetable_slots_tenant_isolation
    ON class_timetable_slots
    FOR ALL TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = class_timetable_slots.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = class_timetable_slots.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    );

CREATE POLICY timetable_substitutions_tenant_isolation
    ON timetable_substitutions
    FOR ALL TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = timetable_substitutions.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = timetable_substitutions.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    );

-- ============================================================================
-- Section 10: Comments (documentation)
-- ============================================================================

COMMENT ON TYPE substitution_status IS 'Lifecycle state for substitutions: PENDING (awaiting assignment), ASSIGNED (cover teacher confirmed), COMPLETED (coverage done), CANCELLED (substitution revoked).';
COMMENT ON TYPE room_type IS 'Category of room: STANDARD (regular classroom), SCIENCE_LAB, COMPUTER_LAB, GYM.';

COMMENT ON TABLE timetable_templates IS 'Parent container for a distinct bell schedule configuration (e.g., "Standard 6-Period Day", "Morning Shift 3-Lesson").';
COMMENT ON COLUMN timetable_templates.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN timetable_templates.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN timetable_templates.name IS 'Name of the template (e.g., "Primary Schedule", "Standard 6-Period Day").';
COMMENT ON COLUMN timetable_templates.description IS 'Optional details about when or who uses this template.';
COMMENT ON COLUMN timetable_templates.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN timetable_templates.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE time_slots IS 'Individual periods or breaks belonging to a specific template (e.g., Period 1, Morning Break).';
COMMENT ON COLUMN time_slots.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN time_slots.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN time_slots.timetable_template_id IS 'FK to timetable_templates(id). Cascades on template delete.';
COMMENT ON COLUMN time_slots.name IS 'Period label (e.g., Period 1, Morning Break).';
COMMENT ON COLUMN time_slots.start_time IS 'Slot start time (e.g., 08:00:00).';
COMMENT ON COLUMN time_slots.end_time IS 'Slot end time (e.g., 08:40:00).';
COMMENT ON COLUMN time_slots.sequence_index IS 'Order of the slot within the template (1, 2, 3...).';
COMMENT ON COLUMN time_slots.is_instructional IS 'True for classes, False for breaks/recess. Defaults to TRUE.';
COMMENT ON COLUMN time_slots.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN time_slots.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE rooms IS 'Physical facilities and campus locations to prevent room overbooking (e.g., Lab A, Room 204). Distinct from class_rooms.';
COMMENT ON COLUMN rooms.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN rooms.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN rooms.name IS 'Room identifier or name (e.g., Lab A, Room 204). Unique per school.';
COMMENT ON COLUMN rooms.capacity IS 'Maximum student capacity the room can hold. Optional.';
COMMENT ON COLUMN rooms.room_type IS 'Category of room (STANDARD, SCIENCE_LAB, COMPUTER_LAB, GYM).';
COMMENT ON COLUMN rooms.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN rooms.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE class_timetable_slots IS 'Maps a classroom to an academic term, assigning subjects, teachers, and rooms to a template time slots for a specific day.';
COMMENT ON COLUMN class_timetable_slots.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN class_timetable_slots.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN class_timetable_slots.class_room_id IS 'FK to class_rooms(id). The operational classroom. Cascades on class_room delete.';
COMMENT ON COLUMN class_timetable_slots.academic_term_id IS 'FK to academic_terms(id). The academic term. Cascades on term delete.';
COMMENT ON COLUMN class_timetable_slots.day_of_week IS 'Day index (1 = Monday through 7 = Sunday).';
COMMENT ON COLUMN class_timetable_slots.time_slot_id IS 'FK to time_slots(id). The period within the day. Cascades on slot delete.';
COMMENT ON COLUMN class_timetable_slots.subject_id IS 'FK to subjects(id). The taught subject. Cascades on subject delete.';
COMMENT ON COLUMN class_timetable_slots.teacher_membership_id IS 'FK to school_memberships(id) for the assigned teacher. Cascades on membership delete.';
COMMENT ON COLUMN class_timetable_slots.room_id IS 'FK to rooms(id). Optional physical location constraint. Cascades on room delete.';
COMMENT ON COLUMN class_timetable_slots.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN class_timetable_slots.updated_at IS 'UTC timestamp of last modification.';

COMMENT ON TABLE timetable_substitutions IS 'Handles emergency or planned teacher absences on specific calendar dates without modifying the master weekly recurring timetable.';
COMMENT ON COLUMN timetable_substitutions.id IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN timetable_substitutions.school_id IS 'FK to schools(id). Cascades on school delete.';
COMMENT ON COLUMN timetable_substitutions.class_timetable_slot_id IS 'FK to class_timetable_slots(id) being overridden. Cascades on slot delete.';
COMMENT ON COLUMN timetable_substitutions.substitution_date IS 'The precise calendar date of the absence/coverage.';
COMMENT ON COLUMN timetable_substitutions.original_teacher_membership_id IS 'FK to school_memberships(id) for the teacher who is away.';
COMMENT ON COLUMN timetable_substitutions.substitute_teacher_membership_id IS 'FK to school_memberships(id) for the covering teacher. NULL if unassigned.';
COMMENT ON COLUMN timetable_substitutions.status IS 'Lifecycle state: PENDING, ASSIGNED, COMPLETED, or CANCELLED.';
COMMENT ON COLUMN timetable_substitutions.reason IS 'Optional explanation (e.g., Medical leave).';
COMMENT ON COLUMN timetable_substitutions.created_at IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN timetable_substitutions.updated_at IS 'UTC timestamp of last modification.';