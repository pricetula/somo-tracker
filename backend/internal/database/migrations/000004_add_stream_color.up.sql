-- Migration: 000004_add_stream_color
-- Adds a color column to cbc_streams for visual identification in the UI.

BEGIN;

ALTER TABLE cbc_streams
    ADD COLUMN IF NOT EXISTS color VARCHAR(50) NOT NULL DEFAULT '';

COMMIT;
