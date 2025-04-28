CREATE TABLE categories (
    id VARCHAR(26) PRIMARY KEY,
    wallet_id VARCHAR(26) NOT NULL,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) NOT NULL,
    icon VARCHAR(100) NOT NULL,
    created_by_id VARCHAR(26) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_categories_wallet_id ON categories(wallet_id);
