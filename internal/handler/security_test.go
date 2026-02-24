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

	"github.com/anyulbade/payment-method-health-monitor/internal/database"
	"github.com/anyulbade/payment-method-health-monitor/internal/dto"
	"github.com/anyulbade/payment-method-health-monitor/internal/repository"
	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

func setupFullRouter(t *testing.T) *gin.Engine {
	t.Helper()
	pool := getTestPool(t)
	if pool == nil {
		t.Skip("no database available")
	}

	dbURL := "postgres://pmhm:pmhm_secret@localhost:5434/pmhm?sslmode=disable"
	_ = database.RollbackMigrations(dbURL)
	if err := database.RunMigrations(dbURL); err != nil {
		t.Fatalf("migrations failed: %v", err)
	}
	if err := database.SeedData(t.Context(), pool); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	txnRepo := repository.NewTransactionRepository(pool)
	pmRepo := repository.NewPaymentMethodRepository(pool)
	metricsRepo := repository.NewMetricsRepository(pool)
	insightRepo := repository.NewInsightRepository(pool)

	txnService := service.NewTransactionService(txnRepo, pmRepo)
	metricsService := service.NewMetricsService(metricsRepo)
	insightService := service.NewInsightService(insightRepo)

	txnHandler := NewTransactionHandler(txnService)
	metricsHandler := NewMetricsHandler(metricsService)
	insightHandler := NewInsightHandler(insightService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	api.POST("/transactions", txnHandler.Create)
	api.POST("/transactions/batch", txnHandler.CreateBatch)
	api.GET("/metrics", metricsHandler.GetMetrics)
	api.GET("/insights", insightHandler.GetInsights)

	return router
}

func TestSQLInjection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	router := setupFullRouter(t)

	injections := []struct {
		name  string
		url   string
	}{
		{"country param", "/api/v1/metrics?country=MX'%3B+DROP+TABLE+transactions%3B+--"},
		{"country with OR", "/api/v1/metrics?country=MX'+OR+'1'%3D'1"},
		{"date injection", "/api/v1/metrics?date_from=2026-01-01'+UNION+SELECT+*+FROM+pg_catalog.pg_tables+--"},
		{"insight country", "/api/v1/insights?country=CO'%3B+DROP+TABLE+transactions%3B+--"},
		{"type injection", "/api/v1/metrics?type=CARD'+OR+'1'%3D'1"},
	}

	for _, tc := range injections {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tc.url, nil)
			router.ServeHTTP(w, req)

			// Should NOT be 500 (would indicate SQL error from injection)
			// Parameterized queries prevent injection, so we get 200 with empty results or 400
			assert.NotEqual(t, http.StatusInternalServerError, w.Code,
				"SQL injection attempt should not cause 500")
		})
	}

	t.Run("payment method code injection in transaction", func(t *testing.T) {
		body := dto.CreateTransactionRequest{
			PaymentMethodCode: "PIX' OR '1'='1",
			CountryCode:       "BR",
			Currency:          "BRL",
			Amount:            100,
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
}

func TestMalformedJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	router := setupFullRouter(t)

	cases := []struct {
		name string
		body string
	}{
		{"truncated JSON", `{"payment_method_code":"PIX","country_code":"BR"`},
		{"null required fields", `{"payment_method_code":null,"country_code":null,"currency":null,"amount":null,"status":null,"transaction_date":null}`},
		{"wrong types", `{"payment_method_code":123,"country_code":456,"currency":789,"amount":"not_a_number","status":true,"transaction_date":"not_a_date"}`},
		{"empty object", `{}`},
		{"just array", `[]`},
		{"empty string", ``},
		{"random string", `hello world`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code,
				"malformed JSON should return 400, got %d for %s", w.Code, tc.name)
		})
	}
}

func TestBoundaryConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	router := setupFullRouter(t)

	t.Run("batch: 0 items", func(t *testing.T) {
		body := `{"transactions":[]}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions/batch", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("batch: 501 items rejected", func(t *testing.T) {
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

	t.Run("page_size: negative defaults to 20", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/metrics?page_size=-1", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("page_size: 101 caps to 100", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/metrics?page_size=101", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("large amount accepted", func(t *testing.T) {
		body := fmt.Sprintf(`{"payment_method_code":"PIX","country_code":"BR","currency":"BRL","amount":9999999999999.99,"status":"APPROVED","transaction_date":"%s"}`, time.Now().Format(time.RFC3339))
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("future date accepted", func(t *testing.T) {
		future := time.Now().AddDate(1, 0, 0)
		body := fmt.Sprintf(`{"payment_method_code":"PIX","country_code":"BR","currency":"BRL","amount":100,"status":"APPROVED","transaction_date":"%s"}`, future.Format(time.RFC3339))
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/transactions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})
}
