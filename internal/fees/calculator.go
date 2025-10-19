package fees

import (
	"fmt"

	"crypto-conversion/internal/logger"
)

// Calculator handles fee calculations for cross-border payments
type Calculator struct {
	// Configuration could be injected here for different fee tiers
}

// FeeResult contains the calculated fee information
type FeeResult struct {
	FeeAmount    int64   `json:"fee_amount"`    // Fee in cents (same currency as input)
	FeeCurrency  string  `json:"fee_currency"`  // Currency of the fee (USD for MVP)
	FeeRate      float64 `json:"fee_rate"`      // Effective percentage rate used
	FixedFee     int64   `json:"fixed_fee"`     // Fixed portion of fee in cents
	BaseAmount   int64   `json:"base_amount"`   // Original amount before fees
	TotalAmount  int64   `json:"total_amount"`  // Base amount + fees
}

// NewCalculator creates a new fee calculator
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CalculateFee calculates the fee for a payment based on amount and destination currency
//
// Fee Structure (USD amounts):
//   - Amount < $100:      2.9% + $0.30
//   - Amount < $1,000:    2.5% + $0.50
//   - Amount >= $1,000:   2.0% + $1.00
//
// Parameters:
//   - amount: Payment amount in cents
//   - currency: Destination currency (affects fee tier, EUR for MVP)
//
// Returns:
//   - FeeResult with calculated fees
func (c *Calculator) CalculateFee(amount int64, currency string) *FeeResult {
	var percentageRate float64
	var fixedFee int64

	// Determine fee tier based on amount
	// All amounts are in cents (USD cents for MVP)
	switch {
	case amount < 10000: // Less than $100
		percentageRate = 0.029 // 2.9%
		fixedFee = 30          // $0.30 in cents

	case amount < 100000: // Less than $1,000
		percentageRate = 0.025 // 2.5%
		fixedFee = 50          // $0.50 in cents

	default: // $1,000 or more
		percentageRate = 0.020 // 2.0%
		fixedFee = 100         // $1.00 in cents
	}

	// Calculate percentage-based fee
	percentageFee := int64(float64(amount) * percentageRate)

	// Total fee = percentage fee + fixed fee
	totalFee := percentageFee + fixedFee

	result := &FeeResult{
		FeeAmount:   totalFee,
		FeeCurrency: "USD", // All fees in USD for MVP
		FeeRate:     percentageRate,
		FixedFee:    fixedFee,
		BaseAmount:  amount,
		TotalAmount: amount + totalFee,
	}

	logger.Info("Fee calculated", logger.Fields{
		"base_amount":    amount,
		"currency":       currency,
		"fee_amount":     totalFee,
		"fee_rate":       fmt.Sprintf("%.1f%%", percentageRate*100),
		"fixed_fee":      fixedFee,
		"total_amount":   result.TotalAmount,
	})

	return result
}

// CalculateFeeForCurrency is a convenience wrapper that logs currency-specific info
// In production, this could apply different fees based on destination country/currency
func (c *Calculator) CalculateFeeForCurrency(amount int64, currency string) *FeeResult {
	// For MVP, we use the same fee structure regardless of destination currency
	// In production, you might have:
	// - Different fees for different corridors (USD->EUR vs USD->GBP)
	// - Country-specific regulatory fees
	// - Currency conversion spreads

	result := c.CalculateFee(amount, currency)

	logger.Info("Currency-specific fee calculation", logger.Fields{
		"destination_currency": currency,
		"fee_amount":          result.FeeAmount,
		"effective_rate":      fmt.Sprintf("%.2f%%", (float64(result.FeeAmount)/float64(amount))*100),
	})

	return result
}

// FormatFeeForDisplay returns a human-readable fee string
func (r *FeeResult) FormatFeeForDisplay() string {
	dollars := float64(r.FeeAmount) / 100.0
	return fmt.Sprintf("$%.2f (%d%% + $%.2f)",
		dollars,
		int(r.FeeRate*100),
		float64(r.FixedFee)/100.0)
}

// GetEffectiveRate returns the actual fee rate as a percentage of the base amount
func (r *FeeResult) GetEffectiveRate() float64 {
	if r.BaseAmount == 0 {
		return 0
	}
	return (float64(r.FeeAmount) / float64(r.BaseAmount)) * 100
}
