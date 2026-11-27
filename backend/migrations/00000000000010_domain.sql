-- +goose Up

-- Teachers: school staff members.
CREATE TABLE teachers (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    employee_number TEXT NOT NULL,
    full_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, employee_number)
);

CREATE INDEX idx_teachers_school ON teachers (school_id);

-- Students: learners enrolled at a school.
CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    admission_number TEXT NOT NULL,
    full_name TEXT NOT NULL,
    date_of_birth DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, admission_number),
    UNIQUE (id, school_id)
);

CREATE INDEX idx_students_school ON students (school_id);

-- Enrollments: links a student to a class for an academic year.
CREATE TABLE enrollments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'withdrawn', 'transferred', 'completed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, academic_year_id),
    FOREIGN KEY (class_id, school_id)
        REFERENCES classes(id, school_id),
    FOREIGN KEY (student_id, school_id)
        REFERENCES students(id, school_id)
);

CREATE INDEX idx_enrollments_student ON enrollments (student_id);
CREATE INDEX idx_enrollments_class ON enrollments (class_id);
CREATE INDEX idx_enrollments_academic_year ON enrollments (academic_year_id);
CREATE INDEX idx_enrollments_school ON enrollments (school_id);

-- Assessment types: categories of assessments (Quiz, Term Exam, etc.).
CREATE TABLE assessment_types (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Assessments: teacher-created assessments for a class, subject, term.
CREATE TABLE assessments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES school_subjects(id) ON DELETE CASCADE,
    assessment_type_id UUID NOT NULL REFERENCES assessment_types(id) ON DELETE CASCADE,
    academic_term_id UUID NOT NULL REFERENCES academic_terms(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    max_score DECIMAL NOT NULL CHECK (max_score > 0),
    date_assessed DATE NOT NULL,
    created_by_teacher_id UUID NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, school_id),
    FOREIGN KEY (class_id, school_id)
        REFERENCES classes(id, school_id)
);

CREATE INDEX idx_assessments_class ON assessments (class_id);
CREATE INDEX idx_assessments_subject ON assessments (subject_id);
CREATE INDEX idx_assessments_term ON assessments (academic_term_id);
CREATE INDEX idx_assessments_created_by ON assessments (created_by_teacher_id);
CREATE INDEX idx_assessments_school ON assessments (school_id);

-- Assessment results: teacher-entered scores for each student.
CREATE TABLE assessment_results (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    assessment_id UUID NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    score DECIMAL NOT NULL CHECK (score >= 0),
    grade TEXT,
    entered_by_teacher_id UUID NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    entered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (assessment_id, student_id),
    FOREIGN KEY (assessment_id, school_id)
        REFERENCES assessments(id, school_id),
    FOREIGN KEY (student_id, school_id)
        REFERENCES students(id, school_id)
);

CREATE INDEX idx_assessment_results_assessment ON assessment_results (assessment_id);
CREATE INDEX idx_assessment_results_student ON assessment_results (student_id);
CREATE INDEX idx_assessment_results_school ON assessment_results (school_id);

-- Guardians: parents or legal caregivers linked to students (ADR-0013).
-- Guardians are contacts, not system users (no login in MVP).
CREATE TABLE guardians (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    full_name TEXT NOT NULL,
    phone TEXT NOT NULL,
    email TEXT,
    relationship TEXT NOT NULL,
    address TEXT,
    is_emergency_contact BOOL NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_guardians_school ON guardians (school_id);

-- Student guardians: many-to-many link between students and guardians (ADR-0013).
CREATE TABLE student_guardians (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    guardian_id UUID NOT NULL REFERENCES guardians(id) ON DELETE CASCADE,
    is_primary BOOL NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, guardian_id)
);

CREATE INDEX idx_student_guardians_student ON student_guardians (student_id);
CREATE INDEX idx_student_guardians_guardian ON student_guardians (guardian_id);

-- Attendance records: one row per student per school day (ADR-0012).
CREATE TABLE attendance_records (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    academic_year_id UUID NOT NULL REFERENCES academic_years(id) ON DELETE CASCADE,
    attendance_date DATE NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('present', 'absent', 'late', 'excused')),
    recorded_by_teacher_id UUID NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (student_id, attendance_date),
    FOREIGN KEY (class_id, school_id)
        REFERENCES classes(id, school_id),
    FOREIGN KEY (student_id, school_id)
        REFERENCES students(id, school_id)
);

CREATE INDEX idx_attendance_records_student ON attendance_records (student_id);
CREATE INDEX idx_attendance_records_class ON attendance_records (class_id);
CREATE INDEX idx_attendance_records_date ON attendance_records (attendance_date);
CREATE INDEX idx_attendance_records_school ON attendance_records (school_id);

-- Teacher qualifications: records a teacher's qualification to teach a subject at a level (ADR-0014).
CREATE TABLE teacher_qualifications (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    teacher_id UUID NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES school_subjects(id) ON DELETE CASCADE,
    education_level_id UUID NOT NULL REFERENCES education_levels(id) ON DELETE CASCADE,
    qualification TEXT NOT NULL,
    registration_number TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_teacher_qualifications_teacher ON teacher_qualifications (teacher_id);
CREATE INDEX idx_teacher_qualifications_subject ON teacher_qualifications (subject_id);

-- Consent records: guardian consent for processing student personal data (ADR-0017).
CREATE TABLE consent_records (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    guardian_id UUID NOT NULL REFERENCES guardians(id) ON DELETE CASCADE,
    consent_type TEXT NOT NULL,
    granted BOOL NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    withdrawn_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consent_records_student ON consent_records (student_id);
CREATE INDEX idx_consent_records_guardian ON consent_records (guardian_id);

-- +goose Down

DROP TABLE IF EXISTS consent_records;

DROP TABLE IF EXISTS teacher_qualifications;

DROP TABLE IF EXISTS attendance_records;

DROP TABLE IF EXISTS student_guardians;

DROP TABLE IF EXISTS guardians;

DROP TABLE IF EXISTS assessment_results;

DROP TABLE IF EXISTS assessments;

DROP TABLE IF EXISTS assessment_types;

DROP TABLE IF EXISTS enrollments;

DROP TABLE IF EXISTS students;

DROP TABLE IF EXISTS teachers;
