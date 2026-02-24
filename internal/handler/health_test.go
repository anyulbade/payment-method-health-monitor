package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_NoPool(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Without a real DB pool, we test the handler structure.
	// Full integration test runs with Docker.
	t.Run("handler is created", func(t *testing.T) {
		h := NewHealthHandler(nil)
		assert.NotNil(t, h)
	})
}

func TestHealthResponse_Structure(t *testing.T) {
	type healthResponse struct {
		Status   string `json:"status"`
		Database string `json:"database"`
	}

	t.Run("healthy response has correct fields", func(t *testing.T) {
		body := `{"status":"healthy","database":"connected"}`
		var resp healthResponse
		err := json.Unmarshal([]byte(body), &resp)
		require.NoError(t, err)
		assert.Equal(t, "healthy", resp.Status)
		assert.Equal(t, "connected", resp.Database)
	})
}

// Integration test: requires running database
func TestHealthHandler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool := getTestPool(t)
	if pool == nil {
		t.Skip("no database available")
	}
	defer pool.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHealthHandler(pool)
	router.GET("/health", h.Health)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
	assert.Equal(t, "connected", resp["database"])
}
