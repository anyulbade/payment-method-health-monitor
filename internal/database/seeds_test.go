package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Tests run from package dir; point to project-root migrations
	MigrationsDir = "file://../../migrations"
	t.Cleanup(func() { MigrationsDir = "file://migrations" })

	dbURL := getTestDBURL()
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skip("no database available")
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		t.Skip("no database available")
	}

	// Clean and migrate
	_ = RollbackMigrations(dbURL)
	require.NoError(t, RunMigrations(dbURL))

	ctx := context.Background()

	t.Run("seed produces correct counts", func(t *testing.T) {
		err := SeedData(ctx, pool)
		require.NoError(t, err)

		// Verify countries
		var countryCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM countries").Scan(&countryCount)
		require.NoError(t, err)
		assert.Equal(t, 6, countryCount, "should have 6 countries")

		// Verify payment methods (21 unique, but VISA_CREDIT_MX maps to VISA_CREDIT)
		var pmCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM payment_methods").Scan(&pmCount)
		require.NoError(t, err)
		assert.Equal(t, 20, pmCount, "should have 20 payment methods")

		// Verify transactions exist with reasonable count
		var txnCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&txnCount)
		require.NoError(t, err)
		assert.Greater(t, txnCount, 350, "should have >350 transactions")
		assert.Less(t, txnCount, 1200, "should have <1200 transactions")

		// Verify status distribution
		var approvedCount, declinedCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE status = 'APPROVED'").Scan(&approvedCount)
		require.NoError(t, err)
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE status = 'DECLINED'").Scan(&declinedCount)
		require.NoError(t, err)

		approvalRate := float64(approvedCount) / float64(txnCount)
		assert.Greater(t, approvalRate, 0.65, "overall approval rate should be >65%")
		assert.Less(t, approvalRate, 0.95, "overall approval rate should be <95%")
		assert.Greater(t, declinedCount, 0, "should have some declined transactions")

		// Verify integration costs
		var costCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM integration_costs").Scan(&costCount)
		require.NoError(t, err)
		assert.Greater(t, costCount, 20, "should have >20 cost entries")

		// Verify catalog
		var catalogCount int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM country_payment_catalog").Scan(&catalogCount)
		require.NoError(t, err)
		assert.Greater(t, catalogCount, 25, "should have >25 catalog entries")
	})

	t.Run("idempotency - running twice does not duplicate", func(t *testing.T) {
		var txnCountBefore int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&txnCountBefore)

		err := SeedData(ctx, pool)
		require.NoError(t, err)

		var txnCountAfter int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&txnCountAfter)
		assert.Equal(t, txnCountBefore, txnCountAfter, "second seed should not add data")
	})

	t.Run("transaction dates are weighted to recent months", func(t *testing.T) {
		var recentCount, totalCount int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE transaction_date >= '2026-01-01'").Scan(&recentCount)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&totalCount)

		recentPct := float64(recentCount) / float64(totalCount)
		assert.Greater(t, recentPct, 0.45, "should have >45% in last 2 months (target 60%)")
	})

	// Clean up
	_ = RollbackMigrations(dbURL)
}
