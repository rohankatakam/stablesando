# Deployment Guide

This guide walks you through deploying the Crypto Conversion Payment API to AWS.

## Prerequisites

### Required Software

1. **Go 1.21+**
   ```bash
   go version
   ```

2. **AWS CLI**
   ```bash
   aws --version
   ```

3. **Terraform 1.0+**
   ```bash
   terraform version
   ```

4. **Make**
   ```bash
   make --version
   ```

### AWS Account Setup

1. **AWS Account**: You need an active AWS account

2. **IAM Permissions**: Your AWS user/role needs permissions for:
   - Lambda (create, update, delete functions)
   - API Gateway (create, update APIs)
   - DynamoDB (create, update tables)
   - SQS (create, manage queues)
   - IAM (create roles and policies)
   - CloudWatch (create log groups)
   - S3 (for Terraform state - optional)

3. **AWS Credentials**: Configure your credentials
   ```bash
   aws configure
   ```

## Quick Start

### 1. Clone and Setup

```bash
cd /path/to/crypto_conversion
make deps
```

### 2. Build Lambda Functions

```bash
make build
```

This creates deployment packages in the `build/` directory:
- `api-handler.zip`
- `worker-handler.zip`
- `webhook-handler.zip`

### 3. Deploy to Development

```bash
./scripts/deploy.sh dev
```

This will:
- Build all Lambda functions
- Initialize Terraform
- Create all AWS resources
- Output the API endpoint URL

### 4. Test the Deployment

```bash
./scripts/test-api.sh <API_ENDPOINT_FROM_STEP_3>
```

Example:
```bash
./scripts/test-api.sh https://abc123.execute-api.us-east-1.amazonaws.com/dev/payments
```

## Detailed Deployment Steps

### Step 1: Configure Terraform Backend (Optional)

For production deployments, store Terraform state in S3:

1. Create an S3 bucket:
   ```bash
   aws s3 mb s3://your-terraform-state-bucket
   ```

2. Enable versioning:
   ```bash
   aws s3api put-bucket-versioning \
     --bucket your-terraform-state-bucket \
     --versioning-configuration Status=Enabled
   ```

3. Update `infrastructure/terraform/main.tf`:
   ```hcl
   backend "s3" {
     bucket = "your-terraform-state-bucket"
     key    = "crypto-conversion/terraform.tfstate"
     region = "us-east-1"
   }
   ```

### Step 2: Review Environment Configuration

Edit environment-specific variables in `infrastructure/terraform/environments/`:

**Development** (`dev.tfvars`):
```hcl
environment                   = "dev"
aws_region                    = "us-east-1"
log_retention_days            = 7
enable_point_in_time_recovery = false
lambda_timeout                = 30
lambda_memory_size            = 512
```

**Production** (`prod.tfvars`):
```hcl
environment                   = "prod"
aws_region                    = "us-east-1"
log_retention_days            = 30
enable_point_in_time_recovery = true
lambda_timeout                = 30
lambda_memory_size            = 1024
```

### Step 3: Build

```bash
# Clean previous builds
make clean

# Build all functions
make build

# Verify build artifacts
ls -lh build/
```

Expected output:
```
-rw-r--r--  1 user  staff   5.2M Jan 15 10:00 api-handler.zip
-rw-r--r--  1 user  staff   5.3M Jan 15 10:00 worker-handler.zip
-rw-r--r--  1 user  staff   5.1M Jan 15 10:00 webhook-handler.zip
```

### Step 4: Deploy Infrastructure

#### Using the Deployment Script

```bash
# Deploy to dev
./scripts/deploy.sh dev

# Deploy to prod
./scripts/deploy.sh prod us-east-1
```

#### Manual Deployment

```bash
cd infrastructure/terraform

# Initialize
terraform init

# Plan
terraform plan -var-file=environments/dev.tfvars -out=tfplan

# Review the plan output carefully

# Apply
terraform apply tfplan

# Get outputs
terraform output
```

### Step 5: Verify Deployment

1. **Check API Gateway**:
   ```bash
   aws apigateway get-rest-apis
   ```

2. **Check Lambda Functions**:
   ```bash
   aws lambda list-functions --query 'Functions[?contains(FunctionName, `crypto-conversion`)]'
   ```

3. **Check DynamoDB Table**:
   ```bash
   aws dynamodb describe-table --table-name crypto-conversion-payments-dev
   ```

4. **Check SQS Queues**:
   ```bash
   aws sqs list-queues | grep crypto-conversion
   ```

### Step 6: Test the API

Run the automated test suite:

```bash
# Get the API endpoint from Terraform outputs
API_ENDPOINT=$(cd infrastructure/terraform && terraform output -raw api_endpoint)

# Run tests
./scripts/test-api.sh $API_ENDPOINT
```

Expected output:
```
===================================
Testing Crypto Conversion API
Endpoint: https://abc123.execute-api.us-east-1.amazonaws.com/dev/payments
===================================

Test 1: Create a payment
✅ Test 1 PASSED: Payment accepted

Test 2: Duplicate idempotency key
✅ Test 2 PASSED: Duplicate request rejected

Test 3: Missing idempotency key
✅ Test 3 PASSED: Missing header rejected

Test 4: Invalid amount (negative)
✅ Test 4 PASSED: Invalid amount rejected
```

## Environment-Specific Deployments

### Development Environment

```bash
./scripts/deploy.sh dev
```

Characteristics:
- Lower memory allocation
- Shorter log retention
- No point-in-time recovery
- Faster iteration

### Production Environment

```bash
./scripts/deploy.sh prod
```

Characteristics:
- Higher memory allocation
- Longer log retention (30 days)
- Point-in-time recovery enabled
- Enhanced monitoring

## Updating the Deployment

### Code Changes

```bash
# Make code changes
# ...

# Build and deploy
make build
./scripts/deploy.sh dev
```

### Infrastructure Changes

```bash
# Edit Terraform files
# ...

cd infrastructure/terraform
terraform plan -var-file=environments/dev.tfvars
terraform apply -var-file=environments/dev.tfvars
```

## Monitoring Deployment

### View Lambda Logs

```bash
# API Handler logs
aws logs tail /aws/lambda/crypto-conversion-api-handler-dev --follow

# Worker logs
aws logs tail /aws/lambda/crypto-conversion-worker-handler-dev --follow

# Webhook logs
aws logs tail /aws/lambda/crypto-conversion-webhook-handler-dev --follow
```

### Check SQS Queue Metrics

```bash
# Payment queue
aws sqs get-queue-attributes \
  --queue-url $(cd infrastructure/terraform && terraform output -raw payment_queue_url) \
  --attribute-names All

# Check DLQ for failed messages
aws sqs get-queue-attributes \
  --queue-url $(cd infrastructure/terraform && terraform output -raw payment_queue_url | sed 's/payment-queue/payment-dlq/') \
  --attribute-names ApproximateNumberOfMessages
```

### Check DynamoDB

```bash
# Scan recent payments (dev only - be careful with large tables)
aws dynamodb scan \
  --table-name crypto-conversion-payments-dev \
  --limit 10
```

## Rollback

If you need to rollback to a previous version:

### Rollback Lambda Code

```bash
# List function versions
aws lambda list-versions-by-function \
  --function-name crypto-conversion-api-handler-dev

# Update alias to point to previous version
aws lambda update-alias \
  --function-name crypto-conversion-api-handler-dev \
  --name live \
  --function-version <previous-version>
```

### Rollback Infrastructure

```bash
cd infrastructure/terraform

# Revert to previous Terraform state
terraform state pull > current.tfstate
# Copy previous state
terraform state push previous.tfstate
terraform apply
```

## Destroying Resources

### Development Environment

```bash
./scripts/destroy.sh dev
```

### Manual Destruction

```bash
cd infrastructure/terraform
terraform destroy -var-file=environments/dev.tfvars
```

**Warning**: This will permanently delete:
- All Lambda functions
- DynamoDB table (and all payment data)
- SQS queues (and all messages)
- API Gateway
- CloudWatch logs

## Troubleshooting

### Build Failures

**Issue**: `make build` fails with Go compilation errors

**Solution**:
```bash
# Clean and rebuild
make clean
go mod download
go mod verify
make build
```

### Terraform Initialization Fails

**Issue**: `terraform init` fails

**Solution**:
```bash
cd infrastructure/terraform
rm -rf .terraform
terraform init -upgrade
```

### Lambda Function Not Found

**Issue**: Terraform can't find the Lambda zip files

**Solution**:
```bash
# Ensure you build before deploying
make build

# Verify zip files exist
ls -la build/*.zip
```

### API Gateway 502 Errors

**Issue**: API returns 502 Bad Gateway

**Possible Causes**:
1. Lambda function timeout
2. Lambda execution error
3. IAM permission issues

**Debug**:
```bash
# Check Lambda logs
aws logs tail /aws/lambda/crypto-conversion-api-handler-dev --follow

# Test Lambda directly
aws lambda invoke \
  --function-name crypto-conversion-api-handler-dev \
  --payload '{}' \
  response.json
```

### DynamoDB Access Denied

**Issue**: Lambda can't access DynamoDB

**Solution**:
```bash
# Verify IAM role has correct permissions
aws iam get-role-policy \
  --role-name crypto-conversion-api-handler-role-dev \
  --policy-name crypto-conversion-api-handler-policy-dev
```

### SQS Messages Not Processing

**Issue**: Messages stuck in queue

**Debug**:
```bash
# Check queue attributes
aws sqs get-queue-attributes \
  --queue-url <queue-url> \
  --attribute-names All

# Check Lambda event source mapping
aws lambda list-event-source-mappings \
  --function-name crypto-conversion-worker-handler-dev

# Check DLQ for failed messages
aws sqs receive-message \
  --queue-url <dlq-url>
```

## Best Practices

### Development

1. Always test locally before deploying
2. Use development environment for testing
3. Review Terraform plan before applying
4. Keep logs of deployments

### Production

1. Use Terraform state locking (S3 + DynamoDB)
2. Enable CloudWatch alarms
3. Set up SNS notifications for failures
4. Regular backups of DynamoDB
5. Use Lambda versions and aliases
6. Implement blue/green deployments
7. Monitor costs with AWS Cost Explorer

### Security

1. Never commit AWS credentials
2. Use IAM roles, not access keys
3. Enable CloudTrail for audit logging
4. Restrict API Gateway access
5. Use VPC endpoints for private APIs
6. Implement API authentication

## Next Steps

After successful deployment:

1. Review [Architecture Documentation](architecture.md)
2. Read [API Reference](api-reference.md)
3. Set up monitoring and alerts
4. Configure webhook endpoints
5. Implement authentication
6. Set up CI/CD pipeline
