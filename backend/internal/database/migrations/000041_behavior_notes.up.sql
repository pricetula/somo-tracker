-- Migration: 000041_behavior_notes
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: behavior_notes

CREATE TABLE IF NOT EXISTS behavior_notes (
    id                UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID                 NOT NULL,
    school_id         UUID                 NOT NULL,
    student_id        UUID                 NOT NULL,
    timetable_slot_id UUID                 NOT NULL,
    date              DATE                 NOT NULL,
    category_id       UUID                 NOT NULL,
    description       TEXT                 NOT NULL,
    is_urgent         BOOLEAN              NOT NULL DEFAULT false,
    status            behavior_note_status NOT NULL DEFAULT 'PENDING_REVIEW',
    authored_by_id    UUID                 NOT NULL,
    reviewed_by_id    UUID                 NULL,
    reviewed_at       TIMESTAMPTZ          NULL,
    created_at        TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ          NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_behavior_notes_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_notes_timetable_slot
        FOREIGN KEY (tenant_id, timetable_slot_id)
        REFERENCES cbc_timetable_slots(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_notes_tenant_category
        FOREIGN KEY (tenant_id, category_id)
        REFERENCES behavior_categories(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_behavior_notes_authored_by
        FOREIGN KEY (tenant_id, authored_by_id)
        REFERENCES users(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_behavior_notes_reviewed_by
        FOREIGN KEY (tenant_id, reviewed_by_id)
        REFERENCES users(tenant_id, id) ON DELETE SET NULL (reviewed_by_id)
);

CREATE INDEX IF NOT EXISTS idx_behavior_notes_student
    ON behavior_notes (student_id);
CREATE INDEX IF NOT EXISTS idx_behavior_notes_status
    ON behavior_notes (status);
CREATE INDEX IF NOT EXISTS idx_behavior_notes_urgent
    ON behavior_notes (is_urgent) WHERE is_urgent = true;
CREATE INDEX IF NOT EXISTS idx_behavior_notes_slot_date
    ON behavior_notes (timetable_slot_id, date);
CREATE INDEX IF NOT EXISTS idx_behavior_notes_tenant
    ON behavior_notes (tenant_id);
CREATE INDEX IF NOT EXISTS idx_behavior_notes_school
    ON behavior_notes (school_id);

COMMENT ON TABLE behavior_notes IS
    'Sparse incident/behavior records logged by teachers. Each note is
     associated with a specific student, timetable slot, and date. Notes
     go through admin approval (PENDING_REVIEW → APPROVED/REJECTED) before
     being included in term reports or reaching parents. Urgent notes bypass
     term-end batching for immediate parent contact.';

COMMENT ON COLUMN behavior_notes.is_urgent IS
    'When true and approved, triggers immediate parent notification instead of
     waiting for term-end compilation.';



CREATE TRIGGER trg_behavior_notes_updated_at
    BEFORE UPDATE ON behavior_notes
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN behavior_notes.updated_at IS
    'Tracks approval workflow: PENDING_REVIEW, APPROVED, REJECTED.';
