# Production Scaling & Operations Guide

## Overview

This document outlines strategies for scaling the Crypto Conversion Payment API to production workloads, handling high concurrency, and maintaining reliability at scale.

---

## Architecture Scalability

### Current Architecture Benefits

The system is built on AWS serverless architecture, which provides inherent scalability:

```
API Gateway → Lambda (API) → DynamoDB + SQS
                ↓
            Lambda (Worker) → Mock On/Off Ramp
                ↓
            Lambda (Webhook) → Webhook Delivery
```

**Key Advantages**:
- **Auto-scaling**: All components scale automatically
- **Pay-per-use**: No idle resource costs
- **High availability**: Multi-AZ by default
- **Managed infrastructure**: AWS handles patches, updates

---

## Concurrency Handling

### API Layer (API Gateway + Lambda)

**Current Limits**:
- API Gateway: 10,000 requests/second (soft limit, can increase)
- Lambda concurrent executions: 1,000 (account-level, can increase to 10,000+)
- Lambda per-function concurrency: Unreserved (shares account pool)

**Scaling Strategies**:

#### 1. Reserved Concurrency
```hcl
# terraform/modules/lambda/main.tf
resource "aws_lambda_function" "api_handler" {
  # ... existing config ...

  reserved_concurrent_executions = 500  # Reserve capacity for API handler
}
```

**When to use**:
- Predictable traffic patterns
- Need guaranteed capacity
- Prevent one function from consuming all quota

#### 2. Provisioned Concurrency (for ultra-low latency)
```hcl
resource "aws_lambda_provisioned_concurrency_config" "api_handler" {
  function_name                     = aws_lambda_function.api_handler.function_name
  provisioned_concurrent_executions = 10
  qualifier                         = aws_lambda_function.api_handler.version
}
```

**When to use**:
- Eliminate cold starts (< 10ms response time)
- Traffic spikes (Black Friday, payroll days)
- Cost: ~$4/month per concurrent execution

#### 3. Throttle Protection
```hcl
# API Gateway throttling (already configured)
throttle_settings {
  burst_limit = 100   # Requests allowed in burst
  rate_limit  = 50    # Steady state req/sec
}
```

**Production recommendations**:
- **Burst**: 1,000-5,000 req/sec
- **Rate**: 500-2,000 req/sec
- **Per-client limits**: Use API keys + usage plans

---

## Database Scaling (DynamoDB)

### Current Configuration
- **Billing Mode**: PAY_PER_REQUEST (on-demand)
- **Capacity**: Scales automatically to workload
- **Indexes**: 1 GSI on `idempotency_key`

### Scaling Considerations

#### 1. Read/Write Patterns

**Current Usage**:
```
API Handler:  1 read (idempotency check) + 1 write per request
Worker:       1 read + 2 writes per payment
Webhook:      1 read per webhook
```

**At 1,000 payments/second**:
- **Writes**: ~4,000 WCU/sec
- **Reads**: ~2,000 RCU/sec

#### 2. Hot Partition Prevention

**Risk**: All new payments use `payment_id` as PK, could create hot partition

**Mitigation**:
```
✅ UUIDs as payment_id (random distribution)
✅ GSI on idempotency_key (separate partition key)
⚠️ Consider composite keys for time-series queries (future)
```

#### 3. Provisioned Capacity (for cost savings at scale)

**When to switch from on-demand to provisioned**:
- Predictable traffic (> 10,000 consistent RCU/WCU)
- Cost optimization (provisioned is ~50% cheaper at steady load)

```hcl
resource "aws_dynamodb_table" "payments" {
  billing_mode = "PROVISIONED"
  read_capacity  = 5000
  write_capacity = 5000

  # Enable auto-scaling
}

resource "aws_appautoscaling_target" "dynamodb_table_read_target" {
  max_capacity       = 40000
  min_capacity       = 5000
  resource_id        = "table/${aws_dynamodb_table.payments.name}"
  scalable_dimension = "dynamodb:table:ReadCapacityUnits"
  service_namespace  = "dynamodb"
}
```

#### 4. Caching Strategy

**For read-heavy workloads**:
```
API Gateway → Lambda → [DAX Cache] → DynamoDB
                           ↑
                   Microsecond latency
                   10x throughput increase
```

**DAX (DynamoDB Accelerator)**:
- In-memory cache
- Sub-millisecond reads
- Cost: ~$0.25/hour (t3.small)

**When to use**:
- Read-heavy queries (payment status checks)
- High-frequency idempotency lookups
- Need < 1ms response times

---

## Queue Scaling (SQS)

### Current Configuration
- **Visibility Timeout**: 300s (payment queue), 60s (webhook queue)
- **Message Retention**: 14 days (payment), 4 days (webhook)
- **Long Polling**: 20s (reduces empty receives)
- **Dead Letter Queues**: 3 retries (payment), 5 retries (webhook)

### Scaling Strategies

#### 1. Processing Throughput

**Current**:
- Worker processes 1 message at a time (`batch_size = 1`)
- Lambda concurrency determines throughput

**At scale**:
```hcl
resource "aws_lambda_event_source_mapping" "worker_sqs" {
  batch_size                         = 10     # Process 10 payments in parallel
  maximum_batching_window_in_seconds = 1      # Wait max 1s to fill batch

  scaling_config {
    maximum_concurrency = 100  # Max concurrent Lambda instances
  }
}
```

**Throughput calculation**:
```
Batch size: 10
Concurrency: 100
Processing time: 5s per payment

Throughput = (10 × 100) / 5 = 200 payments/second
```

#### 2. Back-Pressure Handling

**Scenario**: On/off-ramp providers slow down

**Solution**: SQS automatically handles back-pressure
```
Messages queue up → Lambda retries → DLQ after max attempts
```

**Monitoring alerts**:
```
ApproximateAgeOfOldestMessage > 5 minutes → Alert
ApproximateNumberOfMessagesVisible > 10,000 → Alert
```

#### 3. FIFO Queues (if ordering matters)

**Standard Queue (current)**:
- Best-effort ordering
- At-least-once delivery
- Unlimited throughput

**FIFO Queue (optional)**:
- Strict ordering per user
- Exactly-once processing
- 3,000 msg/sec limit (300 with batching)

**Use case**: Per-user payment sequencing

---

## Error Handling & Retries

### Retry Strategy

#### 1. Lambda Retries (SQS Triggered)
```
Attempt 1: Immediate
Attempt 2: After visibility timeout (300s)
Attempt 3: After visibility timeout
→ Move to DLQ
```

#### 2. DLQ Processing
```python
# Pseudocode for DLQ processor
def process_dlq_message(message):
    reason = analyze_failure(message)

    if reason == "transient_error":
        requeue_with_delay(message, delay=600)
    elif reason == "provider_down":
        alert_ops_team()
        requeue_with_delay(message, delay=3600)
    else:
        mark_payment_failed()
        send_refund_notification()
```

#### 3. Circuit Breaker Pattern

**For on/off-ramp providers**:
```go
type CircuitBreaker struct {
    failures      int
    threshold     int
    timeout       time.Duration
    state         State  // CLOSED, OPEN, HALF_OPEN
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == OPEN {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = HALF_OPEN
        } else {
            return ErrCircuitOpen
        }
    }

    err := fn()
    if err != nil {
        cb.failures++
        if cb.failures >= cb.threshold {
            cb.state = OPEN
            cb.lastFailure = time.Now()
        }
        return err
    }

    cb.failures = 0
    cb.state = CLOSED
    return nil
}
```

---

## Monitoring & Alerting

### Key Metrics to Track

#### 1. API Performance
```
✅ Latency (p50, p95, p99)
✅ Error rate (4xx, 5xx)
✅ Request count
✅ Concurrent executions
```

**CloudWatch Alarms**:
```hcl
resource "aws_cloudwatch_metric_alarm" "api_error_rate" {
  alarm_name          = "crypto-conversion-api-errors-dev"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "5XXError"
  namespace           = "AWS/ApiGateway"
  period              = 60
  statistic           = "Sum"
  threshold           = 10  # 10 errors in 1 minute
  alarm_description   = "API Gateway 5XX errors"

  dimensions = {
    ApiName = "crypto-conversion-api-dev"
  }
}
```

#### 2. Payment Processing
```
✅ Payment success rate
✅ Average processing time
✅ Queue depth
✅ DLQ message count
```

#### 3. Database Health
```
✅ Read/Write throttles (should be 0)
✅ User errors (validation failures)
✅ System errors (AWS issues)
```

### Logging Strategy

**Structured Logging** (already implemented):
```json
{
  "level": "INFO",
  "timestamp": "2025-10-19T01:23:38Z",
  "message": "Fee calculated for payment",
  "payment_id": "9a586bc5-d753-4754-86f3-897b4e8a043f",
  "base_amount": 5000,
  "fee_amount": 175,
  "total_amount": 5175
}
```

**Log Aggregation**:
- **Current**: CloudWatch Logs (7-day retention)
- **Production**: Ship to centralized logging (Datadog, New Relic, ELK)

**Log Sampling** (for cost control):
```go
// Sample 10% of successful requests, 100% of errors
if err != nil || rand.Float32() < 0.1 {
    logger.Info("Request processed", fields)
}
```

---

## Rate Limiting Strategy

### Multi-Layer Rate Limiting

#### 1. API Gateway (Already configured)
```
Burst: 100 req/sec
Rate:  50 req/sec
Daily: 10,000 requests
```

#### 2. Per-User Rate Limiting (Recommended)

**Using API Keys**:
```hcl
resource "aws_api_gateway_api_key" "user_key" {
  name = "user-${var.user_id}"
}

resource "aws_api_gateway_usage_plan" "per_user" {
  name = "per-user-plan"

  quota_settings {
    limit  = 1000   # 1,000 payments per day
    period = "DAY"
  }

  throttle_settings {
    burst_limit = 20   # 20 req/sec burst
    rate_limit  = 10   # 10 req/sec steady
  }
}
```

#### 3. Application-Level Rate Limiting

**Using DynamoDB**:
```go
type RateLimiter struct {
    db *dynamodb.Client
}

func (rl *RateLimiter) AllowRequest(userID string, limit int, window time.Duration) bool {
    key := fmt.Sprintf("rate_limit:%s:%d", userID, time.Now().Unix() / int64(window.Seconds()))

    // Atomic increment
    count, err := rl.db.IncrementCounter(key, 1, window)
    if err != nil {
        return false
    }

    return count <= limit
}
```

---

## Cost Optimization

### Current Costs (Development)
- DynamoDB: ~$2/month (on-demand, low usage)
- Lambda: ~$1/month (free tier)
- API Gateway: ~$0/month (free tier)
- SQS: ~$0/month (free tier)
- **Total**: ~$3-5/month

### Production Cost Projections

#### Scenario: 1M payments/month (33,000/day)

**Breakdown**:
```
API Gateway:
  1M requests × $3.50/million = $3.50

Lambda (API):
  1M invocations × $0.20/million = $0.20
  1M × 200ms × 512MB = 100,000 GB-seconds
  100,000 × $0.0000166667 = $1.67

Lambda (Worker):
  1M × 2s × 512MB = 1,000,000 GB-seconds
  1,000,000 × $0.0000166667 = $16.67

Lambda (Webhook):
  1M × 500ms × 256MB = 125,000 GB-seconds
  125,000 × $0.0000166667 = $2.08

DynamoDB (on-demand):
  4M writes × $1.25/million = $5.00
  2M reads × $0.25/million = $0.50

SQS:
  2M requests × $0.40/million = $0.80

CloudWatch Logs:
  10GB ingested × $0.50/GB = $5.00

Total: ~$35/month
```

### Cost Optimization Strategies

#### 1. Switch to Provisioned Capacity (at scale)
```
Savings: ~50% on DynamoDB
When: > 10M requests/month
```

#### 2. S3 for Cold Data
```
Move payments older than 90 days to S3
Cost: $0.023/GB (vs $0.25/GB in DynamoDB)
```

#### 3. Log Retention Policies
```
Keep 7 days in CloudWatch ($0.50/GB)
Archive to S3 ($0.023/GB)
Lifecycle to Glacier after 1 year ($0.004/GB)
```

#### 4. Reserved Capacity (Savings Plans)
```
Compute Savings Plans: 17-72% off Lambda
1-year commitment
```

---

## Security Hardening

### Current Security Posture
✅ IAM roles (no access keys)
✅ DynamoDB encryption at rest
✅ TLS 1.2+ for all communications
✅ Idempotency key validation
✅ Input validation

### Production Security Enhancements

#### 1. API Authentication
```
Options:
- API Keys (simple, already supported by API Gateway)
- AWS IAM (for B2B integrations)
- OAuth 2.0 / JWT (for B2C)
- Cognito (managed user pools)
```

#### 2. VPC Integration
```hcl
resource "aws_lambda_function" "api_handler" {
  # ... existing config ...

  vpc_config {
    subnet_ids         = var.private_subnet_ids
    security_group_ids = [aws_security_group.lambda.id]
  }
}
```

**Benefits**:
- Isolate Lambdas from internet
- Connect to private RDS/ElastiCache
- Enhanced network security

**Costs**:
- NAT Gateway: ~$32/month
- ENI charges: ~$3.60/month per function

#### 3. WAF (Web Application Firewall)
```hcl
resource "aws_wafv2_web_acl" "api_protection" {
  name  = "crypto-conversion-api-waf"
  scope = "REGIONAL"

  default_action {
    allow {}
  }

  rule {
    name     = "RateLimitRule"
    priority = 1

    statement {
      rate_based_statement {
        limit              = 2000
        aggregate_key_type = "IP"
      }
    }

    action {
      block {}
    }
  }

  rule {
    name     = "AWSManagedRulesCommonRuleSet"
    priority = 2

    override_action {
      none {}
    }

    statement {
      managed_rule_group_statement {
        vendor_name = "AWS"
        name        = "AWSManagedRulesCommonRuleSet"
      }
    }
  }
}
```

**Cost**: ~$5/month base + $1 per million requests

#### 4. Secrets Management
```hcl
resource "aws_secretsmanager_secret" "onramp_api_key" {
  name = "crypto-conversion/onramp/api-key"

  rotation_rules {
    automatically_after_days = 30
  }
}
```

**Access in Lambda**:
```go
func getOnRampAPIKey(ctx context.Context) (string, error) {
    svc := secretsmanager.New(session.New())
    result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
        SecretId: aws.String("crypto-conversion/onramp/api-key"),
    })
    return *result.SecretString, err
}
```

---

## Disaster Recovery

### Backup Strategy

#### 1. DynamoDB Point-in-Time Recovery
```hcl
resource "aws_dynamodb_table" "payments" {
  # ... existing config ...

  point_in_time_recovery {
    enabled = true  # Already set to false in dev
  }
}
```

**Production settings**:
- Enable PITR (recover to any point in last 35 days)
- Cost: ~20% of table storage (~$0.20/GB)

#### 2. Cross-Region Replication
```hcl
resource "aws_dynamodb_global_table" "payments" {
  name = "crypto-conversion-payments-prod"

  replica {
    region_name = "us-west-1"
  }

  replica {
    region_name = "us-east-1"  # DR region
  }
}
```

#### 3. Lambda Versioning
```hcl
resource "aws_lambda_function" "api_handler" {
  # ... existing config ...

  publish = true  # Create version on each deployment
}

resource "aws_lambda_alias" "live" {
  name             = "live"
  function_name    = aws_lambda_function.api_handler.arn
  function_version = aws_lambda_function.api_handler.version
}
```

**Rollback**:
```bash
aws lambda update-alias \
  --function-name crypto-conversion-api-handler-prod \
  --name live \
  --function-version 42  # Previous working version
```

---

## Deployment Strategy

### Blue/Green Deployments

```hcl
resource "aws_lambda_alias" "blue" {
  name             = "blue"
  function_name    = aws_lambda_function.api_handler.arn
  function_version = "42"
}

resource "aws_lambda_alias" "green" {
  name             = "green"
  function_name    = aws_lambda_function.api_handler.arn
  function_version = "43"  # New version
}

# Gradually shift traffic
resource "aws_lambda_alias" "live" {
  name             = "live"
  function_name    = aws_lambda_function.api_handler.arn
  function_version = aws_lambda_alias.green.function_version

  routing_config {
    additional_version_weights = {
      "${aws_lambda_alias.blue.function_version}" = 0.1  # 10% traffic
    }
  }
}
```

**Deployment process**:
1. Deploy new version (green)
2. Route 10% traffic to green
3. Monitor error rates
4. Gradually increase to 50%, 90%, 100%
5. Rollback to blue if errors spike

---

## Performance Optimization

### Cold Start Mitigation

**Current cold start**: ~300ms

**Optimization strategies**:

#### 1. Provisioned Concurrency
```hcl
resource "aws_lambda_provisioned_concurrency_config" "api_handler" {
  function_name                     = aws_lambda_function.api_handler.function_name
  provisioned_concurrent_executions = 5  # Keep 5 warm
  qualifier                         = aws_lambda_alias.live.name
}
```

**Result**: < 10ms latency

#### 2. Package Size Optimization
```bash
# Current sizes:
api-handler.zip:     4.5 MB
worker-handler.zip:  4.5 MB
webhook-handler.zip: 3.1 MB

# Optimization:
- Use Go 1.21+ (smaller binaries)
- Strip debug symbols: go build -ldflags="-s -w"
- Use UPX compression (30-50% size reduction)
```

#### 3. VPC Optimization
If using VPC:
```
Problem: VPC cold starts can add 10+ seconds
Solution: Use Hyperplane ENIs (automatic in newer runtimes)
```

---

## Summary: Production Checklist

### High Priority
- [ ] Enable DynamoDB Point-in-Time Recovery
- [ ] Set up CloudWatch alarms for errors, latency, queue depth
- [ ] Implement WAF for DDoS protection
- [ ] Add API authentication (API keys minimum)
- [ ] Configure cross-region backups
- [ ] Set up centralized logging (Datadog/New Relic)
- [ ] Implement circuit breaker for on/off-ramp providers
- [ ] Create runbooks for common incidents

### Medium Priority
- [ ] Switch to provisioned DynamoDB capacity (if > 10M req/month)
- [ ] Implement DAX caching for hot reads
- [ ] Set up multi-region failover
- [ ] Enable Lambda versioning and blue/green deployments
- [ ] Implement per-user rate limiting
- [ ] Create comprehensive monitoring dashboards

### Low Priority (Nice to Have)
- [ ] VPC integration for enhanced security
- [ ] Provisioned concurrency for ultra-low latency
- [ ] S3 archival for old payments
- [ ] Automated chaos engineering tests
- [ ] Custom CloudWatch metrics for business KPIs

---

## Contact for Scaling Questions

For questions about scaling this system to production:
- Review AWS Well-Architected Framework
- Consult AWS Solutions Architects
- Load test with realistic traffic patterns
- Monitor closely during initial rollout
