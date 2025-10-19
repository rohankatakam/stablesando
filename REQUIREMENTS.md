# Cross-Border Payment API - Requirements Document

## Assignment Overview
Build a REST API demonstrating USD → multi-currency payout flow with a fee engine (2-3 hours using AI tools).

## MVP Scope & Simplifications

### Currency Selection
- **Input**: USD only
- **Output**: EUR only (single destination currency for MVP)
- Future: Support multiple destination currencies

### Payment Flow Architecture
```
Client → API → Fee Calculation → Onramp (USD→USDC) → Offramp (USDC→EUR) → Webhook
```

## Core Requirements

### 1. REST API Endpoints

#### POST /payments
**Request:**
```json
{
  "amount": 100000,           // Amount in USD cents
  "currency": "EUR",          // Destination currency (EUR for MVP)
  "source_account": "user123",
  "destination_account": "merchant456"
}
```

**Headers:**
- `Idempotency-Key`: Required UUID for duplicate prevention

**Response (202 Accepted):**
```json
{
  "payment_id": "uuid",
  "status": "PENDING",
  "message": "Payment accepted for processing"
}
```

**Error Responses:**
- 400: Invalid input, missing idempotency key
- 409: Duplicate request (idempotency key exists)

### 2. Fee Engine

**Requirements:**
- Calculate fees based on:
  - Payment amount (tiered structure)
  - Destination currency/country
- Return fee breakdown in API response or webhook

**MVP Fee Structure:**
```
Amount < $100:     2.9% + $0.30
Amount < $1000:    2.5% + $0.50
Amount >= $1000:   2.0% + $1.00
```

### 3. Mock Integrations

#### Onramp Provider (USD → USDC)
- Mock: Bridge/Circle/Generic
- Simulate: USD collection + USDC minting
- Return: Transaction ID
- Latency: Simulate 1-2s delay

#### Offramp Provider (USDC → EUR)
- Mock: USDC conversion to EUR
- Simulate: Bank transfer initiation
- Return: Transaction ID
- Latency: Simulate 1-2s delay

### 4. Idempotency

**Requirements:**
- Prevent duplicate payments using `Idempotency-Key` header
- Store key with payment record
- Return 409 if key already exists
- Index on idempotency_key for fast lookups

### 5. Async Event Handling

**Requirements:**
- Decouple payment acceptance from processing
- Use queue for async processing
- Send webhook notifications on status changes

**Webhook Payload:**
```json
{
  "event_type": "payment.completed",
  "payment_id": "uuid",
  "status": "COMPLETED",
  "amount": 100000,
  "currency": "EUR",
  "fees": {
    "amount": 2900,
    "currency": "USD"
  },
  "on_ramp_tx_id": "onramp_123",
  "off_ramp_tx_id": "offramp_456",
  "processed_at": "2025-10-19T01:12:49Z"
}
```

### 6. Data Model

**Payment Entity:**
```
- payment_id (PK)
- idempotency_key (indexed)
- status (PENDING, PROCESSING, COMPLETED, FAILED)
- amount
- currency
- source_account
- destination_account
- fees (calculated amount)
- on_ramp_tx_id
- off_ramp_tx_id
- created_at
- updated_at
- processed_at
```

## Deliverables Checklist

### 1. Working Code
- [x] REST API implementation
- [ ] Fee engine implementation
- [x] Mock onramp integration
- [x] Mock offramp integration
- [x] Idempotency handling
- [x] Webhook event system
- [x] Async processing (queue-based)

### 2. README with Setup
- [x] Installation prerequisites
- [x] Setup instructions
- [x] Deployment guide
- [x] API usage examples
- [ ] Fee calculation examples

### 3. Production Scaling Notes
- [ ] Concurrency handling
- [ ] Database scaling strategy
- [ ] Queue scaling approach
- [ ] Error handling & retries
- [ ] Monitoring & alerting
- [ ] Rate limiting strategy

## Evaluation Criteria

### 1. API Design & Payment Flow Orchestration
- RESTful design principles
- Proper HTTP status codes
- Clear request/response contracts
- Async processing architecture

### 2. Onramp/Offramp Architecture Understanding
- Separation of concerns
- Mock implementations demonstrating flow
- Transaction ID tracking
- Error handling at integration points

### 3. Handling at Scale
- Idempotency for reliability
- Queue-based async processing
- Database indexing strategy
- Retry mechanisms (DLQ)
- Monitoring capabilities

### 4. Code Quality & AI Tool Usage
- Clean, readable code
- Proper error handling
- Structured logging
- Infrastructure as Code
- Documentation

## MVP Technical Stack (Current)
- **Language**: Go
- **Cloud**: AWS (Lambda, API Gateway, DynamoDB, SQS)
- **Architecture**: Serverless, event-driven
- **IaC**: Terraform
- **Region**: us-west-1

## Out of Scope for MVP
- Multiple destination currencies (only EUR)
- Payment status query endpoint (GET /payments/:id)
- Authentication/authorization
- Real onramp/offramp integrations
- Currency conversion rate APIs
- Refunds/cancellations
- Historical payment queries
- User account management
