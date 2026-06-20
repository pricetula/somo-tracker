-- ================================================================================
-- 🌱 SEED DATA ROLLBACK
-- ================================================================================

BEGIN;

DELETE FROM assessment_weight_configs;
DELETE FROM cbc_schools;
DELETE FROM tenants;

COMMIT;
