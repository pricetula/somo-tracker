-- Migration: 000002_create_tenants_and_users (down)
-- Purpose: Reverse the creation of tenants and users, removing RLS and indexes
-- before dropping the tables themselves.
--
-- Order of removal is the strict inverse of creation:
--   1. RLS policy (must exist before table drop, but policy lives on table)
--   2. Indexes (must be removed before table drop; dropped implicitly with table,
--      but explicit removal is safer for rollback scripts and avoids errors)
--   3. Users table (depends on tenants via FK)
--   4. Indexes on tenants
--   5. Tenants table
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations down

-- ============================================================================
-- Section 1: Drop RLS policy
-- ============================================================================

-- Drop the isolation policy before disabling RLS or dropping the table.
DROP POLICY IF EXISTS users_tenant_isolation ON users;

-- ============================================================================
-- Section 2: Drop indexes on users (before table drop for explicitness)
-- ============================================================================

DROP INDEX IF EXISTS users_tenant_id_idx;
DROP INDEX IF EXISTS users_email_idx;
DROP INDEX IF EXISTS users_external_auth_id_idx;
DROP INDEX IF EXISTS users_updated_at_idx;

-- ============================================================================
-- Section 3: Drop users table (depends on tenants via ON DELETE CASCADE FK)
-- ============================================================================

DROP TABLE IF EXISTS users;

-- ============================================================================
-- Section 4: Drop indexes on tenants
-- ============================================================================

DROP INDEX IF EXISTS tenants_slug_idx;
DROP INDEX IF EXISTS tenants_stytch_org_id_idx;
DROP INDEX IF EXISTS tenants_created_at_idx;

-- ============================================================================
-- Section 5: Drop tenants table
-- ============================================================================

DROP TABLE IF EXISTS tenants;
