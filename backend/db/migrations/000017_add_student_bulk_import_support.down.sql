-- Migration down: 000017_add_student_bulk_import_support

DROP TRIGGER IF EXISTS student_gender_counts_updated_at_trg ON student_gender_counts;
DROP TABLE IF EXISTS student_gender_counts;

ALTER TABLE bulk_jobs DROP CONSTRAINT IF EXISTS bulk_jobs_job_type_check;
ALTER TABLE bulk_jobs ADD CONSTRAINT bulk_jobs_job_type_check
    CHECK (job_type IN ('ADMIN_INVITATION'));
