-- +goose Up
-- NO-OP: pkce_codes table removed (schema removed in 00000000000002, RLS removed in 00000000000006).
-- Nothing to drop; database is rebuilt from scratch without PKCE.

-- +goose Down
-- NO-OP: pkce_codes table never created; nothing to restore.
