ALTER TABLE catalog_items
    ADD COLUMN internal_code VARCHAR(255);

UPDATE catalog_items AS i
SET internal_code = a.value
FROM (
    SELECT DISTINCT ON (item_id) item_id, value
    FROM catalog_item_aliases
    WHERE scheme = 'internal_sku'
    ORDER BY item_id, created_at ASC
) AS a
WHERE a.item_id = i.id;

DELETE FROM catalog_item_aliases
WHERE scheme = 'internal_sku';

ALTER TABLE catalog_items
    ADD CONSTRAINT catalog_items_internal_code_not_blank
        CHECK (internal_code IS NULL OR btrim(internal_code) <> '');

CREATE UNIQUE INDEX ux_catalog_items_internal_code
    ON catalog_items (internal_code)
    WHERE internal_code IS NOT NULL;
