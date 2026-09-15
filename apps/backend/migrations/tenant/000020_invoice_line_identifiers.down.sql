ALTER TABLE invoice_headers
    ADD CONSTRAINT invoice_headers_issuer_party_id_fkey
    FOREIGN KEY (issuer_party_id) REFERENCES parties(id) ON DELETE SET NULL;

ALTER TABLE invoice_lines
    ADD CONSTRAINT invoice_lines_item_id_fkey
    FOREIGN KEY (item_id) REFERENCES catalog_items(id) ON DELETE SET NULL;

ALTER TABLE invoice_lines ADD COLUMN item_code VARCHAR(100);

UPDATE invoice_lines
SET item_code = COALESCE(NULLIF(trim(seller_sku), ''), NULLIF(trim(gtin), ''), NULLIF(trim(buyer_code), ''));

ALTER TABLE invoice_lines
    DROP COLUMN IF EXISTS buyer_code,
    DROP COLUMN IF EXISTS seller_sku,
    DROP COLUMN IF EXISTS gtin;
