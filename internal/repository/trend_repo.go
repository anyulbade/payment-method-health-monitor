package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TrendRepository struct {
	pool *pgxpool.Pool
}

func NewTrendRepository(pool *pgxpool.Pool) *TrendRepository {
	return &TrendRepository{pool: pool}
}

type TrendBucket struct {
	Period              string
	PaymentMethodCode   string
	PaymentMethodName   string
	CountryCode         string
	TransactionCount    int
	TpvUSD              float64
	ApprovalRate        float64
	AvgTransactionValue float64
}

func (r *TrendRepository) GetTrends(ctx context.Context, country, paymentMethod, period string, periodsBack int) ([]TrendBucket, error) {
	truncFunc := "month"
	if period == "WOW" {
		truncFunc = "week"
	}

	intervalStr := fmt.Sprintf("%d %ss", periodsBack, truncFunc)

	query := fmt.Sprintf(`
		SELECT
			DATE_TRUNC('%s', t.transaction_date)::text AS period,
			t.payment_method_code,
			pm.name,
			t.country_code,
			COUNT(*) AS txn_count,
			COALESCE(SUM(t.amount_usd) FILTER (WHERE t.status = 'APPROVED'), 0) AS tpv_usd,
			CASE WHEN COUNT(*) > 0
				THEN ROUND(COUNT(*) FILTER (WHERE t.status = 'APPROVED')::numeric / COUNT(*)::numeric * 100, 2)
				ELSE 0
			END AS approval_rate,
			CASE WHEN COUNT(*) > 0
				THEN ROUND(AVG(t.amount_usd)::numeric, 2)
				ELSE 0
			END AS avg_txn_value
		FROM transactions t
		JOIN payment_methods pm ON pm.code = t.payment_method_code
		WHERE t.transaction_date >= DATE_TRUNC('%s', NOW()) - $1::interval
			AND ($2 = '' OR t.country_code = $2)
			AND ($3 = '' OR t.payment_method_code = $3)
		GROUP BY DATE_TRUNC('%s', t.transaction_date), t.payment_method_code, pm.name, t.country_code
		ORDER BY period ASC, t.payment_method_code, t.country_code
	`, truncFunc, truncFunc, truncFunc)

	rows, err := r.pool.Query(ctx, query, intervalStr, country, paymentMethod)
	if err != nil {
		return nil, fmt.Errorf("query trends: %w", err)
	}
	defer rows.Close()

	var results []TrendBucket
	for rows.Next() {
		var b TrendBucket
		if err := rows.Scan(&b.Period, &b.PaymentMethodCode, &b.PaymentMethodName,
			&b.CountryCode, &b.TransactionCount, &b.TpvUSD, &b.ApprovalRate, &b.AvgTransactionValue); err != nil {
			return nil, fmt.Errorf("scan trend: %w", err)
		}
		results = append(results, b)
	}
	return results, nil
}
