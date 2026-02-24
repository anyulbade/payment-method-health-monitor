package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MarketGapRepository struct {
	pool *pgxpool.Pool
}

func NewMarketGapRepository(pool *pgxpool.Pool) *MarketGapRepository {
	return &MarketGapRepository{pool: pool}
}

type MarketGap struct {
	CountryCode       string
	PaymentMethodCode string
	MarketSharePct    float64
	IsEssential       bool
	Source            string
}

type CountryCoverage struct {
	CountryCode   string
	TotalCatalog  int
	ActiveMethods int
}

func (r *MarketGapRepository) GetGaps(ctx context.Context, country string, onlyEssential bool) ([]MarketGap, error) {
	query := `
		SELECT cpc.country_code, cpc.payment_method_code,
			COALESCE(cpc.market_share_pct, 0), cpc.is_essential, COALESCE(cpc.source, '')
		FROM country_payment_catalog cpc
		WHERE NOT EXISTS (
			SELECT 1 FROM transactions t
			WHERE t.payment_method_code = cpc.payment_method_code
				AND t.country_code = cpc.country_code
				AND t.transaction_date >= NOW() - INTERVAL '90 days'
		)
		AND ($1 = '' OR cpc.country_code = $1)
		AND ($2 = false OR cpc.is_essential = true)
		ORDER BY cpc.country_code, COALESCE(cpc.market_share_pct, 0) DESC
	`
	rows, err := r.pool.Query(ctx, query, country, onlyEssential)
	if err != nil {
		return nil, fmt.Errorf("query gaps: %w", err)
	}
	defer rows.Close()

	var results []MarketGap
	for rows.Next() {
		var g MarketGap
		if err := rows.Scan(&g.CountryCode, &g.PaymentMethodCode, &g.MarketSharePct, &g.IsEssential, &g.Source); err != nil {
			return nil, fmt.Errorf("scan gap: %w", err)
		}
		results = append(results, g)
	}
	return results, nil
}

func (r *MarketGapRepository) GetCoverage(ctx context.Context, country string) ([]CountryCoverage, error) {
	query := `
		SELECT cpc.country_code,
			COUNT(DISTINCT cpc.payment_method_code) as total_catalog,
			COUNT(DISTINCT t.payment_method_code) as active_methods
		FROM country_payment_catalog cpc
		LEFT JOIN transactions t ON t.payment_method_code = cpc.payment_method_code
			AND t.country_code = cpc.country_code
			AND t.transaction_date >= NOW() - INTERVAL '90 days'
		WHERE ($1 = '' OR cpc.country_code = $1)
		GROUP BY cpc.country_code
	`
	rows, err := r.pool.Query(ctx, query, country)
	if err != nil {
		return nil, fmt.Errorf("query coverage: %w", err)
	}
	defer rows.Close()

	var results []CountryCoverage
	for rows.Next() {
		var c CountryCoverage
		if err := rows.Scan(&c.CountryCode, &c.TotalCatalog, &c.ActiveMethods); err != nil {
			return nil, fmt.Errorf("scan coverage: %w", err)
		}
		results = append(results, c)
	}
	return results, nil
}
