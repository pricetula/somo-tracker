-- Migration Down: 000003_bulk_invite
-- Note: Postgres does not support removing values from an enum.
-- The down migration drops the index but cannot revert the enum type.
-- This is acceptable — the extra enum values are harmless to existing code.

BEGIN;

DROP INDEX IF EXISTS uq_invitations_school_email_pending;

COMMIT;
