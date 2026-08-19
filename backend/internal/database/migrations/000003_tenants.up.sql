-- Migration: 000003_tenants
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: tenants

CREATE TABLE IF NOT EXISTS tenants (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    slug          VARCHAR(255) NOT NULL UNIQUE,
    stytch_org_id VARCHAR(255) NOT NULL UNIQUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug          ON tenants (slug);
CREATE INDEX IF NOT EXISTS idx_tenants_stytch_org_id ON tenants (stytch_org_id);

-- ---------------------------------------------------------------------------
-- USERS
-- ---------------------------------------------------------------------------
