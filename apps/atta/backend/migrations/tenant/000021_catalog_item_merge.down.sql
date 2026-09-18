DROP TABLE IF EXISTS catalog_item_not_duplicates;

DROP INDEX IF EXISTS ix_catalog_items_merged_into_id;

ALTER TABLE catalog_items
    DROP CONSTRAINT IF EXISTS catalog_items_merged_into_chk;

ALTER TABLE catalog_items
    DROP CONSTRAINT IF EXISTS catalog_items_status_check;

ALTER TABLE catalog_items
    DROP COLUMN IF EXISTS merged_into_id;

ALTER TABLE catalog_items
    ADD CONSTRAINT catalog_items_status_check
        CHECK (status IN ('provisional', 'confirmed'));
