package database

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"crypto-conversion/internal/errors"
	"crypto-conversion/internal/logger"
	"crypto-conversion/internal/models"
)

// Client represents a DynamoDB client
type Client struct {
	svc       *dynamodb.DynamoDB
	tableName string
}

// NewClient creates a new DynamoDB client
func NewClient(region, tableName, endpoint string) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	svc := dynamodb.New(sess)

	// Override endpoint for local testing
	if endpoint != "" {
		svc.Endpoint = endpoint
	}

	return &Client{
		svc:       svc,
		tableName: tableName,
	}, nil
}

// CreatePayment creates a new payment record
func (c *Client) CreatePayment(ctx context.Context, payment *models.Payment) error {
	av, err := dynamodbattribute.MarshalMap(payment)
	if err != nil {
		logger.Error("Failed to marshal payment", logger.Fields{"error": err.Error()})
		return errors.ErrDatabaseOperation("marshal", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(c.tableName),
		Item:      av,
		// Ensure idempotency key doesn't already exist
		ConditionExpression: aws.String("attribute_not_exists(idempotency_key)"),
	}

	_, err = c.svc.PutItemWithContext(ctx, input)
	if err != nil {
		// Check if it's a conditional check failure (duplicate)
		if _, ok := err.(*dynamodb.ConditionalCheckFailedException); ok {
			return errors.ErrDuplicateRequest(payment.IdempotencyKey)
		}
		logger.Error("Failed to create payment", logger.Fields{"error": err.Error()})
		return errors.ErrDatabaseOperation("create", err)
	}

	logger.Info("Payment created", logger.Fields{
		"payment_id":      payment.PaymentID,
		"idempotency_key": payment.IdempotencyKey,
	})
	return nil
}

// GetPaymentByID retrieves a payment by its ID
func (c *Client) GetPaymentByID(ctx context.Context, paymentID string) (*models.Payment, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"payment_id": {
				S: aws.String(paymentID),
			},
		},
	}

	result, err := c.svc.GetItemWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to get payment", logger.Fields{"error": err.Error(), "payment_id": paymentID})
		return nil, errors.ErrDatabaseOperation("get", err)
	}

	if result.Item == nil {
		return nil, errors.ErrPaymentNotFound(paymentID)
	}

	var payment models.Payment
	err = dynamodbattribute.UnmarshalMap(result.Item, &payment)
	if err != nil {
		logger.Error("Failed to unmarshal payment", logger.Fields{"error": err.Error()})
		return nil, errors.ErrDatabaseOperation("unmarshal", err)
	}

	return &payment, nil
}

// GetPaymentByIdempotencyKey retrieves a payment by its idempotency key
func (c *Client) GetPaymentByIdempotencyKey(ctx context.Context, idempotencyKey string) (*models.Payment, error) {
	// Create a filter expression
	filt := expression.Name("idempotency_key").Equal(expression.Value(idempotencyKey))
	expr, err := expression.NewBuilder().WithFilter(filt).Build()
	if err != nil {
		logger.Error("Failed to build expression", logger.Fields{"error": err.Error()})
		return nil, errors.ErrDatabaseOperation("build_expression", err)
	}

	input := &dynamodb.ScanInput{
		TableName:                 aws.String(c.tableName),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := c.svc.ScanWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to scan for payment", logger.Fields{"error": err.Error()})
		return nil, errors.ErrDatabaseOperation("scan", err)
	}

	if len(result.Items) == 0 {
		return nil, nil // Not found, but not an error
	}

	var payment models.Payment
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &payment)
	if err != nil {
		logger.Error("Failed to unmarshal payment", logger.Fields{"error": err.Error()})
		return nil, errors.ErrDatabaseOperation("unmarshal", err)
	}

	return &payment, nil
}

// UpdatePaymentStatus updates the status of a payment
func (c *Client) UpdatePaymentStatus(ctx context.Context, paymentID string, status models.PaymentStatus, errorMsg string) error {
	now := time.Now()

	update := expression.Set(expression.Name("status"), expression.Value(status)).
		Set(expression.Name("updated_at"), expression.Value(now))

	if errorMsg != "" {
		update = update.Set(expression.Name("error_message"), expression.Value(errorMsg))
	}

	if status == models.StatusCompleted || status == models.StatusFailed {
		update = update.Set(expression.Name("processed_at"), expression.Value(now))
	}

	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		logger.Error("Failed to build update expression", logger.Fields{"error": err.Error()})
		return errors.ErrDatabaseOperation("build_expression", err)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"payment_id": {
				S: aws.String(paymentID),
			},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	_, err = c.svc.UpdateItemWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to update payment status", logger.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
			"status":     status,
		})
		return errors.ErrDatabaseOperation("update", err)
	}

	logger.Info("Payment status updated", logger.Fields{
		"payment_id": paymentID,
		"status":     status,
	})
	return nil
}

// UpdatePaymentTransactions updates the transaction IDs for a payment
func (c *Client) UpdatePaymentTransactions(ctx context.Context, paymentID, onRampTxID, offRampTxID string) error {
	update := expression.Set(expression.Name("updated_at"), expression.Value(time.Now()))

	if onRampTxID != "" {
		update = update.Set(expression.Name("on_ramp_tx_id"), expression.Value(onRampTxID))
	}
	if offRampTxID != "" {
		update = update.Set(expression.Name("off_ramp_tx_id"), expression.Value(offRampTxID))
	}

	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return errors.ErrDatabaseOperation("build_expression", err)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"payment_id": {
				S: aws.String(paymentID),
			},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	_, err = c.svc.UpdateItemWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to update payment transactions", logger.Fields{
			"error":      err.Error(),
			"payment_id": paymentID,
		})
		return errors.ErrDatabaseOperation("update_transactions", err)
	}

	logger.Info("Payment transactions updated", logger.Fields{
		"payment_id":     paymentID,
		"on_ramp_tx_id":  onRampTxID,
		"off_ramp_tx_id": offRampTxID,
	})
	return nil
}

// UpdatePayment updates the entire payment record
func (c *Client) UpdatePayment(ctx context.Context, payment *models.Payment) error {
	payment.UpdatedAt = time.Now()

	av, err := dynamodbattribute.MarshalMap(payment)
	if err != nil {
		logger.Error("Failed to marshal payment", logger.Fields{"error": err.Error()})
		return errors.ErrDatabaseOperation("marshal", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(c.tableName),
		Item:      av,
	}

	_, err = c.svc.PutItemWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to update payment", logger.Fields{
			"error":      err.Error(),
			"payment_id": payment.PaymentID,
		})
		return errors.ErrDatabaseOperation("update", err)
	}

	logger.Info("Payment updated", logger.Fields{
		"payment_id": payment.PaymentID,
		"status":     payment.Status,
	})
	return nil
}
