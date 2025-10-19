package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/yourusername/crypto-conversion/internal/config"
	"github.com/yourusername/crypto-conversion/internal/logger"
	"github.com/yourusername/crypto-conversion/internal/models"
)

// Handler manages the Webhook Lambda dependencies
type Handler struct {
	httpClient *http.Client
	cfg        *config.Config
}

// NewHandler creates a new webhook handler
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cfg: cfg,
	}
}

// HandleRequest processes SQS messages containing webhook events
func (h *Handler) HandleRequest(ctx context.Context, sqsEvent events.SQSEvent) error {
	logger.Info("Received webhook event", logger.Fields{
		"record_count": len(sqsEvent.Records),
	})

	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			logger.Error("Failed to process webhook record", logger.Fields{
				"error":      err.Error(),
				"message_id": record.MessageId,
			})
			// Continue processing other records even if one fails
			// In production, you might want to send failed webhooks to a DLQ
			continue
		}
	}

	return nil
}

// processRecord processes a single webhook event
func (h *Handler) processRecord(ctx context.Context, record events.SQSMessage) error {
	// Parse webhook event from message body
	var event models.WebhookEvent
	if err := json.Unmarshal([]byte(record.Body), &event); err != nil {
		logger.Error("Failed to unmarshal webhook event", logger.Fields{
			"error": err.Error(),
		})
		return err
	}

	logger.Info("Processing webhook event", logger.Fields{
		"payment_id": event.PaymentID,
		"status":     event.Status,
	})

	// In a real implementation, you would:
	// 1. Fetch the webhook URL from the payment record or a separate configuration
	// 2. Send the webhook with proper authentication/signing
	// 3. Implement retry logic with exponential backoff
	// 4. Track webhook delivery status

	// For now, we'll simulate sending the webhook
	if err := h.sendWebhook(ctx, event); err != nil {
		logger.Error("Failed to send webhook", logger.Fields{
			"error":      err.Error(),
			"payment_id": event.PaymentID,
		})
		return err
	}

	logger.Info("Webhook sent successfully", logger.Fields{
		"payment_id": event.PaymentID,
		"status":     event.Status,
	})

	return nil
}

// sendWebhook sends the webhook to the configured endpoint
func (h *Handler) sendWebhook(ctx context.Context, event models.WebhookEvent) error {
	// In production, fetch this from configuration or database
	// For now, we'll just log the webhook payload
	webhookURL := "https://example.com/webhook" // Placeholder

	// Prepare webhook payload
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	logger.Info("Sending webhook", logger.Fields{
		"url":        webhookURL,
		"payment_id": event.PaymentID,
		"status":     event.Status,
	})

	// In a real implementation, send the actual HTTP request
	// For development/testing, we'll just log it
	logger.Info("Webhook payload", logger.Fields{
		"payload": string(payload),
	})

	// Example of how to send in production:
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payment-ID", event.PaymentID)
	req.Header.Set("X-Payment-Status", string(event.Status))
	// Add signature header for webhook verification
	// req.Header.Set("X-Webhook-Signature", generateSignature(payload))

	// Uncomment in production to actually send the webhook:
	// resp, err := h.httpClient.Do(req)
	// if err != nil {
	// 	return fmt.Errorf("failed to send webhook: %w", err)
	// }
	// defer resp.Body.Close()
	//
	// if resp.StatusCode < 200 || resp.StatusCode >= 300 {
	// 	return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	// }

	logger.Info("Webhook would be sent (mocked in development)", logger.Fields{
		"payment_id": event.PaymentID,
		"url":        webhookURL,
	})

	return nil
}

// generateSignature generates an HMAC signature for webhook verification
// This is a placeholder - implement proper HMAC-SHA256 signing in production
func generateSignature(payload []byte) string {
	// Example:
	// h := hmac.New(sha256.New, []byte(webhookSecret))
	// h.Write(payload)
	// return hex.EncodeToString(h.Sum(nil))
	return "signature-placeholder"
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
	handler := NewHandler(cfg)

	// Start Lambda
	lambda.Start(handler.HandleRequest)
}
