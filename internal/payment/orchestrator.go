package payment

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"crypto-conversion/internal/logger"
	"crypto-conversion/internal/models"
)

// Orchestrator handles the payment processing logic (the "stablecoin sandwich")
type Orchestrator struct {
	onRampClient  OnRampClient
	offRampClient OffRampClient
}

// OnRampClient interface for on-ramp operations
type OnRampClient interface {
	ConvertToStablecoin(ctx context.Context, amount int64, currency string) (txID string, stablecoinAmount int64, err error)
}

// OffRampClient interface for off-ramp operations
type OffRampClient interface {
	ConvertFromStablecoin(ctx context.Context, stablecoinAmount int64, currency string) (txID string, finalAmount int64, err error)
}

// NewOrchestrator creates a new payment orchestrator
func NewOrchestrator(onRamp OnRampClient, offRamp OffRampClient) *Orchestrator {
	return &Orchestrator{
		onRampClient:  onRamp,
		offRampClient: offRamp,
	}
}

// ProcessPayment executes the full payment flow
func (o *Orchestrator) ProcessPayment(ctx context.Context, job *models.PaymentJob) (*PaymentResult, error) {
	logger.Info("Starting payment processing", logger.Fields{
		"payment_id": job.PaymentID,
		"amount":     job.Amount,
		"currency":   job.Currency,
	})

	// Step 1: On-ramp - Convert fiat to stablecoin
	onRampTxID, stablecoinAmount, err := o.onRampClient.ConvertToStablecoin(ctx, job.Amount, job.Currency)
	if err != nil {
		logger.Error("On-ramp conversion failed", logger.Fields{
			"payment_id": job.PaymentID,
			"error":      err.Error(),
		})
		return nil, fmt.Errorf("on-ramp conversion failed: %w", err)
	}

	logger.Info("On-ramp conversion successful", logger.Fields{
		"payment_id":        job.PaymentID,
		"on_ramp_tx_id":     onRampTxID,
		"stablecoin_amount": stablecoinAmount,
	})

	// Step 2: Off-ramp - Convert stablecoin back to target currency
	offRampTxID, finalAmount, err := o.offRampClient.ConvertFromStablecoin(ctx, stablecoinAmount, job.Currency)
	if err != nil {
		logger.Error("Off-ramp conversion failed", logger.Fields{
			"payment_id":    job.PaymentID,
			"on_ramp_tx_id": onRampTxID,
			"error":         err.Error(),
		})
		return nil, fmt.Errorf("off-ramp conversion failed (on-ramp tx: %s): %w", onRampTxID, err)
	}

	logger.Info("Off-ramp conversion successful", logger.Fields{
		"payment_id":     job.PaymentID,
		"off_ramp_tx_id": offRampTxID,
		"final_amount":   finalAmount,
	})

	result := &PaymentResult{
		OnRampTxID:       onRampTxID,
		OffRampTxID:      offRampTxID,
		StablecoinAmount: stablecoinAmount,
		FinalAmount:      finalAmount,
	}

	logger.Info("Payment processing completed", logger.Fields{
		"payment_id":     job.PaymentID,
		"on_ramp_tx_id":  onRampTxID,
		"off_ramp_tx_id": offRampTxID,
	})

	return result, nil
}

// PaymentResult contains the result of payment processing
type PaymentResult struct {
	OnRampTxID       string
	OffRampTxID      string
	StablecoinAmount int64
	FinalAmount      int64
}

// MockOnRampClient is a mock implementation for testing/development
type MockOnRampClient struct{}

// NewMockOnRampClient creates a new mock on-ramp client
func NewMockOnRampClient() *MockOnRampClient {
	return &MockOnRampClient{}
}

// ConvertToStablecoin simulates converting fiat to stablecoin
func (m *MockOnRampClient) ConvertToStablecoin(ctx context.Context, amount int64, currency string) (string, int64, error) {
	// Simulate processing time
	time.Sleep(time.Millisecond * time.Duration(100+rand.Intn(200)))

	// Simulate occasional failures (5% failure rate)
	if rand.Float32() < 0.05 {
		return "", 0, fmt.Errorf("mock on-ramp service unavailable")
	}

	// Generate mock transaction ID
	txID := fmt.Sprintf("onramp_%s_%d", currency, time.Now().UnixNano())

	// Mock conversion: assume 1:1 ratio for simplicity
	// In real implementation, this would use actual exchange rates
	stablecoinAmount := amount

	logger.Info("Mock on-ramp conversion", logger.Fields{
		"tx_id":             txID,
		"amount":            amount,
		"currency":          currency,
		"stablecoin_amount": stablecoinAmount,
	})

	return txID, stablecoinAmount, nil
}

// MockOffRampClient is a mock implementation for testing/development
type MockOffRampClient struct{}

// NewMockOffRampClient creates a new mock off-ramp client
func NewMockOffRampClient() *MockOffRampClient {
	return &MockOffRampClient{}
}

// ConvertFromStablecoin simulates converting stablecoin to fiat
func (m *MockOffRampClient) ConvertFromStablecoin(ctx context.Context, stablecoinAmount int64, currency string) (string, int64, error) {
	// Simulate processing time
	time.Sleep(time.Millisecond * time.Duration(100+rand.Intn(200)))

	// Simulate occasional failures (5% failure rate)
	if rand.Float32() < 0.05 {
		return "", 0, fmt.Errorf("mock off-ramp service unavailable")
	}

	// Generate mock transaction ID
	txID := fmt.Sprintf("offramp_%s_%d", currency, time.Now().UnixNano())

	// Mock conversion: assume 1:1 ratio for simplicity
	// In real implementation, this would use actual exchange rates
	finalAmount := stablecoinAmount

	logger.Info("Mock off-ramp conversion", logger.Fields{
		"tx_id":             txID,
		"stablecoin_amount": stablecoinAmount,
		"currency":          currency,
		"final_amount":      finalAmount,
	})

	return txID, finalAmount, nil
}
