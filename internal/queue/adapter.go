package queue

import (
	"context"

	"crypto-conversion/internal/models"
)

// QueueAdapter wraps the SQS client with a known queue URL
type QueueAdapter struct {
	client   *Client
	queueURL string
}

// NewQueueAdapter creates a new queue adapter
func NewQueueAdapter(client *Client, queueURL string) *QueueAdapter {
	return &QueueAdapter{
		client:   client,
		queueURL: queueURL,
	}
}

// EnqueuePaymentWithDelay sends a payment job with a delay
func (qa *QueueAdapter) EnqueuePaymentWithDelay(ctx context.Context, job *models.PaymentJob, delaySeconds int) error {
	return qa.client.SendPaymentJobWithDelay(ctx, qa.queueURL, job, delaySeconds)
}
