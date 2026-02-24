package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InsightRepository struct {
	pool *pgxpool.Pool
}

func NewInsightRepository(pool *pgxpool.Pool) *InsightRepository {
	return &InsightRepository{pool: pool}
}

type ZombieCandidate struct {
	PaymentMethodCode string
	PaymentMethodName string
	PaymentMethodType string
	CountryCode       string
	TxnCount90d       int
	HistoricalMonthlyAvg float64
	MonthsActive      int
	MonthlyCostUSD    float64
}

func (r *InsightRepository) GetZombieCandidates(ctx context.Context, country string) ([]ZombieCandidate, error) {
	query := `
		WITH txn_90d AS (
			SELECT payment_method_code, country_code, COUNT(*) as cnt
			FROM transactions
			WHERE transaction_date >= NOW() - INTERVAL '90 days'
			GROUP BY payment_method_code, country_code
		),
		historical AS (
			SELECT payment_method_code, country_code,
				COUNT(*)::float / GREATEST(
					EXTRACT(EPOCH FROM (MAX(transaction_date) - MIN(transaction_date))) / (30*86400),
					1
				) as monthly_avg,
				EXTRACT(EPOCH FROM (MAX(transaction_date) - MIN(transaction_date))) / (30*86400) as months_active
			FROM transactions
			GROUP BY payment_method_code, country_code
		)
		SELECT ic.payment_method_code, pm.name, pm.type, ic.country_code,
			COALESCE(t90.cnt, 0) as txn_count_90d,
			COALESCE(h.monthly_avg, 0) as historical_monthly_avg,
			COALESCE(h.months_active, 0)::int as months_active,
			ic.monthly_fixed_cost_usd
		FROM integration_costs ic
		JOIN payment_methods pm ON pm.code = ic.payment_method_code
		LEFT JOIN txn_90d t90 ON t90.payment_method_code = ic.payment_method_code AND t90.country_code = ic.country_code
		LEFT JOIN historical h ON h.payment_method_code = ic.payment_method_code AND h.country_code = ic.country_code
		WHERE ic.effective_to IS NULL
			AND ($1 = '' OR ic.country_code = $1)
	`
	rows, err := r.pool.Query(ctx, query, country)
	if err != nil {
		return nil, fmt.Errorf("query zombie candidates: %w", err)
	}
	defer rows.Close()

	var results []ZombieCandidate
	for rows.Next() {
		var z ZombieCandidate
		if err := rows.Scan(&z.PaymentMethodCode, &z.PaymentMethodName, &z.PaymentMethodType,
			&z.CountryCode, &z.TxnCount90d, &z.HistoricalMonthlyAvg, &z.MonthsActive, &z.MonthlyCostUSD); err != nil {
			return nil, fmt.Errorf("scan zombie: %w", err)
		}
		results = append(results, z)
	}
	return results, nil
}

type HiddenGemCandidate struct {
	PaymentMethodCode   string
	PaymentMethodName   string
	CountryCode         string
	ApprovalRate        float64
	RevenueContribution float64
	VolumeShare         float64
	TpvUSD              float64
	TransactionCount    int
}

func (r *InsightRepository) GetHiddenGemCandidates(ctx context.Context, country string) ([]HiddenGemCandidate, error) {
	query := `
		WITH txn_agg AS (
			SELECT payment_method_code, country_code,
				COUNT(*) as txn_count,
				COALESCE(SUM(amount_usd) FILTER (WHERE status = 'APPROVED'), 0) as tpv_usd,
				CASE WHEN COUNT(*) > 0
					THEN COUNT(*) FILTER (WHERE status = 'APPROVED')::float / COUNT(*)::float * 100
					ELSE 0
				END as approval_rate
			FROM transactions
			WHERE ($1 = '' OR country_code = $1)
			GROUP BY payment_method_code, country_code
		),
		totals AS (
			SELECT SUM(tpv_usd) as total_tpv, SUM(txn_count) as total_txns FROM txn_agg
		)
		SELECT a.payment_method_code, pm.name, a.country_code,
			a.approval_rate,
			CASE WHEN t.total_tpv > 0 THEN a.tpv_usd / t.total_tpv * 100 ELSE 0 END as revenue_contribution,
			CASE WHEN t.total_txns > 0 THEN a.txn_count::float / t.total_txns::float * 100 ELSE 0 END as volume_share,
			a.tpv_usd,
			a.txn_count
		FROM txn_agg a
		JOIN payment_methods pm ON pm.code = a.payment_method_code
		CROSS JOIN totals t
	`
	rows, err := r.pool.Query(ctx, query, country)
	if err != nil {
		return nil, fmt.Errorf("query hidden gems: %w", err)
	}
	defer rows.Close()

	var results []HiddenGemCandidate
	for rows.Next() {
		var h HiddenGemCandidate
		if err := rows.Scan(&h.PaymentMethodCode, &h.PaymentMethodName, &h.CountryCode,
			&h.ApprovalRate, &h.RevenueContribution, &h.VolumeShare, &h.TpvUSD, &h.TransactionCount); err != nil {
			return nil, fmt.Errorf("scan hidden gem: %w", err)
		}
		results = append(results, h)
	}
	return results, nil
}

type PerformanceAlertCandidate struct {
	PaymentMethodCode      string
	PaymentMethodName      string
	PaymentMethodType      string
	CountryCode            string
	ApprovalRate           float64
	CountryTypeAvgApproval float64
	TransactionCount       int
}

func (r *InsightRepository) GetPerformanceAlertCandidates(ctx context.Context, country string) ([]PerformanceAlertCandidate, error) {
	query := `
		WITH method_stats AS (
			SELECT t.payment_method_code, t.country_code, pm.type as pm_type,
				COUNT(*) as txn_count,
				CASE WHEN COUNT(*) > 0
					THEN COUNT(*) FILTER (WHERE t.status = 'APPROVED')::float / COUNT(*)::float * 100
					ELSE 0
				END as approval_rate
			FROM transactions t
			JOIN payment_methods pm ON pm.code = t.payment_method_code
			WHERE ($1 = '' OR t.country_code = $1)
			GROUP BY t.payment_method_code, t.country_code, pm.type
		),
		type_avgs AS (
			SELECT country_code, pm_type, AVG(approval_rate) as avg_approval
			FROM method_stats
			GROUP BY country_code, pm_type
		)
		SELECT ms.payment_method_code, pm.name, ms.pm_type, ms.country_code,
			ms.approval_rate,
			ta.avg_approval as country_type_avg,
			ms.txn_count
		FROM method_stats ms
		JOIN payment_methods pm ON pm.code = ms.payment_method_code
		JOIN type_avgs ta ON ta.country_code = ms.country_code AND ta.pm_type = ms.pm_type
	`
	rows, err := r.pool.Query(ctx, query, country)
	if err != nil {
		return nil, fmt.Errorf("query perf alerts: %w", err)
	}
	defer rows.Close()

	var results []PerformanceAlertCandidate
	for rows.Next() {
		var p PerformanceAlertCandidate
		if err := rows.Scan(&p.PaymentMethodCode, &p.PaymentMethodName, &p.PaymentMethodType,
			&p.CountryCode, &p.ApprovalRate, &p.CountryTypeAvgApproval, &p.TransactionCount); err != nil {
			return nil, fmt.Errorf("scan perf alert: %w", err)
		}
		results = append(results, p)
	}
	return results, nil
}
