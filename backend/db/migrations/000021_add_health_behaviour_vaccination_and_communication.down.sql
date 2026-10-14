-- Migration rollback: 000021

DROP INDEX IF EXISTS idx_messages_thread;
DROP INDEX IF EXISTS idx_comm_threads_student;
DROP INDEX IF EXISTS idx_vaccination_records_student;
DROP INDEX IF EXISTS idx_behaviour_incidents_severity;
DROP INDEX IF EXISTS idx_behaviour_incidents_student;
DROP INDEX IF EXISTS idx_health_notes_student;

DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS thread_participants;
DROP TABLE IF EXISTS communication_threads;
DROP TABLE IF EXISTS vaccination_records;
DROP TABLE IF EXISTS behaviour_incidents;
DROP TABLE IF EXISTS health_notes;
DROP TABLE IF EXISTS incident_types;

DROP TYPE IF EXISTS thread_status;
DROP TYPE IF EXISTS incident_status;
DROP TYPE IF EXISTS incident_severity;
