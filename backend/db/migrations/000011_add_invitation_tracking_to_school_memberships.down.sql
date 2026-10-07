-- Migration: Revert invitation tracking columns
-- File: backend/db/migrations/000011_add_invitation_tracking_to_school_memberships.down.sql

ALTER TABLE school_memberships
    DROP COLUMN IF EXISTS invited_at,
    DROP COLUMN IF EXISTS invited_by,
    DROP COLUMN IF EXISTS accepted_at;

DROP INDEX IF EXISTS school_memberships_invited_by_idx;
DROP INDEX IF EXISTS school_memberships_invited_at_idx;
