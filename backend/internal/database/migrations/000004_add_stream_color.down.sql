-- Migration Down: 000004_add_stream_color

BEGIN;

ALTER TABLE cbc_streams DROP COLUMN IF EXISTS color;

COMMIT;
