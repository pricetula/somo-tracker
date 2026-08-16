-- Migration: 000013_import_job_chunks (rollback)
-- SomoTracker — rollback for 000013_import_job_chunks.


DROP TABLE IF EXISTS import_job_chunks CASCADE;
