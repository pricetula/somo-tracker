-- Migration: 000002_create_tenants_and_users
-- Purpose: Create the core multi-tenant tables (tenants, users) with RLS isolation.
--
-- Dependencies:
--   - 000001_init_extensions (must run first; provides pgcrypto for gen_random_uuid()).
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- ============================================================================
-- Section 1: tenants
-- ============================================================================

-- Maps 1:1 to Stytch organizations. Each organization corresponds to exactly one
-- Somotracker tenant. The stytch_org_id column holds the Stytch OIDC org ID,
-- which is the authoritative source of truth for SSO / SAML membership.
CREATE TABLE tenants (
    id          UUID        NOT NULL    DEFAULT gen_random_uuid()
                                     PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(255) NOT NULL
                                     UNIQUE,
    stytch_org_id VARCHAR(255) NOT NULL
                                     UNIQUE,
    created_at  TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: tenants_slug covers lookups by slug (e.g. subdomain routing, reverse
-- proxy rewrites) and tenant resolution during onboarding.
CREATE INDEX tenants_slug_idx ON tenants (slug);

-- Index: tenants_stytch_org_id covers the unique Stytch OIDC org join, ensuring
-- O(1) lookups during auth token validation.
CREATE INDEX tenants_stytch_org_id_idx ON tenants (stytch_org_id);

-- Index: tenants_created_at supports time-bucketed queries and admin listing.
CREATE INDEX tenants_created_at_idx ON tenants (created_at DESC);

-- ============================================================================
-- Section 2: users
-- ============================================================================

-- Each row belongs to exactly one tenant. Users are identified by email within
-- a tenant scope; the same email address may exist in different tenants.
-- external_auth_id holds the Stytch user ID (or other auth-provider subject)
-- used to link a local user row to an external identity.
CREATE TABLE users (
    id              UUID        NOT NULL    DEFAULT gen_random_uuid()
                                       PRIMARY KEY,
    email           VARCHAR(255) NOT NULL,
    tenant_id       UUID        NOT NULL    REFERENCES tenants(id)
                                       ON DELETE CASCADE,
    full_name       VARCHAR(255) NOT NULL    DEFAULT '',
    is_active       BOOLEAN      NOT NULL    DEFAULT TRUE,
    external_auth_id VARCHAR(255)             UNIQUE,
    created_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: users_tenant_id is the single most important index for this table.
-- It is used by RLS policy evaluation (every row is checked against the session
-- tenant) and by all tenant-scoped queries (user listing, member lookup, etc).
CREATE INDEX users_tenant_id_idx ON users (tenant_id);

-- Index: users_email supports fast lookups during auth flows (magic-link,
-- passwordless, SSO callback) without scanning every tenant.
CREATE INDEX users_email_idx ON users (email);

-- Index: users_external_auth_id enables O(1) lookup when a Stytch token is
-- decoded and we need to hydrate the local user row.
CREATE INDEX users_external_auth_id_idx ON users (external_auth_id);

-- Index: users_updated_at supports stale-user cleanup jobs and last-seen
-- reporting.
CREATE INDEX users_updated_at_idx ON users (updated_at DESC);

-- ============================================================================
-- Section 3: Comments (documentation)
-- ============================================================================

COMMENT ON TABLE  tenants                           IS 'Maps 1:1 to a Stytch OIDC organization. All Somotracker data is scoped under exactly one tenant row.';
COMMENT ON COLUMN tenants.id                        IS 'Auto-generated UUID primary key. No external meaning — treat as opaque.';
COMMENT ON COLUMN tenants.name                      IS 'Human-readable organization display name (e.g. "Acme Corp").';
COMMENT ON COLUMN tenants.slug                      IS 'URL-safe, lowercase, hyphen-separated identifier. Used for subdomains and admin routing. Must be globally unique.';
COMMENT ON COLUMN tenants.stytch_org_id             IS 'The Stytch OIDC organization ID. This is the authoritative identity anchor for SSO / SAML membership. Unique.';
COMMENT ON COLUMN tenants.created_at                IS 'UTC timestamp of row creation.';

COMMENT ON TABLE  users                             IS 'Per-tenant user accounts. Rows are scoped to exactly one tenant via the foreign key on tenant_id. Users are identified by email within a tenant scope; the same email may appear in different tenants.';
COMMENT ON COLUMN users.id                          IS 'Auto-generated UUID primary key. No external meaning — treat as opaque.';
COMMENT ON COLUMN users.email                       IS 'Canonical email address. Unique within a tenant but not globally. Case-insensitive comparison recommended at the application layer.';
COMMENT ON COLUMN users.tenant_id                   IS 'Foreign key to tenants(id). Every user must belong to exactly one tenant. Deleting the tenant cascades this row.';
COMMENT ON COLUMN users.full_name                   IS 'Display name chosen by the user. May be empty.';
COMMENT ON COLUMN users.is_active                   IS 'Soft-disable flag. Inactive users cannot authenticate but their rows are retained for audit purposes.';
COMMENT ON COLUMN users.external_auth_id            IS 'Stytch user ID or equivalent auth-provider subject. Enables linking the local row to the external identity without querying by email.';
COMMENT ON COLUMN users.created_at                  IS 'UTC timestamp of row creation.';
COMMENT ON COLUMN users.updated_at                  IS 'UTC timestamp of last modification (updated by application code, not triggers by default).';

-- ============================================================================
-- Section 4: Row-Level Security (RLS)
-- ============================================================================

-- RLS is enabled but NOT forced, meaning the table owner (and any BYPASSRLS
-- roles used by migration tooling) bypasses the policy. Production application
-- queries run under a role that has RLS enforced.
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy: users_tenant_isolation
--
-- Every row-level operation (SELECT, INSERT, UPDATE, DELETE) is gated by a
-- comparison of the row's tenant_id with the value of the session GUC
-- 'app.current_tenant_id'.  The pattern NULLIF(..., '')::UUID is used so that:
--
--   - If the session variable is unset or empty, the comparison yields NULL,
--     which is never equal to any tenant_id → zero rows are visible (fail-closed).
--   - If the session variable contains a valid UUID string, it is cast and
--     compared; only rows whose tenant_id matches are returned.
--   - If the session variable contains an invalid UUID string, the cast raises
--     a runtime error, which propagates as a 500 and aborts the query — also
--     fail-closed.
--
-- This mirrors the contract established by database.WithTenantTx, which sets
-- 'app.current_tenant_id' via SET LOCAL at the start of every application
-- transaction. Applications must therefore always perform database operations
-- inside a tenant-scoped transaction; raw queries outside that scope will
-- receive zero rows.
--
-- The policy is named with a suffix so that future per-operation policies can
-- coexist (e.g. users_tenant_insert, users_tenant_update) if fine-grained
-- control becomes necessary.
CREATE POLICY users_tenant_isolation ON users
    FOR ALL
    TO   PUBLIC
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::UUID
    );
