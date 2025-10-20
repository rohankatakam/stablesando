package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	secretString, err := GetSecretValue(ctx, secretName, region)
	if err != nil {
		return "", fmt.Errorf("failed to get Anthropic API key: %w", err)
	}

	// Trim any whitespace
	secretString = strings.TrimSpace(secretString)

	// Check if it's JSON (key/value format from AWS Console)
	if strings.HasPrefix(secretString, "{") {
		// Parse as JSON and extract the value
		secretMap, err := ParseJSONSecret(secretString)
		if err != nil {
			return "", fmt.Errorf("failed to parse JSON secret: %w", err)
		}

		// Debug: Print all keys in the JSON
		fmt.Printf("DEBUG: JSON secret keys: ")
		for key := range secretMap {
			fmt.Printf("%s, ", key)
		}
		fmt.Printf("\n")

		// Try to get the API key from the map using the secret name as key
		if apiKey, ok := secretMap[secretName].(string); ok {
			apiKey = strings.TrimSpace(apiKey)
			if len(apiKey) > 20 {
				fmt.Printf("DEBUG: Found API key using key '%s', prefix=%s..., length=%d\n", secretName, apiKey[:20], len(apiKey))
			} else {
				fmt.Printf("DEBUG: Found API key using key '%s', length=%d (too short!)\n", secretName, len(apiKey))
			}
			return apiKey, nil
		}

		// If not found by secret name, try common key names
		for _, keyName := range []string{"api_key", "apiKey", "key", "anthropic_api_key"} {
			if apiKey, ok := secretMap[keyName].(string); ok {
				fmt.Printf("DEBUG: Found API key using key '%s', length=%d\n", keyName, len(apiKey))
				return strings.TrimSpace(apiKey), nil
			}
		}

		return "", fmt.Errorf("could not find API key in JSON secret (tried keys: %s, api_key, apiKey, key, anthropic_api_key)", secretName)
	}

	// Otherwise, treat as plain text
	return secretString, nil
}

// ParseJSONSecret parses a JSON secret into a map
func ParseJSONSecret(secretString string) (map[string]interface{}, error) {
	var secretMap map[string]interface{}
	if err := json.Unmarshal([]byte(secretString), &secretMap); err != nil {
		return nil, fmt.Errorf("failed to parse JSON secret: %w", err)
	}
	return secretMap, nil
}
