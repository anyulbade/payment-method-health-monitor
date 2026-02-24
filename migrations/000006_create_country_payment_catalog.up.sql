CREATE TABLE country_payment_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country_code VARCHAR(2) NOT NULL REFERENCES countries(code),
    payment_method_code VARCHAR(50) NOT NULL,
    market_share_pct DECIMAL(5,2),
    is_essential BOOLEAN NOT NULL DEFAULT false,
    source VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(country_code, payment_method_code),
    CONSTRAINT chk_market_share CHECK (market_share_pct IS NULL OR (market_share_pct >= 0 AND market_share_pct <= 100))
);
