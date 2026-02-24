package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/anyulbade/payment-method-health-monitor/internal/repository"
)

type InsightService struct {
	repo *repository.InsightRepository
}

func NewInsightService(repo *repository.InsightRepository) *InsightService {
	return &InsightService{repo: repo}
}

type Insight struct {
	InsightID         string                 `json:"insight_id"`
	Type              string                 `json:"type"`
	Severity          string                 `json:"severity"`
	PaymentMethodCode string                 `json:"payment_method_code"`
	PaymentMethodName string                 `json:"payment_method_name"`
	CountryCode       string                 `json:"country_code"`
	TriggeringMetric  string                 `json:"triggering_metric"`
	MetricValue       float64                `json:"metric_value"`
	Threshold         float64                `json:"threshold"`
	Description       string                 `json:"description"`
	RecommendedAction string                 `json:"recommended_action"`
	SupportingData    map[string]interface{} `json:"supporting_data"`
	GeneratedAt       time.Time              `json:"generated_at"`
}

func (s *InsightService) DetectInsights(ctx context.Context, country, insightType, severity string) ([]Insight, error) {
	g, gctx := errgroup.WithContext(ctx)

	var zombies, gems, alerts []Insight

	if insightType == "" || insightType == "zombie" {
		g.Go(func() error {
			var err error
			zombies, err = s.detectZombies(gctx, country)
			return err
		})
	}

	if insightType == "" || insightType == "hidden_gem" {
		g.Go(func() error {
			var err error
			gems, err = s.detectHiddenGems(gctx, country)
			return err
		})
	}

	if insightType == "" || insightType == "performance_alert" {
		g.Go(func() error {
			var err error
			alerts, err = s.detectPerformanceAlerts(gctx, country)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var all []Insight
	all = append(all, zombies...)
	all = append(all, gems...)
	all = append(all, alerts...)

	if severity != "" {
		var filtered []Insight
		for _, i := range all {
			if i.Severity == severity {
				filtered = append(filtered, i)
			}
		}
		all = filtered
	}

	return all, nil
}

func (s *InsightService) detectZombies(ctx context.Context, country string) ([]Insight, error) {
	candidates, err := s.repo.GetZombieCandidates(ctx, country)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var insights []Insight

	for _, c := range candidates {
		var isZombie bool
		var sev string
		var threshold float64

		if c.MonthsActive < 3 {
			// New method: absolute threshold
			threshold = 5
			isZombie = c.TxnCount90d < 5
		} else {
			// Established: relative to baseline
			baseline := math.Max(10, c.HistoricalMonthlyAvg*3*0.1)
			threshold = baseline
			isZombie = float64(c.TxnCount90d) < baseline
		}

		if !isZombie {
			continue
		}

		if c.TxnCount90d == 0 || (c.HistoricalMonthlyAvg > 0 && float64(c.TxnCount90d)/(c.HistoricalMonthlyAvg*3) < 0.1) {
			sev = "HIGH"
		} else if c.HistoricalMonthlyAvg > 0 && float64(c.TxnCount90d)/(c.HistoricalMonthlyAvg*3) < 0.3 {
			sev = "MEDIUM"
		} else {
			sev = "LOW"
		}

		insights = append(insights, Insight{
			InsightID:         hashID("zombie", c.PaymentMethodCode, c.CountryCode),
			Type:              "zombie",
			Severity:          sev,
			PaymentMethodCode: c.PaymentMethodCode,
			PaymentMethodName: c.PaymentMethodName,
			CountryCode:       c.CountryCode,
			TriggeringMetric:  "txn_count_90d",
			MetricValue:       float64(c.TxnCount90d),
			Threshold:         threshold,
			Description:       fmt.Sprintf("%s in %s has only %d transactions in 90 days with active integration costing $%.2f/month", c.PaymentMethodName, c.CountryCode, c.TxnCount90d, c.MonthlyCostUSD),
			RecommendedAction: "Review integration cost vs. value. Consider deactivating or renegotiating terms.",
			SupportingData: map[string]interface{}{
				"monthly_cost_usd":       c.MonthlyCostUSD,
				"historical_monthly_avg": c.HistoricalMonthlyAvg,
				"months_active":          c.MonthsActive,
				"payment_method_type":    c.PaymentMethodType,
			},
			GeneratedAt: now,
		})
	}

	return insights, nil
}

func (s *InsightService) detectHiddenGems(ctx context.Context, country string) ([]Insight, error) {
	candidates, err := s.repo.GetHiddenGemCandidates(ctx, country)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var insights []Insight

	for _, c := range candidates {
		if c.ApprovalRate < 90 || c.RevenueContribution < 2 {
			continue
		}
		if c.VolumeShare >= c.RevenueContribution*0.75 {
			continue
		}

		var sev string
		if c.RevenueContribution > 8 {
			sev = "HIGH"
		} else if c.RevenueContribution > 4 {
			sev = "MEDIUM"
		} else {
			sev = "LOW"
		}

		insights = append(insights, Insight{
			InsightID:         hashID("hidden_gem", c.PaymentMethodCode, c.CountryCode),
			Type:              "hidden_gem",
			Severity:          sev,
			PaymentMethodCode: c.PaymentMethodCode,
			PaymentMethodName: c.PaymentMethodName,
			CountryCode:       c.CountryCode,
			TriggeringMetric:  "revenue_contribution_pct",
			MetricValue:       c.RevenueContribution,
			Threshold:         2.0,
			Description:       fmt.Sprintf("%s in %s has %.1f%% approval rate and %.1f%% revenue contribution but only %.1f%% volume share", c.PaymentMethodName, c.CountryCode, c.ApprovalRate, c.RevenueContribution, c.VolumeShare),
			RecommendedAction: "Increase merchant adoption and volume for this high-performing method.",
			SupportingData: map[string]interface{}{
				"approval_rate":        c.ApprovalRate,
				"volume_share_pct":     c.VolumeShare,
				"tpv_usd":             c.TpvUSD,
				"transaction_count":    c.TransactionCount,
			},
			GeneratedAt: now,
		})
	}

	return insights, nil
}

func (s *InsightService) detectPerformanceAlerts(ctx context.Context, country string) ([]Insight, error) {
	candidates, err := s.repo.GetPerformanceAlertCandidates(ctx, country)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var insights []Insight

	for _, c := range candidates {
		if c.TransactionCount < 20 {
			continue
		}

		gap := c.CountryTypeAvgApproval - c.ApprovalRate
		if gap <= 10 {
			continue
		}

		var sev string
		if gap > 15 {
			sev = "HIGH"
		} else {
			sev = "MEDIUM"
		}

		insights = append(insights, Insight{
			InsightID:         hashID("performance_alert", c.PaymentMethodCode, c.CountryCode),
			Type:              "performance_alert",
			Severity:          sev,
			PaymentMethodCode: c.PaymentMethodCode,
			PaymentMethodName: c.PaymentMethodName,
			CountryCode:       c.CountryCode,
			TriggeringMetric:  "approval_rate",
			MetricValue:       c.ApprovalRate,
			Threshold:         c.CountryTypeAvgApproval - 10,
			Description:       fmt.Sprintf("%s in %s has %.1f%% approval rate, %.1fpp below %s average of %.1f%%", c.PaymentMethodName, c.CountryCode, c.ApprovalRate, gap, c.PaymentMethodType, c.CountryTypeAvgApproval),
			RecommendedAction: "Investigate decline reasons. Check provider configuration and fraud rules.",
			SupportingData: map[string]interface{}{
				"country_type_avg_approval": c.CountryTypeAvgApproval,
				"gap_pp":                    gap,
				"payment_method_type":       c.PaymentMethodType,
				"transaction_count":         c.TransactionCount,
			},
			GeneratedAt: now,
		})
	}

	return insights, nil
}

func hashID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
