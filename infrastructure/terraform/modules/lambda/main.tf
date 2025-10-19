# IAM Role for API Handler Lambda
resource "aws_iam_role" "api_handler" {
  name = "${var.project_name}-api-handler-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

# IAM Policy for API Handler
resource "aws_iam_role_policy" "api_handler" {
  name = "${var.project_name}-api-handler-policy-${var.environment}"
  role = aws_iam_role.api_handler.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = [
          var.dynamodb_table_arn,
          "${var.dynamodb_table_arn}/index/*",
          var.quote_table_arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage"
        ]
        Resource = var.payment_queue_arn
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "${var.api_handler_log_group_arn}:*"
      }
    ]
  })
}

# API Handler Lambda Function
resource "aws_lambda_function" "api_handler" {
  filename         = "${path.module}/../../../../build/api-handler.zip"
  function_name    = "${var.project_name}-api-handler-${var.environment}"
  role            = aws_iam_role.api_handler.arn
  handler         = "bootstrap"
  source_code_hash = fileexists("${path.module}/../../../../build/api-handler.zip") ? filebase64sha256("${path.module}/../../../../build/api-handler.zip") : ""
  runtime         = "provided.al2"
  timeout         = 30
  memory_size     = 512

  environment {
    variables = {
      DYNAMODB_TABLE     = var.dynamodb_table_name
      QUOTE_TABLE        = var.quote_table_name
      PAYMENT_QUEUE_URL  = var.payment_queue_url
      WEBHOOK_QUEUE_URL  = var.webhook_queue_url
      LOG_LEVEL          = "INFO"
    }
  }

  depends_on = [
    aws_iam_role_policy.api_handler
  ]
}

# IAM Role for Worker Lambda
resource "aws_iam_role" "worker_handler" {
  name = "${var.project_name}-worker-handler-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

# IAM Policy for Worker Handler
resource "aws_iam_role_policy" "worker_handler" {
  name = "${var.project_name}-worker-handler-policy-${var.environment}"
  role = aws_iam_role.worker_handler.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:UpdateItem",
          "dynamodb:PutItem"
        ]
        Resource = var.dynamodb_table_arn
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:SendMessage"
        ]
        Resource = var.payment_queue_arn
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage"
        ]
        Resource = var.webhook_queue_arn
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "${var.worker_handler_log_group_arn}:*"
      }
    ]
  })
}

# Worker Handler Lambda Function
resource "aws_lambda_function" "worker_handler" {
  filename         = "${path.module}/../../../../build/worker-handler.zip"
  function_name    = "${var.project_name}-worker-handler-${var.environment}"
  role            = aws_iam_role.worker_handler.arn
  handler         = "bootstrap"
  source_code_hash = fileexists("${path.module}/../../../../build/worker-handler.zip") ? filebase64sha256("${path.module}/../../../../build/worker-handler.zip") : ""
  runtime         = "provided.al2"
  timeout         = 300 # 5 minutes for payment processing
  memory_size     = 512

  environment {
    variables = {
      DYNAMODB_TABLE     = var.dynamodb_table_name
      PAYMENT_QUEUE_URL  = var.payment_queue_url
      WEBHOOK_QUEUE_URL  = var.webhook_queue_url
      LOG_LEVEL          = "INFO"
    }
  }

  depends_on = [
    aws_iam_role_policy.worker_handler
  ]
}

# SQS Event Source Mapping for Worker
resource "aws_lambda_event_source_mapping" "worker_sqs" {
  event_source_arn = var.payment_queue_arn
  function_name    = aws_lambda_function.worker_handler.arn
  batch_size       = 1
  enabled          = true
}

# IAM Role for Webhook Lambda
resource "aws_iam_role" "webhook_handler" {
  name = "${var.project_name}-webhook-handler-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

# IAM Policy for Webhook Handler
resource "aws_iam_role_policy" "webhook_handler" {
  name = "${var.project_name}-webhook-handler-policy-${var.environment}"
  role = aws_iam_role.webhook_handler.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes"
        ]
        Resource = var.webhook_queue_arn
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "${var.webhook_handler_log_group_arn}:*"
      }
    ]
  })
}

# Webhook Handler Lambda Function
resource "aws_lambda_function" "webhook_handler" {
  filename         = "${path.module}/../../../../build/webhook-handler.zip"
  function_name    = "${var.project_name}-webhook-handler-${var.environment}"
  role            = aws_iam_role.webhook_handler.arn
  handler         = "bootstrap"
  source_code_hash = fileexists("${path.module}/../../../../build/webhook-handler.zip") ? filebase64sha256("${path.module}/../../../../build/webhook-handler.zip") : ""
  runtime         = "provided.al2"
  timeout         = 30
  memory_size     = 256

  environment {
    variables = {
      LOG_LEVEL          = "INFO"
    }
  }

  depends_on = [
    aws_iam_role_policy.webhook_handler
  ]
}

# SQS Event Source Mapping for Webhook Handler
resource "aws_lambda_event_source_mapping" "webhook_sqs" {
  event_source_arn = var.webhook_queue_arn
  function_name    = aws_lambda_function.webhook_handler.arn
  batch_size       = 10
  enabled          = true
}
