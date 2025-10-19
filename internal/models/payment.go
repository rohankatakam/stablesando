package models

import "time"

// PaymentStatus represents the current state of a payment
type PaymentStatus string

const (
	StatusPending    PaymentStatus = "PENDING"
	StatusProcessing PaymentStatus = "PROCESSING"
	StatusCompleted  PaymentStatus = "COMPLETED"
	StatusFailed     PaymentStatus = "FAILED"
)

// Payment represents a payment record in the system
type Payment struct {
	PaymentID          string        `json:"payment_id" dynamodbav:"payment_id"`
	IdempotencyKey     string        `json:"idempotency_key" dynamodbav:"idempotency_key"`
	Amount             int64         `json:"amount" dynamodbav:"amount"`
	Currency           string        `json:"currency" dynamodbav:"currency"`
	SourceAccount      string        `json:"source_account" dynamodbav:"source_account"`
	DestinationAccount string        `json:"destination_account" dynamodbav:"destination_account"`
	Status             PaymentStatus `json:"status" dynamodbav:"status"`
	FeeAmount          int64         `json:"fee_amount" dynamodbav:"fee_amount"`
	FeeCurrency        string        `json:"fee_currency" dynamodbav:"fee_currency"`
	OnRampTxID         string        `json:"on_ramp_tx_id,omitempty" dynamodbav:"on_ramp_tx_id,omitempty"`
	OffRampTxID        string        `json:"off_ramp_tx_id,omitempty" dynamodbav:"off_ramp_tx_id,omitempty"`
	ErrorMessage       string        `json:"error_message,omitempty" dynamodbav:"error_message,omitempty"`
	CreatedAt          time.Time     `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at" dynamodbav:"updated_at"`
	ProcessedAt        *time.Time    `json:"processed_at,omitempty" dynamodbav:"processed_at,omitempty"`
}

// PaymentRequest represents the incoming API request
type PaymentRequest struct {
	Amount             int64  `json:"amount"`
	Currency           string `json:"currency"`
	SourceAccount      string `json:"source_account"`
	DestinationAccount string `json:"destination_account"`
}

// PaymentResponse represents the API response
type PaymentResponse struct {
	PaymentID string        `json:"payment_id"`
	Status    PaymentStatus `json:"status"`
	Message   string        `json:"message"`
}

// PaymentJob represents a message in the SQS queue
type PaymentJob struct {
	PaymentID          string `json:"payment_id"`
	Amount             int64  `json:"amount"`
	Currency           string `json:"currency"`
	SourceAccount      string `json:"source_account"`
	DestinationAccount string `json:"destination_account"`
}

// WebhookEvent represents a webhook notification payload
type WebhookEvent struct {
	EventType   string         `json:"event_type"`
	PaymentID   string         `json:"payment_id"`
	Status      PaymentStatus  `json:"status"`
	Amount      int64          `json:"amount"`
	Currency    string         `json:"currency"`
	Fees        *FeeBreakdown  `json:"fees,omitempty"`
	OnRampTxID  string         `json:"on_ramp_tx_id,omitempty"`
	OffRampTxID string         `json:"off_ramp_tx_id,omitempty"`
	Error       string         `json:"error,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
}

// FeeBreakdown represents fee information in webhooks and responses
type FeeBreakdown struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}
