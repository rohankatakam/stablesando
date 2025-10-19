package config

import (
	"fmt"
	"os"
)

// Config holds all application configuration
type Config struct {
	AWS      AWSConfig
	Database DatabaseConfig
	Queue    QueueConfig
	Logging  LoggingConfig
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	Region string
}

// DatabaseConfig holds DynamoDB configuration
type DatabaseConfig struct {
	TableName      string
	QuoteTableName string
	Endpoint       string // For local testing
}

// QueueConfig holds SQS configuration
type QueueConfig struct {
	PaymentQueueURL string
	WebhookQueueURL string
	Endpoint        string // For local testing
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		AWS: AWSConfig{
			Region: getEnv("AWS_REGION", "us-east-1"),
		},
		Database: DatabaseConfig{
			TableName:      getEnv("DYNAMODB_TABLE", "payments"),
			QuoteTableName: getEnv("QUOTE_TABLE", "quotes"),
			Endpoint:       getEnv("DYNAMODB_ENDPOINT", ""), // Empty for AWS, set for local
		},
		Queue: QueueConfig{
			PaymentQueueURL: getEnv("PAYMENT_QUEUE_URL", ""),
			WebhookQueueURL: getEnv("WEBHOOK_QUEUE_URL", ""),
			Endpoint:        getEnv("SQS_ENDPOINT", ""), // Empty for AWS, set for local
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "INFO"),
		},
	}

	// Validate required fields
	if cfg.Queue.PaymentQueueURL == "" {
		return nil, fmt.Errorf("PAYMENT_QUEUE_URL is required")
	}

	if cfg.Database.TableName == "" {
		return nil, fmt.Errorf("DYNAMODB_TABLE is required")
	}

	return cfg, nil
}

// getEnv gets an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
