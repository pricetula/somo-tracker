-- Migration: 000017_add_fee_category_name_uniqueness — Down
ALTER TABLE fee_categories
    DROP CONSTRAINT IF EXISTS uq_fee_categories_tenant_school_name;
