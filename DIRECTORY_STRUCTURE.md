# Complete Directory Structure

```
crypto_conversion/
│
├── cmd/                                    # Lambda function entry points
│   ├── api-handler/
│   │   └── main.go                        # API Gateway request handler
│   ├── worker-handler/
│   │   └── main.go                        # SQS payment job processor
│   └── webhook-handler/
│       └── main.go                        # Webhook notification sender
│
├── internal/                               # Private application packages
│   ├── config/
│   │   └── config.go                      # Configuration management
│   ├── database/
│   │   └── dynamodb.go                    # DynamoDB client and operations
│   ├── errors/
│   │   └── errors.go                      # Custom error types
│   ├── logger/
│   │   └── logger.go                      # Structured logging
│   ├── models/
│   │   └── payment.go                     # Data models and DTOs
│   ├── payment/
│   │   └── orchestrator.go                # Payment processing orchestration
│   ├── queue/
│   │   └── sqs.go                         # SQS client and operations
│   └── validator/
│       └── validator.go                   # Request validation logic
│
├── infrastructure/                         # Infrastructure as Code
│   └── terraform/
│       ├── main.tf                        # Root Terraform configuration
│       ├── variables.tf                   # Input variables
│       ├── environments/
│       │   ├── dev.tfvars                # Development environment config
│       │   └── prod.tfvars               # Production environment config
│       └── modules/
│           ├── api-gateway/              # API Gateway module
│           │   ├── main.tf
│           │   ├── variables.tf
│           │   └── outputs.tf
│           └── lambda/                   # Lambda functions module
│               ├── main.tf
│               ├── variables.tf
│               └── outputs.tf
│
├── tests/                                  # Test suites
│   ├── unit/
│   │   └── validator_test.go             # Unit tests for validator
│   ├── integration/                       # Integration tests (to be added)
│   └── mocks/                            # Mock implementations (to be added)
│
├── docs/                                   # Documentation
│   ├── architecture.md                    # System architecture overview
│   ├── api-reference.md                   # API endpoint documentation
│   └── deployment-guide.md                # Deployment instructions
│
├── scripts/                                # Deployment and utility scripts
│   ├── deploy.sh                          # Main deployment script
│   ├── test-api.sh                        # API testing script
│   └── destroy.sh                         # Infrastructure teardown script
│
├── build/                                  # Build artifacts (generated)
│   ├── api-handler.zip
│   ├── worker-handler.zip
│   └── webhook-handler.zip
│
├── .gitignore                              # Git ignore rules
├── go.mod                                  # Go module definition
├── go.sum                                  # Go module checksums
├── Makefile                                # Build automation
├── README.md                               # Project overview
├── spec.md                                 # Original specification
└── DIRECTORY_STRUCTURE.md                  # This file
```

## Component Descriptions

### `/cmd` - Application Entry Points
Contains the `main.go` files for each Lambda function. These are kept minimal, focusing on wiring dependencies and starting the Lambda runtime.

### `/internal` - Private Application Code
All reusable business logic, organized by domain:

- **config**: Environment variable loading and configuration management
- **database**: DynamoDB operations (CRUD, queries, updates)
- **errors**: Custom error types with HTTP status codes
- **logger**: Structured JSON logging for CloudWatch
- **models**: Data models, DTOs, and type definitions
- **payment**: Core payment processing orchestration logic
- **queue**: SQS message sending and receiving
- **validator**: Request validation and business rules

### `/infrastructure` - Terraform IaC
Complete infrastructure definition using Terraform:

- **Root Module**: Main configuration, DynamoDB, SQS queues
- **Lambda Module**: All Lambda functions, IAM roles, event mappings
- **API Gateway Module**: REST API, routes, CORS, logging
- **Environments**: Environment-specific variable files

### `/tests` - Test Suites
Test code organized by type:

- **unit**: Fast, isolated tests with no external dependencies
- **integration**: Tests against local AWS services (DynamoDB Local, LocalStack)
- **mocks**: Mock implementations for testing

### `/docs` - Documentation
Comprehensive documentation:

- **architecture.md**: System design, data flow, scalability
- **api-reference.md**: API endpoints, request/response formats
- **deployment-guide.md**: Step-by-step deployment instructions

### `/scripts` - Automation Scripts
Bash scripts for common operations:

- **deploy.sh**: Build and deploy to AWS
- **test-api.sh**: Automated API testing
- **destroy.sh**: Clean teardown of infrastructure

### `/build` - Build Artifacts
Generated directory containing Lambda deployment packages (ZIP files).

## File Sizes (Approximate)

```
go.mod                          ~800 bytes
Makefile                        ~3 KB
README.md                       ~4 KB
spec.md                         ~2 KB

internal/config/config.go       ~2 KB
internal/database/dynamodb.go   ~7 KB
internal/errors/errors.go       ~4 KB
internal/logger/logger.go       ~5 KB
internal/models/payment.go      ~2 KB
internal/payment/orchestrator.go ~5 KB
internal/queue/sqs.go           ~3 KB
internal/validator/validator.go ~3 KB

cmd/api-handler/main.go         ~5 KB
cmd/worker-handler/main.go      ~6 KB
cmd/webhook-handler/main.go     ~4 KB

infrastructure/terraform/main.tf                      ~6 KB
infrastructure/terraform/modules/lambda/main.tf       ~7 KB
infrastructure/terraform/modules/api-gateway/main.tf  ~6 KB

docs/architecture.md            ~12 KB
docs/api-reference.md           ~10 KB
docs/deployment-guide.md        ~15 KB

Build artifacts (each):         ~5-6 MB (compressed)
```

## Key Design Principles

### 1. Separation of Concerns
- Entry points (`cmd/`) are thin wrappers
- Business logic lives in `internal/`
- Infrastructure is separate from application code

### 2. Dependency Direction
- `cmd/` depends on `internal/`
- `internal/` packages are loosely coupled
- No circular dependencies

### 3. Testability
- Interfaces for external dependencies (database, queue)
- Mock implementations for testing
- Unit tests can run without AWS

### 4. Infrastructure as Code
- Complete infrastructure in Terraform
- Environment-specific configurations
- Reproducible deployments

### 5. Observability
- Structured logging throughout
- CloudWatch integration
- Metrics and tracing support

## Build Process

The build process creates self-contained Lambda deployment packages:

```bash
make build
```

For each Lambda function:
1. Compiles Go code with `GOOS=linux GOARCH=amd64`
2. Creates a binary named `bootstrap` (for custom runtime)
3. Packages the binary into a ZIP file
4. Places ZIP in `build/` directory

## Deployment Flow

```
1. make build
   ↓
2. terraform init
   ↓
3. terraform plan
   ↓
4. terraform apply
   ↓
5. Resources created:
   - DynamoDB table
   - SQS queues
   - Lambda functions
   - API Gateway
   - IAM roles
   - CloudWatch logs
```

## Next Steps

To start development:

1. Review [README.md](README.md) for project overview
2. Read [docs/architecture.md](docs/architecture.md) to understand the system
3. Follow [docs/deployment-guide.md](docs/deployment-guide.md) to deploy
4. Use [docs/api-reference.md](docs/api-reference.md) for API details

To add new features:

1. Add models to `internal/models/`
2. Implement business logic in appropriate `internal/` package
3. Update Lambda handlers in `cmd/`
4. Add tests in `tests/`
5. Update Terraform if infrastructure changes needed
6. Update documentation
