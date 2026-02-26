# Outputs

output "api_gateway_url" {
  description = "API Gateway URL"
  value       = "https://${aws_api_gateway_rest_api.ssm_access_manager.id}.execute-api.${var.aws_region}.amazonaws.com/${aws_api_gateway_stage.ssm_access_manager.stage_name}"
}

output "slack_command_endpoint" {
  description = "Slack slash command endpoint"
  value       = "https://${aws_api_gateway_rest_api.ssm_access_manager.id}.execute-api.${var.aws_region}.amazonaws.com/${aws_api_gateway_stage.ssm_access_manager.stage_name}/slack/command"
}

output "slack_interaction_endpoint" {
  description = "Slack interaction endpoint"
  value       = "https://${aws_api_gateway_rest_api.ssm_access_manager.id}.execute-api.${var.aws_region}.amazonaws.com/${aws_api_gateway_stage.ssm_access_manager.stage_name}/slack/interaction"
}

output "slack_admin_endpoint" {
  description = "Slack admin command endpoint"
  value       = "https://${aws_api_gateway_rest_api.ssm_access_manager.id}.execute-api.${var.aws_region}.amazonaws.com/${aws_api_gateway_stage.ssm_access_manager.stage_name}/slack/admin"
}

output "admin_endpoint" {
  description = "Admin API endpoint"
  value       = "https://${aws_api_gateway_rest_api.ssm_access_manager.id}.execute-api.${var.aws_region}.amazonaws.com/${aws_api_gateway_stage.ssm_access_manager.stage_name}/admin"
}

output "dynamodb_tables" {
  description = "DynamoDB table names"
  value = {
    requests        = aws_dynamodb_table.access_requests.name
    documents       = aws_dynamodb_table.ssm_documents.name
    users           = aws_dynamodb_table.users.name
    accounts        = aws_dynamodb_table.accounts.name
    approval_groups = aws_dynamodb_table.approval_groups.name
  }
}

output "lambda_functions" {
  description = "Lambda function names"
  value = {
    request_handler      = aws_lambda_function.request_handler.function_name
    approval_handler     = aws_lambda_function.approval_handler.function_name
    document_creator     = aws_lambda_function.document_creator.function_name
    expiration_cleanup   = aws_lambda_function.expiration_cleanup.function_name
    admin_handler        = aws_lambda_function.admin_handler.function_name
    admin_slack_handler  = aws_lambda_function.admin_slack_handler.function_name
  }
}
