-- Migration: Add invitation tracking columns to school_memberships
-- File: backend/db/migrations/000011_add_invitation_tracking_to_school_memberships.up.sql

ALTER TABLE school_memberships
    ADD COLUMN IF NOT EXISTS invited_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS accepted_at TIMESTAMPTZ;

-- Optional indexes for bulk invitation lookups
CREATE INDEX IF NOT EXISTS school_memberships_invited_by_idx ON school_memberships (invited_by);
CREATE INDEX IF NOT EXISTS school_memberships_invited_at_idx ON school_memberships (invited_at);
