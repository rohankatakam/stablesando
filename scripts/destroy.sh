#!/bin/bash

set -e

# Destruction script for crypto-conversion API

ENVIRONMENT=${1:-dev}

echo "==================================="
echo "Crypto Conversion API Destruction"
echo "Environment: $ENVIRONMENT"
echo "==================================="
echo ""
echo "⚠️  WARNING: This will destroy all infrastructure!"
echo ""
read -p "Are you sure you want to continue? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Destruction cancelled"
    exit 0
fi

echo ""
echo "Destroying infrastructure..."

cd infrastructure/terraform

# Initialize Terraform (in case it's not initialized)
terraform init

# Destroy infrastructure
terraform destroy -var-file="environments/${ENVIRONMENT}.tfvars" -auto-approve

cd ../..

echo ""
echo "==================================="
echo "Infrastructure destroyed"
echo "==================================="
