package database

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"crypto-conversion/internal/errors"
	"crypto-conversion/internal/logger"
	"crypto-conversion/internal/quotes"
)

// QuoteClient handles quote storage operations
type QuoteClient struct {
	svc       *dynamodb.DynamoDB
	tableName string
}

// NewQuoteClient creates a new quote database client
func NewQuoteClient(region, tableName, endpoint string) (*QuoteClient, error) {
	client, err := NewClient(region, tableName, endpoint)
	if err != nil {
		return nil, err
	}

	return &QuoteClient{
		svc:       client.svc,
		tableName: tableName,
	}, nil
}

// CreateQuote stores a new quote in DynamoDB
func (c *QuoteClient) CreateQuote(ctx context.Context, quote *quotes.Quote) error {
	av, err := dynamodbattribute.MarshalMap(quote)
	if err != nil {
		logger.Error("Failed to marshal quote", logger.Fields{"error": err.Error()})
		return errors.ErrDatabaseOperation("marshal", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(c.tableName),
		Item:      av,
	}

	_, err = c.svc.PutItemWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to create quote", logger.Fields{"error": err.Error()})
		return errors.ErrDatabaseOperation("create", err)
	}

	logger.Info("Quote created", logger.Fields{
		"quote_id":   quote.QuoteID,
		"amount":     quote.Amount,
		"expires_at": quote.ExpiresAt,
	})
	return nil
}

// GetQuote retrieves a quote by ID
func (c *QuoteClient) GetQuote(ctx context.Context, quoteID string) (*quotes.Quote, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(c.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"quote_id": {
				S: aws.String(quoteID),
			},
		},
	}

	result, err := c.svc.GetItemWithContext(ctx, input)
	if err != nil {
		logger.Error("Failed to get quote", logger.Fields{"error": err.Error(), "quote_id": quoteID})
		return nil, errors.ErrDatabaseOperation("get", err)
	}

	if result.Item == nil {
		return nil, errors.ErrQuoteNotFound(quoteID)
	}

	var quote quotes.Quote
	err = dynamodbattribute.UnmarshalMap(result.Item, &quote)
	if err != nil {
		logger.Error("Failed to unmarshal quote", logger.Fields{"error": err.Error()})
		return nil, errors.ErrDatabaseOperation("unmarshal", err)
	}

	return &quote, nil
}
