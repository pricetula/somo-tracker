-- Migration: 000026_enable_rls_students
-- Purpose: Enable Row Level Security on students table and add tenant isolation policy
-- for new structured fields added in 000023. Ensures PII is scoped to current tenant.

ALTER TABLE students ENABLE ROW LEVEL SECURITY;
ALTER TABLE students FORCE ROW LEVEL SECURITY;

CREATE POLICY students_tenant_isolation
    ON students
    FOR ALL TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = students.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM schools
            WHERE schools.id = students.school_id
              AND schools.tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
        )
    );

COMMENT ON POLICY students_tenant_isolation ON students IS 'Tenant isolation for students table, scoping by schools.tenant_id via app.current_tenant_id';
