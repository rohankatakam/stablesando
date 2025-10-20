package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// GetSecretValue retrieves a secret from AWS Secrets Manager
func GetSecretValue(ctx context.Context, secretName, region string) (string, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return "", fmt.Errorf("unable to create AWS session: %w", err)
	}

	client := secretsmanager.New(sess)

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := client.GetSecretValueWithContext(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// Secrets Manager can store secrets as SecretString or SecretBinary
	var secretString string
	if result.SecretString != nil {
		secretString = *result.SecretString
	} else {
		return "", fmt.Errorf("secret is stored as binary, expected string")
	}

	return secretString, nil
}

// GetAnthropicAPIKey retrieves the Anthropic API key from Secrets Manager or environment
func GetAnthropicAPIKey(ctx context.Context, region string) (string, error) {
	// First, try to get from environment variable (for local development)
	if apiKey := getEnv("ANTHROPIC_API_KEY", ""); apiKey != "" {
		return apiKey, nil
	}

	// Otherwise, fetch from Secrets Manager
	secretName := "crypto-conversion/anthropic-api-key"
	apiKey, err := GetSecretValue(ctx, secretName, region)
	if err != nil {
		return "", fmt.Errorf("failed to get Anthropic API key: %w", err)
	}

	return apiKey, nil
}

// ParseJSONSecret parses a JSON secret into a map
func ParseJSONSecret(secretString string) (map[string]interface{}, error) {
	var secretMap map[string]interface{}
	if err := json.Unmarshal([]byte(secretString), &secretMap); err != nil {
		return nil, fmt.Errorf("failed to parse JSON secret: %w", err)
	}
	return secretMap, nil
}
