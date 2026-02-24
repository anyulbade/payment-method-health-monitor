package dto

import "time"

type CreateTransactionRequest struct {
	PaymentMethodCode string    `json:"payment_method_code" binding:"required"`
	CountryCode       string    `json:"country_code" binding:"required"`
	Currency          string    `json:"currency" binding:"required"`
	Amount            float64   `json:"amount" binding:"required,gt=0"`
	Status            string    `json:"status" binding:"required,oneof=APPROVED DECLINED PENDING REFUNDED"`
	MerchantID        string    `json:"merchant_id"`
	CustomerID        string    `json:"customer_id"`
	TransactionDate   time.Time `json:"transaction_date" binding:"required"`
}

type BatchTransactionRequest struct {
	Transactions []CreateTransactionRequest `json:"transactions" binding:"required,min=1,max=500,dive"`
}
