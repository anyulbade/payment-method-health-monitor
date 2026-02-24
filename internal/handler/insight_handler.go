package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/anyulbade/payment-method-health-monitor/internal/dto"
	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

type InsightHandler struct {
	svc *service.InsightService
}

func NewInsightHandler(svc *service.InsightService) *InsightHandler {
	return &InsightHandler{svc: svc}
}

func (h *InsightHandler) GetInsights(c *gin.Context) {
	country := c.Query("country")
	insightType := c.Query("insight_type")
	severity := c.Query("severity")
	p := dto.ParsePagination(c)

	insights, err := h.svc.DetectInsights(c.Request.Context(), country, insightType, severity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to detect insights: " + err.Error()})
		return
	}

	totalItems := len(insights)
	start := p.Offset
	end := start + p.PageSize
	if start > totalItems {
		start = totalItems
	}
	if end > totalItems {
		end = totalItems
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       insights[start:end],
		"pagination": dto.NewPagination(p.Page, p.PageSize, totalItems),
	})
}
