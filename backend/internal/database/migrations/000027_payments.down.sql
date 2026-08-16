-- Migration: 000027_payments (rollback)
-- SomoTracker — rollback for 000027_payments.

DROP FUNCTION IF EXISTS fn_sync_invoice_payment_status_insert CASCADE;
DROP FUNCTION IF EXISTS fn_sync_invoice_payment_status_delete CASCADE;
DROP FUNCTION IF EXISTS fn_sync_invoice_payment_status_update CASCADE;

DROP TABLE IF EXISTS payments CASCADE;
