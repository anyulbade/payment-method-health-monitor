package service

import (
	"context"
	"fmt"
	"math"

	"github.com/anyulbade/payment-method-health-monitor/internal/dto"
	"github.com/anyulbade/payment-method-health-monitor/internal/model"
	"github.com/anyulbade/payment-method-health-monitor/internal/repository"
)

type TransactionService struct {
	txnRepo *repository.TransactionRepository
	pmRepo  *repository.PaymentMethodRepository
}

func NewTransactionService(txnRepo *repository.TransactionRepository, pmRepo *repository.PaymentMethodRepository) *TransactionService {
	return &TransactionService{txnRepo: txnRepo, pmRepo: pmRepo}
}

func (s *TransactionService) CreateTransaction(ctx context.Context, req *dto.CreateTransactionRequest) (*model.Transaction, error) {
	if err := s.validateTransaction(ctx, req, 0); err != nil {
		return nil, err
	}

	fxRate, _, err := s.pmRepo.GetFxRate(ctx, req.CountryCode)
	if err != nil {
		return nil, fmt.Errorf("get fx rate: %w", err)
	}

	amountUSD := math.Round(req.Amount*fxRate*100) / 100

	txn := &model.Transaction{
		PaymentMethodCode: req.PaymentMethodCode,
		CountryCode:       req.CountryCode,
		Currency:          req.Currency,
		Amount:            req.Amount,
		AmountUSD:         amountUSD,
		Status:            req.Status,
		MerchantID:        req.MerchantID,
		CustomerID:        req.CustomerID,
		TransactionDate:   req.TransactionDate,
	}

	if err := s.txnRepo.Insert(ctx, txn); err != nil {
		return nil, err
	}

	return txn, nil
}

func (s *TransactionService) CreateBatch(ctx context.Context, req *dto.BatchTransactionRequest) ([]*model.Transaction, []dto.ValidationError, error) {
	var validationErrors []dto.ValidationError

	for i, txnReq := range req.Transactions {
		if err := s.validateTransaction(ctx, &txnReq, i); err != nil {
			if ve, ok := err.(*validationErr); ok {
				validationErrors = append(validationErrors, dto.ValidationError{
					Index:   i,
					Field:   ve.field,
					Message: ve.message,
				})
			} else {
				return nil, nil, err
			}
		}
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors, nil
	}

	txns := make([]*model.Transaction, len(req.Transactions))
	for i, txnReq := range req.Transactions {
		fxRate, _, err := s.pmRepo.GetFxRate(ctx, txnReq.CountryCode)
		if err != nil {
			return nil, nil, fmt.Errorf("get fx rate for item %d: %w", i, err)
		}

		amountUSD := math.Round(txnReq.Amount*fxRate*100) / 100

		txns[i] = &model.Transaction{
			PaymentMethodCode: txnReq.PaymentMethodCode,
			CountryCode:       txnReq.CountryCode,
			Currency:          txnReq.Currency,
			Amount:            txnReq.Amount,
			AmountUSD:         amountUSD,
			Status:            txnReq.Status,
			MerchantID:        txnReq.MerchantID,
			CustomerID:        txnReq.CustomerID,
			TransactionDate:   txnReq.TransactionDate,
		}
	}

	if err := s.txnRepo.InsertBatch(ctx, txns); err != nil {
		return nil, nil, err
	}

	return txns, nil, nil
}

type validationErr struct {
	field   string
	message string
}

func (e *validationErr) Error() string {
	return fmt.Sprintf("%s: %s", e.field, e.message)
}

func (s *TransactionService) validateTransaction(ctx context.Context, req *dto.CreateTransactionRequest, index int) error {
	exists, err := s.pmRepo.Exists(ctx, req.PaymentMethodCode)
	if err != nil {
		return fmt.Errorf("check payment method: %w", err)
	}
	if !exists {
		return &validationErr{field: "payment_method_code", message: fmt.Sprintf("payment method '%s' not found", req.PaymentMethodCode)}
	}

	countryExists, err := s.pmRepo.CountryExists(ctx, req.CountryCode)
	if err != nil {
		return fmt.Errorf("check country: %w", err)
	}
	if !countryExists {
		return &validationErr{field: "country_code", message: fmt.Sprintf("country '%s' not found", req.CountryCode)}
	}

	inCountry, err := s.pmRepo.ExistsInCountry(ctx, req.PaymentMethodCode, req.CountryCode)
	if err != nil {
		return fmt.Errorf("check pm in country: %w", err)
	}
	if !inCountry {
		return &validationErr{field: "country_code", message: fmt.Sprintf("payment method '%s' not available in country '%s'", req.PaymentMethodCode, req.CountryCode)}
	}

	return nil
}
