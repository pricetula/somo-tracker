-- Migration: 000001_auth_tables (rollback)

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
DROP TYPE IF EXISTS user_role;
