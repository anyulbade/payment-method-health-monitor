package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/anyulbade/payment-method-health-monitor/internal/model"
)

type PaymentMethodRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentMethodRepository(pool *pgxpool.Pool) *PaymentMethodRepository {
	return &PaymentMethodRepository{pool: pool}
}

func (r *PaymentMethodRepository) FindByCode(ctx context.Context, code string) (*model.PaymentMethod, error) {
	pm := &model.PaymentMethod{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, name, type, COALESCE(provider, '') as provider, created_at
		FROM payment_methods WHERE code = $1`, code).
		Scan(&pm.ID, &pm.Code, &pm.Name, &pm.Type, &pm.Provider, &pm.CreatedAt)
	if err != nil {
		return nil, err
	}
	return pm, nil
}

func (r *PaymentMethodRepository) Exists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM payment_methods WHERE code = $1)`, code).Scan(&exists)
	return exists, err
}

func (r *PaymentMethodRepository) ExistsInCountry(ctx context.Context, pmCode, countryCode string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM payment_method_countries WHERE payment_method_code = $1 AND country_code = $2)`,
		pmCode, countryCode).Scan(&exists)
	return exists, err
}

func (r *PaymentMethodRepository) CountryExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM countries WHERE code = $1)`, code).Scan(&exists)
	return exists, err
}

func (r *PaymentMethodRepository) GetFxRate(ctx context.Context, countryCode string) (float64, string, error) {
	var fxRate float64
	var currency string
	err := r.pool.QueryRow(ctx,
		`SELECT fx_rate_to_usd, currency FROM countries WHERE code = $1`, countryCode).
		Scan(&fxRate, &currency)
	return fxRate, currency, err
}
