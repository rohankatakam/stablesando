#!/bin/bash

set -e

# Deployment script for crypto-conversion API

ENVIRONMENT=${1:-dev}
AWS_REGION=${2:-us-east-1}

echo "==================================="
echo "Crypto Conversion API Deployment"
echo "Environment: $ENVIRONMENT"
echo "Region: $AWS_REGION"
echo "==================================="

# Check prerequisites
check_prerequisites() {
    echo "Checking prerequisites..."

    if ! command -v go &> /dev/null; then
        echo "Error: Go is not installed"
        exit 1
    fi

    if ! command -v terraform &> /dev/null; then
        echo "Error: Terraform is not installed"
        exit 1
    fi

    if ! command -v aws &> /dev/null; then
        echo "Error: AWS CLI is not installed"
        exit 1
    fi

    echo "All prerequisites satisfied"
}

# Build Lambda functions
build_lambdas() {
    echo ""
    echo "Building Lambda functions..."
    make build
    echo "Build complete"
}

# Deploy infrastructure
deploy_infrastructure() {
    echo ""
    echo "Deploying infrastructure..."

    cd infrastructure/terraform

    # Initialize Terraform
    terraform init

    # Plan deployment
    echo "Creating deployment plan..."
    terraform plan -var-file="environments/${ENVIRONMENT}.tfvars" -out=tfplan

    # Apply deployment
    echo "Applying deployment..."
    terraform apply tfplan

    # Clean up plan file
    rm tfplan

    cd ../..

    echo "Infrastructure deployed successfully"
}

# Get deployment outputs
get_outputs() {
    echo ""
    echo "==================================="
    echo "Deployment Outputs"
    echo "==================================="

    cd infrastructure/terraform

    API_ENDPOINT=$(terraform output -raw api_endpoint 2>/dev/null || echo "N/A")
    DYNAMODB_TABLE=$(terraform output -raw dynamodb_table_name 2>/dev/null || echo "N/A")
    PAYMENT_QUEUE=$(terraform output -raw payment_queue_url 2>/dev/null || echo "N/A")
    WEBHOOK_QUEUE=$(terraform output -raw webhook_queue_url 2>/dev/null || echo "N/A")

    cd ../..

    echo "API Endpoint: $API_ENDPOINT"
    echo "DynamoDB Table: $DYNAMODB_TABLE"
    echo "Payment Queue: $PAYMENT_QUEUE"
    echo "Webhook Queue: $WEBHOOK_QUEUE"
    echo ""
}

# Main deployment flow
main() {
    check_prerequisites
    build_lambdas
    deploy_infrastructure
    get_outputs

    echo "==================================="
    echo "Deployment completed successfully!"
    echo "==================================="
}

main
