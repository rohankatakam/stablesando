terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Using local state for development
  # For production, configure S3 backend for remote state storage
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "crypto-conversion"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

# DynamoDB Table for Payments
resource "aws_dynamodb_table" "payments" {
  name           = "${var.project_name}-payments-${var.environment}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "payment_id"

  attribute {
    name = "payment_id"
    type = "S"
  }

  attribute {
    name = "idempotency_key"
    type = "S"
  }

  global_secondary_index {
    name            = "idempotency-key-index"
    hash_key        = "idempotency_key"
    projection_type = "ALL"
  }

  point_in_time_recovery {
    enabled = var.enable_point_in_time_recovery
  }

  server_side_encryption {
    enabled = true
  }

  tags = {
    Name = "${var.project_name}-payments-${var.environment}"
  }
}

# DynamoDB Table for Quotes
resource "aws_dynamodb_table" "quotes" {
  name           = "${var.project_name}-quotes-${var.environment}"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "quote_id"

  attribute {
    name = "quote_id"
    type = "S"
  }

  # TTL configuration - DynamoDB will automatically delete expired quotes
  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  point_in_time_recovery {
    enabled = var.enable_point_in_time_recovery
  }

  server_side_encryption {
    enabled = true
  }

  tags = {
    Name = "${var.project_name}-quotes-${var.environment}"
  }
}

# SQS Queue for Payment Jobs
resource "aws_sqs_queue" "payment_queue" {
  name                       = "${var.project_name}-payment-queue-${var.environment}"
  visibility_timeout_seconds = 300 # 5 minutes - should be 6x Lambda timeout
  message_retention_seconds  = 1209600 # 14 days
  receive_wait_time_seconds  = 20 # Long polling

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.payment_dlq.arn
    maxReceiveCount     = 3
  })

  tags = {
    Name = "${var.project_name}-payment-queue-${var.environment}"
  }
}

# Dead Letter Queue for failed payment jobs
resource "aws_sqs_queue" "payment_dlq" {
  name                      = "${var.project_name}-payment-dlq-${var.environment}"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Name = "${var.project_name}-payment-dlq-${var.environment}"
  }
}

# SQS Queue for Webhook Events
resource "aws_sqs_queue" "webhook_queue" {
  name                       = "${var.project_name}-webhook-queue-${var.environment}"
  visibility_timeout_seconds = 60 # 1 minute
  message_retention_seconds  = 345600 # 4 days
  receive_wait_time_seconds  = 20

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.webhook_dlq.arn
    maxReceiveCount     = 5
  })

  tags = {
    Name = "${var.project_name}-webhook-queue-${var.environment}"
  }
}

# Dead Letter Queue for failed webhooks
resource "aws_sqs_queue" "webhook_dlq" {
  name                      = "${var.project_name}-webhook-dlq-${var.environment}"
  message_retention_seconds = 1209600 # 14 days

  tags = {
    Name = "${var.project_name}-webhook-dlq-${var.environment}"
  }
}

# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "api_handler" {
  name              = "/aws/lambda/${var.project_name}-api-handler-${var.environment}"
  retention_in_days = var.log_retention_days
}

resource "aws_cloudwatch_log_group" "worker_handler" {
  name              = "/aws/lambda/${var.project_name}-worker-handler-${var.environment}"
  retention_in_days = var.log_retention_days
}

resource "aws_cloudwatch_log_group" "webhook_handler" {
  name              = "/aws/lambda/${var.project_name}-webhook-handler-${var.environment}"
  retention_in_days = var.log_retention_days
}

# Import Lambda functions and API Gateway from separate modules
module "lambda_functions" {
  source = "./modules/lambda"

  project_name                  = var.project_name
  environment                   = var.environment
  aws_region                    = var.aws_region
  dynamodb_table_name           = aws_dynamodb_table.payments.name
  dynamodb_table_arn            = aws_dynamodb_table.payments.arn
  quote_table_name              = aws_dynamodb_table.quotes.name
  quote_table_arn               = aws_dynamodb_table.quotes.arn
  payment_queue_url             = aws_sqs_queue.payment_queue.url
  payment_queue_arn             = aws_sqs_queue.payment_queue.arn
  webhook_queue_url             = aws_sqs_queue.webhook_queue.url
  webhook_queue_arn             = aws_sqs_queue.webhook_queue.arn
  api_handler_log_group_arn     = aws_cloudwatch_log_group.api_handler.arn
  worker_handler_log_group_arn  = aws_cloudwatch_log_group.worker_handler.arn
  webhook_handler_log_group_arn = aws_cloudwatch_log_group.webhook_handler.arn
}

module "api_gateway" {
  source = "./modules/api-gateway"

  project_name            = var.project_name
  environment             = var.environment
  api_handler_invoke_arn  = module.lambda_functions.api_handler_invoke_arn
  api_handler_function_name = module.lambda_functions.api_handler_function_name
}

# Outputs
output "api_endpoint" {
  description = "API Gateway endpoint URL"
  value       = module.api_gateway.api_endpoint
}

output "dynamodb_table_name" {
  description = "DynamoDB payments table name"
  value       = aws_dynamodb_table.payments.name
}

output "quote_table_name" {
  description = "DynamoDB quotes table name"
  value       = aws_dynamodb_table.quotes.name
}

output "payment_queue_url" {
  description = "Payment SQS queue URL"
  value       = aws_sqs_queue.payment_queue.url
}

output "webhook_queue_url" {
  description = "Webhook SQS queue URL"
  value       = aws_sqs_queue.webhook_queue.url
}
