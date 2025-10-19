package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"crypto-conversion/internal/models"
	"crypto-conversion/internal/validator"
)

func TestValidatePaymentRequest(t *testing.T) {
	tests := []struct {
		name    string
		request *models.PaymentRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: &models.PaymentRequest{
				Amount:             100000,
				Currency:           "EUR",
				SourceAccount:      "user123",
				DestinationAccount: "merchant456",
			},
			wantErr: false,
		},
		{
			name: "zero amount",
			request: &models.PaymentRequest{
				Amount:             0,
				Currency:           "EUR",
				SourceAccount:      "user123",
				DestinationAccount: "merchant456",
			},
			wantErr: true,
			errMsg:  "amount",
		},
		{
			name: "negative amount",
			request: &models.PaymentRequest{
				Amount:             -1000,
				Currency:           "EUR",
				SourceAccount:      "user123",
				DestinationAccount: "merchant456",
			},
			wantErr: true,
			errMsg:  "amount",
		},
		{
			name: "amount too large",
			request: &models.PaymentRequest{
				Amount:             2000000000,
				Currency:           "EUR",
				SourceAccount:      "user123",
				DestinationAccount: "merchant456",
			},
			wantErr: true,
			errMsg:  "amount",
		},
		{
			name: "empty currency",
			request: &models.PaymentRequest{
				Amount:             100000,
				Currency:           "",
				SourceAccount:      "user123",
				DestinationAccount: "merchant456",
			},
			wantErr: true,
			errMsg:  "currency",
		},
		{
			name: "unsupported currency",
			request: &models.PaymentRequest{
				Amount:             100000,
				Currency:           "XXX",
				SourceAccount:      "user123",
				DestinationAccount: "merchant456",
			},
			wantErr: true,
			errMsg:  "currency",
		},
		{
			name: "empty source account",
			request: &models.PaymentRequest{
				Amount:             100000,
				Currency:           "EUR",
				SourceAccount:      "",
				DestinationAccount: "merchant456",
			},
			wantErr: true,
			errMsg:  "source_account",
		},
		{
			name: "source account too short",
			request: &models.PaymentRequest{
				Amount:             100000,
				Currency:           "EUR",
				SourceAccount:      "ab",
				DestinationAccount: "merchant456",
			},
			wantErr: true,
			errMsg:  "source_account",
		},
		{
			name: "empty destination account",
			request: &models.PaymentRequest{
				Amount:             100000,
				Currency:           "EUR",
				SourceAccount:      "user123",
				DestinationAccount: "",
			},
			wantErr: true,
			errMsg:  "destination_account",
		},
		{
			name: "same source and destination",
			request: &models.PaymentRequest{
				Amount:             100000,
				Currency:           "EUR",
				SourceAccount:      "user123",
				DestinationAccount: "user123",
			},
			wantErr: true,
			errMsg:  "destination_account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePaymentRequest(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIdempotencyKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			key:     "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "valid alphanumeric with hyphens",
			key:     "payment-abc123-xyz789",
			wantErr: false,
		},
		{
			name:    "valid with underscores",
			key:     "payment_abc123_xyz789",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "too short",
			key:     "abc123",
			wantErr: true,
		},
		{
			name:    "contains special characters",
			key:     "payment@abc123",
			wantErr: true,
		},
		{
			name:    "contains spaces",
			key:     "payment abc123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateIdempotencyKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsSupportedCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		want     bool
	}{
		{"USD", "USD", true},
		{"EUR", "EUR", true},
		{"GBP", "GBP", true},
		{"lowercase eur", "eur", true},
		{"unsupported", "XXX", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsSupportedCurrency(tt.currency)
			assert.Equal(t, tt.want, result)
		})
	}
}
