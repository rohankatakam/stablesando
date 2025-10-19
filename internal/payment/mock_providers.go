package payment

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"crypto-conversion/internal/logger"
)

// TransferStatus represents the status of a transfer
type TransferStatus string

const (
	TransferStatusPending  TransferStatus = "PENDING"
	TransferStatusSettled  TransferStatus = "SETTLED"
	TransferStatusFailed   TransferStatus = "FAILED"
)

// Transfer represents an in-flight transfer
type Transfer struct {
	TxID             string
	Status           TransferStatus
	Amount           int64
	Currency         string
	StablecoinAmount int64
	CreatedAt        time.Time
	SettledAt        *time.Time
	PollCount        int
	SettlesAfterPoll int // Settles after this many poll attempts
}

// StatefulOnRampClient is a mock that simulates async settlement
type StatefulOnRampClient struct {
	transfers map[string]*Transfer
	mu        sync.RWMutex
}

// NewStatefulOnRampClient creates a new stateful on-ramp client
func NewStatefulOnRampClient() *StatefulOnRampClient {
	return &StatefulOnRampClient{
		transfers: make(map[string]*Transfer),
	}
}

// InitiateTransfer starts an on-ramp transfer (returns immediately)
func (c *StatefulOnRampClient) InitiateTransfer(ctx context.Context, amount int64, currency string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate transaction ID
	txID := fmt.Sprintf("onramp_%s_%d", currency, time.Now().UnixNano())

	// Simulate 2% immediate failure rate
	if rand.Float32() < 0.02 {
		return "", fmt.Errorf("mock on-ramp initiation failed")
	}

	// Create pending transfer
	// Settles after 2-4 poll attempts (simulating 4-8 minutes at 2-min polling)
	settlesAfter := 2 + rand.Intn(3)

	transfer := &Transfer{
		TxID:             txID,
		Status:           TransferStatusPending,
		Amount:           amount,
		Currency:         currency,
		StablecoinAmount: amount, // 1:1 for simplicity
		CreatedAt:        time.Now(),
		PollCount:        0,
		SettlesAfterPoll: settlesAfter,
	}

	c.transfers[txID] = transfer

	logger.Info("On-ramp transfer initiated", logger.Fields{
		"tx_id":              txID,
		"amount":             amount,
		"currency":           currency,
		"settles_after_poll": settlesAfter,
	})

	return txID, nil
}

// GetTransferStatus polls the status of a transfer
func (c *StatefulOnRampClient) GetTransferStatus(ctx context.Context, txID string) (*Transfer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	transfer, exists := c.transfers[txID]
	if !exists {
		return nil, fmt.Errorf("transfer not found: %s", txID)
	}

	// Increment poll count
	transfer.PollCount++

	// Check if it should settle now
	if transfer.Status == TransferStatusPending && transfer.PollCount >= transfer.SettlesAfterPoll {
		// Simulate 5% failure rate on settlement
		if rand.Float32() < 0.05 {
			transfer.Status = TransferStatusFailed
			logger.Warn("On-ramp transfer failed", logger.Fields{
				"tx_id":      txID,
				"poll_count": transfer.PollCount,
			})
		} else {
			transfer.Status = TransferStatusSettled
			now := time.Now()
			transfer.SettledAt = &now
			logger.Info("On-ramp transfer settled", logger.Fields{
				"tx_id":             txID,
				"poll_count":        transfer.PollCount,
				"stablecoin_amount": transfer.StablecoinAmount,
			})
		}
	}

	logger.Info("On-ramp status polled", logger.Fields{
		"tx_id":      txID,
		"status":     transfer.Status,
		"poll_count": transfer.PollCount,
	})

	// Return a copy to avoid external modification
	return &Transfer{
		TxID:             transfer.TxID,
		Status:           transfer.Status,
		Amount:           transfer.Amount,
		Currency:         transfer.Currency,
		StablecoinAmount: transfer.StablecoinAmount,
		CreatedAt:        transfer.CreatedAt,
		SettledAt:        transfer.SettledAt,
		PollCount:        transfer.PollCount,
		SettlesAfterPoll: transfer.SettlesAfterPoll,
	}, nil
}

// StatefulOffRampClient is a mock that simulates async settlement
type StatefulOffRampClient struct {
	transfers map[string]*Transfer
	mu        sync.RWMutex
}

// NewStatefulOffRampClient creates a new stateful off-ramp client
func NewStatefulOffRampClient() *StatefulOffRampClient {
	return &StatefulOffRampClient{
		transfers: make(map[string]*Transfer),
	}
}

// InitiateTransfer starts an off-ramp transfer (returns immediately)
func (c *StatefulOffRampClient) InitiateTransfer(ctx context.Context, stablecoinAmount int64, currency string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate transaction ID
	txID := fmt.Sprintf("offramp_%s_%d", currency, time.Now().UnixNano())

	// Simulate 2% immediate failure rate
	if rand.Float32() < 0.02 {
		return "", fmt.Errorf("mock off-ramp initiation failed")
	}

	// Create pending transfer
	// Settles after 2-4 poll attempts
	settlesAfter := 2 + rand.Intn(3)

	transfer := &Transfer{
		TxID:             txID,
		Status:           TransferStatusPending,
		StablecoinAmount: stablecoinAmount,
		Amount:           stablecoinAmount, // 1:1 for simplicity
		Currency:         currency,
		CreatedAt:        time.Now(),
		PollCount:        0,
		SettlesAfterPoll: settlesAfter,
	}

	c.transfers[txID] = transfer

	logger.Info("Off-ramp transfer initiated", logger.Fields{
		"tx_id":              txID,
		"stablecoin_amount":  stablecoinAmount,
		"currency":           currency,
		"settles_after_poll": settlesAfter,
	})

	return txID, nil
}

// GetTransferStatus polls the status of a transfer
func (c *StatefulOffRampClient) GetTransferStatus(ctx context.Context, txID string) (*Transfer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	transfer, exists := c.transfers[txID]
	if !exists {
		return nil, fmt.Errorf("transfer not found: %s", txID)
	}

	// Increment poll count
	transfer.PollCount++

	// Check if it should settle now
	if transfer.Status == TransferStatusPending && transfer.PollCount >= transfer.SettlesAfterPoll {
		// Simulate 5% failure rate on settlement
		if rand.Float32() < 0.05 {
			transfer.Status = TransferStatusFailed
			logger.Warn("Off-ramp transfer failed", logger.Fields{
				"tx_id":      txID,
				"poll_count": transfer.PollCount,
			})
		} else {
			transfer.Status = TransferStatusSettled
			now := time.Now()
			transfer.SettledAt = &now
			logger.Info("Off-ramp transfer settled", logger.Fields{
				"tx_id":        txID,
				"poll_count":   transfer.PollCount,
				"final_amount": transfer.Amount,
			})
		}
	}

	logger.Info("Off-ramp status polled", logger.Fields{
		"tx_id":      txID,
		"status":     transfer.Status,
		"poll_count": transfer.PollCount,
	})

	// Return a copy
	return &Transfer{
		TxID:             transfer.TxID,
		Status:           transfer.Status,
		Amount:           transfer.Amount,
		Currency:         transfer.Currency,
		StablecoinAmount: transfer.StablecoinAmount,
		CreatedAt:        transfer.CreatedAt,
		SettledAt:        transfer.SettledAt,
		PollCount:        transfer.PollCount,
		SettlesAfterPoll: transfer.SettlesAfterPoll,
	}, nil
}
