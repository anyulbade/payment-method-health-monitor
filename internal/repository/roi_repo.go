package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ROIRepository struct {
	pool *pgxpool.Pool
}

func NewROIRepository(pool *pgxpool.Pool) *ROIRepository {
	return &ROIRepository{pool: pool}
}

type ROIRow struct {
	PaymentMethodCode     string
	PaymentMethodName     string
	CountryCode           string
	ApprovedTPV           float64
	ApprovedCount         int
	TotalTxnCount         int
	MonthlyFixedCost      float64
	PerTransactionCost    float64
	PercentageFee         float64
	MonthsInRange         float64
}

func (r *ROIRepository) GetROIData(ctx context.Context, country, dateFrom, dateTo string) ([]ROIRow, error) {
	query := `
		WITH txn_agg AS (
			SELECT t.payment_method_code, t.country_code,
				COALESCE(SUM(t.amount_usd) FILTER (WHERE t.status = 'APPROVED'), 0) AS approved_tpv,
				COUNT(*) FILTER (WHERE t.status = 'APPROVED') AS approved_count,
				COUNT(*) AS total_count,
				GREATEST(
					EXTRACT(EPOCH FROM (
						COALESCE(NULLIF($3, '')::timestamptz, MAX(t.transaction_date)) -
						COALESCE(NULLIF($2, '')::timestamptz, MIN(t.transaction_date))
					)) / (30*86400),
					1
				) AS months_in_range
			FROM transactions t
			WHERE ($1 = '' OR t.country_code = $1)
				AND ($2 = '' OR t.transaction_date >= $2::timestamptz)
				AND ($3 = '' OR t.transaction_date <= $3::timestamptz)
			GROUP BY t.payment_method_code, t.country_code
		)
		SELECT a.payment_method_code, pm.name, a.country_code,
			a.approved_tpv, a.approved_count, a.total_count,
			COALESCE(ic.monthly_fixed_cost_usd, 0),
			COALESCE(ic.per_transaction_cost_usd, 0),
			COALESCE(ic.percentage_fee, 0),
			a.months_in_range
		FROM txn_agg a
		JOIN payment_methods pm ON pm.code = a.payment_method_code
		LEFT JOIN integration_costs ic ON ic.payment_method_code = a.payment_method_code
			AND ic.country_code = a.country_code AND ic.effective_to IS NULL
	`
	rows, err := r.pool.Query(ctx, query, country, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("query ROI: %w", err)
	}
	defer rows.Close()

	var results []ROIRow
	for rows.Next() {
		var r ROIRow
		if err := rows.Scan(&r.PaymentMethodCode, &r.PaymentMethodName, &r.CountryCode,
			&r.ApprovedTPV, &r.ApprovedCount, &r.TotalTxnCount,
			&r.MonthlyFixedCost, &r.PerTransactionCost, &r.PercentageFee, &r.MonthsInRange); err != nil {
			return nil, fmt.Errorf("scan ROI: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}
