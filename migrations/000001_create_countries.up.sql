CREATE TABLE countries (
    code VARCHAR(2) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    fx_rate_to_usd DECIMAL(12,8) NOT NULL
);

ALTER TABLE countries ADD CONSTRAINT chk_country_code CHECK (code ~ '^[A-Z]{2}$');
ALTER TABLE countries ADD CONSTRAINT chk_currency CHECK (currency ~ '^[A-Z]{3}$');
ALTER TABLE countries ADD CONSTRAINT chk_fx_rate CHECK (fx_rate_to_usd > 0);
