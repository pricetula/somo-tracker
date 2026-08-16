-- Migration: 000014_import_job_failures (rollback)
-- SomoTracker — rollback for 000014_import_job_failures.


DROP TABLE IF EXISTS import_job_failures CASCADE;
