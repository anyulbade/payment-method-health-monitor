package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anyulbade/payment-method-health-monitor/internal/database"
	"github.com/anyulbade/payment-method-health-monitor/internal/dto"
	"github.com/anyulbade/payment-method-health-monitor/internal/repository"
	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

func setupTransactionRouter(t *testing.T) *gin.Engine {
	t.Helper()
	pool := getTestPool(t)
	if pool == nil {
		t.Skip("no database available")
	}

	dbURL := "postgres://pmhm:pmhm_secret@localhost:5434/pmhm?sslmode=disable"
	_ = database.RollbackMigrations(dbURL)
	require.NoError(t, database.RunMigrations(dbURL))
	require.NoError(t, database.SeedData(t.Context(), pool))

	txnRepo := repository.NewTransactionRepository(pool)
	pmRepo := repository.NewPaymentMethodRepository(pool)
	txnService := service.NewTransactionService(txnRepo, pmRepo)
	txnHandler := NewTransactionHandler(txnService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/transactions", txnHandler.Create)
	api.POST("/transactions/batch", txnHandler.CreateBatch)

	return router
}

func TestTransactionHandler_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	router := setupTransactionRouter(t)

	t.Run("happy: single insert", func(t *testing.T) {
		body := dto.CreateTransactionRequest{
			PaymentMethodCode: "PIX",
			CountryCode:       "BR",
			Currency:          "BRL",
			Amount:            250.50,
			Status:            "APPROVED",
			MerchantID:        "merchant_001",
			CustomerID:        "customer_001",
			TransactionDate:   time.Now(),
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp dto.TransactionResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, "PIX", resp.PaymentMethodCode)
		assert.Greater(t, resp.AmountUSD, 0.0, "should have USD conversion")
	})

	t.Run("happy: USD conversion correctness", func(t *testing.T) {
		body := dto.CreateTransactionRequest{
			PaymentMethodCode: "PIX",
			CountryCode:       "BR",
			Currency:          "BRL",
			Amount:            1000.00,
			Status:            "APPROVED",
			TransactionDate:   time.Now(),
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp dto.TransactionResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		// BRL fx_rate = 0.1960, so 1000 BRL = 196.00 USD
		assert.Equal(t, 196.00, resp.AmountUSD)
	})

	t.Run("bad: missing required fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad: invalid payment method", func(t *testing.T) {
		body := dto.CreateTransactionRequest{
			PaymentMethodCode: "NONEXISTENT",
			CountryCode:       "BR",
			Currency:          "BRL",
			Amount:            100.00,
			Status:            "APPROVED",
			TransactionDate:   time.Now(),
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad: invalid country", func(t *testing.T) {
		body := dto.CreateTransactionRequest{
			PaymentMethodCode: "PIX",
			CountryCode:       "XX",
			Currency:          "XXX",
			Amount:            100.00,
			Status:            "APPROVED",
			TransactionDate:   time.Now(),
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad: pm not available in country", func(t *testing.T) {
		body := dto.CreateTransactionRequest{
			PaymentMethodCode: "PIX",
			CountryCode:       "MX",
			Currency:          "MXN",
			Amount:            100.00,
			Status:            "APPROVED",
			TransactionDate:   time.Now(),
		}
		jsonBody, _ := json.Marshal(body)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad: invalid status", func(t *testing.T) {
		body := `{"payment_method_code":"PIX","country_code":"BR","currency":"BRL","amount":100,"status":"INVALID","transaction_date":"2026-01-01T00:00:00Z"}`

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad: negative amount", func(t *testing.T) {
		body := `{"payment_method_code":"PIX","country_code":"BR","currency":"BRL","amount":-10,"status":"APPROVED","transaction_date":"2026-01-01T00:00:00Z"}`

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTransactionHandler_Batch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	router := setupTransactionRouter(t)

	t.Run("happy: batch insert", func(t *testing.T) {
		txns := []dto.CreateTransactionRequest{
			{PaymentMethodCode: "PIX", CountryCode: "BR", Currency: "BRL", Amount: 100, Status: "APPROVED", TransactionDate: time.Now()},
			{PaymentMethodCode: "PIX", CountryCode: "BR", Currency: "BRL", Amount: 200, Status: "DECLINED", TransactionDate: time.Now()},
		}
		body, _ := json.Marshal(dto.BatchTransactionRequest{Transactions: txns})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp dto.BatchTransactionResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 2, resp.Inserted)
	})

	t.Run("bad: empty batch", func(t *testing.T) {
		body := `{"transactions":[]}`

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions/batch", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad: batch exceeds 500", func(t *testing.T) {
		txns := make([]dto.CreateTransactionRequest, 501)
		for i := range txns {
			txns[i] = dto.CreateTransactionRequest{
				PaymentMethodCode: "PIX", CountryCode: "BR", Currency: "BRL",
				Amount: 100, Status: "APPROVED", TransactionDate: time.Now(),
			}
		}
		body, _ := json.Marshal(dto.BatchTransactionRequest{Transactions: txns})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad: batch with invalid item rejects all", func(t *testing.T) {
		txns := []dto.CreateTransactionRequest{
			{PaymentMethodCode: "PIX", CountryCode: "BR", Currency: "BRL", Amount: 100, Status: "APPROVED", TransactionDate: time.Now()},
			{PaymentMethodCode: "NONEXISTENT", CountryCode: "BR", Currency: "BRL", Amount: 100, Status: "APPROVED", TransactionDate: time.Now()},
		}
		body, _ := json.Marshal(dto.BatchTransactionRequest{Transactions: txns})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp dto.ErrorListResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp.Errors)
	})

	t.Run("edge: batch of exactly 500", func(t *testing.T) {
		txns := make([]dto.CreateTransactionRequest, 500)
		for i := range txns {
			txns[i] = dto.CreateTransactionRequest{
				PaymentMethodCode: "PIX", CountryCode: "BR", Currency: "BRL",
				Amount: float64(i+1) * 0.5, Status: "APPROVED",
				TransactionDate: time.Now().Add(time.Duration(i) * time.Minute),
				MerchantID:      fmt.Sprintf("m_%d", i),
			}
		}
		body, _ := json.Marshal(dto.BatchTransactionRequest{Transactions: txns})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}
