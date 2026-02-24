package dto

import "time"

type TransactionResponse struct {
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

type BatchTransactionResponse struct {
	Inserted int                   `json:"inserted"`
	Results  []TransactionResponse `json:"results"`
}

type ValidationError struct {
	Index   int    `json:"index,omitempty"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorListResponse struct {
	Error  string            `json:"error"`
	Errors []ValidationError `json:"errors,omitempty"`
}

type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}
