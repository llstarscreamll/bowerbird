DROP INDEX IF EXISTS ix_catalog_items_name_id;
DROP TABLE IF EXISTS catalog_import_errors;
DROP TABLE IF EXISTS catalog_imports;

ALTER TABLE catalog_items
    DROP CONSTRAINT IF EXISTS catalog_items_creation_source_check;

ALTER TABLE catalog_items
    ADD CONSTRAINT catalog_items_creation_source_check
        CHECK (creation_source IN ('manual', 'invoice'));
