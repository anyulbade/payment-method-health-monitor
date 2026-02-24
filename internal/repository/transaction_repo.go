package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/anyulbade/payment-method-health-monitor/internal/model"
)

type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

func (r *TransactionRepository) Insert(ctx context.Context, txn *model.Transaction) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO transactions (payment_method_code, country_code, currency, amount, amount_usd, status, merchant_id, customer_id, transaction_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`,
		txn.PaymentMethodCode, txn.CountryCode, txn.Currency, txn.Amount, txn.AmountUSD,
		txn.Status, txn.MerchantID, txn.CustomerID, txn.TransactionDate,
	).Scan(&txn.ID, &txn.CreatedAt)
}

func (r *TransactionRepository) InsertBatch(ctx context.Context, txns []*model.Transaction) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin batch transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	for _, txn := range txns {
		batch.Queue(
			`INSERT INTO transactions (payment_method_code, country_code, currency, amount, amount_usd, status, merchant_id, customer_id, transaction_date)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id, created_at`,
			txn.PaymentMethodCode, txn.CountryCode, txn.Currency, txn.Amount, txn.AmountUSD,
			txn.Status, txn.MerchantID, txn.CustomerID, txn.TransactionDate,
		)
	}

	br := tx.SendBatch(ctx, batch)
	for i := range txns {
		if err := br.QueryRow().Scan(&txns[i].ID, &txns[i].CreatedAt); err != nil {
			br.Close()
			return fmt.Errorf("insert transaction %d: %w", i, err)
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("close batch: %w", err)
	}

	return tx.Commit(ctx)
}
