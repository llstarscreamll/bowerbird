ALTER TABLE transactions
    DROP COLUMN category_id;

DROP INDEX idx_transactions_category_id;
