# Architecture Documentation

## System Overview

The Crypto Conversion Payment API is a serverless, event-driven system built on AWS that processes cryptocurrency payments using a "stablecoin sandwich" approach.

## Architecture Diagram

```
┌─────────┐
│ Client  │
└────┬────┘
     │
     │ POST /payments
     │ (Idempotency-Key header)
     ▼
┌─────────────────┐
│  API Gateway    │
└────┬────────────┘
     │
     │ triggers
     ▼
┌──────────────────────────┐
│ API Handler Lambda       │
│ • Validate request       │
│ • Check idempotency key  │
│ • Create payment record  │
│ • Enqueue job           │
│ • Return 202 Accepted   │
└──────┬───────────────────┘
       │
       │ writes to          reads/writes
       ▼                    ▼
┌─────────────┐      ┌──────────────┐
│  DynamoDB   │      │  SQS Queue   │
│  (payments) │      │  (jobs)      │
└─────────────┘      └──────┬───────┘
                            │
                            │ triggers
                            ▼
                     ┌──────────────────────┐
                     │ Worker Lambda        │
                     │ • Update to          │
                     │   PROCESSING         │
                     │ • On-ramp (fiat→USD) │
                     │ • Off-ramp (USD→fiat)│
                     │ • Update to          │
                     │   COMPLETED/FAILED   │
                     │ • Enqueue webhook    │
                     └──────┬───────────────┘
                            │
                            │ writes event
                            ▼
                     ┌──────────────┐
                     │ Webhook Queue│
                     └──────┬───────┘
                            │
                            │ triggers
                            ▼
                     ┌──────────────────┐
                     │ Webhook Lambda   │
                     │ • Send webhook   │
                     │   notification   │
                     └──────────────────┘
```

## Components

### 1. API Gateway

- **Purpose**: Public-facing HTTP endpoint
- **Responsibilities**:
  - Route incoming requests to Lambda
  - Handle CORS
  - Rate limiting and throttling
  - Access logging

### 2. API Handler Lambda

- **Runtime**: Go (provided.al2)
- **Timeout**: 30 seconds
- **Memory**: 512 MB
- **Responsibilities**:
  - Request validation
  - Idempotency key checking
  - Payment record creation
  - Job enqueueing
  - Fast response (< 1 second)

**Key Operations**:
1. Extract and validate `Idempotency-Key` header
2. Query DynamoDB for existing payment with same key
3. Validate payment request data
4. Create payment record with status `PENDING`
5. Send job to SQS payment queue
6. Return `202 Accepted` response

### 3. DynamoDB Table

- **Table**: `payments`
- **Primary Key**: `payment_id` (String)
- **Global Secondary Index**: `idempotency_key` (String)
- **Features**:
  - On-demand billing
  - Point-in-time recovery (production)
  - Server-side encryption

**Schema**:
```json
{
  "payment_id": "uuid",
  "idempotency_key": "string",
  "amount": "number",
  "currency": "string",
  "source_account": "string",
  "destination_account": "string",
  "status": "PENDING|PROCESSING|COMPLETED|FAILED",
  "on_ramp_tx_id": "string (optional)",
  "off_ramp_tx_id": "string (optional)",
  "error_message": "string (optional)",
  "created_at": "timestamp",
  "updated_at": "timestamp",
  "processed_at": "timestamp (optional)"
}
```

### 4. SQS Payment Queue

- **Purpose**: Decouple API from processing
- **Configuration**:
  - Visibility timeout: 300 seconds (5 minutes)
  - Message retention: 14 days
  - Long polling enabled
  - Dead letter queue after 3 retries

### 5. Worker Lambda

- **Runtime**: Go (provided.al2)
- **Timeout**: 300 seconds (5 minutes)
- **Memory**: 512 MB
- **Trigger**: SQS payment queue (batch size: 1)
- **Responsibilities**:
  - Payment orchestration
  - On-ramp/off-ramp execution
  - Status updates
  - Webhook event creation

**Processing Flow**:
1. Receive job from SQS
2. Update payment status to `PROCESSING`
3. Execute on-ramp: Convert fiat → stablecoin
4. Execute off-ramp: Convert stablecoin → fiat
5. Update payment with transaction IDs
6. Update payment status to `COMPLETED` or `FAILED`
7. Send webhook event to queue

### 6. SQS Webhook Queue

- **Purpose**: Decouple payment processing from webhook delivery
- **Configuration**:
  - Visibility timeout: 60 seconds
  - Message retention: 4 days
  - Dead letter queue after 5 retries

### 7. Webhook Lambda

- **Runtime**: Go (provided.al2)
- **Timeout**: 30 seconds
- **Memory**: 256 MB
- **Trigger**: SQS webhook queue (batch size: 10)
- **Responsibilities**:
  - Send webhook notifications to clients
  - Retry logic with exponential backoff
  - Webhook signature generation

## Data Flow

### Successful Payment Flow

1. **Client Request** (t=0ms)
   - POST /payments with idempotency key
   - Request body contains payment details

2. **API Handler** (t=0-500ms)
   - Validates request
   - Checks for duplicate idempotency key
   - Creates payment record (status: PENDING)
   - Enqueues job
   - Returns 202 Accepted

3. **Worker Processing** (t=500ms-60s)
   - Updates status to PROCESSING
   - Executes on-ramp (mock: 100-300ms)
   - Executes off-ramp (mock: 100-300ms)
   - Updates status to COMPLETED
   - Enqueues webhook event

4. **Webhook Delivery** (t=60s+)
   - Sends webhook to client endpoint
   - Includes payment status and transaction IDs

### Failure Scenarios

**Duplicate Request**:
- API Handler detects existing idempotency key
- Returns 409 Conflict immediately
- No processing occurs

**Validation Failure**:
- API Handler validates request
- Returns 400 Bad Request with error details
- No database write occurs

**Processing Failure**:
- Worker catches error during on-ramp/off-ramp
- Updates status to FAILED with error message
- Sends webhook with failure details

**Retry Logic**:
- SQS automatically retries failed messages
- Payment queue: 3 retries → DLQ
- Webhook queue: 5 retries → DLQ

## Scalability

### Horizontal Scaling
- All components are serverless and auto-scale
- Lambda concurrency scales to demand
- DynamoDB on-demand scaling
- SQS handles unlimited messages

### Performance Characteristics
- API response time: < 500ms (p95)
- Payment processing: 1-5 seconds
- Throughput: Limited by Lambda concurrency (default 1000)

### Cost Optimization
- Pay-per-use pricing model
- No idle costs
- DynamoDB on-demand billing
- Lambda execution-based billing

## Security

### Authentication & Authorization
- API Gateway can be configured with:
  - API Keys
  - AWS IAM
  - Lambda Authorizers
  - Cognito User Pools

### Data Protection
- DynamoDB encryption at rest
- TLS 1.2+ for all communications
- Idempotency keys prevent duplicate charges

### IAM Roles
- Least privilege access
- Separate roles for each Lambda
- No hard-coded credentials

## Monitoring & Observability

### CloudWatch Logs
- All Lambda functions log to CloudWatch
- Structured JSON logging
- Log retention configurable per environment

### CloudWatch Metrics
- Lambda invocations, duration, errors
- API Gateway request counts, latency
- SQS message counts, age
- DynamoDB read/write capacity

### Alarms (Recommended)
- Lambda error rate > 5%
- API Gateway 5xx errors
- SQS DLQ message count > 0
- DynamoDB throttling events

### X-Ray Tracing
- Enabled on API Gateway
- Can be enabled on Lambda functions
- End-to-end request tracing

## Deployment

### Infrastructure as Code
- Terraform for all AWS resources
- Modular design (Lambda, API Gateway)
- Environment-specific configurations
- State stored in S3

### CI/CD Pipeline (Recommended)
1. Build: `make build`
2. Test: `make test`
3. Deploy: `./scripts/deploy.sh <env>`
4. Verify: `./scripts/test-api.sh <endpoint>`

### Blue/Green Deployment
- Lambda versions and aliases
- API Gateway stages
- Zero-downtime deployments

## Local Development

### Testing Components
- DynamoDB Local for database testing
- LocalStack for SQS testing
- Unit tests with mocks
- Integration tests with local services

### Running Locally
```bash
# Start local DynamoDB
make local-dynamodb

# Start local SQS
make local-sqs

# Run tests
make test

# Run integration tests
make integration-test
```

## Best Practices

### Idempotency
- Always use unique idempotency keys
- Keys should be deterministic for retries
- DynamoDB conditional writes prevent duplicates

### Error Handling
- Graceful degradation
- Comprehensive error logging
- Dead letter queues for failed messages

### Performance
- Minimal API handler logic (< 1s)
- Async processing for heavy work
- Efficient DynamoDB access patterns

### Monitoring
- Log all state transitions
- Track payment lifecycle metrics
- Alert on anomalies
