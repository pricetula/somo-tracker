-- Migration: 000026_invoice_items (rollback)
-- SomoTracker — rollback for 000026_invoice_items.


DROP TABLE IF EXISTS invoice_items CASCADE;
