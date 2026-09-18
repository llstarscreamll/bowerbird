ALTER TABLE catalog_item_aliases
    ADD COLUMN source VARCHAR(32) NOT NULL DEFAULT 'invoice';

ALTER TABLE catalog_item_aliases
    ADD CONSTRAINT catalog_item_aliases_source_check CHECK (source IN ('invoice', 'manual'));

ALTER TABLE catalog_item_aliases DROP CONSTRAINT IF EXISTS catalog_item_aliases_party_id_fkey;
ALTER TABLE catalog_match_memories DROP CONSTRAINT IF EXISTS catalog_match_memories_party_id_fkey;
