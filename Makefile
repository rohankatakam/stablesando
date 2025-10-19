.PHONY: help build test clean deploy lint format

# Variables
FUNCTIONS := api-handler worker-handler webhook-handler
BUILD_DIR := build
COVERAGE_FILE := coverage.out

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	go mod download
	go mod verify

build: clean ## Build all Lambda functions
	@echo "Building Lambda functions..."
	@mkdir -p $(BUILD_DIR)
	@for func in $(FUNCTIONS); do \
		echo "Building $$func..."; \
		GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$$func/bootstrap ./cmd/$$func; \
		cd $(BUILD_DIR)/$$func && zip -q ../$$func.zip bootstrap && cd ../..; \
	done
	@echo "Build complete! Artifacts in $(BUILD_DIR)/"

test: ## Run tests with coverage
	go test -v -race -coverprofile=$(COVERAGE_FILE) ./...
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-short: ## Run tests without coverage
	go test -v -short ./...

lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

format: ## Format code
	go fmt ./...
	gofmt -s -w .

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) coverage.html
	go clean

local-dynamodb: ## Start local DynamoDB for testing
	docker run -d -p 8000:8000 --name local-dynamodb amazon/dynamodb-local

local-sqs: ## Start local SQS for testing (using localstack)
	docker run -d -p 4566:4566 --name localstack localstack/localstack

deploy-dev: build ## Deploy to development environment
	cd infrastructure/terraform && terraform apply -var-file=environments/dev.tfvars

deploy-prod: build ## Deploy to production environment
	cd infrastructure/terraform && terraform apply -var-file=environments/prod.tfvars

integration-test: ## Run integration tests
	go test -v -tags=integration ./tests/integration/...

benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./...

docker-clean: ## Stop and remove local Docker containers
	-docker stop local-dynamodb localstack
	-docker rm local-dynamodb localstack

.DEFAULT_GOAL := help
