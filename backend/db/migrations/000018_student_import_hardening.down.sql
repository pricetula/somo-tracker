-- Drop trigger and helper functions
DROP TRIGGER IF EXISTS students_gender_counts_tri ON students;
DROP FUNCTION IF EXISTS recompute_student_gender_counts_for_row();
DROP FUNCTION IF EXISTS recompute_student_gender_counts_for_school(uuid);

-- Drop functional unique index
DROP INDEX IF EXISTS students_school_admission_number_ci;

-- Revert created_by FK to CASCADE? Original was CASCADE. We'll set back to CASCADE.
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'bulk_jobs_created_by_fkey') THEN
    ALTER TABLE bulk_jobs DROP CONSTRAINT bulk_jobs_created_by_fkey;
  END IF;
END $$;

ALTER TABLE bulk_jobs
  ADD CONSTRAINT bulk_jobs_created_by_fkey
  FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE;

-- Drop unique constraint on idempotency
ALTER TABLE bulk_jobs DROP CONSTRAINT IF EXISTS bulk_jobs_school_id_idempotency_key_key;

-- Revert students.gender back to text
ALTER TABLE students
  ALTER COLUMN gender TYPE text USING gender::text;

-- Drop enum type if no longer used
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'student_gender') THEN
    DROP TYPE student_gender;
  END IF;
END $$;
