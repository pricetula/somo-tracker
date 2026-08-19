-- Migration: 000012_import_jobs (rollback)
-- SomoTracker — rollback for 000012_import_jobs.


DROP TABLE IF EXISTS import_jobs CASCADE;
