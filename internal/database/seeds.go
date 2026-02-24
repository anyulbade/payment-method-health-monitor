package database

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/anyulbade/payment-method-health-monitor/seeddata"
)

type catalogEntry struct {
	CountryCode       string  `json:"country_code"`
	PaymentMethodCode string  `json:"payment_method_code"`
	MarketSharePct    float64 `json:"market_share_pct"`
	IsEssential       bool    `json:"is_essential"`
	Source            string  `json:"source"`
}

type pmProfile struct {
	Code         string
	Name         string
	Type         string
	Provider     string
	Countries    []string
	TxnRange     [2]int     // min, max txns per country
	ApprovalRate [2]float64 // min, max approval rate
	AvgAmount    [2]float64 // min, max avg amount in local currency
	Category     string     // champion, average, zombie, hidden_gem, problem_child
}

var countries = []struct {
	Code     string
	Name     string
	Currency string
	FxRate   float64
}{
	{"MX", "Mexico", "MXN", 0.0580},
	{"BR", "Brazil", "BRL", 0.1960},
	{"CO", "Colombia", "COP", 0.000245},
	{"AR", "Argentina", "ARS", 0.00115},
	{"CL", "Chile", "CLP", 0.00108},
	{"PE", "Peru", "PEN", 0.2680},
}

var paymentMethods = []pmProfile{
	// Champions
	{Code: "PIX", Name: "PIX Instant Payment", Type: "BANK_TRANSFER", Provider: "BCB", Countries: []string{"BR"}, TxnRange: [2]int{60, 80}, ApprovalRate: [2]float64{0.93, 0.97}, AvgAmount: [2]float64{100, 500}, Category: "champion"},
	{Code: "VISA_CREDIT", Name: "Visa Credit Card", Type: "CARD", Provider: "Visa", Countries: []string{"BR", "CO", "AR", "CL", "PE"}, TxnRange: [2]int{50, 70}, ApprovalRate: [2]float64{0.85, 0.95}, AvgAmount: [2]float64{200, 800}, Category: "champion"},
	{Code: "SPEI", Name: "SPEI Bank Transfer", Type: "BANK_TRANSFER", Provider: "Banxico", Countries: []string{"MX"}, TxnRange: [2]int{55, 75}, ApprovalRate: [2]float64{0.90, 0.97}, AvgAmount: [2]float64{2000, 8000}, Category: "champion"},
	{Code: "OXXO", Name: "OXXO Cash Payment", Type: "CASH", Provider: "OXXO", Countries: []string{"MX"}, TxnRange: [2]int{50, 70}, ApprovalRate: [2]float64{0.85, 0.92}, AvgAmount: [2]float64{500, 3000}, Category: "champion"},

	// Average
	{Code: "MC_CREDIT", Name: "Mastercard Credit", Type: "CARD", Provider: "Mastercard", Countries: []string{"BR", "CO", "AR", "CL"}, TxnRange: [2]int{25, 40}, ApprovalRate: [2]float64{0.75, 0.88}, AvgAmount: [2]float64{150, 600}, Category: "average"},
	{Code: "PSE", Name: "PSE Bank Transfer", Type: "BANK_TRANSFER", Provider: "ACH Colombia", Countries: []string{"CO"}, TxnRange: [2]int{30, 40}, ApprovalRate: [2]float64{0.78, 0.85}, AvgAmount: [2]float64{200000, 800000}, Category: "average"},
	{Code: "BOLETO", Name: "Boleto Bancário", Type: "CASH", Provider: "FEBRABAN", Countries: []string{"BR"}, TxnRange: [2]int{20, 35}, ApprovalRate: [2]float64{0.70, 0.80}, AvgAmount: [2]float64{200, 1000}, Category: "average"},
	{Code: "MERCADOPAGO", Name: "MercadoPago Wallet", Type: "WALLET", Provider: "MercadoLibre", Countries: []string{"MX", "AR"}, TxnRange: [2]int{25, 35}, ApprovalRate: [2]float64{0.80, 0.88}, AvgAmount: [2]float64{500, 2000}, Category: "average"},
	{Code: "NUBANK_CREDIT", Name: "Nubank Credit Card", Type: "CARD", Provider: "Nubank", Countries: []string{"BR"}, TxnRange: [2]int{20, 30}, ApprovalRate: [2]float64{0.82, 0.88}, AvgAmount: [2]float64{150, 500}, Category: "average"},
	{Code: "WEBPAY", Name: "WebPay Plus", Type: "BANK_TRANSFER", Provider: "Transbank", Countries: []string{"CL"}, TxnRange: [2]int{30, 45}, ApprovalRate: [2]float64{0.80, 0.88}, AvgAmount: [2]float64{30000, 120000}, Category: "average"},

	// Zombies
	{Code: "RAPIPAGO", Name: "Rapipago Cash", Type: "CASH", Provider: "Rapipago", Countries: []string{"AR"}, TxnRange: [2]int{3, 8}, ApprovalRate: [2]float64{0.60, 0.75}, AvgAmount: [2]float64{5000, 15000}, Category: "zombie"},
	{Code: "DAVIPLATA", Name: "DaviPlata Wallet", Type: "WALLET", Provider: "Davivienda", Countries: []string{"CO"}, TxnRange: [2]int{2, 6}, ApprovalRate: [2]float64{0.65, 0.78}, AvgAmount: [2]float64{100000, 300000}, Category: "zombie"},
	{Code: "KUESKI", Name: "Kueski Pay BNPL", Type: "BNPL", Provider: "Kueski", Countries: []string{"MX"}, TxnRange: [2]int{3, 7}, ApprovalRate: [2]float64{0.55, 0.70}, AvgAmount: [2]float64{2000, 6000}, Category: "zombie"},
	{Code: "FPAY", Name: "FPAY Wallet", Type: "WALLET", Provider: "Falabella", Countries: []string{"CL"}, TxnRange: [2]int{2, 5}, ApprovalRate: [2]float64{0.60, 0.72}, AvgAmount: [2]float64{20000, 60000}, Category: "zombie"},

	// Hidden Gems
	{Code: "NEQUI", Name: "Nequi Wallet", Type: "WALLET", Provider: "Bancolombia", Countries: []string{"CO"}, TxnRange: [2]int{15, 25}, ApprovalRate: [2]float64{0.93, 0.97}, AvgAmount: [2]float64{300000, 900000}, Category: "hidden_gem"},
	{Code: "ADDI", Name: "Addi BNPL", Type: "BNPL", Provider: "Addi", Countries: []string{"CO"}, TxnRange: [2]int{12, 20}, ApprovalRate: [2]float64{0.93, 0.96}, AvgAmount: [2]float64{500000, 1500000}, Category: "hidden_gem"},
	{Code: "YAPE", Name: "Yape Mobile Wallet", Type: "WALLET", Provider: "BCP", Countries: []string{"PE"}, TxnRange: [2]int{15, 22}, ApprovalRate: [2]float64{0.94, 0.97}, AvgAmount: [2]float64{200, 600}, Category: "hidden_gem"},

	// Problem child (VISA in MX specifically)
	{Code: "VISA_CREDIT_MX", Name: "Visa Credit Card", Type: "CARD", Provider: "Visa", Countries: []string{"MX"}, TxnRange: [2]int{50, 65}, ApprovalRate: [2]float64{0.60, 0.65}, AvgAmount: [2]float64{1500, 5000}, Category: "problem_child"},

	// Additional methods
	{Code: "EFECTY", Name: "Efecty Cash Payment", Type: "CASH", Provider: "Efecty", Countries: []string{"CO"}, TxnRange: [2]int{15, 25}, ApprovalRate: [2]float64{0.75, 0.82}, AvgAmount: [2]float64{100000, 400000}, Category: "average"},
	{Code: "VISA_DEBIT", Name: "Visa Debit Card", Type: "CARD", Provider: "Visa", Countries: []string{"BR", "MX", "AR"}, TxnRange: [2]int{20, 35}, ApprovalRate: [2]float64{0.78, 0.88}, AvgAmount: [2]float64{100, 400}, Category: "average"},
	{Code: "PAGOFACIL", Name: "Pago Fácil Cash", Type: "CASH", Provider: "PagoFácil", Countries: []string{"AR"}, TxnRange: [2]int{10, 18}, ApprovalRate: [2]float64{0.72, 0.80}, AvgAmount: [2]float64{8000, 25000}, Category: "average"},
}

func SeedData(ctx context.Context, pool *pgxpool.Pool) error {
	rng := rand.New(rand.NewSource(42))

	// Check if data already exists (idempotency)
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM countries").Scan(&count)
	if err != nil {
		return fmt.Errorf("check existing data: %w", err)
	}
	if count > 0 {
		log.Info().Msg("seed data already exists, skipping")
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert countries
	for _, c := range countries {
		_, err := tx.Exec(ctx,
			"INSERT INTO countries (code, name, currency, fx_rate_to_usd) VALUES ($1, $2, $3, $4)",
			c.Code, c.Name, c.Currency, c.FxRate)
		if err != nil {
			return fmt.Errorf("insert country %s: %w", c.Code, err)
		}
	}
	log.Info().Int("count", len(countries)).Msg("inserted countries")

	// Insert payment methods and country mappings
	// VISA_CREDIT_MX is actually VISA_CREDIT in MX, so we handle it specially
	insertedMethods := make(map[string]bool)
	for _, pm := range paymentMethods {
		actualCode := pm.Code
		if pm.Code == "VISA_CREDIT_MX" {
			actualCode = "VISA_CREDIT"
		}

		if !insertedMethods[actualCode] {
			_, err := tx.Exec(ctx,
				"INSERT INTO payment_methods (code, name, type, provider) VALUES ($1, $2, $3, $4)",
				actualCode, pm.Name, pm.Type, pm.Provider)
			if err != nil {
				return fmt.Errorf("insert payment method %s: %w", actualCode, err)
			}
			insertedMethods[actualCode] = true
		}

		for _, cc := range pm.Countries {
			_, err := tx.Exec(ctx,
				"INSERT INTO payment_method_countries (payment_method_code, country_code) VALUES ($1, $2) ON CONFLICT DO NOTHING",
				actualCode, cc)
			if err != nil {
				return fmt.Errorf("insert pm_country %s-%s: %w", actualCode, cc, err)
			}
		}
	}
	log.Info().Int("count", len(insertedMethods)).Msg("inserted payment methods")

	// Generate transactions across 6 months (Sep 2025 - Feb 2026)
	// 60% weight in last 2 months
	baseDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	totalTxns := 0

	for _, pm := range paymentMethods {
		actualCode := pm.Code
		if pm.Code == "VISA_CREDIT_MX" {
			actualCode = "VISA_CREDIT"
		}

		for _, cc := range pm.Countries {
			// For VISA_CREDIT in non-MX countries, skip VISA_CREDIT_MX profile
			if pm.Code == "VISA_CREDIT_MX" && cc != "MX" {
				continue
			}
			if pm.Code == "VISA_CREDIT" && cc == "MX" {
				continue // MX handled by VISA_CREDIT_MX profile
			}

			numTxns := pm.TxnRange[0] + rng.Intn(pm.TxnRange[1]-pm.TxnRange[0]+1)

			// Get country's currency and fx rate
			var currency string
			var fxRate float64
			for _, c := range countries {
				if c.Code == cc {
					currency = c.Currency
					fxRate = c.FxRate
					break
				}
			}

			for i := 0; i < numTxns; i++ {
				// Weight 60% in last 2 months
				var monthOffset int
				if rng.Float64() < 0.6 {
					monthOffset = 4 + rng.Intn(2) // months 4-5 (Jan-Feb 2026)
				} else {
					monthOffset = rng.Intn(4) // months 0-3 (Sep-Dec 2025)
				}

				txnDate := baseDate.AddDate(0, monthOffset, rng.Intn(28)).
					Add(time.Duration(rng.Intn(24)) * time.Hour).
					Add(time.Duration(rng.Intn(60)) * time.Minute)

				// Amount in local currency
				amtRange := pm.AvgAmount[1] - pm.AvgAmount[0]
				amount := pm.AvgAmount[0] + rng.Float64()*amtRange
				amount = math.Round(amount*100) / 100
				amountUSD := math.Round(amount*fxRate*100) / 100

				// Status based on approval rate
				approvalRate := pm.ApprovalRate[0] + rng.Float64()*(pm.ApprovalRate[1]-pm.ApprovalRate[0])
				status := "APPROVED"
				roll := rng.Float64()
				if roll > approvalRate {
					if rng.Float64() < 0.85 {
						status = "DECLINED"
					} else if rng.Float64() < 0.5 {
						status = "PENDING"
					} else {
						status = "REFUNDED"
					}
				}

				merchantID := fmt.Sprintf("merchant_%03d", rng.Intn(50)+1)
				customerID := fmt.Sprintf("customer_%05d", rng.Intn(5000)+1)

				_, err := tx.Exec(ctx,
					`INSERT INTO transactions (payment_method_code, country_code, currency, amount, amount_usd, status, merchant_id, customer_id, transaction_date)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
					actualCode, cc, currency, amount, amountUSD, status, merchantID, customerID, txnDate)
				if err != nil {
					return fmt.Errorf("insert transaction: %w", err)
				}
				totalTxns++
			}
		}
	}
	log.Info().Int("count", totalTxns).Msg("inserted transactions")

	// Insert integration costs
	costProfiles := []struct {
		PMCode    string
		Country   string
		Monthly   float64
		PerTxn    float64
		PctFee    float64
	}{
		// Champions - low relative cost
		{"PIX", "BR", 50.00, 0.02, 0.0010},
		{"VISA_CREDIT", "BR", 200.00, 0.15, 0.0250},
		{"VISA_CREDIT", "CO", 200.00, 0.15, 0.0250},
		{"VISA_CREDIT", "AR", 200.00, 0.15, 0.0250},
		{"VISA_CREDIT", "CL", 200.00, 0.15, 0.0250},
		{"VISA_CREDIT", "PE", 200.00, 0.15, 0.0250},
		{"VISA_CREDIT", "MX", 200.00, 0.15, 0.0250},
		{"SPEI", "MX", 75.00, 0.03, 0.0015},
		{"OXXO", "MX", 100.00, 0.10, 0.0200},

		// Average
		{"MC_CREDIT", "BR", 180.00, 0.12, 0.0230},
		{"MC_CREDIT", "CO", 180.00, 0.12, 0.0230},
		{"MC_CREDIT", "AR", 180.00, 0.12, 0.0230},
		{"MC_CREDIT", "CL", 180.00, 0.12, 0.0230},
		{"PSE", "CO", 80.00, 0.05, 0.0020},
		{"BOLETO", "BR", 60.00, 0.08, 0.0015},
		{"MERCADOPAGO", "MX", 100.00, 0.08, 0.0350},
		{"MERCADOPAGO", "AR", 100.00, 0.08, 0.0350},
		{"NUBANK_CREDIT", "BR", 120.00, 0.10, 0.0200},
		{"WEBPAY", "CL", 90.00, 0.06, 0.0180},
		{"EFECTY", "CO", 70.00, 0.07, 0.0015},
		{"VISA_DEBIT", "BR", 150.00, 0.10, 0.0150},
		{"VISA_DEBIT", "MX", 150.00, 0.10, 0.0150},
		{"VISA_DEBIT", "AR", 150.00, 0.10, 0.0150},
		{"PAGOFACIL", "AR", 60.00, 0.06, 0.0012},

		// Zombies - high relative cost (still paying monthly for low volume)
		{"RAPIPAGO", "AR", 80.00, 0.08, 0.0015},
		{"DAVIPLATA", "CO", 90.00, 0.06, 0.0020},
		{"KUESKI", "MX", 150.00, 0.20, 0.0300},
		{"FPAY", "CL", 85.00, 0.07, 0.0025},

		// Hidden gems
		{"NEQUI", "CO", 70.00, 0.04, 0.0015},
		{"ADDI", "CO", 120.00, 0.15, 0.0280},
		{"YAPE", "PE", 60.00, 0.03, 0.0012},
	}

	for _, cp := range costProfiles {
		_, err := tx.Exec(ctx,
			`INSERT INTO integration_costs (payment_method_code, country_code, monthly_fixed_cost_usd, per_transaction_cost_usd, percentage_fee, effective_from)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			cp.PMCode, cp.Country, cp.Monthly, cp.PerTxn, cp.PctFee, "2025-01-01")
		if err != nil {
			return fmt.Errorf("insert cost %s-%s: %w", cp.PMCode, cp.Country, err)
		}
	}
	log.Info().Int("count", len(costProfiles)).Msg("inserted integration costs")

	// Insert country payment catalog from embedded JSON
	var catalog []catalogEntry
	if err := json.Unmarshal(seeddata.CountryCatalogJSON, &catalog); err != nil {
		return fmt.Errorf("parse catalog JSON: %w", err)
	}

	for _, entry := range catalog {
		_, err := tx.Exec(ctx,
			`INSERT INTO country_payment_catalog (country_code, payment_method_code, market_share_pct, is_essential, source)
			VALUES ($1, $2, $3, $4, $5)`,
			entry.CountryCode, entry.PaymentMethodCode, entry.MarketSharePct, entry.IsEssential, entry.Source)
		if err != nil {
			return fmt.Errorf("insert catalog %s-%s: %w", entry.CountryCode, entry.PaymentMethodCode, err)
		}
	}
	log.Info().Int("count", len(catalog)).Msg("inserted country payment catalog")

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed data: %w", err)
	}

	log.Info().Msg("seed data generation complete")
	return nil
}
