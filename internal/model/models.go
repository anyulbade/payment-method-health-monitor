package model

import (
	"time"
)

type Country struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Currency   string  `json:"currency"`
	FxRateUSD  float64 `json:"fx_rate_to_usd"`
}

type PaymentMethod struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Provider  string    `json:"provider,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PaymentMethodCountry struct {
	PaymentMethodCode string `json:"payment_method_code"`
	CountryCode       string `json:"country_code"`
}

type Transaction struct {
	ID                string    `json:"id"`
	PaymentMethodCode string    `json:"payment_method_code"`
	CountryCode       string    `json:"country_code"`
	Currency          string    `json:"currency"`
	Amount            float64   `json:"amount"`
	AmountUSD         float64   `json:"amount_usd"`
	Status            string    `json:"status"`
	MerchantID        string    `json:"merchant_id,omitempty"`
	CustomerID        string    `json:"customer_id,omitempty"`
	TransactionDate   time.Time `json:"transaction_date"`
	CreatedAt         time.Time `json:"created_at"`
}

type IntegrationCost struct {
	ID                    string    `json:"id"`
	PaymentMethodCode     string    `json:"payment_method_code"`
	CountryCode           string    `json:"country_code"`
	MonthlyFixedCostUSD   float64   `json:"monthly_fixed_cost_usd"`
	PerTransactionCostUSD float64   `json:"per_transaction_cost_usd"`
	PercentageFee         float64   `json:"percentage_fee"`
	EffectiveFrom         string    `json:"effective_from"`
	EffectiveTo           *string   `json:"effective_to,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

type CountryPaymentCatalog struct {
	ID                string  `json:"id"`
	CountryCode       string  `json:"country_code"`
	PaymentMethodCode string  `json:"payment_method_code"`
	MarketSharePct    float64 `json:"market_share_pct,omitempty"`
	IsEssential       bool    `json:"is_essential"`
	Source            string  `json:"source,omitempty"`
}
