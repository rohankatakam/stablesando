# API Reference

## Base URL

```
https://<api-id>.execute-api.<region>.amazonaws.com/<stage>/
```

Example:
```
https://abc123xyz.execute-api.us-east-1.amazonaws.com/dev/
```

## Authentication

Currently, the API does not require authentication. In production, you should implement one of:
- API Keys
- AWS IAM Signature
- Lambda Authorizers
- Amazon Cognito

## Headers

### Required Headers

| Header | Type | Description |
|--------|------|-------------|
| `Idempotency-Key` | string | Unique identifier for request deduplication (10-255 characters, alphanumeric, hyphens, underscores) |
| `Content-Type` | string | Must be `application/json` |

### Response Headers

| Header | Type | Description |
|--------|------|-------------|
| `Content-Type` | string | Always `application/json` |

## Endpoints

### POST /payments

Create a new payment request.

#### Request

```http
POST /payments HTTP/1.1
Host: <api-endpoint>
Content-Type: application/json
Idempotency-Key: payment-abc123-xyz789

{
  "amount": 100000,
  "currency": "EUR",
  "source_account": "user123",
  "destination_account": "merchant456"
}
```

#### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `amount` | integer | Yes | Payment amount in smallest currency unit (e.g., cents). Must be > 0 and ≤ 1,000,000,000 |
| `currency` | string | Yes | ISO 4217 currency code. Supported: USD, EUR, GBP, JPY, AUD, CAD |
| `source_account` | string | Yes | Source account identifier (3-100 characters) |
| `destination_account` | string | Yes | Destination account identifier (3-100 characters, must differ from source) |

**Note**: Fees are automatically calculated based on the payment amount and destination currency. See [Fee Structure](#fee-structure) below.

#### Success Response

**Status Code:** `202 Accepted`

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING",
  "message": "Payment accepted for processing"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `payment_id` | string | UUID of the created payment |
| `status` | string | Current payment status (will be "PENDING") |
| `message` | string | Human-readable status message |

#### Error Responses

##### 400 Bad Request

Invalid request data or validation failure.

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed for field 'amount': must be greater than 0"
  }
}
```

**Common Error Codes:**
- `VALIDATION_ERROR`: Field validation failed
- `MISSING_HEADER`: Required header is missing
- `INVALID_JSON`: Request body is not valid JSON
- `INVALID_REQUEST`: General request validation failure

##### 409 Conflict

Duplicate idempotency key.

```json
{
  "error": {
    "code": "DUPLICATE_REQUEST",
    "message": "Request with idempotency key 'payment-abc123-xyz789' already exists"
  }
}
```

##### 500 Internal Server Error

Server-side error during processing.

```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Failed to process request"
  }
}
```

**Common Error Codes:**
- `INTERNAL_ERROR`: General server error
- `DATABASE_ERROR`: Database operation failed
- `QUEUE_ERROR`: Failed to enqueue payment job

## Payment Status Lifecycle

```
PENDING → PROCESSING → COMPLETED
                    ↘ FAILED
```

| Status | Description |
|--------|-------------|
| `PENDING` | Payment created and queued for processing |
| `PROCESSING` | Payment is being processed (on-ramp/off-ramp in progress) |
| `COMPLETED` | Payment successfully completed |
| `FAILED` | Payment failed (error details in `error_message` field) |

## Idempotency

The API uses idempotency keys to prevent duplicate payments. The `Idempotency-Key` header is required for all payment creation requests.

### Best Practices

1. **Generate Unique Keys**: Use UUIDs or similar unique identifiers
2. **Deterministic Retries**: Use the same key for retries of the same logical operation
3. **Key Format**: Alphanumeric characters, hyphens, and underscores only
4. **Key Length**: Between 10 and 255 characters

### Behavior

- If a request with a new idempotency key succeeds, a `202 Accepted` response is returned
- If a request with a duplicate idempotency key is received, a `409 Conflict` response is returned
- Idempotency keys are stored permanently in DynamoDB

## Rate Limiting

The API implements rate limiting via API Gateway Usage Plans:

| Limit | Value |
|-------|-------|
| Requests per second | 50 |
| Burst | 100 |
| Daily quota | 10,000 |

When rate limits are exceeded, you'll receive a `429 Too Many Requests` response:

```json
{
  "message": "Too Many Requests"
}
```

## Webhooks

After payment processing completes, the system sends a webhook notification to your configured endpoint.

### Webhook Payload

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "COMPLETED",
  "amount": 100000,
  "currency": "EUR",
  "on_ramp_tx_id": "onramp_EUR_1234567890",
  "off_ramp_tx_id": "offramp_EUR_1234567891",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

For failed payments:

```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "FAILED",
  "amount": 100000,
  "currency": "EUR",
  "error": "mock on-ramp service unavailable",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

### Webhook Headers

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/json` |
| `X-Payment-ID` | Payment identifier |
| `X-Payment-Status` | Payment status |
| `X-Webhook-Signature` | HMAC signature for verification (when implemented) |

### Webhook Retry Policy

- Retries: Up to 5 attempts
- Backoff: Exponential
- Failed webhooks are sent to a Dead Letter Queue for manual review

## Examples

### cURL

```bash
curl -X POST https://abc123.execute-api.us-east-1.amazonaws.com/dev/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "amount": 100000,
    "currency": "EUR",
    "source_account": "user123",
    "destination_account": "merchant456"
  }'
```

### JavaScript (fetch)

```javascript
const response = await fetch('https://abc123.execute-api.us-east-1.amazonaws.com/dev/payments', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Idempotency-Key': crypto.randomUUID()
  },
  body: JSON.stringify({
    amount: 100000,
    currency: 'EUR',
    source_account: 'user123',
    destination_account: 'merchant456'
  })
});

const data = await response.json();
console.log(data);
```

### Python (requests)

```python
import requests
import uuid

response = requests.post(
    'https://abc123.execute-api.us-east-1.amazonaws.com/dev/payments',
    headers={
        'Content-Type': 'application/json',
        'Idempotency-Key': str(uuid.uuid4())
    },
    json={
        'amount': 100000,
        'currency': 'EUR',
        'source_account': 'user123',
        'destination_account': 'merchant456'
    }
)

print(response.status_code)
print(response.json())
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "github.com/google/uuid"
)

func createPayment() error {
    payload := map[string]interface{}{
        "amount": 100000,
        "currency": "EUR",
        "source_account": "user123",
        "destination_account": "merchant456",
    }

    body, _ := json.Marshal(payload)

    req, _ := http.NewRequest(
        "POST",
        "https://abc123.execute-api.us-east-1.amazonaws.com/dev/payments",
        bytes.NewBuffer(body),
    )

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Idempotency-Key", uuid.New().String())

    client := &http.Client{}
    resp, err := client.Do(req)

    // Handle response...
    return err
}
```

## Fee Structure

The API automatically calculates fees based on the payment amount and destination currency. Fees are charged in USD.

### Tiered Fee Model

| Amount Range | Percentage Fee | Fixed Fee | Example (USD) |
|--------------|----------------|-----------|---------------|
| $0 - $99.99 | 2.9% | $0.30 | $50 → $1.75 fee ($1.45 + $0.30) |
| $100 - $999.99 | 2.5% | $0.50 | $500 → $13.00 fee ($12.50 + $0.50) |
| $1,000+ | 2.0% | $1.00 | $5,000 → $101.00 fee ($100 + $1.00) |

**Formula**: `Total Fee = (Amount × Percentage) + Fixed Fee`

### Fee Examples

#### Example 1: Small Payment
```json
{
  "amount": 5000,  // $50.00
  "currency": "EUR"
}
```
**Fee Calculation**:
- Percentage: $50.00 × 2.9% = $1.45
- Fixed: $0.30
- **Total Fee: $1.75** (175 cents)

#### Example 2: Medium Payment
```json
{
  "amount": 50000,  // $500.00
  "currency": "EUR"
}
```
**Fee Calculation**:
- Percentage: $500.00 × 2.5% = $12.50
- Fixed: $0.50
- **Total Fee: $13.00** (1,300 cents)

#### Example 3: Large Payment
```json
{
  "amount": 500000,  // $5,000.00
  "currency": "EUR"
}
```
**Fee Calculation**:
- Percentage: $5,000.00 × 2.0% = $100.00
- Fixed: $1.00
- **Total Fee: $101.00** (10,100 cents)

### Fee Information in Responses

Fees are stored with each payment and included in webhook notifications:

```json
{
  "event_type": "payment.completed",
  "payment_id": "9a586bc5-d753-4754-86f3-897b4e8a043f",
  "status": "COMPLETED",
  "amount": 5000,
  "currency": "EUR",
  "fees": {
    "amount": 175,
    "currency": "USD"
  },
  "on_ramp_tx_id": "onramp_EUR_1760837018830172901",
  "off_ramp_tx_id": "offramp_EUR_1760837019049612817",
  "timestamp": "2025-10-19T01:23:39Z"
}
```

### Querying Fee Information

Fees are automatically calculated and stored in the DynamoDB payment record:

```bash
aws dynamodb get-item \
  --table-name crypto-conversion-payments-dev \
  --key '{"payment_id": {"S": "<payment-id>"}}'
```

Response includes:
```json
{
  "Item": {
    "payment_id": {"S": "9a586bc5-d753-4754-86f3-897b4e8a043f"},
    "amount": {"N": "5000"},
    "fee_amount": {"N": "175"},
    "fee_currency": {"S": "USD"},
    ...
  }
}
```

## Testing

Use the provided test script:

```bash
./scripts/test-api.sh https://abc123.execute-api.us-east-1.amazonaws.com/dev/payments
```

This will run a series of tests including:
- Successful payment creation
- Duplicate idempotency key handling
- Missing header validation
- Invalid data validation

### Testing Fee Calculations

Test different payment amounts to verify fee tiers:

```bash
# Test tier 1: < $100 (2.9% + $0.30)
curl -X POST $API_ENDPOINT/payments \
  -H "Idempotency-Key: test-$(date +%s)" \
  -H "Content-Type: application/json" \
  -d '{"amount": 5000, "currency": "EUR", "source_account": "test", "destination_account": "merchant"}'

# Test tier 2: $100-$999 (2.5% + $0.50)
curl -X POST $API_ENDPOINT/payments \
  -H "Idempotency-Key: test-$(date +%s)" \
  -H "Content-Type: application/json" \
  -d '{"amount": 50000, "currency": "EUR", "source_account": "test", "destination_account": "merchant"}'

# Test tier 3: $1000+ (2.0% + $1.00)
curl -X POST $API_ENDPOINT/payments \
  -H "Idempotency-Key: test-$(date +%s)" \
  -H "Content-Type: application/json" \
  -d '{"amount": 500000, "currency": "EUR", "source_account": "test", "destination_account": "merchant"}'
```

Check the fee amounts in DynamoDB after processing (wait 5-10 seconds for async processing)
