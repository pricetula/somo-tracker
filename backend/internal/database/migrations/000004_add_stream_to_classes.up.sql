-- Migration: 000004_add_stream_to_classes
-- ============================================================================
-- Adds a `stream` column to the `classes` table to track the stream/section
-- name (e.g., "East", "West") used during onboarding bulk generation.
-- ============================================================================

ALTER TABLE classes
    ADD COLUMN IF NOT EXISTS stream VARCHAR(100) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_classes_stream ON classes (stream);
