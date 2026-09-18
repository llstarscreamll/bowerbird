DROP TABLE IF EXISTS party_addresses;
DROP TABLE IF EXISTS party_phones;
DROP TABLE IF EXISTS party_emails;

ALTER TABLE parties
    DROP CONSTRAINT IF EXISTS parties_scheme_id_check,
    DROP CONSTRAINT IF EXISTS parties_taxpayer_kind_check;

ALTER TABLE parties
    DROP COLUMN IF EXISTS scheme_id,
    DROP COLUMN IF EXISTS taxpayer_kind,
    DROP COLUMN IF EXISTS tax_level_codes;
