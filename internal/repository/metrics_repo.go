package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MetricRow struct {
	PaymentMethodCode    string
	PaymentMethodName    string
	PaymentMethodType    string
	CountryCode          string
	TransactionCount     int
	ApprovedCount        int
	DeclinedCount        int
	TpvUSD               float64
	ApprovalRate         float64
	AvgTransactionValue  float64
	RevenueContribution  float64
	MonthlyCostUSD       float64
	CostEfficiencyRatio  float64
	ActivityStatus       string
}

type MetricsRepository struct {
	pool *pgxpool.Pool
}

func NewMetricsRepository(pool *pgxpool.Pool) *MetricsRepository {
	return &MetricsRepository{pool: pool}
}

func (r *MetricsRepository) GetMetrics(ctx context.Context, country, pmType, dateFrom, dateTo, sortBy, order string, limit, offset int) ([]MetricRow, int, error) {
	baseQuery := `
		WITH txn_agg AS (
			SELECT
				t.payment_method_code,
				t.country_code,
				COUNT(*) AS transaction_count,
				COUNT(*) FILTER (WHERE t.status = 'APPROVED') AS approved_count,
				COUNT(*) FILTER (WHERE t.status = 'DECLINED') AS declined_count,
				COALESCE(SUM(t.amount_usd) FILTER (WHERE t.status = 'APPROVED'), 0) AS tpv_usd,
				CASE WHEN COUNT(*) > 0
					THEN ROUND(COUNT(*) FILTER (WHERE t.status = 'APPROVED')::numeric / COUNT(*)::numeric * 100, 2)
					ELSE 0
				END AS approval_rate,
				CASE WHEN COUNT(*) > 0
					THEN ROUND(AVG(t.amount_usd)::numeric, 2)
					ELSE 0
				END AS avg_transaction_value
			FROM transactions t
			WHERE ($1 = '' OR t.country_code = $1)
				AND ($3 = '' OR t.transaction_date >= $3::timestamptz)
				AND ($4 = '' OR t.transaction_date <= $4::timestamptz)
			GROUP BY t.payment_method_code, t.country_code
		),
		total_tpv AS (
			SELECT COALESCE(SUM(tpv_usd), 0) AS total FROM txn_agg
		),
		txn_90d AS (
			SELECT
				t.payment_method_code,
				t.country_code,
				COUNT(*) AS txn_count_90d
			FROM transactions t
			WHERE t.transaction_date >= NOW() - INTERVAL '90 days'
				AND ($1 = '' OR t.country_code = $1)
			GROUP BY t.payment_method_code, t.country_code
		)
		SELECT
			a.payment_method_code,
			pm.name AS payment_method_name,
			pm.type AS payment_method_type,
			a.country_code,
			a.transaction_count,
			a.approved_count,
			a.declined_count,
			a.tpv_usd,
			a.approval_rate,
			a.avg_transaction_value,
			CASE WHEN tt.total > 0
				THEN ROUND(a.tpv_usd / tt.total * 100, 2)
				ELSE 0
			END AS revenue_contribution_pct,
			COALESCE(ic.monthly_fixed_cost_usd, 0) AS monthly_cost_usd,
			CASE WHEN a.tpv_usd > 0
				THEN ROUND(COALESCE(ic.monthly_fixed_cost_usd, 0)::numeric / a.tpv_usd::numeric * 100, 2)
				ELSE 0
			END AS cost_efficiency_ratio,
			CASE
				WHEN COALESCE(t90.txn_count_90d, 0) >= 10 THEN 'ACTIVE'
				WHEN COALESCE(t90.txn_count_90d, 0) >= 1 THEN 'LOW_ACTIVITY'
				ELSE 'INACTIVE'
			END AS activity_status
		FROM txn_agg a
		JOIN payment_methods pm ON pm.code = a.payment_method_code
		CROSS JOIN total_tpv tt
		LEFT JOIN integration_costs ic ON ic.payment_method_code = a.payment_method_code
			AND ic.country_code = a.country_code
			AND ic.effective_to IS NULL
		LEFT JOIN txn_90d t90 ON t90.payment_method_code = a.payment_method_code
			AND t90.country_code = a.country_code
		WHERE ($2 = '' OR pm.type = $2)
	`

	validSorts := map[string]string{
		"transaction_count":    "a.transaction_count",
		"tpv_usd":             "a.tpv_usd",
		"approval_rate":       "a.approval_rate",
		"revenue_contribution": "revenue_contribution_pct",
		"payment_method_code": "a.payment_method_code",
	}

	sortCol, ok := validSorts[sortBy]
	if !ok {
		sortCol = "a.tpv_usd"
	}

	orderDir := "DESC"
	if order == "asc" {
		orderDir = "ASC"
	}

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM (%s) sub`, baseQuery)
	var totalItems int
	err := r.pool.QueryRow(ctx, countQuery, country, pmType, dateFrom, dateTo).Scan(&totalItems)
	if err != nil {
		return nil, 0, fmt.Errorf("count metrics: %w", err)
	}

	// Data query
	dataQuery := fmt.Sprintf(`%s ORDER BY %s %s LIMIT $5 OFFSET $6`, baseQuery, sortCol, orderDir)

	rows, err := r.pool.Query(ctx, dataQuery, country, pmType, dateFrom, dateTo, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	var results []MetricRow
	for rows.Next() {
		var m MetricRow
		err := rows.Scan(
			&m.PaymentMethodCode, &m.PaymentMethodName, &m.PaymentMethodType,
			&m.CountryCode, &m.TransactionCount, &m.ApprovedCount, &m.DeclinedCount,
			&m.TpvUSD, &m.ApprovalRate, &m.AvgTransactionValue,
			&m.RevenueContribution, &m.MonthlyCostUSD, &m.CostEfficiencyRatio,
			&m.ActivityStatus,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan metric row: %w", err)
		}
		results = append(results, m)
	}

	return results, totalItems, nil
}
