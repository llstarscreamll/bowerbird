ALTER TABLE transactions
ALTER COLUMN category_setter_id SET DEFAULT '';
ALTER TABLE transactions
ALTER COLUMN category_setter_id DROP NOT NULL;

ALTER TABLE transactions
ALTER COLUMN category_id SET DEFAULT '';
ALTER TABLE transactions
ALTER COLUMN category_id DROP NOT NULL;

