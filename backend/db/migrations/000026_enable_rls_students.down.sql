DROP POLICY IF EXISTS students_tenant_isolation ON students;
ALTER TABLE students DISABLE ROW LEVEL SECURITY;
