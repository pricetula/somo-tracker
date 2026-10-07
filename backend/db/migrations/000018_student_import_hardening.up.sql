-- Create enum for student gender to standardize values across backend and frontend
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'student_gender') THEN
    CREATE TYPE student_gender AS ENUM ('M', 'F', 'OTHER');
  END IF;
END $$;

-- Alter students.gender to use enum with default OTHER
ALTER TABLE students
  ALTER COLUMN gender TYPE student_gender USING upper(trim(coalesce(gender,'OTHER')))::text::student_gender,
  ALTER COLUMN gender SET DEFAULT 'OTHER';

-- Add unique constraint for idempotency per school
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'bulk_jobs_school_id_idempotency_key_key'
  ) THEN
    ALTER TABLE bulk_jobs
      ADD CONSTRAINT bulk_jobs_school_id_idempotency_key_key
      UNIQUE (school_id, idempotency_key);
  END IF;
END $$;

-- Change created_by FK to SET NULL on delete to preserve audit trail
DO $$
BEGIN
  -- Drop existing FK if exists and recreate with SET NULL
  -- First find constraint name
  PERFORM 1 FROM pg_constraint WHERE conname = 'bulk_jobs_created_by_fkey';
  IF FOUND THEN
    ALTER TABLE bulk_jobs DROP CONSTRAINT bulk_jobs_created_by_fkey;
  END IF;
END $$;

ALTER TABLE bulk_jobs
  ADD CONSTRAINT bulk_jobs_created_by_fkey
  FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

-- Normalize admission_number via functional unique index for case/space insensitivity
CREATE UNIQUE INDEX IF NOT EXISTS students_school_admission_number_ci
ON students (school_id, upper(trim(admission_number)));

-- Add trigger to keep student_gender_counts in sync for all writes outside bulk import
CREATE OR REPLACE FUNCTION recompute_student_gender_counts_for_row() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  PERFORM recompute_student_gender_counts_for_school(
    COALESCE(NEW.school_id, OLD.school_id)
  );
  RETURN NULL;
END $$;

-- Helper function to recompute counts for a school
CREATE OR REPLACE FUNCTION recompute_student_gender_counts_for_school(p_school_id uuid) RETURNS VOID LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO student_gender_counts (school_id, male_count, female_count, other_count, total_count, updated_at)
  SELECT p_school_id,
         COUNT(*) FILTER (WHERE gender = 'M'::student_gender),
         COUNT(*) FILTER (WHERE gender = 'F'::student_gender),
         COUNT(*) FILTER (WHERE gender = 'OTHER'::student_gender),
         COUNT(*),
         NOW()
  FROM students
  WHERE school_id = p_school_id
  ON CONFLICT (school_id) DO UPDATE SET
    male_count = EXCLUDED.male_count,
    female_count = EXCLUDED.female_count,
    other_count = EXCLUDED.other_count,
    total_count = EXCLUDED.total_count,
    updated_at = NOW();
END $$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'students_gender_counts_tri') THEN
    CREATE TRIGGER students_gender_counts_tri
    AFTER INSERT OR UPDATE OF gender, school_id OR DELETE ON students
    FOR EACH ROW EXECUTE FUNCTION recompute_student_gender_counts_for_row();
  END IF;
END $$;
