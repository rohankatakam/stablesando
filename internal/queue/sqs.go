package queue

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"crypto-conversion/internal/errors"
	"crypto-conversion/internal/logger"
	"crypto-conversion/internal/models"
)

// Client represents an SQS client
type Client struct {
	svc *sqs.SQS
}

// NewClient creates a new SQS client
func NewClient(region, endpoint string) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	svc := sqs.New(sess)

	// Override endpoint for local testing
	if endpoint != "" {
		svc.Endpoint = endpoint
	}

	return &Client{
		svc: svc,
	}, nil
}

// SendPaymentJob sends a payment job to the queue
func (c *Client) SendPaymentJob(ctx context.Context, queueURL string, job *models.PaymentJob) error {
	return c.SendPaymentJobWithDelay(ctx, queueURL, job, 0)
}

// SendPaymentJobWithDelay sends a payment job to the queue with a delay
func (c *Client) SendPaymentJobWithDelay(ctx context.Context, queueURL string, job *models.PaymentJob, delaySeconds int) error {
	body, err := json.Marshal(job)
	if err != nil {
		logger.Error("Failed to marshal payment job", logger.Fields{"error": err.Error()})
		return errors.ErrQueueOperation("marshal", err)
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(body)),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"PaymentID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(job.PaymentID),
			},
			"Currency": {
				DataType:    aws.String("String"),
				StringValue: aws.String(job.Currency),
			},
		},
	}

	// Add delay if specified (max 900 seconds = 15 minutes for standard SQS)
	if delaySeconds > 0 {
		if delaySeconds > 900 {
			delaySeconds = 900 // Cap at SQS max
		}
		input.DelaySeconds = aws.Int64(int64(delaySeconds))
	}

	result, err := c.svc.SendMessageWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to send payment job", logger.Fields{
			"error":        err.Error(),
			"payment_id":   job.PaymentID,
			"delay_seconds": delaySeconds,
		})
		return errors.ErrQueueOperation("send", err)
	}

	logger.Info("Payment job sent to queue", logger.Fields{
		"payment_id":    job.PaymentID,
		"message_id":    *result.MessageId,
		"delay_seconds": delaySeconds,
	})
	return nil
}

// EnqueuePaymentWithDelay is an alias for compatibility with state machine interface
func (c *Client) EnqueuePaymentWithDelay(ctx context.Context, job *models.PaymentJob, delaySeconds int) error {
	// This will be set by the worker handler which knows the queue URL
	// For now, this is a placeholder - will be properly wired in worker handler
	return nil
}

// SendWebhookEvent sends a webhook event to the queue
func (c *Client) SendWebhookEvent(ctx context.Context, queueURL string, event *models.WebhookEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		logger.Error("Failed to marshal webhook event", logger.Fields{"error": err.Error()})
		return errors.ErrQueueOperation("marshal", err)
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(body)),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"PaymentID": {
				DataType:    aws.String("String"),
				StringValue: aws.String(event.PaymentID),
			},
			"Status": {
				DataType:    aws.String("String"),
				StringValue: aws.String(string(event.Status)),
			},
		},
	}

	result, err := c.svc.SendMessageWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to send webhook event", logger.Fields{
			"error":      err.Error(),
			"payment_id": event.PaymentID,
		})
		return errors.ErrQueueOperation("send", err)
	}

	logger.Info("Webhook event sent to queue", logger.Fields{
		"payment_id": event.PaymentID,
		"message_id": *result.MessageId,
	})
	return nil
}

// DeleteMessage deletes a message from the queue
func (c *Client) DeleteMessage(ctx context.Context, queueURL, receiptHandle string) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	}

	_, err := c.svc.DeleteMessageWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to delete message", logger.Fields{"error": err.Error()})
		return errors.ErrQueueOperation("delete", err)
	}

	return nil
}
