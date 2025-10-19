package quotes

import "time"

// Quote represents a locked-in exchange rate and fee quote
type Quote struct {
	QuoteID              string    `json:"quote_id" dynamodbav:"quote_id"`
	FromCurrency         string    `json:"from_currency" dynamodbav:"from_currency"`
	ToCurrency           string    `json:"to_currency" dynamodbav:"to_currency"`
	Amount               int64     `json:"amount" dynamodbav:"amount"`                   // Amount in cents
	ExchangeRate         float64   `json:"exchange_rate" dynamodbav:"exchange_rate"`     // e.g., 0.92 for USD to EUR
	PlatformFee          int64     `json:"platform_fee" dynamodbav:"platform_fee"`       // Platform fee in cents
	OnrampFee            int64     `json:"onramp_fee" dynamodbav:"onramp_fee"`           // Estimated onramp fee
	OfframpFee           int64     `json:"offramp_fee" dynamodbav:"offramp_fee"`         // Estimated offramp fee
	TotalFees            int64     `json:"total_fees" dynamodbav:"total_fees"`           // Sum of all fees
	GuaranteedPayout     int64     `json:"guaranteed_payout" dynamodbav:"guaranteed_payout"` // Final amount recipient gets
	PayoutCurrency       string    `json:"payout_currency" dynamodbav:"payout_currency"` // Same as ToCurrency
	CreatedAt            time.Time `json:"created_at" dynamodbav:"created_at"`
	ExpiresAt            time.Time `json:"expires_at" dynamodbav:"expires_at"`
	ValidForSeconds      int       `json:"valid_for_seconds" dynamodbav:"valid_for_seconds"`
	ProviderRate         string    `json:"provider_rate,omitempty" dynamodbav:"provider_rate,omitempty"` // Which provider gave best rate
	TTL                  int64     `json:"-" dynamodbav:"ttl"` // DynamoDB TTL attribute (unix timestamp)
}

// QuoteRequest represents a request for a payment quote
type QuoteRequest struct {
	FromCurrency string `json:"from_currency"`
	ToCurrency   string `json:"to_currency"`
	Amount       int64  `json:"amount"` // Amount in cents
}

// QuoteResponse represents the API response for a quote
type QuoteResponse struct {
	QuoteID          string    `json:"quote_id"`
	Amount           int64     `json:"amount"`
	Currency         string    `json:"currency"` // From currency
	ExchangeRate     float64   `json:"exchange_rate"`
	Fees             FeeDetail `json:"fees"`
	GuaranteedPayout int64     `json:"guaranteed_payout"`
	PayoutCurrency   string    `json:"payout_currency"`
	ExpiresAt        time.Time `json:"expires_at"`
	ValidForSeconds  int       `json:"valid_for_seconds"`
}

// FeeDetail breaks down the fee structure
type FeeDetail struct {
	PlatformFee int64  `json:"platform_fee"`
	OnrampFee   int64  `json:"onramp_fee"`
	OfframpFee  int64  `json:"offramp_fee"`
	TotalFees   int64  `json:"total_fees"`
	Currency    string `json:"currency"` // USD for MVP
}
