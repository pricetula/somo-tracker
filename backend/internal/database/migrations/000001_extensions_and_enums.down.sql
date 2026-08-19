-- Migration: 000001_extensions_and_enums (rollback)
-- SomoTracker — rollback for 000001_extensions_and_enums.

DROP TYPE IF EXISTS assessment_evaluation_method CASCADE;
DROP TYPE IF EXISTS assessment_session_status CASCADE;
DROP TYPE IF EXISTS cbc_performance_level CASCADE;
DROP TYPE IF EXISTS behavior_severity CASCADE;
DROP TYPE IF EXISTS behavior_note_status CASCADE;
DROP TYPE IF EXISTS attendance_status CASCADE;
DROP TYPE IF EXISTS behavior_category_type CASCADE;
DROP TYPE IF EXISTS parent_relationship_type CASCADE;
DROP TYPE IF EXISTS payment_method_type CASCADE;
DROP TYPE IF EXISTS import_failure_type CASCADE;
DROP TYPE IF EXISTS block_type CASCADE;
DROP TYPE IF EXISTS import_chunk_status CASCADE;
DROP TYPE IF EXISTS import_staging_status CASCADE;
DROP TYPE IF EXISTS import_job_type CASCADE;
DROP TYPE IF EXISTS import_job_status CASCADE;
DROP TYPE IF EXISTS invoice_payment_status CASCADE;
DROP TYPE IF EXISTS cbc_learning_pathway CASCADE;
DROP TYPE IF EXISTS cbc_school_type CASCADE;
DROP TYPE IF EXISTS teacher_role CASCADE;
DROP TYPE IF EXISTS cbc_education_level CASCADE;
DROP TYPE IF EXISTS cbc_grade_level CASCADE;
DROP TYPE IF EXISTS cbc_enrollment_status CASCADE;
DROP TYPE IF EXISTS gender_type CASCADE;
DROP TYPE IF EXISTS invitation_status CASCADE;
DROP TYPE IF EXISTS user_role CASCADE;

DROP EXTENSION IF EXISTS btree_gist CASCADE;
DROP EXTENSION IF EXISTS pgcrypto CASCADE;
