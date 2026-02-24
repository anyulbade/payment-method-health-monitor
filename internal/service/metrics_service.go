package service

import (
	"context"

	"github.com/anyulbade/payment-method-health-monitor/internal/repository"
)

type MetricsService struct {
	repo *repository.MetricsRepository
}

func NewMetricsService(repo *repository.MetricsRepository) *MetricsService {
	return &MetricsService{repo: repo}
}

type MetricResult struct {
	PaymentMethodCode   string  `json:"payment_method_code"`
	PaymentMethodName   string  `json:"payment_method_name"`
	PaymentMethodType   string  `json:"payment_method_type"`
	CountryCode         string  `json:"country_code"`
	TransactionCount    int     `json:"transaction_count"`
	ApprovedCount       int     `json:"approved_count"`
	DeclinedCount       int     `json:"declined_count"`
	TpvUSD              float64 `json:"tpv_usd"`
	ApprovalRate        float64 `json:"approval_rate"`
	AvgTransactionValue float64 `json:"avg_transaction_value_usd"`
	RevenueContribution float64 `json:"revenue_contribution_pct"`
	MonthlyCostUSD      float64 `json:"monthly_cost_usd"`
	CostEfficiencyRatio float64 `json:"cost_efficiency_ratio"`
	ActivityStatus      string  `json:"activity_status"`
}

type MetricsSummary struct {
	TotalTransactions int     `json:"total_transactions"`
	TotalApproved     int     `json:"total_approved"`
	TotalTPVUSD       float64 `json:"total_tpv_usd"`
	OverallApproval   float64 `json:"overall_approval_rate"`
	ActiveMethods     int     `json:"active_methods"`
	LowActivityCount  int     `json:"low_activity_methods"`
	InactiveCount     int     `json:"inactive_methods"`
}

func (s *MetricsService) GetMetrics(ctx context.Context, country, pmType, dateFrom, dateTo, sortBy, order string, limit, offset int) ([]MetricResult, MetricsSummary, int, error) {
	rows, totalItems, err := s.repo.GetMetrics(ctx, country, pmType, dateFrom, dateTo, sortBy, order, limit, offset)
	if err != nil {
		return nil, MetricsSummary{}, 0, err
	}

	results := make([]MetricResult, len(rows))
	var summary MetricsSummary

	for i, row := range rows {
		results[i] = MetricResult{
			PaymentMethodCode:   row.PaymentMethodCode,
			PaymentMethodName:   row.PaymentMethodName,
			PaymentMethodType:   row.PaymentMethodType,
			CountryCode:         row.CountryCode,
			TransactionCount:    row.TransactionCount,
			ApprovedCount:       row.ApprovedCount,
			DeclinedCount:       row.DeclinedCount,
			TpvUSD:              row.TpvUSD,
			ApprovalRate:        row.ApprovalRate,
			AvgTransactionValue: row.AvgTransactionValue,
			RevenueContribution: row.RevenueContribution,
			MonthlyCostUSD:      row.MonthlyCostUSD,
			CostEfficiencyRatio: row.CostEfficiencyRatio,
			ActivityStatus:      row.ActivityStatus,
		}

		summary.TotalTransactions += row.TransactionCount
		summary.TotalApproved += row.ApprovedCount
		summary.TotalTPVUSD += row.TpvUSD

		switch row.ActivityStatus {
		case "ACTIVE":
			summary.ActiveMethods++
		case "LOW_ACTIVITY":
			summary.LowActivityCount++
		case "INACTIVE":
			summary.InactiveCount++
		}
	}

	if summary.TotalTransactions > 0 {
		summary.OverallApproval = float64(summary.TotalApproved) / float64(summary.TotalTransactions) * 100
		summary.OverallApproval = float64(int(summary.OverallApproval*100)) / 100
	}

	return results, summary, totalItems, nil
}
