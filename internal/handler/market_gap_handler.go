package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/anyulbade/payment-method-health-monitor/internal/dto"
	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

type MarketGapHandler struct {
	svc *service.MarketGapService
}

func NewMarketGapHandler(svc *service.MarketGapService) *MarketGapHandler {
	return &MarketGapHandler{svc: svc}
}

func (h *MarketGapHandler) GetMarketGaps(c *gin.Context) {
	country := c.Query("country")
	onlyEssential := c.Query("only_essential") == "true"
	p := dto.ParsePagination(c)

	gaps, coverage, err := h.svc.GetMarketGaps(c.Request.Context(), country, onlyEssential)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to detect market gaps: " + err.Error()})
		return
	}

	totalItems := len(gaps)
	start := p.Offset
	end := start + p.PageSize
	if start > totalItems {
		start = totalItems
	}
	if end > totalItems {
		end = totalItems
	}

	c.JSON(http.StatusOK, gin.H{
		"gaps":       gaps[start:end],
		"coverage":   coverage,
		"pagination": dto.NewPagination(p.Page, p.PageSize, totalItems),
	})
}
