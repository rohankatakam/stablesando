# Crypto Conversion Payment API

A serverless, event-driven cryptocurrency payment processing system built with Go and AWS services.

## Architecture Overview

This system implements a "stablecoin sandwich" payment flow using a fully serverless architecture:

```
Client → API Gateway → Go API Lambda → SQS Queue → Go Worker Lambda → DynamoDB
                                                          ↓
                                                    Webhook Lambda
```

### Components

- **API Handler Lambda**: Validates requests, checks idempotency, and enqueues payment jobs
- **Worker Lambda**: Processes payments through on-ramp/off-ramp orchestration
- **Webhook Lambda**: Sends payment status updates to clients
- **DynamoDB**: Stores payment records and idempotency keys
- **SQS**: Decouples API from processing for async execution

## Project Structure

```
.
├── cmd/                          # Lambda function entry points
│   ├── api-handler/             # API Gateway handler
│   ├── worker-handler/          # SQS worker handler
│   └── webhook-handler/         # Webhook sender handler
├── internal/                     # Private application code
│   ├── config/                  # Configuration management
│   ├── database/                # DynamoDB operations
│   ├── errors/                  # Custom error types
│   ├── logger/                  # Structured logging
│   ├── models/                  # Data models
│   ├── queue/                   # SQS operations
│   ├── validator/               # Request validation
│   └── payment/                 # Payment orchestration logic
├── infrastructure/              # Infrastructure as Code
│   └── terraform/               # Terraform configurations
├── tests/                       # Test suites
│   ├── integration/            # Integration tests
│   └── mocks/                  # Mock implementations
├── docs/                        # Documentation
├── scripts/                     # Deployment and utility scripts
├── go.mod                       # Go module definition
├── Makefile                     # Build and deployment tasks
└── README.md                    # This file
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- AWS CLI configured with appropriate credentials
- Terraform (for infrastructure deployment)
- Docker (for local testing)

### Installation

1. Clone the repository
2. Install dependencies:
   ```bash
   make deps
   ```

### Development

Build all Lambda functions:
```bash
make build
```

Run tests:
```bash
make test
```

Run linter:
```bash
make lint
```

Format code:
```bash
make format
```

### Local Testing

Start local DynamoDB:
```bash
make local-dynamodb
```

Start local SQS:
```bash
make local-sqs
```

Run integration tests:
```bash
make integration-test
```

### Deployment

Deploy to development:
```bash
make deploy-dev
```

Deploy to production:
```bash
make deploy-prod
```

## API Endpoints

### POST /payments

Create a new payment request.

**Headers:**
- `Idempotency-Key`: Unique identifier for request deduplication (required)
- `Content-Type`: application/json

**Request Body:**
```json
{
  "amount": 1000,
  "currency": "EUR",
  "source_account": "user123",
  "destination_account": "merchant456"
}
```

**Response:**
- `202 Accepted`: Payment accepted for processing
- `400 Bad Request`: Invalid request data
- `409 Conflict`: Duplicate idempotency key

## Environment Variables

- `DYNAMODB_TABLE`: DynamoDB table name for payment records
- `SQS_QUEUE_URL`: SQS queue URL for payment jobs
- `WEBHOOK_QUEUE_URL`: SQS queue URL for webhook notifications
- `AWS_REGION`: AWS region for services
- `LOG_LEVEL`: Logging level (DEBUG, INFO, WARN, ERROR)

## Testing

The project includes comprehensive test coverage:

- Unit tests for all components
- Integration tests with local AWS services
- Benchmarks for performance-critical paths

Run all tests with coverage:
```bash
make test
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

## License

MIT License
