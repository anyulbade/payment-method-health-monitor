package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/anyulbade/payment-method-health-monitor/internal/dto"
	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

type MetricsHandler struct {
	svc *service.MetricsService
}

func NewMetricsHandler(svc *service.MetricsService) *MetricsHandler {
	return &MetricsHandler{svc: svc}
}

func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	country := c.Query("country")
	pmType := c.Query("type")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	sortBy := c.DefaultQuery("sort_by", "tpv_usd")
	order := c.DefaultQuery("order", "desc")

	p := dto.ParsePagination(c)

	// Validate date formats
	if dateFrom != "" {
		if _, err := time.Parse(time.RFC3339, dateFrom); err != nil {
			if _, err2 := time.Parse("2006-01-02", dateFrom); err2 != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_from format"})
				return
			}
		}
	}
	if dateTo != "" {
		if _, err := time.Parse(time.RFC3339, dateTo); err != nil {
			if _, err2 := time.Parse("2006-01-02", dateTo); err2 != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_to format"})
				return
			}
		}
	}

	// Validate date range
	if dateFrom != "" && dateTo != "" && dateFrom > dateTo {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date_from must be before date_to"})
		return
	}

	results, summary, totalItems, err := h.svc.GetMetrics(
		c.Request.Context(), country, pmType, dateFrom, dateTo, sortBy, order,
		p.PageSize, p.Offset,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute metrics: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       results,
		"summary":    summary,
		"pagination": dto.NewPagination(p.Page, p.PageSize, totalItems),
	})
}
