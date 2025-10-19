variable "project_name" {
  description = "Project name"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
}

variable "dynamodb_table_name" {
  description = "DynamoDB table name"
  type        = string
}

variable "dynamodb_table_arn" {
  description = "DynamoDB table ARN"
  type        = string
}

variable "payment_queue_url" {
  description = "Payment queue URL"
  type        = string
}

variable "payment_queue_arn" {
  description = "Payment queue ARN"
  type        = string
}

variable "webhook_queue_url" {
  description = "Webhook queue URL"
  type        = string
}

variable "webhook_queue_arn" {
  description = "Webhook queue ARN"
  type        = string
}

variable "api_handler_log_group_arn" {
  description = "API handler log group ARN"
  type        = string
}

variable "worker_handler_log_group_arn" {
  description = "Worker handler log group ARN"
  type        = string
}

variable "webhook_handler_log_group_arn" {
  description = "Webhook handler log group ARN"
  type        = string
}
