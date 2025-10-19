# Crypto Conversion Payment API

A serverless cryptocurrency payment processing system that demonstrates solving the **Synchronization Problem** - coordinating two blockchain transactions with guaranteed exchange rates during async settlement.

## The Challenge

Cross-chain cryptocurrency payments face timing inconsistency: USD→USDC (onramp) and USDC→EUR (offramp) transactions settle minutes apart. Exchange rate volatility during this window creates uncertainty. This system demonstrates a production-ready solution using quote-based rate locking and asynchronous state machine orchestration.

## Solution Architecture

### Quote System
Locks exchange rate for 60 seconds with guaranteed payout amount, including all fees.

### State Machine
Async orchestration using SQS re-enqueuing pattern:
```
PENDING → ONRAMP_PENDING → ONRAMP_COMPLETE → OFFRAMP_PENDING → COMPLETED
```
Each Lambda execution processes one state, updates DynamoDB, and re-enqueues with delay.

### Key Features
- **Rate Locking**: 60-second guaranteed payout quotes
- **Async Processing**: No long-running processes (Lambda <1s per execution)
- **Scalability**: Serverless auto-scaling
- **Fault Tolerance**: SQS retries + dead letter queues
- **Audit Trail**: Complete state history tracking

## System Flow

```
POST /quotes → Rate Lock (60s) → POST /payments (with quote_id)
    ↓
API Gateway → API Lambda → DynamoDB + SQS
    ↓
Worker Lambda (State Machine) → Poll onramp → Poll offramp → Webhook
```

## Project Structure

```
.
├── cmd/                          # Lambda function entry points
│   ├── api-handler/             # API Gateway handler (quotes + payments)
│   ├── worker-handler/          # State machine orchestrator
│   └── webhook-handler/         # Webhook sender handler
├── internal/                     # Private application code
│   ├── config/                  # Configuration management
│   ├── database/                # DynamoDB operations
│   ├── errors/                  # Custom error types
│   ├── logger/                  # Structured logging
│   ├── models/                  # Data models (Payment, Quote, etc.)
│   ├── queue/                   # SQS operations (with delay support)
│   ├── validator/               # Request validation
│   ├── quotes/                  # Quote generation and validation
│   └── payment/                 # State machine + mock providers
│       ├── state_handlers.go   # State machine implementation
│       └── mock_providers.go   # Stateful onramp/offramp clients
├── infrastructure/              # Infrastructure as Code
│   └── terraform/               # Terraform configurations
│       ├── main.tf             # DynamoDB tables (payments + quotes)
│       ├── modules/
│       │   ├── lambda/         # Lambda functions + IAM roles
│       │   └── api-gateway/    # API Gateway (quotes + payments)
├── docs/                        # Documentation
├── scripts/                     # Deployment and utility scripts
├── go.mod                       # Go module definition
├── Makefile                     # Build and deployment tasks
└── README.md                    # This file
```

## Quick Start

### Prerequisites
- Go 1.21+, AWS CLI, Terraform 1.0+, Make

### Deploy
```bash
make build
cd infrastructure/terraform
terraform init
terraform apply -var-file=environments/dev.tfvars
```

## API Endpoints

### POST /quotes

Generate a rate-locked quote with guaranteed payout amount.

**Request Body:**
```json
{
  "from_currency": "USD",
  "to_currency": "EUR",
  "amount": 100000
}
```

**Response (200 OK):**
```json
{
  "quote_id": "quote_a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "amount": 100000,
  "currency": "USD",
  "exchange_rate": 0.9205,
  "fees": {
    "platform_fee": 2100,
    "onramp_fee": 1050,
    "offramp_fee": 1575,
    "total_fees": 4725,
    "currency": "USD"
  },
  "guaranteed_payout": 87699,
  "payout_currency": "EUR",
  "expires_at": "2025-10-19T05:11:35Z",
  "valid_for_seconds": 60
}
```

Notes:
- Quote expires after 60 seconds
- DynamoDB TTL auto-deletes expired quotes
- Amounts in cents (100000 = $1000.00)

### POST /payments

Create a new payment request using a quote.

**Headers:**
- `Idempotency-Key`: Unique identifier for request deduplication (optional)
- `Content-Type`: application/json

**Request Body:**
```json
{
  "amount": 100000,
  "currency": "EUR",
  "source_account": "user123",
  "destination_account": "merchant456",
  "quote_id": "quote_a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

**Response (202 Accepted):**
```json
{
  "payment_id": "d910ce80-3f54-46bf-a1b0-256234c6c08a",
  "status": "PENDING",
  "message": "Payment accepted for processing"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request data or quote expired
  ```json
  {
    "error": "QUOTE_EXPIRED",
    "message": "Quote has expired, please request a new quote"
  }
  ```
- `409 Conflict`: Duplicate idempotency key

## State Machine Flow

| State | Action | Duration |
|-------|--------|----------|
| PENDING | Initiate onramp | <1s |
| ONRAMP_PENDING | Poll settlement | 90-120s |
| ONRAMP_COMPLETE | Initiate offramp | <1s |
| OFFRAMP_PENDING | Poll settlement | 90-120s |
| COMPLETED | Send webhook | Terminal |

## Configuration

All environment variables are managed via Terraform. Key configs:
- DynamoDB tables (payments, quotes)
- SQS queues (payments, webhooks, DLQs)
- Lambda timeouts and memory
- Log levels

## Testing

1. Create quote: `POST /quotes`
2. Create payment: `POST /payments` with quote_id
3. Monitor: Check DynamoDB for payment state
4. View logs: CloudWatch logs for each Lambda

## Performance

- API response: <200ms
- Worker execution: <1s per state
- Total processing: 3-8 minutes
- Quote validity: 60 seconds

## Technical Design

**Patterns:**
- Async state machine with SQS re-enqueuing
- Quote-based rate locking
- Polling-based settlement tracking
- Idempotency via header validation
- Complete audit trail

**Scalability:**
- Lambda auto-scales based on queue depth
- DynamoDB on-demand capacity
- Queue-based decoupling
- Dead letter queues for failures

## Documentation

- [spec.md](spec.md) - Original system specification
- [docs/architecture.md](docs/architecture.md) - Detailed system design
- [docs/api-reference.md](docs/api-reference.md) - Complete API documentation
- [docs/deployment-guide.md](docs/deployment-guide.md) - Deployment instructions
- [docs/production-scaling.md](docs/production-scaling.md) - Production scaling guide
