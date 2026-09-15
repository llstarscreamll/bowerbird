ALTER TABLE catalog_item_aliases
    ADD CONSTRAINT catalog_item_aliases_party_id_fkey
    FOREIGN KEY (party_id) REFERENCES parties(id) ON DELETE SET NULL;

ALTER TABLE catalog_match_memories
    ADD CONSTRAINT catalog_match_memories_party_id_fkey
    FOREIGN KEY (party_id) REFERENCES parties(id) ON DELETE SET NULL;

ALTER TABLE catalog_item_aliases DROP CONSTRAINT IF EXISTS catalog_item_aliases_source_check;
ALTER TABLE catalog_item_aliases DROP COLUMN IF EXISTS source;
