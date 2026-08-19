-- Migration: 000027_payments
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: payments

CREATE TABLE IF NOT EXISTS payments (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID          NOT NULL,
    invoice_id     UUID          NOT NULL,
    amount         NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    parent_id      UUID          NULL,
    payment_method payment_method_type NULL,
    reference_code VARCHAR(100)  NULL,
    recorded_by    UUID          NOT NULL,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_payments_tenant_invoice
        FOREIGN KEY (tenant_id, invoice_id)
        REFERENCES invoices(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_payments_tenant_parent
        FOREIGN KEY (tenant_id, parent_id)
        REFERENCES cbc_parents(tenant_id, id) ON DELETE SET NULL (parent_id),
    CONSTRAINT fk_payments_tenant_recorded_by
        FOREIGN KEY (tenant_id, recorded_by)
        REFERENCES users(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_payments_tenant     ON payments (tenant_id);
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id ON payments (invoice_id);
CREATE INDEX IF NOT EXISTS idx_payments_parent     ON payments (parent_id);
-- IMPROVE: M-Pesa reconciliation lookups by reference_code; partial keeps index small
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_reference_code
    ON payments (reference_code) WHERE reference_code IS NOT NULL;

COMMENT ON COLUMN payments.payment_method IS
    'Payment method type enum. Covers the four real Kenyan payment channels
     plus OTHER. Original free-text column migrated to enum in 000003_fix.';

-- ============================================================
-- TRIGGER: Sync invoice payment_status and amount_paid
-- BUG FIX: Split into 3 separate functions so each trigger only accesses the
--          transition table(s) available to it. The original single function
--          referenced both inserted_rows and deleted_rows regardless of the
--          trigger event, which would fail at runtime for INSERT and DELETE.
-- ============================================================

CREATE OR REPLACE FUNCTION fn_sync_invoice_payment_status_insert()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE invoices i
    SET
        amount_paid    = COALESCE(p.total_paid, 0),
        payment_status = (CASE
            WHEN i.payment_status = 'WAIVED'               THEN 'WAIVED'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) = 0             THEN 'UNPAID'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) >= i.amount_due THEN 'PAID'::invoice_payment_status
            ELSE 'PARTIAL'::invoice_payment_status
        END)
    FROM (SELECT DISTINCT invoice_id FROM inserted_rows) ai
    LEFT JOIN (
        SELECT invoice_id, SUM(amount) AS total_paid
        FROM payments
        GROUP BY invoice_id
    ) p ON p.invoice_id = ai.invoice_id
    WHERE i.id = ai.invoice_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_invoice_payment_status_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE invoices i
    SET
        amount_paid    = COALESCE(p.total_paid, 0),
        payment_status = (CASE
            WHEN i.payment_status = 'WAIVED'               THEN 'WAIVED'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) = 0             THEN 'UNPAID'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) >= i.amount_due THEN 'PAID'::invoice_payment_status
            ELSE 'PARTIAL'::invoice_payment_status
        END)
    FROM (SELECT DISTINCT invoice_id FROM deleted_rows) ai
    LEFT JOIN (
        SELECT invoice_id, SUM(amount) AS total_paid
        FROM payments
        GROUP BY invoice_id
    ) p ON p.invoice_id = ai.invoice_id
    WHERE i.id = ai.invoice_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_invoice_payment_status_update()
RETURNS TRIGGER AS $$
BEGIN
    WITH affected_invoices AS (
        SELECT DISTINCT invoice_id FROM inserted_rows
        UNION
        SELECT DISTINCT invoice_id FROM deleted_rows
    )
    UPDATE invoices i
    SET
        amount_paid    = COALESCE(p.total_paid, 0),
        payment_status = (CASE
            WHEN i.payment_status = 'WAIVED'               THEN 'WAIVED'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) = 0             THEN 'UNPAID'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) >= i.amount_due THEN 'PAID'::invoice_payment_status
            ELSE 'PARTIAL'::invoice_payment_status
        END)
    FROM affected_invoices ai
    LEFT JOIN (
        SELECT invoice_id, SUM(amount) AS total_paid
        FROM payments
        GROUP BY invoice_id
    ) p ON p.invoice_id = ai.invoice_id
    WHERE i.id = ai.invoice_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Fires on INSERT
CREATE TRIGGER trg_sync_invoice_payment_status_insert
    AFTER INSERT ON payments
    REFERENCING NEW TABLE AS inserted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_invoice_payment_status_insert();

-- Fires on DELETE
CREATE TRIGGER trg_sync_invoice_payment_status_delete
    AFTER DELETE ON payments
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_invoice_payment_status_delete();

-- Fires on UPDATE
CREATE TRIGGER trg_sync_invoice_payment_status_update
    AFTER UPDATE ON payments
    REFERENCING NEW TABLE AS inserted_rows OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_invoice_payment_status_update();



CREATE TRIGGER trg_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN payments.updated_at IS
    'Tracks payment record corrections and reconciliations.';
