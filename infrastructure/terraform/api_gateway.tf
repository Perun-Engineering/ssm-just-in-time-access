# API Gateway REST API
resource "aws_api_gateway_rest_api" "ssm_access_manager" {
  name        = "${var.environment}-ssm-access-manager-api"
  description = "API Gateway for SSM Access Manager Slack webhooks"

  endpoint_configuration {
    types = ["REGIONAL"]
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# /slack resource
resource "aws_api_gateway_resource" "slack" {
  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id
  parent_id   = aws_api_gateway_rest_api.ssm_access_manager.root_resource_id
  path_part   = "slack"
}

# /slack/command resource
resource "aws_api_gateway_resource" "slack_command" {
  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id
  parent_id   = aws_api_gateway_resource.slack.id
  path_part   = "command"
}

# /slack/interaction resource
resource "aws_api_gateway_resource" "slack_interaction" {
  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id
  parent_id   = aws_api_gateway_resource.slack.id
  path_part   = "interaction"
}

# /slack/admin resource
resource "aws_api_gateway_resource" "slack_admin" {
  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id
  parent_id   = aws_api_gateway_resource.slack.id
  path_part   = "admin"
}

# /admin resource
resource "aws_api_gateway_resource" "admin" {
  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id
  parent_id   = aws_api_gateway_rest_api.ssm_access_manager.root_resource_id
  path_part   = "admin"
}

# POST /slack/command
resource "aws_api_gateway_method" "slack_command_post" {
  rest_api_id   = aws_api_gateway_rest_api.ssm_access_manager.id
  resource_id   = aws_api_gateway_resource.slack_command.id
  http_method   = "POST"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "slack_command_lambda" {
  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id
  resource_id = aws_api_gateway_resource.slack_command.id
  http_method = aws_api_gateway_method.slack_command_post.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.request_handler.invoke_arn
}

# POST /slack/interaction
resource "aws_api_gateway_method" "slack_interaction_post" {
  rest_api_id   = aws_api_gateway_rest_api.ssm_access_manager.id
  resource_id   = aws_api_gateway_resource.slack_interaction.id
  http_method   = "POST"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "slack_interaction_lambda" {
  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id
  resource_id = aws_api_gateway_resource.slack_interaction.id
  http_method = aws_api_gateway_method.slack_interaction_post.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.request_handler.invoke_arn
}

# POST /slack/admin
resource "aws_api_gateway_method" "slack_admin_post" {
  rest_api_id   = aws_api_gateway_rest_api.ssm_access_manager.id
  resource_id   = aws_api_gateway_resource.slack_admin.id
  http_method   = "POST"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "slack_admin_lambda" {
  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id
  resource_id = aws_api_gateway_resource.slack_admin.id
  http_method = aws_api_gateway_method.slack_admin_post.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.admin_slack_handler.invoke_arn
}

# POST /admin
resource "aws_api_gateway_method" "admin_post" {
  rest_api_id   = aws_api_gateway_rest_api.ssm_access_manager.id
  resource_id   = aws_api_gateway_resource.admin.id
  http_method   = "POST"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "admin_lambda" {
  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id
  resource_id = aws_api_gateway_resource.admin.id
  http_method = aws_api_gateway_method.admin_post.http_method

  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.admin_handler.invoke_arn
}

# API Gateway Deployment
resource "aws_api_gateway_deployment" "ssm_access_manager" {
  depends_on = [
    aws_api_gateway_integration.slack_command_lambda,
    aws_api_gateway_integration.slack_interaction_lambda,
    aws_api_gateway_integration.slack_admin_lambda,
    aws_api_gateway_integration.admin_lambda
  ]

  rest_api_id = aws_api_gateway_rest_api.ssm_access_manager.id

  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.slack.id,
      aws_api_gateway_resource.slack_command.id,
      aws_api_gateway_resource.slack_interaction.id,
      aws_api_gateway_resource.slack_admin.id,
      aws_api_gateway_resource.admin.id,
      aws_api_gateway_method.slack_command_post.id,
      aws_api_gateway_method.slack_interaction_post.id,
      aws_api_gateway_method.slack_admin_post.id,
      aws_api_gateway_method.admin_post.id,
      aws_api_gateway_integration.slack_command_lambda.id,
      aws_api_gateway_integration.slack_interaction_lambda.id,
      aws_api_gateway_integration.slack_admin_lambda.id,
      aws_api_gateway_integration.admin_lambda.id,
      aws_lambda_function.request_handler.source_code_hash,
      aws_lambda_function.approval_handler.source_code_hash,
      aws_lambda_function.admin_handler.source_code_hash,
      aws_lambda_function.admin_slack_handler.source_code_hash,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }
}

# API Gateway Stage
resource "aws_api_gateway_stage" "ssm_access_manager" {
  deployment_id = aws_api_gateway_deployment.ssm_access_manager.id
  rest_api_id   = aws_api_gateway_rest_api.ssm_access_manager.id
  stage_name    = var.environment

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Lambda Permissions for API Gateway
resource "aws_lambda_permission" "api_gateway_request_handler" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.request_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.ssm_access_manager.execution_arn}/*/*"
}

resource "aws_lambda_permission" "api_gateway_approval_handler" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.approval_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.ssm_access_manager.execution_arn}/*/*"
}

resource "aws_lambda_permission" "api_gateway_admin_handler" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.admin_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.ssm_access_manager.execution_arn}/*/*"
}

resource "aws_lambda_permission" "api_gateway_admin_slack_handler" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.admin_slack_handler.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.ssm_access_manager.execution_arn}/*/*"
}
