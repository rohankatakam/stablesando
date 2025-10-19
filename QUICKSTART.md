# Quick Start Guide

Get the Crypto Conversion Payment API running in under 10 minutes!

## Prerequisites Check

```bash
# Check Go
go version
# Should show: go version go1.21 or higher

# Check AWS CLI
aws --version
# Should show: aws-cli/2.x.x

# Check Terraform
terraform version
# Should show: Terraform v1.0 or higher

# Check AWS credentials
aws sts get-caller-identity
# Should show your AWS account details
```

## 5-Minute Deployment

### 1. Install Dependencies (30 seconds)

```bash
cd /Users/rohankatakam/Documents/crypto_conversion
make deps
```

### 2. Build Lambda Functions (1 minute)

```bash
make build
```

You should see:
```
Building Lambda functions...
Building api-handler...
Building worker-handler...
Building webhook-handler...
Build complete! Artifacts in build/
```

### 3. Deploy to AWS (3-5 minutes)

```bash
./scripts/deploy.sh dev
```

Watch as it:
- ✅ Checks prerequisites
- ✅ Builds Lambda functions
- ✅ Initializes Terraform
- ✅ Creates infrastructure:
  - DynamoDB table
  - SQS queues
  - Lambda functions
  - API Gateway
  - IAM roles
  - CloudWatch logs

At the end, you'll see:
```
===================================
Deployment Outputs
===================================
API Endpoint: https://abc123xyz.execute-api.us-east-1.amazonaws.com/dev/payments
DynamoDB Table: crypto-conversion-payments-dev
Payment Queue: https://sqs.us-east-1.amazonaws.com/123456789012/crypto-conversion-payment-queue-dev
Webhook Queue: https://sqs.us-east-1.amazonaws.com/123456789012/crypto-conversion-webhook-queue-dev

===================================
Deployment completed successfully!
===================================
```

**Save the API Endpoint URL!** You'll need it for testing.

### 4. Test Your API (30 seconds)

```bash
./scripts/test-api.sh https://YOUR-API-ENDPOINT/payments
```

You should see all tests pass:
```
✅ Test 1 PASSED: Payment accepted
✅ Test 2 PASSED: Duplicate request rejected
✅ Test 3 PASSED: Missing header rejected
✅ Test 4 PASSED: Invalid amount rejected
```

## Your First API Call

### Using cURL

```bash
curl -X POST https://YOUR-API-ENDPOINT/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "amount": 100000,
    "currency": "EUR",
    "source_account": "user123",
    "destination_account": "merchant456"
  }'
```

Expected response:
```json
{
  "payment_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING",
  "message": "Payment accepted for processing"
}
```

### Using HTTPie (if installed)

```bash
http POST https://YOUR-API-ENDPOINT/payments \
  Idempotency-Key:$(uuidgen) \
  amount:=100000 \
  currency=EUR \
  source_account=user123 \
  destination_account=merchant456
```

## What Just Happened?

1. **API Gateway** received your request
2. **API Handler Lambda** validated it and created a payment record
3. Payment was queued for async processing
4. **Worker Lambda** processed the payment (on-ramp → off-ramp)
5. **Webhook Lambda** sent a notification (mocked in dev)

## View the Payment in DynamoDB

```bash
aws dynamodb scan \
  --table-name crypto-conversion-payments-dev \
  --limit 5
```

## View Lambda Logs

```bash
# API Handler logs
aws logs tail /aws/lambda/crypto-conversion-api-handler-dev --follow

# Worker logs (where processing happens)
aws logs tail /aws/lambda/crypto-conversion-worker-handler-dev --follow
```

## Check Queue Status

```bash
# How many messages waiting?
aws sqs get-queue-attributes \
  --queue-url $(cd infrastructure/terraform && terraform output -raw payment_queue_url) \
  --attribute-names ApproximateNumberOfMessages
```

## Common Issues & Solutions

### Issue: "AWS credentials not found"

**Solution:**
```bash
aws configure
# Enter your AWS Access Key ID, Secret Access Key, and region
```

### Issue: Build fails with "command not found: go"

**Solution:**
```bash
# Install Go from https://golang.org/dl/
# Or use Homebrew on macOS:
brew install go
```

### Issue: Terraform command not found

**Solution:**
```bash
# Install Terraform from https://www.terraform.io/downloads
# Or use Homebrew on macOS:
brew install terraform
```

### Issue: "Error creating Lambda function"

**Possible causes:**
1. Insufficient IAM permissions
2. Build artifacts not found

**Solution:**
```bash
# Ensure build completed successfully
make clean
make build
ls -la build/

# Check IAM permissions
aws iam get-user
```

### Issue: API returns 502 Bad Gateway

**Debug:**
```bash
# Check Lambda logs for errors
aws logs tail /aws/lambda/crypto-conversion-api-handler-dev --follow

# Test Lambda directly
aws lambda invoke \
  --function-name crypto-conversion-api-handler-dev \
  --payload file://test-event.json \
  response.json
```

## Development Workflow

### Make Code Changes

1. Edit files in `internal/` or `cmd/`
2. Run tests: `make test`
3. Rebuild: `make build`
4. Redeploy: `./scripts/deploy.sh dev`

### View Logs While Developing

```bash
# Terminal 1: API logs
aws logs tail /aws/lambda/crypto-conversion-api-handler-dev --follow

# Terminal 2: Worker logs
aws logs tail /aws/lambda/crypto-conversion-worker-handler-dev --follow

# Terminal 3: Make API calls
curl -X POST ...
```

## Clean Up (When Done Testing)

To remove all AWS resources and avoid charges:

```bash
./scripts/destroy.sh dev
```

Type `yes` when prompted. This will delete:
- All Lambda functions
- DynamoDB table
- SQS queues
- API Gateway
- IAM roles
- CloudWatch logs

**Warning:** All payment data will be permanently deleted!

## Next Steps

Now that you have the API running:

1. **Understand the Architecture**: Read [docs/architecture.md](docs/architecture.md)
2. **Explore the API**: Review [docs/api-reference.md](docs/api-reference.md)
3. **Deploy to Production**: Follow [docs/deployment-guide.md](docs/deployment-guide.md)
4. **Add Features**:
   - Implement real on-ramp/off-ramp integrations
   - Add authentication (API keys, OAuth)
   - Configure webhook endpoints
   - Set up monitoring and alerts

## Project Structure at a Glance

```
crypto_conversion/
├── cmd/                    # Lambda entry points
├── internal/               # Business logic
├── infrastructure/         # Terraform IaC
├── tests/                  # Test suites
├── docs/                   # Documentation
├── scripts/                # Deployment scripts
└── Makefile               # Build automation
```

## Useful Commands

```bash
# Build everything
make build

# Run tests
make test

# Run linter
make lint

# Format code
make format

# Clean build artifacts
make clean

# Deploy to dev
./scripts/deploy.sh dev

# Deploy to prod
./scripts/deploy.sh prod

# Destroy infrastructure
./scripts/destroy.sh dev

# Test API
./scripts/test-api.sh <endpoint>
```

## Getting Help

- Check [README.md](README.md) for project overview
- Read [docs/](docs/) for detailed documentation
- View [spec.md](spec.md) for original requirements
- Check [DIRECTORY_STRUCTURE.md](DIRECTORY_STRUCTURE.md) for code organization

## Support

For issues or questions:
1. Check the documentation in `docs/`
2. Review CloudWatch logs for errors
3. Verify AWS resource status in AWS Console

Happy coding! 🚀
