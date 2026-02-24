package service

import (
	"context"
	"math"

	"github.com/anyulbade/payment-method-health-monitor/internal/repository"
)

type ROIService struct {
	repo *repository.ROIRepository
}

func NewROIService(repo *repository.ROIRepository) *ROIService {
	return &ROIService{repo: repo}
}

type ROIResult struct {
	PaymentMethodCode    string   `json:"payment_method_code"`
	PaymentMethodName    string   `json:"payment_method_name"`
	CountryCode          string   `json:"country_code"`
	ApprovedTPVUSD       float64  `json:"approved_tpv_usd"`
	TotalCostUSD         float64  `json:"total_cost_usd"`
	ROIPct               *float64 `json:"roi_pct"`
	CostPerApprovedTxn   float64  `json:"cost_per_approved_txn"`
	RevenuePerCostDollar float64  `json:"revenue_per_cost_dollar"`
	BreakEvenTxnCount    int      `json:"break_even_txn_count"`
	Recommendation       string   `json:"recommendation"`
}

func (s *ROIService) GetROI(ctx context.Context, country, dateFrom, dateTo string) ([]ROIResult, error) {
	rows, err := s.repo.GetROIData(ctx, country, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	var results []ROIResult
	for _, r := range rows {
		totalCost := (r.MonthsInRange * r.MonthlyFixedCost) +
			(float64(r.TotalTxnCount) * r.PerTransactionCost) +
			(r.ApprovedTPV * r.PercentageFee)
		totalCost = math.Round(totalCost*100) / 100

		var roiPct *float64
		if totalCost > 0 {
			v := math.Round((r.ApprovedTPV-totalCost)/totalCost*10000) / 100
			roiPct = &v
		}

		costPerApproved := 0.0
		if r.ApprovedCount > 0 {
			costPerApproved = math.Round(totalCost/float64(r.ApprovedCount)*100) / 100
		}

		revPerCost := 0.0
		if totalCost > 0 {
			revPerCost = math.Round(r.ApprovedTPV/totalCost*100) / 100
		}

		breakEven := 0
		if r.PerTransactionCost > 0 || r.PercentageFee > 0 {
			// Simplified break-even: fixed costs / (avg revenue per txn - variable cost per txn)
			if r.ApprovedCount > 0 {
				avgRevPerTxn := r.ApprovedTPV / float64(r.ApprovedCount)
				varCostPerTxn := r.PerTransactionCost + (avgRevPerTxn * r.PercentageFee)
				margin := avgRevPerTxn - varCostPerTxn
				if margin > 0 {
					breakEven = int(math.Ceil(r.MonthsInRange * r.MonthlyFixedCost / margin))
				}
			}
		}

		rec := "UNPROFITABLE"
		if roiPct != nil {
			switch {
			case *roiPct > 500:
				rec = "HIGHLY_PROFITABLE"
			case *roiPct > 100:
				rec = "PROFITABLE"
			case *roiPct > 0:
				rec = "MARGINAL"
			}
		}

		results = append(results, ROIResult{
			PaymentMethodCode:    r.PaymentMethodCode,
			PaymentMethodName:    r.PaymentMethodName,
			CountryCode:          r.CountryCode,
			ApprovedTPVUSD:       r.ApprovedTPV,
			TotalCostUSD:         totalCost,
			ROIPct:               roiPct,
			CostPerApprovedTxn:   costPerApproved,
			RevenuePerCostDollar: revPerCost,
			BreakEvenTxnCount:    breakEven,
			Recommendation:       rec,
		})
	}

	return results, nil
}
