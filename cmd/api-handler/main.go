package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	"crypto-conversion/internal/config"
	"crypto-conversion/internal/database"
	"crypto-conversion/internal/errors"
	"crypto-conversion/internal/fees"
	"crypto-conversion/internal/logger"
	"crypto-conversion/internal/models"
	"crypto-conversion/internal/queue"
	"crypto-conversion/internal/quotes"
	"crypto-conversion/internal/validator"
)

// Handler manages the API Lambda dependencies
type Handler struct {
	db        *database.Client
	quoteDB   *database.QuoteClient
	queue     *queue.Client
	feeCalc   *fees.Calculator
	quoteCalc *quotes.Calculator
	cfg       *config.Config
}

// NewHandler creates a new API handler
func NewHandler(cfg *config.Config) (*Handler, error) {
	// Initialize database client
	db, err := database.NewClient(cfg.AWS.Region, cfg.Database.TableName, cfg.Database.Endpoint)
	if err != nil {
		return nil, err
	}

	// Initialize quote database client
	quoteDB, err := database.NewQuoteClient(cfg.AWS.Region, cfg.Database.QuoteTableName, cfg.Database.Endpoint)
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

	// Initialize quote calculator
	quoteCalc := quotes.NewCalculator(feeCalc)

	return &Handler{
		db:        db,
		quoteDB:   quoteDB,
		queue:     q,
		feeCalc:   feeCalc,
		quoteCalc: quoteCalc,
		cfg:       cfg,
	}, nil
}

// HandleRequest handles the API Gateway request
func (h *Handler) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.Info("Received API request", logger.Fields{
		"path":   request.Path,
		"method": request.HTTPMethod,
	})

	// Route to appropriate handler
	if request.HTTPMethod == http.MethodPost && request.Path == "/quotes" {
		return h.handleCreateQuote(ctx, request)
	}

	if request.HTTPMethod == http.MethodPost && request.Path == "/payments" {
		return h.handleCreatePayment(ctx, request)
	}

	// Handle GET /payments/{payment_id}
	if request.HTTPMethod == http.MethodGet && len(request.PathParameters) > 0 {
		if paymentID, ok := request.PathParameters["payment_id"]; ok {
			return h.handleGetPayment(ctx, paymentID)
		}
	}

	return errorResponse(http.StatusNotFound, "NOT_FOUND", "Endpoint not found")
}

// handleCreateQuote handles POST /quotes
func (h *Handler) handleCreateQuote(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse request body
	var quoteReq quotes.QuoteRequest
	if err := json.Unmarshal([]byte(request.Body), &quoteReq); err != nil {
		logger.Error("Failed to parse quote request body", logger.Fields{"error": err.Error()})
		return errorResponse(http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
	}

	// Generate quote
	quote, err := h.quoteCalc.GenerateQuote(&quoteReq)
	if err != nil {
		logger.Warn("Quote generation failed", logger.Fields{"error": err.Error()})
		return errorResponse(http.StatusBadRequest, "QUOTE_ERROR", err.Error())
	}

	// Store quote in database
	if err := h.quoteDB.CreateQuote(ctx, quote); err != nil {
		logger.Error("Failed to store quote", logger.Fields{
			"error":    err.Error(),
			"quote_id": quote.QuoteID,
		})
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create quote")
	}

	// Return quote response
	responseBody, _ := json.Marshal(quote.ToResponse())

	logger.Info("Quote created successfully", logger.Fields{
		"quote_id":          quote.QuoteID,
		"amount":            quote.Amount,
		"guaranteed_payout": quote.GuaranteedPayout,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST,OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
		},
		Body: string(responseBody),
	}, nil
}

// handleCreatePayment handles POST /payments
func (h *Handler) handleCreatePayment(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

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

	// Check if quote_id is provided and validate it
	var guaranteedPayout int64
	if paymentReq.QuoteID != "" {
		quote, err := h.quoteDB.GetQuote(ctx, paymentReq.QuoteID)
		if err != nil {
			logger.Error("Failed to fetch quote", logger.Fields{
				"error":    err.Error(),
				"quote_id": paymentReq.QuoteID,
			})
			return errorResponse(http.StatusBadRequest, "INVALID_QUOTE", "Quote not found or expired")
		}

		// Validate quote hasn't expired
		if time.Now().After(quote.ExpiresAt) {
			logger.Warn("Quote expired", logger.Fields{
				"quote_id":   paymentReq.QuoteID,
				"expires_at": quote.ExpiresAt,
			})
			return errorResponse(http.StatusBadRequest, "QUOTE_EXPIRED", "Quote has expired")
		}

		// Validate amount matches quote
		if quote.Amount != paymentReq.Amount {
			logger.Warn("Amount mismatch with quote", logger.Fields{
				"quote_id":       paymentReq.QuoteID,
				"quote_amount":   quote.Amount,
				"payment_amount": paymentReq.Amount,
			})
			return errorResponse(http.StatusBadRequest, "AMOUNT_MISMATCH", "Payment amount does not match quote")
		}

		guaranteedPayout = quote.GuaranteedPayout
		logger.Info("Using quote for payment", logger.Fields{
			"quote_id":          paymentReq.QuoteID,
			"guaranteed_payout": guaranteedPayout,
		})
	}

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
		PaymentID:              paymentID,
		IdempotencyKey:         idempotencyKey,
		Amount:                 paymentReq.Amount,
		Currency:               paymentReq.Currency,
		SourceAccount:          paymentReq.SourceAccount,
		DestinationAccount:     paymentReq.DestinationAccount,
		Status:                 models.StatusPending,
		FeeAmount:              feeResult.FeeAmount,
		FeeCurrency:            feeResult.FeeCurrency,
		QuoteID:                paymentReq.QuoteID,
		GuaranteedPayoutAmount: guaranteedPayout,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
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
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST,OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,Idempotency-Key",
		},
		Body: string(responseBody),
	}, nil
}

// handleGetPayment handles GET /payments/{payment_id}
func (h *Handler) handleGetPayment(ctx context.Context, paymentID string) (events.APIGatewayProxyResponse, error) {
	logger.Info("Fetching payment", logger.Fields{"payment_id": paymentID})

	// Get payment from database
	payment, err := h.db.GetPaymentByID(ctx, paymentID)
	if err != nil {
		logger.Error("Failed to fetch payment", logger.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		})
		return errorResponse(http.StatusNotFound, "PAYMENT_NOT_FOUND", "Payment not found")
	}

	// Marshal payment to JSON
	responseBody, err := json.Marshal(payment)
	if err != nil {
		logger.Error("Failed to marshal payment response", logger.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		})
		return errorResponse(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process payment data")
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET,OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token",
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
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET,POST,OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,Idempotency-Key",
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
