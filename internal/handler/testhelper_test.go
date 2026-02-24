package handler

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://pmhm:pmhm_secret@localhost:5434/pmhm?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil
	}

	return pool
}
