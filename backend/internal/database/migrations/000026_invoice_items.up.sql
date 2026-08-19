-- Migration: 000026_invoice_items
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: invoice_items

CREATE TABLE IF NOT EXISTS invoice_items (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID          NOT NULL,
    invoice_id      UUID          NOT NULL,
    fee_category_id UUID          NOT NULL,
    description     VARCHAR(255)  NULL,
    amount          NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_invoice_items_tenant_invoice
        FOREIGN KEY (tenant_id, invoice_id)
        REFERENCES invoices(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_items_tenant_fee_category
        FOREIGN KEY (tenant_id, fee_category_id)
        REFERENCES fee_categories(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_invoice_items_tenant       ON invoice_items (tenant_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id   ON invoice_items (invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_fee_category ON invoice_items (fee_category_id);



CREATE TRIGGER trg_invoice_items_updated_at
    BEFORE UPDATE ON invoice_items
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN invoice_items.updated_at IS
    'Tracks invoice line-item corrections.';
