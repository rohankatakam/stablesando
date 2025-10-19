# API Gateway REST API
resource "aws_api_gateway_rest_api" "main" {
  name        = "${var.project_name}-api-${var.environment}"
  description = "Crypto Conversion Payment API"

  endpoint_configuration {
    types = ["REGIONAL"]
  }
}

# /payments resource
resource "aws_api_gateway_resource" "payments" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  parent_id   = aws_api_gateway_rest_api.main.root_resource_id
  path_part   = "payments"
}

# /quotes resource
resource "aws_api_gateway_resource" "quotes" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  parent_id   = aws_api_gateway_rest_api.main.root_resource_id
  path_part   = "quotes"
}

# POST method on /payments
resource "aws_api_gateway_method" "post_payments" {
  rest_api_id   = aws_api_gateway_rest_api.main.id
  resource_id   = aws_api_gateway_resource.payments.id
  http_method   = "POST"
  authorization = "NONE"

  request_parameters = {
    "method.request.header.Idempotency-Key" = true
  }
}

# POST method on /quotes
resource "aws_api_gateway_method" "post_quotes" {
  rest_api_id   = aws_api_gateway_rest_api.main.id
  resource_id   = aws_api_gateway_resource.quotes.id
  http_method   = "POST"
  authorization = "NONE"
}

# Lambda integration for /payments
resource "aws_api_gateway_integration" "lambda_payments" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.payments.id
  http_method = aws_api_gateway_method.post_payments.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = var.api_handler_invoke_arn
}

# Lambda integration for /quotes
resource "aws_api_gateway_integration" "lambda_quotes" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.quotes.id
  http_method = aws_api_gateway_method.post_quotes.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = var.api_handler_invoke_arn
}

# GET method on /payments/{payment_id}
resource "aws_api_gateway_resource" "payment_id" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  parent_id   = aws_api_gateway_resource.payments.id
  path_part   = "{payment_id}"
}

resource "aws_api_gateway_method" "get_payment" {
  rest_api_id   = aws_api_gateway_rest_api.main.id
  resource_id   = aws_api_gateway_resource.payment_id.id
  http_method   = "GET"
  authorization = "NONE"

  request_parameters = {
    "method.request.path.payment_id" = true
  }
}

resource "aws_api_gateway_integration" "lambda_get_payment" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.payment_id.id
  http_method = aws_api_gateway_method.get_payment.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = var.api_handler_invoke_arn
}

# CORS support - OPTIONS method for /payments
resource "aws_api_gateway_method" "options_payments" {
  rest_api_id   = aws_api_gateway_rest_api.main.id
  resource_id   = aws_api_gateway_resource.payments.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options_payments" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.payments.id
  http_method = aws_api_gateway_method.options_payments.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options_payments_200" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.payments.id
  http_method = aws_api_gateway_method.options_payments.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_integration_response" "options_payments" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.payments.id
  http_method = aws_api_gateway_method.options_payments.http_method
  status_code = aws_api_gateway_method_response.options_payments_200.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,Idempotency-Key'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET,POST,OPTIONS'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }
}

# CORS support - OPTIONS method for /payments/{payment_id}
resource "aws_api_gateway_method" "options_payment_id" {
  rest_api_id   = aws_api_gateway_rest_api.main.id
  resource_id   = aws_api_gateway_resource.payment_id.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options_payment_id" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.payment_id.id
  http_method = aws_api_gateway_method.options_payment_id.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options_payment_id_200" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.payment_id.id
  http_method = aws_api_gateway_method.options_payment_id.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_integration_response" "options_payment_id" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.payment_id.id
  http_method = aws_api_gateway_method.options_payment_id.http_method
  status_code = aws_api_gateway_method_response.options_payment_id_200.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET,OPTIONS'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }
}

# CORS support - OPTIONS method for /quotes
resource "aws_api_gateway_method" "options_quotes" {
  rest_api_id   = aws_api_gateway_rest_api.main.id
  resource_id   = aws_api_gateway_resource.quotes.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options_quotes" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.quotes.id
  http_method = aws_api_gateway_method.options_quotes.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options_quotes_200" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.quotes.id
  http_method = aws_api_gateway_method.options_quotes.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_integration_response" "options_quotes" {
  rest_api_id = aws_api_gateway_rest_api.main.id
  resource_id = aws_api_gateway_resource.quotes.id
  http_method = aws_api_gateway_method.options_quotes.http_method
  status_code = aws_api_gateway_method_response.options_quotes_200.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token'"
    "method.response.header.Access-Control-Allow-Methods" = "'POST,OPTIONS'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }
}

# Lambda permission for API Gateway
resource "aws_lambda_permission" "api_gateway" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.api_handler_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.main.execution_arn}/*/*"
}

# API Deployment
resource "aws_api_gateway_deployment" "main" {
  rest_api_id = aws_api_gateway_rest_api.main.id

  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.payments.id,
      aws_api_gateway_resource.quotes.id,
      aws_api_gateway_resource.payment_id.id,
      aws_api_gateway_method.post_payments.id,
      aws_api_gateway_method.post_quotes.id,
      aws_api_gateway_method.get_payment.id,
      aws_api_gateway_integration.lambda_payments.id,
      aws_api_gateway_integration.lambda_quotes.id,
      aws_api_gateway_integration.lambda_get_payment.id,
      aws_api_gateway_integration.options_payments.id,
      aws_api_gateway_integration.options_quotes.id,
      aws_api_gateway_integration.options_payment_id.id,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [
    aws_api_gateway_integration.lambda_payments,
    aws_api_gateway_integration.lambda_quotes,
    aws_api_gateway_integration.lambda_get_payment,
    aws_api_gateway_integration.options_payments,
    aws_api_gateway_integration.options_quotes,
    aws_api_gateway_integration.options_payment_id
  ]
}

# API Stage
resource "aws_api_gateway_stage" "main" {
  deployment_id = aws_api_gateway_deployment.main.id
  rest_api_id   = aws_api_gateway_rest_api.main.id
  stage_name    = var.environment

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_gateway.arn
    format = jsonencode({
      requestId      = "$context.requestId"
      ip             = "$context.identity.sourceIp"
      caller         = "$context.identity.caller"
      user           = "$context.identity.user"
      requestTime    = "$context.requestTime"
      httpMethod     = "$context.httpMethod"
      resourcePath   = "$context.resourcePath"
      status         = "$context.status"
      protocol       = "$context.protocol"
      responseLength = "$context.responseLength"
    })
  }

  xray_tracing_enabled = true
}

# CloudWatch Log Group for API Gateway
resource "aws_cloudwatch_log_group" "api_gateway" {
  name              = "/aws/apigateway/${var.project_name}-${var.environment}"
  retention_in_days = 7
}

# API Gateway account settings (required for logging)
resource "aws_api_gateway_account" "main" {
  cloudwatch_role_arn = aws_iam_role.api_gateway_cloudwatch.arn
}

# IAM role for API Gateway CloudWatch logging
resource "aws_iam_role" "api_gateway_cloudwatch" {
  name = "${var.project_name}-api-gateway-cloudwatch-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "apigateway.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "api_gateway_cloudwatch" {
  role       = aws_iam_role.api_gateway_cloudwatch.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"
}

# Usage Plan (optional - for rate limiting)
resource "aws_api_gateway_usage_plan" "main" {
  name = "${var.project_name}-usage-plan-${var.environment}"

  api_stages {
    api_id = aws_api_gateway_rest_api.main.id
    stage  = aws_api_gateway_stage.main.stage_name
  }

  quota_settings {
    limit  = 10000
    period = "DAY"
  }

  throttle_settings {
    burst_limit = 100
    rate_limit  = 50
  }
}
