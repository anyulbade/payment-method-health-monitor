CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE integration_costs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_method_code VARCHAR(50) NOT NULL REFERENCES payment_methods(code),
    country_code VARCHAR(2) NOT NULL REFERENCES countries(code),
    monthly_fixed_cost_usd DECIMAL(10,2) NOT NULL DEFAULT 0,
    per_transaction_cost_usd DECIMAL(8,4) NOT NULL DEFAULT 0,
    percentage_fee DECIMAL(5,4) NOT NULL DEFAULT 0,
    effective_from DATE NOT NULL,
    effective_to DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_cost_positive CHECK (monthly_fixed_cost_usd >= 0 AND per_transaction_cost_usd >= 0 AND percentage_fee >= 0),
    CONSTRAINT chk_cost_dates CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT excl_cost_overlap EXCLUDE USING gist (
        payment_method_code WITH =,
        country_code WITH =,
        daterange(effective_from, effective_to, '[)') WITH &&
    )
);
