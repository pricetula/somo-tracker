-- Migration: 000002_core_functions (rollback)
-- SomoTracker — rollback for 000002_core_functions.

DROP FUNCTION IF EXISTS fn_current_tenant_id CASCADE;
DROP FUNCTION IF EXISTS fn_timerange CASCADE;
DROP FUNCTION IF EXISTS fn_set_updated_at CASCADE;
