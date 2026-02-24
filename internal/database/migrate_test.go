package database

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestDBURL() string {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://pmhm:pmhm_secret@localhost:5434/pmhm?sslmode=disable"
	}
	return url
}

func TestMigrations_ApplyAndRollback(t *testing.T) {
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

	// Clean slate
	_ = RollbackMigrations(dbURL)

	// Apply all migrations
	err = RunMigrations(dbURL)
	require.NoError(t, err, "migrations should apply cleanly")

	// Verify tables exist
	tables := []string{"countries", "payment_methods", "payment_method_countries", "transactions", "integration_costs", "country_payment_catalog"}
	for _, table := range tables {
		var exists bool
		err := pool.QueryRow(context.Background(),
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)", table).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "table %s should exist", table)
	}

	// Rollback all
	err = RollbackMigrations(dbURL)
	require.NoError(t, err, "rollback should succeed")

	// Re-apply (idempotency)
	err = RunMigrations(dbURL)
	require.NoError(t, err, "re-apply should succeed")

	// Verify CHECK constraints
	t.Run("country code constraint", func(t *testing.T) {
		_, err := pool.Exec(context.Background(),
			"INSERT INTO countries (code, name, currency, fx_rate_to_usd) VALUES ($1, $2, $3, $4)",
			"xx", "Bad", "USD", 1.0)
		assert.Error(t, err, "lowercase country code should be rejected")
	})

	t.Run("negative fx rate constraint", func(t *testing.T) {
		_, err := pool.Exec(context.Background(),
			"INSERT INTO countries (code, name, currency, fx_rate_to_usd) VALUES ($1, $2, $3, $4)",
			"ZZ", "Bad", "USD", -1.0)
		assert.Error(t, err, "negative fx rate should be rejected")
	})

	t.Run("invalid payment method type", func(t *testing.T) {
		_, err := pool.Exec(context.Background(),
			"INSERT INTO payment_methods (code, name, type) VALUES ($1, $2, $3)",
			"TEST_PM", "Test", "INVALID_TYPE")
		assert.Error(t, err, "invalid pm type should be rejected")
	})

	t.Run("negative transaction amount", func(t *testing.T) {
		// First insert valid country and payment method
		pool.Exec(context.Background(),
			"INSERT INTO countries (code, name, currency, fx_rate_to_usd) VALUES ('US', 'United States', 'USD', 1.0) ON CONFLICT DO NOTHING")
		pool.Exec(context.Background(),
			"INSERT INTO payment_methods (code, name, type) VALUES ('TEST_CARD', 'Test Card', 'CARD') ON CONFLICT DO NOTHING")

		_, err := pool.Exec(context.Background(),
			"INSERT INTO transactions (payment_method_code, country_code, currency, amount, amount_usd, status, transaction_date) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			"TEST_CARD", "US", "USD", -10.00, -10.00, "APPROVED", "2025-01-01T00:00:00Z")
		assert.Error(t, err, "negative amount should be rejected")
	})

	// Clean up
	_ = RollbackMigrations(dbURL)
}
