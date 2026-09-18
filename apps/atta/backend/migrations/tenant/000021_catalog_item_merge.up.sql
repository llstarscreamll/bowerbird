ALTER TABLE catalog_items
    DROP CONSTRAINT IF EXISTS catalog_items_status_check;

ALTER TABLE catalog_items
    ADD COLUMN merged_into_id CHAR(26) REFERENCES catalog_items(id);

ALTER TABLE catalog_items
    ADD CONSTRAINT catalog_items_status_check
        CHECK (status IN ('provisional', 'confirmed', 'merged'));

ALTER TABLE catalog_items
    ADD CONSTRAINT catalog_items_merged_into_chk
        CHECK (
            (status = 'merged' AND merged_into_id IS NOT NULL AND merged_into_id <> id)
            OR (status <> 'merged' AND merged_into_id IS NULL)
        );

CREATE INDEX ix_catalog_items_merged_into_id ON catalog_items (merged_into_id)
    WHERE merged_into_id IS NOT NULL;

CREATE TABLE catalog_item_not_duplicates (
    item_lo CHAR(26) NOT NULL REFERENCES catalog_items(id) ON DELETE CASCADE,
    item_hi CHAR(26) NOT NULL REFERENCES catalog_items(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (item_lo, item_hi),
    CONSTRAINT catalog_item_not_duplicates_order CHECK (item_lo < item_hi)
);
