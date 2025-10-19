package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yourusername/crypto-conversion/internal/config"
	"github.com/yourusername/crypto-conversion/internal/database"
	"github.com/yourusername/crypto-conversion/internal/logger"
	"github.com/yourusername/crypto-conversion/internal/models"
	"github.com/yourusername/crypto-conversion/internal/payment"
	"github.com/yourusername/crypto-conversion/internal/queue"
)

// Handler manages the Worker Lambda dependencies
type Handler struct {
	db           *database.Client
	queue        *queue.Client
	orchestrator *payment.Orchestrator
	cfg          *config.Config
}

// NewHandler creates a new worker handler
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

	// Initialize payment orchestrator with mock clients
	// In production, replace with real on-ramp/off-ramp clients
	onRamp := payment.NewMockOnRampClient()
	offRamp := payment.NewMockOffRampClient()
	orchestrator := payment.NewOrchestrator(onRamp, offRamp)

	return &Handler{
		db:           db,
		queue:        q,
		orchestrator: orchestrator,
		cfg:          cfg,
	}, nil
}

// HandleRequest processes SQS messages containing payment jobs
func (h *Handler) HandleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	logger.Info("Received SQS event", logger.Fields{
		"record_count": len(sqsEvent.Records),
	})

	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			logger.Error("Failed to process record", logger.Fields{
				"error":      err.Error(),
				"message_id": record.MessageId,
			})
			// Return error to retry the message
			// Note: In production, you might want more sophisticated retry logic
			return err
		}
	}

	return nil
}

// processRecord processes a single SQS record
func (h *Handler) processRecord(ctx context.Context, record events.SQSMessage) error {
	// Parse payment job from message body
	var job models.PaymentJob
	if err := json.Unmarshal([]byte(record.Body), &job); err != nil {
		logger.Error("Failed to unmarshal payment job", logger.Fields{
			"error": err.Error(),
		})
		return err
	}

	logger.Info("Processing payment job", logger.Fields{
		"payment_id": job.PaymentID,
		"amount":     job.Amount,
		"currency":   job.Currency,
	})

	// Update status to PROCESSING
	if err := h.db.UpdatePaymentStatus(ctx, job.PaymentID, models.StatusProcessing, ""); err != nil {
		logger.Error("Failed to update payment status to PROCESSING", logger.Fields{
			"error":      err.Error(),
			"payment_id": job.PaymentID,
		})
		return err
	}

	// Execute payment orchestration
	result, err := h.orchestrator.ProcessPayment(ctx, &job)
	if err != nil {
		// Payment failed - update status
		errorMsg := err.Error()
		logger.Error("Payment processing failed", logger.Fields{
			"error":      errorMsg,
			"payment_id": job.PaymentID,
		})

		if updateErr := h.db.UpdatePaymentStatus(ctx, job.PaymentID, models.StatusFailed, errorMsg); updateErr != nil {
			logger.Error("Failed to update payment status to FAILED", logger.Fields{
				"error":      updateErr.Error(),
				"payment_id": job.PaymentID,
			})
			return updateErr
		}

		// Send webhook notification for failure
		h.sendWebhookNotification(ctx, job.PaymentID, models.StatusFailed, "", "", errorMsg)

		return err
	}

	// Update payment with transaction IDs
	if err := h.db.UpdatePaymentTransactions(ctx, job.PaymentID, result.OnRampTxID, result.OffRampTxID); err != nil {
		logger.Error("Failed to update payment transactions", logger.Fields{
			"error":      err.Error(),
			"payment_id": job.PaymentID,
		})
		return err
	}

	// Update status to COMPLETED
	if err := h.db.UpdatePaymentStatus(ctx, job.PaymentID, models.StatusCompleted, ""); err != nil {
		logger.Error("Failed to update payment status to COMPLETED", logger.Fields{
			"error":      err.Error(),
			"payment_id": job.PaymentID,
		})
		return err
	}

	// Send webhook notification for success
	h.sendWebhookNotification(ctx, job.PaymentID, models.StatusCompleted, result.OnRampTxID, result.OffRampTxID, "")

	logger.Info("Payment processing completed successfully", logger.Fields{
		"payment_id":     job.PaymentID,
		"on_ramp_tx_id":  result.OnRampTxID,
		"off_ramp_tx_id": result.OffRampTxID,
	})

	return nil
}

// sendWebhookNotification sends a webhook event to the webhook queue
func (h *Handler) sendWebhookNotification(ctx context.Context, paymentID string, status models.PaymentStatus, onRampTxID, offRampTxID, errorMsg string) {
	// Fetch full payment details
	payment, err := h.db.GetPaymentByID(ctx, paymentID)
	if err != nil {
		logger.Error("Failed to fetch payment for webhook", logger.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		})
		return
	}

	// Determine event type
	eventType := "payment.completed"
	if status == models.StatusFailed {
		eventType = "payment.failed"
	}

	// Create webhook event with fee information
	event := &models.WebhookEvent{
		EventType:   eventType,
		PaymentID:   paymentID,
		Status:      status,
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		OnRampTxID:  onRampTxID,
		OffRampTxID: offRampTxID,
		Error:       errorMsg,
		Timestamp:   time.Now(),
	}

	// Include fee information if available
	if payment.FeeAmount > 0 {
		event.Fees = &models.FeeBreakdown{
			Amount:   payment.FeeAmount,
			Currency: payment.FeeCurrency,
		}
	}

	// Send to webhook queue
	if err := h.queue.SendWebhookEvent(ctx, h.cfg.Queue.WebhookQueueURL, event); err != nil {
		logger.Error("Failed to send webhook event", logger.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		})
		// We don't return error here as the payment is processed successfully
		// Webhook delivery failure should be handled separately
	} else {
		logger.Info("Webhook event sent", logger.Fields{
			"payment_id": paymentID,
			"status":     status,
		})
	}
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
