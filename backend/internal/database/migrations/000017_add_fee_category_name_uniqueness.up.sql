-- Migration: 000017_add_fee_category_name_uniqueness
-- Enforce uniqueness of (tenant_id, school_id, name) on fee_categories
-- so duplicate category creation is caught at the DB layer.

ALTER TABLE fee_categories
    ADD CONSTRAINT uq_fee_categories_tenant_school_name
        UNIQUE (tenant_id, school_id, name);
