-- Migration: 000015_import_job_staging (rollback)
-- SomoTracker — rollback for 000015_import_job_staging.


DROP TABLE IF EXISTS import_job_staging CASCADE;
