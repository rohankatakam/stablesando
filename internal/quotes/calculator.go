package quotes

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"crypto-conversion/internal/fees"
	"crypto-conversion/internal/logger"
)

// Calculator handles quote generation and exchange rate fetching
type Calculator struct {
	feeCalc *fees.Calculator
}

// NewCalculator creates a new quote calculator
func NewCalculator(feeCalc *fees.Calculator) *Calculator {
	return &Calculator{
		feeCalc: feeCalc,
	}
}

// GenerateQuote creates a new quote with locked-in rates and fees
func (c *Calculator) GenerateQuote(req *QuoteRequest) (*Quote, error) {
	// Validate currencies (MVP: only USD -> EUR)
	if req.FromCurrency != "USD" {
		return nil, fmt.Errorf("only USD source currency supported in MVP")
	}
	if req.ToCurrency != "EUR" {
		return nil, fmt.Errorf("only EUR destination currency supported in MVP")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	// Generate quote ID
	quoteID := fmt.Sprintf("quote_%s", uuid.New().String())

	// Fetch exchange rate (mock - simulates checking multiple providers)
	exchangeRate, providerName := c.fetchBestExchangeRate(req.FromCurrency, req.ToCurrency, req.Amount)

	// Calculate platform fee
	feeResult := c.feeCalc.CalculateFee(req.Amount, req.ToCurrency)
	platformFee := feeResult.FeeAmount

	// Estimate onramp fee (mock - would come from provider APIs)
	onrampFee := c.estimateOnrampFee(req.Amount)

	// Estimate offramp fee (mock - would come from provider APIs)
	offrampFee := c.estimateOfframpFee(req.Amount)

	// Calculate total fees
	totalFees := platformFee + onrampFee + offrampFee

	// Calculate guaranteed payout
	// Amount after fees, converted at locked rate
	amountAfterFees := req.Amount - totalFees
	guaranteedPayout := int64(float64(amountAfterFees) * exchangeRate)

	// Quote valid for 60 seconds
	validForSeconds := 60
	createdAt := time.Now()
	expiresAt := createdAt.Add(time.Duration(validForSeconds) * time.Second)

	quote := &Quote{
		QuoteID:          quoteID,
		FromCurrency:     req.FromCurrency,
		ToCurrency:       req.ToCurrency,
		Amount:           req.Amount,
		ExchangeRate:     exchangeRate,
		PlatformFee:      platformFee,
		OnrampFee:        onrampFee,
		OfframpFee:       offrampFee,
		TotalFees:        totalFees,
		GuaranteedPayout: guaranteedPayout,
		PayoutCurrency:   req.ToCurrency,
		CreatedAt:        createdAt,
		ExpiresAt:        expiresAt,
		ValidForSeconds:  validForSeconds,
		ProviderRate:     providerName,
		TTL:              expiresAt.Unix(), // DynamoDB will auto-delete after expiration
	}

	logger.Info("Quote generated", logger.Fields{
		"quote_id":          quoteID,
		"amount":            req.Amount,
		"exchange_rate":     exchangeRate,
		"total_fees":        totalFees,
		"guaranteed_payout": guaranteedPayout,
		"provider":          providerName,
		"expires_at":        expiresAt.Format(time.RFC3339),
	})

	return quote, nil
}

// fetchBestExchangeRate simulates fetching rates from multiple providers
// In production, this would query Circle, Bridge, Coinbase APIs
func (c *Calculator) fetchBestExchangeRate(from, to string, amount int64) (float64, string) {
	// Mock: Simulate checking 3 providers
	providers := []struct {
		name string
		rate float64
	}{
		{"Circle", 0.9200 + (rand.Float64()-0.5)*0.005},  // 0.9175 - 0.9225
		{"Bridge", 0.9195 + (rand.Float64()-0.5)*0.005},  // 0.9170 - 0.9220
		{"Coinbase", 0.9190 + (rand.Float64()-0.5)*0.005}, // 0.9165 - 0.9215
	}

	// Find best rate (highest for USD -> EUR)
	bestProvider := providers[0]
	for _, p := range providers {
		if p.rate > bestProvider.rate {
			bestProvider = p
		}
	}

	logger.Info("Exchange rate fetched", logger.Fields{
		"from":     from,
		"to":       to,
		"rate":     bestProvider.rate,
		"provider": bestProvider.name,
	})

	return bestProvider.rate, bestProvider.name
}

// estimateOnrampFee calculates estimated onramp provider fee
// In production, would call provider quote APIs
func (c *Calculator) estimateOnrampFee(amount int64) int64 {
	// Mock: Onramp typically charges ~1% + fixed fee
	percentageFee := int64(float64(amount) * 0.01) // 1%
	fixedFee := int64(50)                          // $0.50
	return percentageFee + fixedFee
}

// estimateOfframpFee calculates estimated offramp provider fee
// In production, would call provider quote APIs
func (c *Calculator) estimateOfframpFee(amount int64) int64 {
	// Mock: Offramp typically charges ~1.5% + fixed fee
	percentageFee := int64(float64(amount) * 0.015) // 1.5%
	fixedFee := int64(75)                           // $0.75
	return percentageFee + fixedFee
}

// ToResponse converts a Quote to a QuoteResponse for API
func (q *Quote) ToResponse() *QuoteResponse {
	return &QuoteResponse{
		QuoteID:      q.QuoteID,
		Amount:       q.Amount,
		Currency:     q.FromCurrency,
		ExchangeRate: q.ExchangeRate,
		Fees: FeeDetail{
			PlatformFee: q.PlatformFee,
			OnrampFee:   q.OnrampFee,
			OfframpFee:  q.OfframpFee,
			TotalFees:   q.TotalFees,
			Currency:    "USD", // MVP: all fees in USD
		},
		GuaranteedPayout: q.GuaranteedPayout,
		PayoutCurrency:   q.PayoutCurrency,
		ExpiresAt:        q.ExpiresAt,
		ValidForSeconds:  q.ValidForSeconds,
	}
}
