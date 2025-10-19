package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP status code
type AppError struct {
	Code       string // Machine-readable error code
	Message    string // Human-readable error message
	StatusCode int    // HTTP status code
	Err        error  // Underlying error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (underlying: %v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(code, message string, statusCode int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// Common error constructors

// ErrInvalidRequest creates an invalid request error
func ErrInvalidRequest(message string, err error) *AppError {
	return &AppError{
		Code:       "INVALID_REQUEST",
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Err:        err,
	}
}

// ErrDuplicateRequest creates a duplicate request error
func ErrDuplicateRequest(idempotencyKey string) *AppError {
	return &AppError{
		Code:       "DUPLICATE_REQUEST",
		Message:    fmt.Sprintf("Request with idempotency key '%s' already exists", idempotencyKey),
		StatusCode: http.StatusConflict,
		Err:        nil,
	}
}

// ErrPaymentNotFound creates a payment not found error
func ErrPaymentNotFound(paymentID string) *AppError {
	return &AppError{
		Code:       "PAYMENT_NOT_FOUND",
		Message:    fmt.Sprintf("Payment '%s' not found", paymentID),
		StatusCode: http.StatusNotFound,
		Err:        nil,
	}
}

// ErrInternalServer creates an internal server error
func ErrInternalServer(message string, err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// ErrDatabaseOperation creates a database operation error
func ErrDatabaseOperation(operation string, err error) *AppError {
	return &AppError{
		Code:       "DATABASE_ERROR",
		Message:    fmt.Sprintf("Database operation '%s' failed", operation),
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// ErrQueueOperation creates a queue operation error
func ErrQueueOperation(operation string, err error) *AppError {
	return &AppError{
		Code:       "QUEUE_ERROR",
		Message:    fmt.Sprintf("Queue operation '%s' failed", operation),
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// ErrPaymentProcessing creates a payment processing error
func ErrPaymentProcessing(message string, err error) *AppError {
	return &AppError{
		Code:       "PAYMENT_PROCESSING_ERROR",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// ErrValidation creates a validation error
func ErrValidation(field, reason string) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    fmt.Sprintf("Validation failed for field '%s': %s", field, reason),
		StatusCode: http.StatusBadRequest,
		Err:        nil,
	}
}

// ErrMissingHeader creates a missing header error
func ErrMissingHeader(headerName string) *AppError {
	return &AppError{
		Code:       "MISSING_HEADER",
		Message:    fmt.Sprintf("Required header '%s' is missing", headerName),
		StatusCode: http.StatusBadRequest,
		Err:        nil,
	}
}

// ErrQuoteNotFound creates a quote not found error
func ErrQuoteNotFound(quoteID string) *AppError {
	return &AppError{
		Code:       "QUOTE_NOT_FOUND",
		Message:    fmt.Sprintf("Quote '%s' not found or expired", quoteID),
		StatusCode: http.StatusNotFound,
		Err:        nil,
	}
}

// ErrQuoteExpired creates a quote expired error
func ErrQuoteExpired(quoteID string) *AppError {
	return &AppError{
		Code:       "QUOTE_EXPIRED",
		Message:    fmt.Sprintf("Quote '%s' has expired", quoteID),
		StatusCode: http.StatusBadRequest,
		Err:        nil,
	}
}

// ErrorResponse represents an error response structure
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details for API responses
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ToErrorResponse converts an AppError to an ErrorResponse
func ToErrorResponse(err *AppError) ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Code:    err.Code,
			Message: err.Message,
		},
	}
}
