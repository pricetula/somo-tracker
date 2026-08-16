# Somotracker Database Schema

This document outlines the complete database schema for the Somotracker platform, including all tables, fields, relationships, and enumerations.

## Enumerations

### User Roles
- `user_role`: SYSTEM_ADMIN, SCHOOL_ADMIN, TEACHER, NURSE, FINANCE, PARENT

### Invitation Status
- `invitation_status`: pending, accepted, expired, revoked, invite_failed

### Gender Type
- `gender_type`: M, F

### CBC Enrollment Status
- `cbc_enrollment_status`: ACTIVE, SUSPENDED, TRANSFERRED, COMPLETED_CYCLE

### CBC Grade Level
- `cbc_grade_level`: PP1, PP2, G1, G2, G3, G4, G5, G6, G7, G8, G9, G10, G11, G12

### CBC Education Level
- `cbc_education_level`: Early_Years, Upper_Primary, Junior_Secondary, Senior_School

### Teacher Role
- `teacher_role`: PRIMARY_CLASS_TEACHER, SUBJECT_TEACHER, SUBSTITUTE_TEACHER

### School Type
- `cbc_school_type`: Public, Private, Special_Needs_School

### Learning Pathway
- `cbc_learning_pathway`: Age_Based, Stage_Based

### Invoice Payment Status
- `invoice_payment_status`: UNPAID, PARTIAL, PAID, WAIVED

### Import Job Status
- `import_job_status`: pending, processing, completed, failed, cancelled, completed_with_errors, cancelling

### Import Job Type
- `import_job_type`: STAFF_INVITE, STUDENT_IMPORT, PARENT_INVITE

### Import Staging Status
- `import_staging_status`: pending, succeeded, failed

### Import Chunk Status
- `import_chunk_status`: pending, processing, completed, cancelled

### Block Type
- `block_type`: Lesson, Break, Assembly, ExtraCurricular

### Import Failure Type
- `import_failure_type`: SCHEMA_VALIDATION, DATABASE_CONSTRAINT, BUSINESS_RULE_VIOLATION, DUPLICATE_EMAIL, INVALID_EMAIL_FORMAT, STYTCH_API_ERROR, INVITATION_INSERT_FAILED

### Payment Method Type
- `payment_method_type`: MPESA, CASH, BANK_TRANSFER, CHEQUE, OTHER

### Parent Relationship Type
- `parent_relationship_type`: FATHER, MOTHER, GUARDIAN, OTHER

### Behavior Category Type
- `behavior_category_type`: COMMENDATION, DISCIPLINARY, OTHER

## Core Tables

### Tenants
- `id`: UUID (Primary Key)
- `name`: VARCHAR(255) NOT NULL
- `slug`: VARCHAR(255) NOT NULL UNIQUE
- `stytch_org_id`: VARCHAR(255) NOT NULL UNIQUE
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Users
- `id`: UUID (Primary Key)
- `email`: VARCHAR(255) NOT NULL
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `full_name`: VARCHAR(255) NOT NULL DEFAULT ''
- `is_active`: BOOLEAN NOT NULL DEFAULT TRUE
- `external_auth_id`: VARCHAR(255) UNIQUE
- `tsc_number`: VARCHAR(15) NULL
- `knec_panel_assessor_id`: VARCHAR(20) NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Sessions
- `id`: UUID (Primary Key)
- `token`: VARCHAR(128) NULL
- `token_hash`: TEXT NULL
- `user_id`: UUID NOT NULL
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `stytch_member_id`: VARCHAR(255) NOT NULL
- `stytch_org_id`: VARCHAR(255) NOT NULL
- `stytch_session_token`: VARCHAR(512) NOT NULL DEFAULT ''
- `device_fingerprint`: VARCHAR(128) NOT NULL DEFAULT ''
- `expires_at`: TIMESTAMPTZ NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### CBC Schools
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `name`: VARCHAR(255) NOT NULL
- `knec_school_code`: VARCHAR(15) NULL
- `nemis_institution_code`: VARCHAR(20) NULL
- `county`: VARCHAR(50) NOT NULL
- `sub_county`: VARCHAR(50) NOT NULL
- `ward`: VARCHAR(50) NULL
- `school_type`: cbc_school_type NOT NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Academic Years
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `name`: VARCHAR(50) NOT NULL
- `start_date`: DATE NOT NULL
- `end_date`: DATE NOT NULL
- `is_current`: BOOLEAN NOT NULL DEFAULT false
- `version`: INTEGER NOT NULL DEFAULT 1
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_by`: UUID NOT NULL
- `updated_by`: UUID NOT NULL

### Academic Terms
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `name`: VARCHAR(100) NOT NULL
- `term_number`: SMALLINT NOT NULL
- `start_date`: DATE NOT NULL
- `end_date`: DATE NOT NULL
- `is_current`: BOOLEAN NOT NULL DEFAULT false
- `is_final`: BOOLEAN NOT NULL DEFAULT false
- `version`: INTEGER NOT NULL DEFAULT 1
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_by`: UUID NOT NULL
- `updated_by`: UUID NOT NULL

### CBC Streams
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `name`: VARCHAR(100) NOT NULL
- `color`: VARCHAR(50) NOT NULL DEFAULT ''
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### CBC Classes
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `grade_level`: cbc_grade_level NOT NULL
- `stream_id`: UUID NOT NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Memberships
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `user_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `role`: user_role NOT NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Import Jobs
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `school_id`: UUID NOT NULL
- `job_type`: import_job_type NOT NULL
- `role`: user_role NULL
- `created_by`: UUID NULL
- `status`: import_job_status NOT NULL DEFAULT 'pending'
- `total_records`: INT NOT NULL DEFAULT 0
- `processed_records`: INT NOT NULL DEFAULT 0
- `success_count`: INT NOT NULL DEFAULT 0
- `failed_count`: INT NOT NULL DEFAULT 0
- `idempotency_key`: TEXT NULL
- `payload_hash`: TEXT NULL
- `total_chunks`: INT NOT NULL DEFAULT 0
- `processed_chunks`: INT NOT NULL DEFAULT 0
- `metadata`: JSONB NOT NULL DEFAULT '{}'::jsonb
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `started_at`: TIMESTAMPTZ NULL
- `completed_at`: TIMESTAMPTZ NULL
- `last_progress_at`: TIMESTAMPTZ NULL
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Import Job Chunks
- `id`: UUID (Primary Key)
- `job_id`: UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE
- `chunk_index`: INT NOT NULL
- `status`: import_chunk_status NOT NULL DEFAULT 'pending'
- `row_start`: INT NOT NULL DEFAULT 0
- `row_end`: INT NOT NULL DEFAULT 0
- `claimed_at`: TIMESTAMPTZ NULL
- `completed_at`: TIMESTAMPTZ NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Import Job Failures
- `id`: BIGSERIAL (Primary Key)
- `import_job_id`: UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE
- `raw_payload`: JSONB NOT NULL
- `error_message`: TEXT NOT NULL
- `error_type`: import_failure_type NOT NULL DEFAULT 'DATABASE_CONSTRAINT'
- `row_number`: INT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Import Job Staging
- `id`: UUID (Primary Key)
- `job_id`: UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `row_number`: INT NOT NULL
- `raw_data`: JSONB NOT NULL
- `status`: import_staging_status NOT NULL DEFAULT 'pending'
- `processed_at`: TIMESTAMPTZ NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT now()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Invitations
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `school_id`: UUID NOT NULL
- `email`: VARCHAR(255) NOT NULL
- `role`: user_role NOT NULL
- `status`: invitation_status NOT NULL DEFAULT 'pending'
- `invited_by`: UUID NULL
- `token`: TEXT NOT NULL
- `token_hash`: TEXT NULL
- `expires_at`: TIMESTAMPTZ NOT NULL
- `accepted_at`: TIMESTAMPTZ NULL
- `full_name`: VARCHAR(255) NOT NULL
- `phone`: VARCHAR(50) NULL
- `registration_number`: VARCHAR(100) NULL
- `stytch_member_id`: VARCHAR(255) NULL
- `import_job_id`: UUID NULL
- `error_message`: TEXT NULL
- `attempt_count`: INT NOT NULL DEFAULT 0
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### CBC Parents
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL
- `user_id`: UUID NOT NULL
- `phone_number`: VARCHAR(20) NOT NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### CBC Students
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `school_id`: UUID NOT NULL
- `full_name`: VARCHAR(255) NOT NULL
- `gender`: gender_type NOT NULL
- `date_of_birth`: DATE NULL
- `upi_number`: VARCHAR(20) NULL
- `knec_assessment_number`: VARCHAR(15) NULL
- `admission_number`: VARCHAR(20) NULL
- `learning_pathway`: cbc_learning_pathway NOT NULL DEFAULT 'Age_Based'
- `staging_row_id`: UUID NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### CBC Student Parents Junction
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `student_id`: UUID NOT NULL
- `parent_id`: UUID NOT NULL
- `relationship`: parent_relationship_type NULL
- `is_primary`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### CBC Student Enrollments
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `class_id`: UUID NULL
- `status`: cbc_enrollment_status NOT NULL DEFAULT 'ACTIVE'
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Medical Incidents
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `student_id`: UUID NOT NULL
- `incident_timestamp`: TIMESTAMPTZ NOT NULL
- `symptoms`: TEXT NOT NULL
- `action_taken`: TEXT NOT NULL
- `logged_by`: UUID NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Student Health Profiles
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `student_id`: UUID UNIQUE NOT NULL
- `blood_group`: VARCHAR(5)
- `allergies`: TEXT[]
- `chronic_conditions`: TEXT[]
- `emergency_instructions`: TEXT
- `logged_by`: UUID NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Fee Categories
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `name`: VARCHAR(150) NOT NULL
- `is_mandatory`: BOOLEAN NOT NULL DEFAULT true
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

### Fee Templates
- `id`: UUID (Primary Key)
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `grade_level`: cbc_grade_level NOT NULL
- `fee_category_id`: UUID NOT NULL
- `amount`: NUMERIC(12,2) NOT NULL CHECK (amount >= 0)
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

## Relationships Summary

### One-to-Many Relationships
- Tenants → Users (via tenant_id)
- Tenants → Users (sessions)
- Tenants → Schools
- Tenants → Students
- Tenants → Parents
- Schools → Users (memberships)
- Schools → Students
- Schools → CBC Classes
- Schools → CBC Streams
- Schools → Academic Years
- Schools → Academic Terms
- Academic Terms → Academic Years
- Students → Enrollments
- Students → Medical Incidents
- Students → Health Profiles
- Students → Parent Relationships
- Students → Import Job Staging (via staging_row_id)
- Import Jobs → Import Job Chunks
- Import Jobs → Import Job Failures
- Users → Invitations (invited_by)
- Users → Sessions
- Users → Medical Incidents (logged_by)
- Users → Student Health Profiles (logged_by)

### Many-to-Many Relationships
- Students ↔ Parents (via cbc_student_parents junction table)
- Schools ↔ Students (via enrollments)
- Schools ↔ Classes (via academic_year_id)
- Classes ↔ Students (via enrollments)

### Composite Keys (for tenant scoping)
- Tenants (id)
- Users (tenant_id, id)
- Schools (tenant_id, id)
- Students (tenant_id, id)
- Memberships (tenant_id, user_id, school_id)
- Invitations (tenant_id, school_id, email) - unique pending
- Academic Years (tenant_id, school_id, id)
- Academic Terms (tenant_id, school_id, id)
- CBC Classes (tenant_id, id)
- Memberships (user_id, school_id) - unique per user/school

## Relationships with ON DELETE Actions

- **CASCADE**: 
  - Tenants → Users, Schools, Memberships
  - Schools → Students, CBC Classes, CBC Streams, Academic Terms
  - Academic Years → Academic Terms
  - Academic Terms → Academic Years (reference)
  - Students → Medical Incidents, Health Profiles
  - Import Jobs → Import Job Chunks, Staging, Failures

- **SET NULL**:
  - Sessions → Users (token_hash)
  - Invitations → Import Jobs
  - Enrollments → Classes (class_id)

- **RESTRICT**:
  - Medical Incidents → Users (logged_by)
  - Health Profiles → Users (logged_by)

## Key Constraints and Indexes

### Unique Constraints
- Tenants (slug), (stytch_org_id)
- Users (tenant_id, LOWER(email)), (tenant_id, tsc_number), (tenant_id, knec_panel_assessor_id)
- Sessions (token_hash)
- Academic Years (tenant_id, school_id, id), EXCL constraint on start/end dates
- Academic Terms (tenant_id, school_id, id), EXCL constraint on start/end dates
- CBC Classes (school_id, academic_year_id, grade_level, stream_id), (tenant_id, id)
- Memberships (user_id, school_id)
- Invitations (tenant_id, school_id, email) - pending status only
- Invitations (tenant_id, idempotency_key) - unique when not null
- Import Jobs (tenant_id, idempotency_key), (school_id) - one active at a time
- Import Job Chunks (job_id, chunk_index)
- Import Job Staging (job_id, row_number)
- Students (upi_number), (knec_assessment_number), (school_id, staging_row_id)
- Student Parents (tenant_id, student_id, parent_id) - primary key
- One primary parent per student (is_primary = true)

### Foreign Keys
- Users → Tenants (ON DELETE CASCADE)
- Sessions → Users, Tenants (COMPOSITE)
- Schools → Tenants (ON DELETE CASCADE)
- Academic Years → Tenants, Schools (COMPOSITE)
- Academic Terms → Academic Years (COMPOSITE)
- Classes → Academic Years (COMPOSITE)
- Classes → Schools (COMPOSITE)
- Memberships → Tenants, Users, Schools
- Import Jobs → Tenants, Schools, Import Jobs (self-reference)
- Import Job Chunks → Import Jobs
- Import Job Staging → Import Jobs, Schools, Tenants
- Invitations → Tenants, Schools, Import Jobs
- Students → Tenants, Schools
- Student Parents → Tenants, Students, Parents
- Enrollments → Students, Schools, Academic Terms, Classes
- Medical Incidents → Tenants, Students
- Health Profiles → Tenants, Students

### Check Constraints
- `users`: is_active boolean default true
- `academic_years`: start_date < end_date
- `academic_terms`: term_number between 1-3, start_date < end_date
- `cbc_students`: gender IN ('M', 'F')
- `fee_templates`: amount >= 0
- `import_jobs`: role required for staff/parent invites
- `invitation_status`: default 'pending'
- `cbc_enrollment_status`: default 'ACTIVE'

## Notes
1. All tables use UUID primary keys for scalability and security
2. Strong tenant isolation throughout the schema
3. Extensive use of composite keys for tenant scoping
4. Comprehensive audit columns (created_at, updated_at) on most tables
5. GiST exclusion constraints prevent overlapping date ranges for academic years/terms
6. Composite foreign keys enforce tenant-scoped referential integrity
7. ON DELETE actions are carefully chosen to preserve historical data where needed
8. Enums are used extensively for data validation and consistency
9. JSONB fields are used for flexible metadata storage
10. Triggers automatically manage updated_at timestamps across the schema