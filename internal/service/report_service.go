package service

import (
	"bytes"
	"context"
	"html/template"
	"strings"
	"time"
)

type ReportService struct {
	metricsSvc *MetricsService
	insightSvc *InsightService
}

func NewReportService(metricsSvc *MetricsService, insightSvc *InsightService) *ReportService {
	return &ReportService{metricsSvc: metricsSvc, insightSvc: insightSvc}
}

type ReportData struct {
	GeneratedAt string
	Summary     MetricsSummary
	Metrics     []MetricResult
	Insights    []Insight
}

func (s *ReportService) GenerateReport(ctx context.Context, country, dateFrom, dateTo string) (*ReportData, error) {
	metrics, summary, _, err := s.metricsSvc.GetMetrics(ctx, country, "", dateFrom, dateTo, "tpv_usd", "desc", 100, 0)
	if err != nil {
		return nil, err
	}

	insights, err := s.insightSvc.DetectInsights(ctx, country, "", "")
	if err != nil {
		return nil, err
	}

	return &ReportData{
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05 MST"),
		Summary:     summary,
		Metrics:     metrics,
		Insights:    insights,
	}, nil
}

var ReportTemplate string // Set from main via embed

func (s *ReportService) RenderHTML(data *ReportData) (string, error) {
	funcMap := template.FuncMap{
		"toLower": strings.ToLower,
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(ReportTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
