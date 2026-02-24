package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/anyulbade/payment-method-health-monitor/internal/dto"
	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

type TransactionHandler struct {
	svc *service.TransactionService
}

func NewTransactionHandler(svc *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

func (h *TransactionHandler) Create(c *gin.Context) {
	var req dto.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorListResponse{
			Error: "validation failed: " + err.Error(),
		})
		return
	}

	txn, err := h.svc.CreateTransaction(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorListResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.TransactionResponse{
		ID:                txn.ID,
		PaymentMethodCode: txn.PaymentMethodCode,
		CountryCode:       txn.CountryCode,
		Currency:          txn.Currency,
		Amount:            txn.Amount,
		AmountUSD:         txn.AmountUSD,
		Status:            txn.Status,
		MerchantID:        txn.MerchantID,
		CustomerID:        txn.CustomerID,
		TransactionDate:   txn.TransactionDate,
		CreatedAt:         txn.CreatedAt,
	})
}

func (h *TransactionHandler) CreateBatch(c *gin.Context) {
	var req dto.BatchTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorListResponse{
			Error: "validation failed: " + err.Error(),
		})
		return
	}

	txns, validationErrors, err := h.svc.CreateBatch(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorListResponse{
			Error: "batch insert failed: " + err.Error(),
		})
		return
	}

	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorListResponse{
			Error:  "batch validation failed",
			Errors: validationErrors,
		})
		return
	}

	results := make([]dto.TransactionResponse, len(txns))
	for i, txn := range txns {
		results[i] = dto.TransactionResponse{
			ID:                txn.ID,
			PaymentMethodCode: txn.PaymentMethodCode,
			CountryCode:       txn.CountryCode,
			Currency:          txn.Currency,
			Amount:            txn.Amount,
			AmountUSD:         txn.AmountUSD,
			Status:            txn.Status,
			MerchantID:        txn.MerchantID,
			CustomerID:        txn.CustomerID,
			TransactionDate:   txn.TransactionDate,
			CreatedAt:         txn.CreatedAt,
		}
	}

	c.JSON(http.StatusCreated, dto.BatchTransactionResponse{
		Inserted: len(txns),
		Results:  results,
	})
}
