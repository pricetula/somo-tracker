-- +goose Up
-- RLS and Role Setup for Tenant Isolation
-- This migration creates the app role, enables RLS, and adds policies

-- 1. Create the application role (no BYPASSRLS, no superuser)
-- This role will be used by the application at runtime
-- Run this as a superuser (e.g., postgres) in a separate setup step, not via Goose
-- DO NOT run this in the standard migration flow - it requires SUPERUSER
-- 
-- CREATE ROLE somotracker_app WITH LOGIN PASSWORD '...' NOINHERIT;
-- GRANT USAGE ON SCHEMA public TO somotracker_app;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO somotracker_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO somotracker_app;
-- ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO somotracker_app;
-- ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO somotracker_app;

-- 2. Enable RLS on all tenant-scoped tables
-- Core domain tables
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE schools ENABLE ROW LEVEL SECURITY;
ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;

-- Auth tables
ALTER TABLE pending_onboarding ENABLE ROW LEVEL SECURITY;
ALTER TABLE app_session ENABLE ROW LEVEL SECURITY;
ALTER TABLE onboarding_ticket ENABLE ROW LEVEL SECURITY;
ALTER TABLE pkce_codes ENABLE ROW LEVEL SECURITY;

-- Academic tables (if they exist)
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'students') THEN
        ALTER TABLE students ENABLE ROW LEVEL SECURITY;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'enrollments') THEN
        ALTER TABLE enrollments ENABLE ROW LEVEL SECURITY;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'class_levels') THEN
        ALTER TABLE class_levels ENABLE ROW LEVEL SECURITY;
    END IF;
END $$;

-- 3. Create RLS policies using current_setting('app.current_tenant_id', true)
-- The application must SET LOCAL app.current_tenant_id = '<tenant-uuid>' at transaction start

-- Helper function to get current tenant from GUC
-- This avoids repeating the current_setting call in every policy
CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS UUID LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
$$;

-- tenants table: tenant can only see their own row
CREATE POLICY tenant_isolation ON tenants
    USING (id = current_tenant_id());

-- users table: users scoped to tenant
CREATE POLICY user_tenant_isolation ON users
    USING (tenant_id = current_tenant_id());

-- schools table: schools scoped to tenant
CREATE POLICY school_tenant_isolation ON schools
    USING (tenant_id = current_tenant_id());

-- memberships table: memberships scoped to tenant via school
CREATE POLICY membership_tenant_isolation ON memberships
    USING (school_id IN (SELECT id FROM schools WHERE tenant_id = current_tenant_id()));

-- pending_onboarding: scoped to tenant (when tenant_id is set)
CREATE POLICY pending_onboarding_tenant_isolation ON pending_onboarding
    USING (tenant_id = current_tenant_id() OR tenant_id IS NULL);

-- app_session: sessions scoped to tenant
CREATE POLICY app_session_tenant_isolation ON app_session
    USING (tenant_id = current_tenant_id());

-- onboarding_ticket: tickets scoped to tenant via pending_onboarding
CREATE POLICY onboarding_ticket_tenant_isolation ON onboarding_ticket
    USING (pending_onboarding_id IN (
        SELECT id FROM pending_onboarding WHERE tenant_id = current_tenant_id() OR tenant_id IS NULL
    ));

-- pkce_codes: codes scoped to tenant via pending_onboarding
CREATE POLICY pkce_codes_tenant_isolation ON pkce_codes
    USING (pending_onboarding_id IN (
        SELECT id FROM pending_onboarding WHERE tenant_id = current_tenant_id() OR tenant_id IS NULL
    ));

-- 4. FORCE ROW LEVEL SECURITY on all tables so even table owners must comply
-- This is critical - without FORCE RLS, the table owner (migration role) bypasses policies
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE schools FORCE ROW LEVEL SECURITY;
ALTER TABLE memberships FORCE ROW LEVEL SECURITY;
ALTER TABLE pending_onboarding FORCE ROW LEVEL SECURITY;
ALTER TABLE app_session FORCE ROW LEVEL SECURITY;
ALTER TABLE onboarding_ticket FORCE ROW LEVEL SECURITY;
ALTER TABLE pkce_codes FORCE ROW LEVEL SECURITY;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'students') THEN
        ALTER TABLE students FORCE ROW LEVEL SECURITY;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'enrollments') THEN
        ALTER TABLE enrollments FORCE ROW LEVEL SECURITY;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'class_levels') THEN
        ALTER TABLE class_levels FORCE ROW LEVEL SECURITY;
    END IF;
END $$;

-- 5. Grant necessary permissions to app role
-- (Run as superuser after creating role)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON tenants, users, schools, memberships,
--     pending_onboarding, app_session, onboarding_ticket, pkce_codes TO somotracker_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO somotracker_app;

-- 6. Create index to support RLS policy lookups
CREATE INDEX IF NOT EXISTS idx_memberships_school_tenant ON memberships (school_id, user_id);
CREATE INDEX IF NOT EXISTS idx_app_session_tenant_user ON app_session (tenant_id, user_id);

-- +goose Down
-- Drop policies (IF EXISTS for safety)
DROP POLICY IF EXISTS tenant_isolation ON tenants;
DROP POLICY IF EXISTS user_tenant_isolation ON users;
DROP POLICY IF EXISTS school_tenant_isolation ON schools;
DROP POLICY IF EXISTS membership_tenant_isolation ON memberships;
DROP POLICY IF EXISTS pending_onboarding_tenant_isolation ON pending_onboarding;
DROP POLICY IF EXISTS app_session_tenant_isolation ON app_session;
DROP POLICY IF EXISTS onboarding_ticket_tenant_isolation ON onboarding_ticket;
DROP POLICY IF EXISTS pkce_codes_tenant_isolation ON pkce_codes;

-- Disable RLS
ALTER TABLE tenants DISABLE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE schools DISABLE ROW LEVEL SECURITY;
ALTER TABLE memberships DISABLE ROW LEVEL SECURITY;
ALTER TABLE pending_onboarding DISABLE ROW LEVEL SECURITY;
ALTER TABLE app_session DISABLE ROW LEVEL SECURITY;
ALTER TABLE onboarding_ticket DISABLE ROW LEVEL SECURITY;
ALTER TABLE pkce_codes DISABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'students') THEN
        ALTER TABLE students DISABLE ROW LEVEL SECURITY;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'enrollments') THEN
        ALTER TABLE enrollments DISABLE ROW LEVEL SECURITY;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'class_levels') THEN
        ALTER TABLE class_levels DISABLE ROW LEVEL SECURITY;
    END IF;
END $$;

DROP FUNCTION IF EXISTS current_tenant_id();
DROP INDEX IF EXISTS idx_memberships_school_tenant;
DROP INDEX IF EXISTS idx_app_session_tenant_user;