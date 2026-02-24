package service

import (
	"context"
	"math"

	"github.com/anyulbade/payment-method-health-monitor/internal/repository"
)

type MarketGapService struct {
	repo *repository.MarketGapRepository
}

func NewMarketGapService(repo *repository.MarketGapRepository) *MarketGapService {
	return &MarketGapService{repo: repo}
}

type GapResult struct {
	CountryCode       string  `json:"country_code"`
	PaymentMethodCode string  `json:"payment_method_code"`
	MarketSharePct    float64 `json:"market_share_pct"`
	IsEssential       bool    `json:"is_essential"`
	Source            string  `json:"source"`
	OpportunityScore  float64 `json:"opportunity_score"`
}

type CoverageResult struct {
	CountryCode   string  `json:"country_code"`
	TotalCatalog  int     `json:"total_catalog_methods"`
	ActiveMethods int     `json:"active_methods"`
	CoveragePct   float64 `json:"coverage_pct"`
	GapCount      int     `json:"gap_count"`
}

func (s *MarketGapService) GetMarketGaps(ctx context.Context, country string, onlyEssential bool) ([]GapResult, []CoverageResult, error) {
	gaps, err := s.repo.GetGaps(ctx, country, onlyEssential)
	if err != nil {
		return nil, nil, err
	}

	coverage, err := s.repo.GetCoverage(ctx, country)
	if err != nil {
		return nil, nil, err
	}

	gapResults := make([]GapResult, len(gaps))
	for i, g := range gaps {
		score := g.MarketSharePct * 0.5
		if g.IsEssential {
			score += 30
		}
		estTPV := g.MarketSharePct * 1000 // rough estimate
		score += estTPV / 1000 * 0.2
		score = math.Min(score, 100)
		score = math.Round(score*100) / 100

		gapResults[i] = GapResult{
			CountryCode:       g.CountryCode,
			PaymentMethodCode: g.PaymentMethodCode,
			MarketSharePct:    g.MarketSharePct,
			IsEssential:       g.IsEssential,
			Source:            g.Source,
			OpportunityScore:  score,
		}
	}

	coverageResults := make([]CoverageResult, len(coverage))
	for i, c := range coverage {
		covPct := 0.0
		if c.TotalCatalog > 0 {
			covPct = math.Round(float64(c.ActiveMethods)/float64(c.TotalCatalog)*10000) / 100
		}
		coverageResults[i] = CoverageResult{
			CountryCode:   c.CountryCode,
			TotalCatalog:  c.TotalCatalog,
			ActiveMethods: c.ActiveMethods,
			CoveragePct:   covPct,
			GapCount:      c.TotalCatalog - c.ActiveMethods,
		}
	}

	return gapResults, coverageResults, nil
}
