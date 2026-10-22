-- Migration: 000021_add_health_behaviour_vaccination_and_communication
-- Purpose: Health, Behaviour, Vaccination, and In-App Communication
-- Platform: Somotracker (Multi-Tenant School Management System)

CREATE TYPE incident_severity AS ENUM ('LOW', 'MEDIUM', 'HIGH');
CREATE TYPE incident_status AS ENUM ('OPEN', 'PARENT_NOTIFIED', 'MEETING_SCHEDULED', 'RESOLVED');
CREATE TYPE thread_status AS ENUM ('ACTIVE', 'RESOLVED', 'ARCHIVED');

CREATE TABLE IF NOT EXISTS incident_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'DISCIPLINARY',
    is_system_default BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE incident_types IS 'Global/reference lookup table standardizing incident categories for multi-tenant behavioral analytics.';

CREATE TABLE IF NOT EXISTS health_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(student_id) ON DELETE CASCADE,
    note_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    recorded_by_membership_id UUID NOT NULL REFERENCES school_memberships(id) ON DELETE RESTRICT,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE health_notes IS 'Stores confidential clinical notes, illnesses, and first-aid interventions administered to students.';

CREATE TABLE IF NOT EXISTS behaviour_incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(student_id) ON DELETE CASCADE,
    incident_type_id UUID NOT NULL REFERENCES incident_types(id) ON DELETE RESTRICT,
    severity incident_severity NOT NULL DEFAULT 'LOW',
    status incident_status NOT NULL DEFAULT 'OPEN',
    description TEXT NOT NULL,
    action_taken TEXT,
    witness_count INT NOT NULL DEFAULT 0,
    recorded_by_membership_id UUID NOT NULL REFERENCES school_memberships(id) ON DELETE RESTRICT,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE behaviour_incidents IS 'Main incident log for tracking positive and disciplinary student behavior events.';

CREATE TABLE IF NOT EXISTS vaccination_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(student_id) ON DELETE CASCADE,
    vaccine_name VARCHAR(100) NOT NULL,
    batch_number VARCHAR(100),
    date_given DATE,
    next_due_date DATE,
    status VARCHAR(50) NOT NULL DEFAULT 'COMPLETED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE vaccination_records IS 'Structured student immunization ledger for official health compliance and government reporting.';

CREATE TABLE IF NOT EXISTS communication_threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(student_id) ON DELETE CASCADE,
    behaviour_incident_id UUID REFERENCES behaviour_incidents(id) ON DELETE SET NULL,
    subject VARCHAR(255) NOT NULL,
    status thread_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE communication_threads IS 'Active communication channels between staff and guardians, optionally anchored to specific incidents.';

CREATE TABLE IF NOT EXISTS thread_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL REFERENCES communication_threads(id) ON DELETE CASCADE,
    school_membership_id UUID NOT NULL REFERENCES school_memberships(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (thread_id, school_membership_id)
);
COMMENT ON TABLE thread_participants IS 'Participant roster for messaging threads, ensuring strict RLS enforcement.';

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL REFERENCES communication_threads(id) ON DELETE CASCADE,
    sender_membership_id UUID NOT NULL REFERENCES school_memberships(id) ON DELETE RESTRICT,
    content TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ
);
COMMENT ON TABLE messages IS 'Individual message entities supporting full read-receipt tracking and accountability.';

CREATE INDEX idx_health_notes_student ON health_notes(school_id, student_id);
CREATE INDEX idx_behaviour_incidents_student ON behaviour_incidents(school_id, student_id);
CREATE INDEX idx_behaviour_incidents_severity ON behaviour_incidents(school_id, severity);
CREATE INDEX idx_vaccination_records_student ON vaccination_records(school_id, student_id);
CREATE INDEX idx_comm_threads_student ON communication_threads(school_id, student_id);
CREATE INDEX idx_messages_thread ON messages(thread_id, sent_at);
