CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_method_code VARCHAR(50) NOT NULL REFERENCES payment_methods(code),
    country_code VARCHAR(2) NOT NULL REFERENCES countries(code),
    currency VARCHAR(3) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    amount_usd DECIMAL(15,2) NOT NULL,
    status VARCHAR(20) NOT NULL,
    merchant_id VARCHAR(100),
    customer_id VARCHAR(100),
    transaction_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_txn_status CHECK (status IN ('APPROVED','DECLINED','PENDING','REFUNDED')),
    CONSTRAINT chk_txn_amount CHECK (amount > 0),
    CONSTRAINT chk_txn_amount_usd CHECK (amount_usd > 0)
);

CREATE INDEX idx_txn_date_pm_country_status
    ON transactions(transaction_date, payment_method_code, country_code, status)
    INCLUDE (amount_usd);

CREATE INDEX idx_txn_month ON transactions(DATE_TRUNC('month', transaction_date));
CREATE INDEX idx_txn_week ON transactions(DATE_TRUNC('week', transaction_date));
