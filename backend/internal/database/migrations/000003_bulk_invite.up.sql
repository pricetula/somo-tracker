-- Migration: 000003_bulk_invite
-- Extends import_failure_type with bulk-invitation failure categories,
-- and adds a school-scoped email uniqueness index on invitations.

BEGIN;

-- Extend the import_failure_type enum with new values for bulk staff invitations.
-- ALTER TYPE ... ADD VALUE cannot be done inside a transaction block in older
-- Postgres versions, but Postgres 12+ allows it IF the enum is not referenced
-- by any table in the same transaction. Since import_job_failures references
-- it and that table already exists, we do each ADD VALUE in its own DDL statement.
-- These run outside the transaction because ALTER TYPE ... ADD VALUE is not
-- allowed inside a transaction block for existing enums in some Postgres versions.
ALTER TYPE import_failure_type ADD VALUE IF NOT EXISTS 'DUPLICATE_EMAIL';
ALTER TYPE import_failure_type ADD VALUE IF NOT EXISTS 'INVALID_EMAIL_FORMAT';
ALTER TYPE import_failure_type ADD VALUE IF NOT EXISTS 'STYTCH_API_ERROR';
ALTER TYPE import_failure_type ADD VALUE IF NOT EXISTS 'INVITATION_INSERT_FAILED';

-- Add a school-scoped unique index on pending invitations.
-- This prevents race conditions where two concurrent chunks try to invite
-- the same email for the same school. The index only covers pending invitations,
-- so the same email can be invited again after the first invite was accepted,
-- expired, or revoked.
CREATE UNIQUE INDEX IF NOT EXISTS uq_invitations_school_email_pending
    ON invitations (school_id, email)
    WHERE status = 'pending';

COMMIT;
