package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"github.com/yourusername/crypto-conversion/internal/config"
	"github.com/yourusername/crypto-conversion/internal/database"
	"github.com/yourusername/crypto-conversion/internal/errors"
	"github.com/yourusername/crypto-conversion/internal/fees"
	"github.com/yourusername/crypto-conversion/internal/logger"
	"github.com/yourusername/crypto-conversion/internal/models"
	"github.com/yourusername/crypto-conversion/internal/queue"
	"github.com/yourusername/crypto-conversion/internal/validator"
)

// Handler manages the API Lambda dependencies
type Handler struct {
	db      *database.Client
	queue   *queue.Client
	feeCalc *fees.Calculator
	cfg     *config.Config
}

// NewHandler creates a new API handler
func NewHandler(cfg *config.Config) (*Handler, error) {
	// Initialize database client
	db, err := database.NewClient(cfg.AWS.Region, cfg.Database.TableName, cfg.Database.Endpoint)
	if err != nil {
		return nil, err
	}

	// Initialize queue client
	q, err := queue.NewClient(cfg.AWS.Region, cfg.Queue.Endpoint)
	if err != nil {
		return nil, err
	}

	// Initialize fee calculator
	feeCalc := fees.NewCalculator()

	return &Handler{
		db:      db,
		queue:   q,
		feeCalc: feeCalc,
		cfg:     cfg,
	}, nil
}

// HandleRequest handles the API Gateway request
func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.Info("Received payment request", logger.Fields{
		"path":   request.Path,
		"method": request.HTTPMethod,
	})

	// Only handle POST /payments
	if request.HTTPMethod != http.MethodPost || request.Path != "/payments" {
		return errorResponse(http.StatusNotFound, "NOT_FOUND", "Endpoint not found")
	}

	// Extract idempotency key from headers
	idempotencyKey := request.Headers["Idempotency-Key"]
	if idempotencyKey == "" {
		// Try lowercase header name (API Gateway can normalize headers)
		idempotencyKey = request.Headers["idempotency-key"]
	}

	// Validate idempotency key
	if err := validator.ValidateIdempotencyKey(idempotencyKey); err != nil {
		appErr := err.(*errors.AppError)
		return errorResponse(appErr.StatusCode, appErr.Code, appErr.Message)
	}

	// Check if payment with this idempotency key already exists
	existingPayment, err := h.db.GetPaymentByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		logger.Error("Failed to check idempotency key", logger.Fields{
			"error":           err.Error(),
			"idempotency_key": idempotencyKey,
		})
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process request")
	}

	if existingPayment != nil {
		logger.Warn("Duplicate idempotency key", logger.Fields{
			"idempotency_key": idempotencyKey,
			"payment_id":      existingPayment.PaymentID,
		})
		return errorResponse(http.StatusConflict, "DUPLICATE_REQUEST",
			"A payment with this idempotency key already exists")
	}

	// Parse request body
	var paymentReq models.PaymentRequest
	if err := json.Unmarshal([]byte(request.Body), &paymentReq); err != nil {
		logger.Error("Failed to parse request body", logger.Fields{"error": err.Error()})
		return errorResponse(http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	// Validate payment request
	if err := validator.ValidatePaymentRequest(&paymentReq); err != nil {
		appErr := err.(*errors.AppError)
		logger.Warn("Validation failed", logger.Fields{
			"error": appErr.Message,
		})
		return errorResponse(appErr.StatusCode, appErr.Code, appErr.Message)
	}

	// Generate payment ID
	paymentID := uuid.New().String()

	// Calculate fees
	feeResult := h.feeCalc.CalculateFeeForCurrency(paymentReq.Amount, paymentReq.Currency)

	logger.Info("Fee calculated for payment", logger.Fields{
		"payment_id":   paymentID,
		"base_amount":  paymentReq.Amount,
		"fee_amount":   feeResult.FeeAmount,
		"total_amount": feeResult.TotalAmount,
	})

	// Create payment record
	payment := &models.Payment{
		PaymentID:          paymentID,
		IdempotencyKey:     idempotencyKey,
		Amount:             paymentReq.Amount,
		Currency:           paymentReq.Currency,
		SourceAccount:      paymentReq.SourceAccount,
		DestinationAccount: paymentReq.DestinationAccount,
		Status:             models.StatusPending,
		FeeAmount:          feeResult.FeeAmount,
		FeeCurrency:        feeResult.FeeCurrency,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Save to database
	if err := h.db.CreatePayment(ctx, payment); err != nil {
		logger.Error("Failed to create payment", logger.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		})
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create payment")
	}

	// Create payment job
	job := &models.PaymentJob{
		PaymentID:          paymentID,
		Amount:             paymentReq.Amount,
		Currency:           paymentReq.Currency,
		SourceAccount:      paymentReq.SourceAccount,
		DestinationAccount: paymentReq.DestinationAccount,
	}

	// Send job to queue
	if err := h.queue.SendPaymentJob(ctx, h.cfg.Queue.PaymentQueueURL, job); err != nil {
		logger.Error("Failed to enqueue payment job", logger.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		})
		// Payment is created but not queued - this is a critical error
		// In production, you might want to implement a retry mechanism or dead letter queue
		return errorResponse(http.StatusInternalServerError, "QUEUE_ERROR", "Failed to process payment")
	}

	// Return 202 Accepted response
	response := models.PaymentResponse{
		PaymentID: paymentID,
		Status:    models.StatusPending,
		Message:   "Payment accepted for processing",
	}

	responseBody, _ := json.Marshal(response)

	logger.Info("Payment accepted", logger.Fields{
		"payment_id":      paymentID,
		"idempotency_key": idempotencyKey,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(responseBody),
	}, nil
}

// errorResponse creates an error response
func errorResponse(statusCode int, code, message string) (events.APIGatewayProxyResponse, error) {
	errResp := errors.ErrorResponse{
		Error: errors.ErrorDetail{
			Code:    code,
			Message: message,
		},
	}

	body, _ := json.Marshal(errResp)

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", logger.Fields{"error": err.Error()})
		panic(err)
	}

	// Initialize logger
	log := logger.NewFromString(cfg.Logging.Level)
	logger.SetDefault(log)

	// Create handler
	handler, err := NewHandler(cfg)
	if err != nil {
		logger.Error("Failed to create handler", logger.Fields{"error": err.Error()})
		panic(err)
	}

	// Start Lambda
	lambda.Start(handler.HandleRequest)
}
