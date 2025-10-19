output "api_handler_function_name" {
  description = "API handler Lambda function name"
  value       = aws_lambda_function.api_handler.function_name
}

output "api_handler_arn" {
  description = "API handler Lambda function ARN"
  value       = aws_lambda_function.api_handler.arn
}

output "api_handler_invoke_arn" {
  description = "API handler Lambda invoke ARN"
  value       = aws_lambda_function.api_handler.invoke_arn
}

output "worker_handler_function_name" {
  description = "Worker handler Lambda function name"
  value       = aws_lambda_function.worker_handler.function_name
}

output "webhook_handler_function_name" {
  description = "Webhook handler Lambda function name"
  value       = aws_lambda_function.webhook_handler.function_name
}
