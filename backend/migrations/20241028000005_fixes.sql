-- +goose Up

-- 10) pending_onboarding status as enum
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_type t JOIN pg_attribute a ON a.attrelid = t.typid
        WHERE t.typname = 'pending_onboarding_status' AND a.attname = 'status'
    ) THEN
        CREATE TYPE pending_onboarding_status AS ENUM ('pending','provisioning','completed','failed_retryable');
    END IF;
END $$;
ALTER TABLE pending_onboarding ALTER COLUMN status TYPE pending_onboarding_status;

-- 2) All UUID defaults migrated to uuidv7 (already used in other migrations).
--    Assumes latest PostgreSQL (17+) which ships pg_uuidv7 natively.

-- 11) Schools should not cascade-delete countries / education_systems.
ALTER TABLE schools DROP CONSTRAINT IF EXISTS schools_country_id_fkey;
ALTER TABLE schools ADD CONSTRAINT schools_country_id_fkey FOREIGN KEY (country_id) REFERENCES countries(id) ON DELETE RESTRICT;
ALTER TABLE schools DROP CONSTRAINT IF EXISTS schools_education_system_id_fkey;
ALTER TABLE schools ADD CONSTRAINT schools_education_system_id_fkey FOREIGN KEY (education_system_id) REFERENCES education_systems(id) ON DELETE RESTRICT;

-- 3) updated_at trigger function + apply to all tables
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE t TEXT[];
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'pending_onboarding','app_session','onboarding_ticket',
        'countries','education_systems','class_levels',
        'tenants','users','schools','memberships','pkce_codes',
        'students','guardian_student_links'
    ] LOOP
        EXECUTE format(
            'DROP TRIGGER IF EXISTS trigger_set_timestamp ON %I;'
            'CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp()',
            t, t);
    END LOOP;
END $$;

-- 4) Composite index users(tenant_id, email_normalized)
CREATE INDEX IF NOT EXISTS idx_users_tenant_email ON users (tenant_id, email_normalized);

-- 6) Membership role + composite index
CREATE INDEX IF NOT EXISTS idx_memberships_role ON memberships (role);
CREATE INDEX IF NOT EXISTS idx_memberships_school_role ON memberships (school_id, role);
CREATE INDEX IF NOT EXISTS idx_memberships_is_active ON memberships (is_active) WHERE is_active = true;

-- 5) Row Level Security (tenant isolation, model FR-FND-001).
--    Note: requires app middleware to set app.current_tenant_id per session.
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE users     ENABLE ROW LEVEL SECURITY;
ALTER TABLE schools  ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE students  ENABLE ROW LEVEL SECURITY;
ALTER TABLE guardian_student_links ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenants_isolate ON tenants USING (true);

CREATE POLICY users_isolate ON users
    USING (EXISTS (SELECT 1 FROM tenants WHERE tenants.id = users.tenant_id));

CREATE POLICY schools_isolate ON schools
    USING (EXISTS (SELECT 1 FROM tenants WHERE tenants.id = schools.tenant_id));

CREATE POLICY memberships_isolate ON memberships
    USING (EXISTS (SELECT 1 FROM schools WHERE schools.id = memberships.school_id));

CREATE POLICY students_isolate ON students
    USING (EXISTS (SELECT 1 FROM schools WHERE schools.id = students.school_id));

CREATE POLICY guardian_isolate ON guardian_student_links
    USING (EXISTS (SELECT 1 FROM students WHERE students.id = guardian_student_links.student_id));

-- 13) No schema validation on JSONB columns.

-- Soft delete (deleted_at) for all domain tables.
DO $$
DECLARE t TEXT[];
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'pending_onboarding','app_session','onboarding_ticket',
        'countries','education_systems','class_levels',
        'tenants','users','schools','memberships','pkce_codes',
        'students','guardian_student_links'
    ] LOOP
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = t AND column_name = 'deleted_at') THEN CONTINUE; END IF;
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = t) THEN
            EXECUTE format('ALTER TABLE %I ADD COLUMN deleted_at TIMESTAMPTZ', t);
            EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_deleted_at ON %I (deleted_at) WHERE deleted_at IS NULL', t, t);
        END IF;
    END LOOP;
END $$;

-- 9) Skills/strands/sub_strands deferred (see domain model: LearningArea hierarchy).

-- 12) Audit log deferred (model entity in docs/domain/model.md, table not yet required).

-- +goose Down
DROP POLICY IF EXISTS guardian_isolate ON guardian_student_links;
DROP POLICY IF EXISTS students_isolate ON students;
DROP POLICY IF EXISTS memberships_isolate ON memberships;
DROP POLICY IF EXISTS schools_isolate ON schools;
DROP POLICY IF EXISTS users_isolate ON users;
DROP POLICY IF EXISTS tenants_isolate ON tenants;
ALTER TABLE tenants DISABLE ROW LEVEL SECURITY;
ALTER TABLE users      DISABLE ROW LEVEL SECURITY;
ALTER TABLE schools    DISABLE ROW LEVEL SECURITY;
ALTER TABLE memberships DISABLE ROW LEVEL SECURITY;
ALTER TABLE students   DISABLE ROW LEVEL SECURITY;
ALTER TABLE guardian_student_links DISABLE ROW LEVEL SECURITY;

DO $$
DECLARE t TEXT[];
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'pending_onboarding','app_session','onboarding_ticket',
        'countries','education_systems','class_levels',
        'tenants','users','schools','memberships','pkce_codes',
        'students','guardian_student_links'
    ] LOOP
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = t AND column_name = 'deleted_at') THEN
            EXECUTE format('ALTER TABLE %I DROP COLUMN deleted_at', t);
        END IF;
    END LOOP;
END $$;

DROP TRIGGER IF EXISTS trigger_set_timestamp ON pending_onboarding;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON app_session;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON onboarding_ticket;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON countries;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON education_systems;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON class_levels;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON tenants;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON users;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON schools;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON memberships;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON pkce_codes;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON students;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON guardian_student_links;
DROP FUNCTION IF EXISTS trigger_set_timestamp();

ALTER TABLE schools DROP CONSTRAINT IF EXISTS schools_country_id_fkey;
ALTER TABLE schools ADD CONSTRAINT schools_country_id_fkey FOREIGN KEY (country_id) REFERENCES countries(id) ON DELETE CASCADE;
ALTER TABLE schools DROP CONSTRAINT IF EXISTS schools_education_system_id_fkey;
ALTER TABLE schools ADD CONSTRAINT schools_education_system_id_fkey FOREIGN KEY (education_system_id) REFERENCES education_systems(id) ON DELETE CASCADE;

ALTER TABLE pending_onboarding ALTER COLUMN status TYPE TEXT;
ALTER TABLE pending_onboarding ALTER COLUMN status SET DEFAULT 'pending';
ALTER TABLE pending_onboarding
    ADD CONSTRAINT pending_onboarding_status_check CHECK (status IN ('pending','provisioning','completed','failed_retryable'));
DROP TYPE IF EXISTS pending_onboarding_status;
