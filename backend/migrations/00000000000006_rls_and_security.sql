-- +goose Up
-- updated_at trigger function (applied to all domain tables).
CREATE OR REPLACE FUNCTION trigger_set_timestamp() RETURNS TRIGGER LANGUAGE plpgsql AS $func$ BEGIN NEW.updated_at = NOW(); RETURN NEW; END; $func$;

CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON pending_onboarding FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON app_session FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON onboarding_ticket FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON pkce_codes FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON countries FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON education_systems FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON class_levels FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON tenants FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON schools FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON memberships FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON students FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON guardian_student_links FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON subjects FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON topics FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON subtopics FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON streams FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON classrooms FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON academic_years FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON terms FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON enrollments FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- RLS helper: resolves the current tenant from a session GUC.
-- The application must SET LOCAL app.current_tenant_id = '<tenant-uuid>' at transaction start.
CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS UUID LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
$$;

-- Enable RLS on all tenant-scoped tables.
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE schools ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE students ENABLE ROW LEVEL SECURITY;
ALTER TABLE guardian_student_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE pending_onboarding ENABLE ROW LEVEL SECURITY;
ALTER TABLE app_session ENABLE ROW LEVEL SECURITY;
ALTER TABLE onboarding_ticket ENABLE ROW LEVEL SECURITY;
ALTER TABLE pkce_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE countries ENABLE ROW LEVEL SECURITY;
ALTER TABLE education_systems ENABLE ROW LEVEL SECURITY;
ALTER TABLE class_levels ENABLE ROW LEVEL SECURITY;
ALTER TABLE subjects ENABLE ROW LEVEL SECURITY;
ALTER TABLE topics ENABLE ROW LEVEL SECURITY;
ALTER TABLE subtopics ENABLE ROW LEVEL SECURITY;
ALTER TABLE streams ENABLE ROW LEVEL SECURITY;
ALTER TABLE classrooms ENABLE ROW LEVEL SECURITY;
ALTER TABLE academic_years ENABLE ROW LEVEL SECURITY;
ALTER TABLE terms ENABLE ROW LEVEL SECURITY;
ALTER TABLE enrollments ENABLE ROW LEVEL SECURITY;

-- RLS policies using current_tenant_id().
CREATE POLICY tenant_isolation ON tenants USING (id = current_tenant_id());
CREATE POLICY user_tenant_isolation ON users USING (tenant_id = current_tenant_id());
CREATE POLICY school_tenant_isolation ON schools USING (tenant_id = current_tenant_id());
CREATE POLICY membership_tenant_isolation ON memberships
    USING (school_id IN (SELECT id FROM schools WHERE tenant_id = current_tenant_id()));
CREATE POLICY students_isolate ON students
    USING (EXISTS (SELECT 1 FROM schools WHERE schools.id = students.school_id AND schools.tenant_id = current_tenant_id()));
CREATE POLICY guardian_isolate ON guardian_student_links
    USING (EXISTS (SELECT 1 FROM students WHERE students.student_id = guardian_student_links.student_id));
CREATE POLICY pending_onboarding_tenant_isolation ON pending_onboarding
    USING (tenant_id = current_tenant_id() OR tenant_id IS NULL);
CREATE POLICY app_session_tenant_isolation ON app_session USING (tenant_id = current_tenant_id());
CREATE POLICY onboarding_ticket_tenant_isolation ON onboarding_ticket
    USING (pending_onboarding_id IN (
        SELECT id FROM pending_onboarding WHERE tenant_id = current_tenant_id() OR tenant_id IS NULL
    ));
CREATE POLICY pkce_codes_tenant_isolation ON pkce_codes
    USING (pending_onboarding_id IN (
        SELECT id FROM pending_onboarding WHERE tenant_id = current_tenant_id() OR tenant_id IS NULL
    ));
CREATE POLICY subjects_isolate ON subjects
    USING (EXISTS (SELECT 1 FROM education_systems WHERE education_systems.id = subjects.education_system_id));
CREATE POLICY topics_isolate ON topics
    USING (EXISTS (SELECT 1 FROM subjects WHERE subjects.id = topics.subject_id));
CREATE POLICY subtopics_isolate ON subtopics
    USING (EXISTS (SELECT 1 FROM topics WHERE topics.id = subtopics.topic_id));
CREATE POLICY streams_isolate ON streams
    USING (EXISTS (SELECT 1 FROM schools WHERE schools.id = streams.school_id AND schools.tenant_id = current_tenant_id()));
CREATE POLICY classrooms_isolate ON classrooms
    USING (EXISTS (SELECT 1 FROM schools WHERE schools.id = classrooms.school_id AND schools.tenant_id = current_tenant_id()));
CREATE POLICY academic_years_isolate ON academic_years
    USING (EXISTS (SELECT 1 FROM schools WHERE schools.id = academic_years.school_id AND schools.tenant_id = current_tenant_id()));
CREATE POLICY terms_isolate ON terms
    USING (EXISTS (SELECT 1 FROM academic_years WHERE academic_years.id = terms.academic_year_id));
CREATE POLICY enrollments_isolate ON enrollments
    USING (EXISTS (SELECT 1 FROM students WHERE students.student_id = enrollments.student_id));

-- FORCE ROW LEVEL SECURITY so even table owners must comply.
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE schools FORCE ROW LEVEL SECURITY;
ALTER TABLE memberships FORCE ROW LEVEL SECURITY;
ALTER TABLE students FORCE ROW LEVEL SECURITY;
ALTER TABLE guardian_student_links FORCE ROW LEVEL SECURITY;
ALTER TABLE pending_onboarding FORCE ROW LEVEL SECURITY;
ALTER TABLE app_session FORCE ROW LEVEL SECURITY;
ALTER TABLE onboarding_ticket FORCE ROW LEVEL SECURITY;
ALTER TABLE pkce_codes FORCE ROW LEVEL SECURITY;
ALTER TABLE countries FORCE ROW LEVEL SECURITY;
ALTER TABLE education_systems FORCE ROW LEVEL SECURITY;
ALTER TABLE class_levels FORCE ROW LEVEL SECURITY;
ALTER TABLE subjects FORCE ROW LEVEL SECURITY;
ALTER TABLE topics FORCE ROW LEVEL SECURITY;
ALTER TABLE subtopics FORCE ROW LEVEL SECURITY;
ALTER TABLE streams FORCE ROW LEVEL SECURITY;
ALTER TABLE classrooms FORCE ROW LEVEL SECURITY;
ALTER TABLE academic_years FORCE ROW LEVEL SECURITY;
ALTER TABLE terms FORCE ROW LEVEL SECURITY;
ALTER TABLE enrollments FORCE ROW LEVEL SECURITY;

-- +goose Down
DROP POLICY IF EXISTS enrollments_isolate ON enrollments;
DROP POLICY IF EXISTS terms_isolate ON terms;
DROP POLICY IF EXISTS academic_years_isolate ON academic_years;
DROP POLICY IF EXISTS classrooms_isolate ON classrooms;
DROP POLICY IF EXISTS streams_isolate ON streams;
DROP POLICY IF EXISTS subtopics_isolate ON subtopics;
DROP POLICY IF EXISTS topics_isolate ON topics;
DROP POLICY IF EXISTS subjects_isolate ON subjects;
DROP POLICY IF EXISTS pkce_codes_tenant_isolation ON pkce_codes;
DROP POLICY IF EXISTS onboarding_ticket_tenant_isolation ON onboarding_ticket;
DROP POLICY IF EXISTS app_session_tenant_isolation ON app_session;
DROP POLICY IF EXISTS pending_onboarding_tenant_isolation ON pending_onboarding;
DROP POLICY IF EXISTS guardian_isolate ON guardian_student_links;
DROP POLICY IF EXISTS students_isolate ON students;
DROP POLICY IF EXISTS membership_tenant_isolation ON memberships;
DROP POLICY IF EXISTS school_tenant_isolation ON schools;
DROP POLICY IF EXISTS user_tenant_isolation ON users;
DROP POLICY IF EXISTS tenant_isolation ON tenants;

ALTER TABLE enrollments DISABLE ROW LEVEL SECURITY;
ALTER TABLE terms DISABLE ROW LEVEL SECURITY;
ALTER TABLE academic_years DISABLE ROW LEVEL SECURITY;
ALTER TABLE classrooms DISABLE ROW LEVEL SECURITY;
ALTER TABLE streams DISABLE ROW LEVEL SECURITY;
ALTER TABLE class_levels DISABLE ROW LEVEL SECURITY;
ALTER TABLE subjects DISABLE ROW LEVEL SECURITY;
ALTER TABLE topics DISABLE ROW LEVEL SECURITY;
ALTER TABLE subtopics DISABLE ROW LEVEL SECURITY;
ALTER TABLE education_systems DISABLE ROW LEVEL SECURITY;
ALTER TABLE countries DISABLE ROW LEVEL SECURITY;
ALTER TABLE pkce_codes DISABLE ROW LEVEL SECURITY;
ALTER TABLE onboarding_ticket DISABLE ROW LEVEL SECURITY;
ALTER TABLE app_session DISABLE ROW LEVEL SECURITY;
ALTER TABLE pending_onboarding DISABLE ROW LEVEL SECURITY;
ALTER TABLE guardian_student_links DISABLE ROW LEVEL SECURITY;
ALTER TABLE students DISABLE ROW LEVEL SECURITY;
ALTER TABLE memberships DISABLE ROW LEVEL SECURITY;
ALTER TABLE schools DISABLE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE tenants DISABLE ROW LEVEL SECURITY;

DROP TRIGGER IF EXISTS trigger_set_timestamp ON enrollments;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON terms;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON academic_years;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON classrooms;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON streams;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON subtopics;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON topics;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON subjects;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON guardian_student_links;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON students;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON memberships;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON schools;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON users;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON tenants;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON class_levels;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON education_systems;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON countries;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON pkce_codes;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON onboarding_ticket;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON app_session;
DROP TRIGGER IF EXISTS trigger_set_timestamp ON pending_onboarding;

DROP FUNCTION IF EXISTS current_tenant_id();
DROP FUNCTION IF EXISTS trigger_set_timestamp();
