ALTER TABLE invoice_lines
    ADD COLUMN buyer_code VARCHAR(255),
    ADD COLUMN seller_sku VARCHAR(255),
    ADD COLUMN gtin VARCHAR(32);

UPDATE invoice_lines
SET gtin = regexp_replace(trim(item_code), '\s', '', 'g')
WHERE item_code IS NOT NULL
  AND regexp_replace(trim(item_code), '\s', '', 'g') ~ '^[0-9]+$'
  AND length(regexp_replace(trim(item_code), '\s', '', 'g')) IN (8, 12, 13, 14);

UPDATE invoice_lines
SET seller_sku = trim(item_code)
WHERE COALESCE(gtin, '') = ''
  AND item_code IS NOT NULL
  AND trim(item_code) <> '';

ALTER TABLE invoice_lines DROP COLUMN item_code;

ALTER TABLE invoice_headers DROP CONSTRAINT IF EXISTS invoice_headers_issuer_party_id_fkey;
ALTER TABLE invoice_lines DROP CONSTRAINT IF EXISTS invoice_lines_item_id_fkey;
