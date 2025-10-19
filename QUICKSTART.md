# Quick Start Guide

Deploy the Crypto Conversion Payment API in minutes.

## Prerequisites

```bash
go version    # 1.21+
aws --version # AWS CLI 2.x
terraform version # 1.0+
aws sts get-caller-identity # Verify AWS credentials
```

## Deploy

```bash
# Build
make build

# Deploy to AWS
cd infrastructure/terraform
terraform init
terraform apply -var-file=environments/dev.tfvars
```

Save the API endpoint from the output.

## Test

### Create Quote
```bash
curl -X POST https://YOUR-API/quotes \
  -H "Content-Type: application/json" \
  -d '{"from_currency": "USD", "to_currency": "EUR", "amount": 100000}'
```

### Create Payment
```bash
curl -X POST https://YOUR-API/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{
    "amount": 100000,
    "currency": "EUR",
    "source_account": "user123",
    "destination_account": "merchant456",
    "quote_id": "QUOTE_ID_FROM_ABOVE"
  }'
```

## Monitor

```bash
# View payment in DynamoDB
aws dynamodb scan --table-name crypto-conversion-payments-dev --limit 5

# View Lambda logs
aws logs tail /aws/lambda/crypto-conversion-worker-handler-dev --follow
```

## Troubleshooting

**AWS credentials not found:** Run `aws configure`

**Build fails:** Ensure Go 1.21+ installed: `brew install go`

**Lambda errors:** Check CloudWatch logs: `aws logs tail /aws/lambda/crypto-conversion-api-handler-dev --follow`

## Clean Up

```bash
cd infrastructure/terraform
terraform destroy -var-file=environments/dev.tfvars
```

## Documentation

- [README.md](README.md) - Project overview
- [docs/architecture.md](docs/architecture.md) - System design
- [docs/api-reference.md](docs/api-reference.md) - API docs
- [docs/deployment-guide.md](docs/deployment-guide.md) - Deployment details
