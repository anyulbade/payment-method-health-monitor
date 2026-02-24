package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/anyulbade/payment-method-health-monitor/internal/dto"
	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

type ROIHandler struct {
	svc *service.ROIService
}

func NewROIHandler(svc *service.ROIService) *ROIHandler {
	return &ROIHandler{svc: svc}
}

func (h *ROIHandler) GetROI(c *gin.Context) {
	country := c.Query("country")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	p := dto.ParsePagination(c)

	results, err := h.svc.GetROI(c.Request.Context(), country, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute ROI: " + err.Error()})
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
