package service

import (
	"context"
	"math"

	"github.com/anyulbade/payment-method-health-monitor/internal/repository"
)

type TrendService struct {
	repo *repository.TrendRepository
}

func NewTrendService(repo *repository.TrendRepository) *TrendService {
	return &TrendService{repo: repo}
}

type TrendPoint struct {
	Period           string  `json:"period"`
	Value            float64 `json:"value"`
	PreviousValue    float64 `json:"previous_value,omitempty"`
	AbsoluteChange   float64 `json:"absolute_change"`
	PercentageChange float64 `json:"percentage_change"`
	Direction        string  `json:"direction"`
}

type TrendSummary struct {
	PaymentMethodCode string       `json:"payment_method_code"`
	PaymentMethodName string       `json:"payment_method_name"`
	CountryCode       string       `json:"country_code"`
	Metric            string       `json:"metric"`
	Points            []TrendPoint `json:"points"`
	OverallTrend      string       `json:"overall_trend"`
	Slope             float64      `json:"slope"`
	RSquared          float64      `json:"r_squared"`
}

func (s *TrendService) GetTrends(ctx context.Context, country, paymentMethod, period, metric string, periodsBack int) ([]TrendSummary, error) {
	if periodsBack < 1 {
		periodsBack = 6
	}

	buckets, err := s.repo.GetTrends(ctx, country, paymentMethod, period, periodsBack)
	if err != nil {
		return nil, err
	}

	// Group by (payment_method, country)
	type key struct{ pm, cc string }
	grouped := make(map[key][]repository.TrendBucket)
	nameMap := make(map[key]string)
	for _, b := range buckets {
		k := key{b.PaymentMethodCode, b.CountryCode}
		grouped[k] = append(grouped[k], b)
		nameMap[k] = b.PaymentMethodName
	}

	var results []TrendSummary
	for k, points := range grouped {
		values := extractMetricValues(points, metric)

		trendPoints := make([]TrendPoint, len(values))
		for i, v := range values {
			tp := TrendPoint{
				Period: points[i].Period,
				Value:  v,
			}
			if i > 0 {
				tp.PreviousValue = values[i-1]
				tp.AbsoluteChange = v - values[i-1]
				if values[i-1] != 0 {
					tp.PercentageChange = math.Round(tp.AbsoluteChange/values[i-1]*10000) / 100
				}
				if math.Abs(tp.PercentageChange) < 1 {
					tp.Direction = "FLAT"
				} else if tp.AbsoluteChange > 0 {
					tp.Direction = "UP"
				} else {
					tp.Direction = "DOWN"
				}
			}
			trendPoints[i] = tp
		}

		slope, r2 := linearRegression(values)
		overallTrend := "VOLATILE"
		if len(values) >= 2 {
			if r2 >= 0.5 {
				if slope > 0 {
					overallTrend = "GROWING"
				} else {
					overallTrend = "DECLINING"
				}
			}
		}

		results = append(results, TrendSummary{
			PaymentMethodCode: k.pm,
			PaymentMethodName: nameMap[k],
			CountryCode:       k.cc,
			Metric:            metric,
			Points:            trendPoints,
			OverallTrend:      overallTrend,
			Slope:             math.Round(slope*100) / 100,
			RSquared:          math.Round(r2*10000) / 10000,
		})
	}

	return results, nil
}

func extractMetricValues(buckets []repository.TrendBucket, metric string) []float64 {
	values := make([]float64, len(buckets))
	for i, b := range buckets {
		switch metric {
		case "tpv_usd":
			values[i] = b.TpvUSD
		case "transaction_count":
			values[i] = float64(b.TransactionCount)
		case "approval_rate":
			values[i] = b.ApprovalRate
		case "avg_transaction_value":
			values[i] = b.AvgTransactionValue
		default:
			values[i] = b.TpvUSD
		}
	}
	return values
}

func linearRegression(values []float64) (slope, rSquared float64) {
	n := float64(len(values))
	if n < 2 {
		return 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i, v := range values {
		x := float64(i)
		sumX += x
		sumY += v
		sumXY += x * v
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0, 0
	}

	slope = (n*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / n

	// R²
	meanY := sumY / n
	var ssRes, ssTot float64
	for i, v := range values {
		predicted := slope*float64(i) + intercept
		ssRes += (v - predicted) * (v - predicted)
		ssTot += (v - meanY) * (v - meanY)
	}

	if ssTot == 0 {
		return slope, 1.0
	}
	rSquared = 1 - ssRes/ssTot
	return slope, rSquared
}
