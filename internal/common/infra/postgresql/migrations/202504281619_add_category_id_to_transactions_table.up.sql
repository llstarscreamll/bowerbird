
ALTER TABLE transactions
ADD COLUMN category_id VARCHAR(26);

CREATE INDEX idx_transactions_category_id ON transactions(category_id);
