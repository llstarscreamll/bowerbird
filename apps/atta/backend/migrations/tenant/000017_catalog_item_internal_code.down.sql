DROP INDEX IF EXISTS ux_catalog_items_internal_code;

ALTER TABLE catalog_items
    DROP CONSTRAINT IF EXISTS catalog_items_internal_code_not_blank;

ALTER TABLE catalog_items
    DROP COLUMN IF EXISTS internal_code;
