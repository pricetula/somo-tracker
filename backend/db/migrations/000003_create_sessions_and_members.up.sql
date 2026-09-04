-- Migration: 000003_create_sessions_and_members
-- Purpose: Persistent session tokens and Member-first-class model for B2B SSO.
--
-- sessions:
--   Stores opaque, server-issued session tokens. The raw Stytch session token
--   is NOT stored here; only a local opaque token that maps to the cached
--   Stytch token in Redis. This keeps the DB harmless even if leaked — the
--   attacker still needs the Redis token to authenticate.
--
-- members:
--   Mirrors Stytch's Member model so we can store roles/permissions locally
--   without querying Stytch on every request. The stytch_member_id is the
--   canonical external identity; the local user_id is the Somotracker user
--   row (which participates in RLS).  The member row lives in the same tenant
--   as the user row but is NOT RLS-protected because it is cross-tenant for
--   admin lookups (e.g. listing all members of an org).  We rely on the API
--   handler / service layer to enforce org-scoped access instead.
--
-- Dependencies:
--   - 000002_create_tenants_and_users (must run first; provides tenants + users).
--
-- Usage:
--   migrate -database "$DATABASE_URL" -path backend/db/migrations up

-- ============================================================================
-- Section 1: sessions
-- ============================================================================

CREATE TABLE sessions (
    id                  UUID        NOT NULL    DEFAULT gen_random_uuid()
                                         PRIMARY KEY,
    -- The opaque local token. 256 bits of crypto/rand, hex-encoded = 64 chars.
    -- Never expose this in URLs; it belongs only in HttpOnly cookies.
    token               VARCHAR(64) NOT NULL
                                         UNIQUE,
    -- Opaque reference to the Stytch session. Used to validate / revoke in Stytch.
    stytch_session_id   VARCHAR(255) NOT NULL,
    user_id             UUID        NOT NULL    REFERENCES users(id)
                                         ON DELETE CASCADE,
    tenant_id           UUID        NOT NULL    REFERENCES tenants(id)
                                         ON DELETE CASCADE,
    -- Rolling expiry. Default 7 days; refreshed on each valid request.
    expires_at          TIMESTAMPTZ  NOT NULL,
    -- UTC timestamp of creation.
    created_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    -- UTC timestamp of last use (last-valid-request refresh).
    last_seen_at        TIMESTAMPTZ  NOT NULL    DEFAULT NOW()
);

-- Index: sessions_token covers the primary lookup path (cookie value → session row).
CREATE INDEX sessions_token_idx ON sessions (token);

-- Index: sessions_stytch_session_id enables global Stytch revocation (e.g. logout-all).
CREATE INDEX sessions_stytch_session_id_idx ON sessions (stytch_session_id);

-- Index: sessions_user_id supports per-user session invalidation (e.g. password change).
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

-- Index: sessions_expires_at supports the background expired-session cleanup job.
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- Trigger: automatically update last_seen_at on row update.
CREATE OR REPLACE FUNCTION update_last_seen()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_seen_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sessions_last_seen_trg
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_last_seen();

-- ============================================================================
-- Section 2: members
-- ============================================================================

-- Member rows are created / updated atomically with the user row during the
-- magic-link provisioning transaction. The member is the B2B identity;
-- the user row is the Somotracker application identity.
CREATE TABLE members (
    id                  UUID        NOT NULL    DEFAULT gen_random_uuid()
                                         PRIMARY KEY,
    -- Maps to Stytch B2B Member.member_id.
    stytch_member_id    VARCHAR(255) NOT NULL
                                         UNIQUE,
    user_id             UUID        NOT NULL    REFERENCES users(id)
                                         ON DELETE CASCADE,
    tenant_id           UUID        NOT NULL    REFERENCES tenants(id)
                                         ON DELETE CASCADE,
    -- Roles are stored as an array of text for flexibility. Somotracker
    -- currently uses: "admin", "member". Future roles can be added without
    -- schema changes.  The application service layer enforces permissions.
    roles               TEXT[]       NOT NULL    DEFAULT '{member}',
    -- Raw Stytch member object stored as JSONB for audit / debugging.
    -- Sensitive fields (e.g. untrusted third-party metadata) are stripped
    -- before storage; only the fields we explicitly read are persisted.
    stytch_member_raw   JSONB,
    created_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL    DEFAULT NOW(),
    -- Prevent duplicate member records for the same (tenant, stytch_member_id).
    CONSTRAINT members_tenant_member_uniq UNIQUE (tenant_id, stytch_member_id)
);

-- Index: members_stytch_member_id covers lookups during auth provisioning.
CREATE INDEX members_stytch_member_id_idx ON members (stytch_member_id);

-- Index: members_user_id supports user→member reverse lookups (e.g. permission checks).
CREATE INDEX members_user_id_idx ON members (user_id);

-- Trigger: automatically update updated_at on row change.
CREATE TRIGGER members_updated_at_trg
    BEFORE UPDATE ON members
    FOR EACH ROW
    EXECUTE FUNCTION update_last_seen();

-- ============================================================================
-- Section 3: Comments
-- ============================================================================

COMMENT ON TABLE  sessions                      IS 'Server-issued opaque session tokens. Tokens are stored in HttpOnly cookies; the raw Stytch session token is cached only in Redis.';
COMMENT ON COLUMN sessions.id                   IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN sessions.token                IS 'Opaque 256-bit token, hex-encoded. Never exposed to JavaScript or URLs.';
COMMENT ON COLUMN sessions.stytch_session_id    IS 'Opaque Stytch session ID used for revocation / validation against Stytch.';
COMMENT ON COLUMN sessions.user_id              IS 'FK to users(id). Deleting the user cascades all sessions.';
COMMENT ON COLUMN sessions.tenant_id            IS 'FK to tenants(id). Used for fast tenant-scoped session lookup.';
COMMENT ON COLUMN sessions.expires_at            IS 'Rolling expiry timestamp. Default 7 days.';
COMMENT ON COLUMN sessions.created_at           IS 'UTC timestamp of session creation.';
COMMENT ON COLUMN sessions.last_seen_at          IS 'UTC timestamp of last validated request. Used for rolling expiry refresh.';

COMMENT ON TABLE  members                       IS 'B2B member identity mirroring Stytch. Created/updated atomically with users during magic-link provisioning.';
COMMENT ON COLUMN members.id                     IS 'Auto-generated UUID primary key.';
COMMENT ON COLUMN members.stytch_member_id       IS 'Stytch B2B Member.member_id. Unique across the entire system.';
COMMENT ON COLUMN members.user_id                IS 'FK to users(id). The Somotracker application identity.';
COMMENT ON COLUMN members.tenant_id              IS 'FK to tenants(id). The B2B organization.';
COMMENT ON COLUMN members.roles                  IS 'Array of Somotracker role names. Somotracker currently uses: admin, member.';
COMMENT ON COLUMN members.stytch_member_raw      IS 'Cached Stytch member object (JSONB) for audit/debugging. Sensitive metadata fields are stripped before storage.';
COMMENT ON COLUMN members.created_at             IS 'UTC timestamp of member creation.';
COMMENT ON COLUMN members.updated_at            IS 'UTC timestamp of last modification.';
