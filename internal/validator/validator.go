package validator

import (
	"fmt"
	"strings"

	"github.com/yourusername/crypto-conversion/internal/errors"
	"github.com/yourusername/crypto-conversion/internal/models"
)

// Supported currencies
var supportedCurrencies = map[string]bool{
	"USD": true,
	"EUR": true,
	"GBP": true,
	"JPY": true,
	"AUD": true,
	"CAD": true,
}

// ValidatePaymentRequest validates a payment request
func ValidatePaymentRequest(req *models.PaymentRequest) error {
	// Validate amount
	if req.Amount <= 0 {
		return errors.ErrValidation("amount", "must be greater than 0")
	}

	// Maximum amount check (e.g., 1 million in smallest unit)
	if req.Amount > 1000000000 {
		return errors.ErrValidation("amount", "exceeds maximum allowed amount")
	}

	// Validate currency
	if req.Currency == "" {
		return errors.ErrValidation("currency", "is required")
	}

	currency := strings.ToUpper(req.Currency)
	if !supportedCurrencies[currency] {
		return errors.ErrValidation("currency", fmt.Sprintf("'%s' is not supported", req.Currency))
	}

	// Validate source account
	if req.SourceAccount == "" {
		return errors.ErrValidation("source_account", "is required")
	}

	if len(req.SourceAccount) < 3 || len(req.SourceAccount) > 100 {
		return errors.ErrValidation("source_account", "must be between 3 and 100 characters")
	}

	// Validate destination account
	if req.DestinationAccount == "" {
		return errors.ErrValidation("destination_account", "is required")
	}

	if len(req.DestinationAccount) < 3 || len(req.DestinationAccount) > 100 {
		return errors.ErrValidation("destination_account", "must be between 3 and 100 characters")
	}

	// Ensure source and destination are different
	if req.SourceAccount == req.DestinationAccount {
		return errors.ErrValidation("destination_account", "must be different from source_account")
	}

	return nil
}

// ValidateIdempotencyKey validates an idempotency key
func ValidateIdempotencyKey(key string) error {
	if key == "" {
		return errors.ErrMissingHeader("Idempotency-Key")
	}

	if len(key) < 10 || len(key) > 255 {
		return errors.ErrValidation("Idempotency-Key", "must be between 10 and 255 characters")
	}

	// Only allow alphanumeric, hyphens, and underscores
	for _, c := range key {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return errors.ErrValidation("Idempotency-Key", "must contain only alphanumeric characters, hyphens, and underscores")
		}
	}

	return nil
}

// IsSupportedCurrency checks if a currency is supported
func IsSupportedCurrency(currency string) bool {
	return supportedCurrencies[strings.ToUpper(currency)]
}

// GetSupportedCurrencies returns a list of supported currencies
func GetSupportedCurrencies() []string {
	currencies := make([]string, 0, len(supportedCurrencies))
	for currency := range supportedCurrencies {
		currencies = append(currencies, currency)
	}
	return currencies
}
