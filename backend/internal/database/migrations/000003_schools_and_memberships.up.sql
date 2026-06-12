-- Migration: 000003_schools_and_memberships
-- Creates schools and memberships tables for multi-school role-based access.

-- ============================================================================
-- TABLE: schools
-- Represents individual physical school campuses under a tenant.
-- ============================================================================
CREATE TABLE IF NOT EXISTS schools (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name               VARCHAR(255) NOT NULL,
    education_system_id UUID,
    tenant_id          UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_schools_tenant_id ON schools (tenant_id);

-- ============================================================================
-- TABLE: memberships
-- Maps users to specific schools with granular access roles.
-- A single user can have different roles at different schools.
-- ============================================================================
CREATE TABLE IF NOT EXISTS memberships (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    role       user_role   NOT NULL,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id  UUID        NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, school_id)
);

CREATE INDEX IF NOT EXISTS idx_memberships_user_id ON memberships (user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_school_id ON memberships (school_id);
CREATE INDEX IF NOT EXISTS idx_memberships_active ON memberships (is_active) WHERE is_active = TRUE;
