-- Migration: 000036_school_member_counts
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: school_member_counts

CREATE TABLE IF NOT EXISTS school_member_counts (
    tenant_id  UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id  UUID         NOT NULL,
    admins     INTEGER      NOT NULL DEFAULT 0,
    teachers   INTEGER      NOT NULL DEFAULT 0,
    nurses     INTEGER      NOT NULL DEFAULT 0,
    finance    INTEGER      NOT NULL DEFAULT 0,
    parents    INTEGER      NOT NULL DEFAULT 0,
    students   INTEGER      NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_school_member_counts PRIMARY KEY (tenant_id, school_id),
    CONSTRAINT fk_school_member_counts_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
);

-- ============================================================================
-- TRIGGER: Sync school staff counts from memberships
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_sync_school_staff_counts_insert()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO school_member_counts (tenant_id, school_id, admins, teachers, nurses, finance, parents, updated_at)
    SELECT
        COALESCE(m.tenant_id, sc.tenant_id),
        s.school_id,
        COUNT(*) FILTER (WHERE m.role = 'SCHOOL_ADMIN') AS admins,
        COUNT(*) FILTER (WHERE m.role = 'TEACHER')      AS teachers,
        COUNT(*) FILTER (WHERE m.role = 'NURSE')        AS nurses,
        COUNT(*) FILTER (WHERE m.role = 'FINANCE')      AS finance,
        COUNT(*) FILTER (WHERE m.role = 'PARENT')       AS parents,
        NOW()
    FROM (SELECT DISTINCT school_id FROM inserted_rows) s
    LEFT JOIN memberships m
        ON m.school_id = s.school_id
       AND m.is_active  = true
    LEFT JOIN cbc_schools sc ON sc.id = s.school_id
    GROUP BY s.school_id, m.tenant_id, sc.tenant_id
    ON CONFLICT (tenant_id, school_id) DO UPDATE SET
        admins     = EXCLUDED.admins,
        teachers   = EXCLUDED.teachers,
        nurses     = EXCLUDED.nurses,
        finance    = EXCLUDED.finance,
        parents    = EXCLUDED.parents,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_school_staff_counts_delete()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO school_member_counts (tenant_id, school_id, admins, teachers, nurses, finance, parents, updated_at)
    SELECT
        COALESCE(m.tenant_id, sc.tenant_id),
        s.school_id,
        COUNT(*) FILTER (WHERE m.role = 'SCHOOL_ADMIN') AS admins,
        COUNT(*) FILTER (WHERE m.role = 'TEACHER')      AS teachers,
        COUNT(*) FILTER (WHERE m.role = 'NURSE')        AS nurses,
        COUNT(*) FILTER (WHERE m.role = 'FINANCE')      AS finance,
        COUNT(*) FILTER (WHERE m.role = 'PARENT')       AS parents,
        NOW()
    FROM (SELECT DISTINCT school_id FROM deleted_rows) s
    LEFT JOIN memberships m
        ON m.school_id = s.school_id
       AND m.is_active  = true
    LEFT JOIN cbc_schools sc ON sc.id = s.school_id
    GROUP BY s.school_id, m.tenant_id, sc.tenant_id
    ON CONFLICT (tenant_id, school_id) DO UPDATE SET
        admins     = EXCLUDED.admins,
        teachers   = EXCLUDED.teachers,
        nurses     = EXCLUDED.nurses,
        finance    = EXCLUDED.finance,
        parents    = EXCLUDED.parents,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_school_staff_counts_update()
RETURNS TRIGGER AS $$
BEGIN
    WITH affected_schools AS (
        SELECT DISTINCT school_id FROM inserted_rows
        UNION
        SELECT DISTINCT school_id FROM deleted_rows
    )
    INSERT INTO school_member_counts (tenant_id, school_id, admins, teachers, nurses, finance, parents, updated_at)
    SELECT
        COALESCE(m.tenant_id, sc.tenant_id),
        s.school_id,
        COUNT(*) FILTER (WHERE m.role = 'SCHOOL_ADMIN') AS admins,
        COUNT(*) FILTER (WHERE m.role = 'TEACHER')      AS teachers,
        COUNT(*) FILTER (WHERE m.role = 'NURSE')        AS nurses,
        COUNT(*) FILTER (WHERE m.role = 'FINANCE')      AS finance,
        COUNT(*) FILTER (WHERE m.role = 'PARENT')       AS parents,
        NOW()
    FROM affected_schools s
    LEFT JOIN memberships m
        ON m.school_id = s.school_id
       AND m.is_active  = true
    LEFT JOIN cbc_schools sc ON sc.id = s.school_id
    GROUP BY s.school_id, m.tenant_id, sc.tenant_id
    ON CONFLICT (tenant_id, school_id) DO UPDATE SET
        admins     = EXCLUDED.admins,
        teachers   = EXCLUDED.teachers,
        nurses     = EXCLUDED.nurses,
        finance    = EXCLUDED.finance,
        parents    = EXCLUDED.parents,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Fires on INSERT
CREATE TRIGGER trg_memberships_counts_insert
    AFTER INSERT ON memberships
    REFERENCING NEW TABLE AS inserted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_staff_counts_insert();

-- Fires on DELETE
CREATE TRIGGER trg_memberships_counts_delete
    AFTER DELETE ON memberships
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_staff_counts_delete();

-- Fires on UPDATE
CREATE TRIGGER trg_memberships_counts_update
    AFTER UPDATE ON memberships
    REFERENCING NEW TABLE AS inserted_rows OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_staff_counts_update();

-- ============================================================
-- TRIGGER: Sync school student counts from cbc_students
-- ============================================================

CREATE OR REPLACE FUNCTION fn_sync_school_student_counts_insert()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO school_member_counts (tenant_id, school_id, students, updated_at)
    SELECT
        COALESCE(st.tenant_id, sc.tenant_id),
        s.school_id,
        COUNT(st.id) AS students,
        NOW()
    FROM (SELECT DISTINCT school_id FROM inserted_rows) s
    LEFT JOIN cbc_students st
        ON st.school_id = s.school_id
       AND st.is_active  = true
    LEFT JOIN cbc_schools sc ON sc.id = s.school_id
    GROUP BY s.school_id, st.tenant_id, sc.tenant_id
    ON CONFLICT (tenant_id, school_id) DO UPDATE SET
        students   = EXCLUDED.students,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_school_student_counts_delete()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO school_member_counts (tenant_id, school_id, students, updated_at)
    SELECT
        COALESCE(st.tenant_id, sc.tenant_id),
        s.school_id,
        COUNT(st.id) AS students,
        NOW()
    FROM (SELECT DISTINCT school_id FROM deleted_rows) s
    LEFT JOIN cbc_students st
        ON st.school_id = s.school_id
       AND st.is_active  = true
    LEFT JOIN cbc_schools sc ON sc.id = s.school_id
    GROUP BY s.school_id, st.tenant_id, sc.tenant_id
    ON CONFLICT (tenant_id, school_id) DO UPDATE SET
        students   = EXCLUDED.students,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_school_student_counts_update()
RETURNS TRIGGER AS $$
BEGIN
    WITH affected_schools AS (
        SELECT DISTINCT school_id FROM inserted_rows
        UNION
        SELECT DISTINCT school_id FROM deleted_rows
    )
    INSERT INTO school_member_counts (tenant_id, school_id, students, updated_at)
    SELECT
        COALESCE(st.tenant_id, sc.tenant_id),
        s.school_id,
        COUNT(st.id) AS students,
        NOW()
    FROM affected_schools s
    LEFT JOIN cbc_students st
        ON st.school_id = s.school_id
       AND st.is_active  = true
    LEFT JOIN cbc_schools sc ON sc.id = s.school_id
    GROUP BY s.school_id, st.tenant_id, sc.tenant_id
    ON CONFLICT (tenant_id, school_id) DO UPDATE SET
        students   = EXCLUDED.students,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Fires on INSERT
CREATE TRIGGER trg_cbc_students_counts_insert
    AFTER INSERT ON cbc_students
    REFERENCING NEW TABLE AS inserted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_student_counts_insert();

-- Fires on DELETE
CREATE TRIGGER trg_cbc_students_counts_delete
    AFTER DELETE ON cbc_students
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_student_counts_delete();

-- Fires on UPDATE
CREATE TRIGGER trg_cbc_students_counts_update
    AFTER UPDATE ON cbc_students
    REFERENCING NEW TABLE AS inserted_rows OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_student_counts_update();
