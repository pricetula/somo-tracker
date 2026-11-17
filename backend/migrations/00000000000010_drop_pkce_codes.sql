-- +goose Up
DROP TRIGGER IF EXISTS trigger_set_timestamp ON pkce_codes;
DROP POLICY IF EXISTS pkce_codes_tenant_isolation ON pkce_codes;
ALTER TABLE pkce_codes DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS pkce_codes;

-- +goose Down
CREATE TABLE pkce_codes (
    pending_onboarding_id UUID NOT NULL REFERENCES pending_onboarding(id) ON DELETE CASCADE,
    code_verifier_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (pending_onboarding_id)
);

CREATE INDEX IF NOT EXISTS idx_pkce_expires ON pkce_codes (expires_at);
CREATE INDEX IF NOT EXISTS idx_pkce_codes_deleted_at ON pkce_codes (deleted_at) WHERE deleted_at IS NULL;

CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON pkce_codes FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
ALTER TABLE pkce_codes ENABLE ROW LEVEL SECURITY;
CREATE POLICY pkce_codes_tenant_isolation ON pkce_codes
    FOR ALL
    TO PUBLIC
    USING (
        pending_onboarding_id IN (
            SELECT id FROM pending_onboarding WHERE tenant_id = current_setting('app.current_tenant_id', true)::uuid
        )
    );
ALTER TABLE pkce_codes FORCE ROW LEVEL SECURITY;
