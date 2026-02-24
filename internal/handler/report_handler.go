package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/anyulbade/payment-method-health-monitor/internal/service"
)

type ReportHandler struct {
	svc *service.ReportService
}

func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) GetReport(c *gin.Context) {
	country := c.Query("country")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	format := c.Query("format")

	data, err := h.svc.GenerateReport(c.Request.Context(), country, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report: " + err.Error()})
		return
	}

	wantsHTML := format == "html" || strings.Contains(c.GetHeader("Accept"), "text/html")

	if wantsHTML {
		html, err := h.svc.RenderHTML(data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to render HTML: " + err.Error()})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
		return
	}

	c.JSON(http.StatusOK, data)
}
