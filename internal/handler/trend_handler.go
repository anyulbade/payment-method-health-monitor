package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/anyulbade/payment-method-health-monitor/internal/dto"
	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

type TrendHandler struct {
	svc *service.TrendService
}

func NewTrendHandler(svc *service.TrendService) *TrendHandler {
	return &TrendHandler{svc: svc}
}

func (h *TrendHandler) GetTrends(c *gin.Context) {
	country := c.Query("country")
	paymentMethod := c.Query("payment_method")
	period := c.DefaultQuery("period", "MOM")
	metric := c.DefaultQuery("metric", "tpv_usd")
	periodsBack, _ := strconv.Atoi(c.DefaultQuery("periods_back", "6"))
	p := dto.ParsePagination(c)

	if period != "WOW" && period != "MOM" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period must be WOW or MOM"})
		return
	}

	validMetrics := map[string]bool{
		"tpv_usd": true, "transaction_count": true,
		"approval_rate": true, "avg_transaction_value": true,
	}
	if !validMetrics[metric] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric, use: tpv_usd, transaction_count, approval_rate, avg_transaction_value"})
		return
	}

	results, err := h.svc.GetTrends(c.Request.Context(), country, paymentMethod, period, metric, periodsBack)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute trends: " + err.Error()})
		return
	}

	totalItems := len(results)
	start := p.Offset
	end := start + p.PageSize
	if start > totalItems {
		start = totalItems
	}
	if end > totalItems {
		end = totalItems
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       results[start:end],
		"pagination": dto.NewPagination(p.Page, p.PageSize, totalItems),
	})
}
