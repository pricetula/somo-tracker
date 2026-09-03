-- Migration: 000001_init_extensions (down)
-- Purpose: Reverse the extensions enabled by the up migration.
--
-- Extensions are dropped in reverse order of creation. IF EXISTS guards against
-- the case where pg_uuidv7 was never installed in the first place (e.g. a
-- lean image), preventing a rollback failure on that object.

-- pg_uuidv7 must be dropped before pgcrypto/uuid-ossp because other objects
-- (e.g. columns typed as uuidv7) may depend on it.
DROP EXTENSION IF EXISTS "pg_uuidv7";

-- pgcrypto provides gen_random_uuid(); dropping it may break existing columns
-- typed as uuid if no other extension provides the underlying type.
-- Operators should verify no production columns depend on pgcrypto before
-- rolling back this migration.
DROP EXTENSION IF EXISTS "pgcrypto";

-- uuid-ossp is dropped last.
DROP EXTENSION IF EXISTS "uuid-ossp";
