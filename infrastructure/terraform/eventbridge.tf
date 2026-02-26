# Use the default event bus (no need to create it)
data "aws_cloudwatch_event_bus" "default" {
  name = "default"
}

# EventBridge Rule for Expiration Cleanup (hourly)
resource "aws_cloudwatch_event_rule" "expiration_cleanup" {
  name                = "${var.environment}-ssm-expiration-cleanup"
  description         = "Trigger expiration cleanup Lambda hourly"
  schedule_expression = "rate(1 hour)"
  event_bus_name      = data.aws_cloudwatch_event_bus.default.name

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_cloudwatch_event_target" "expiration_cleanup" {
  rule           = aws_cloudwatch_event_rule.expiration_cleanup.name
  target_id      = "ExpirationCleanupLambda"
  arn            = aws_lambda_function.expiration_cleanup.arn
  event_bus_name = data.aws_cloudwatch_event_bus.default.name
}

resource "aws_lambda_permission" "eventbridge_expiration_cleanup" {
  statement_id  = "AllowEventBridgeInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.expiration_cleanup.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.expiration_cleanup.arn
}

# EventBridge Rule for Document Creation (triggered by approval)
resource "aws_cloudwatch_event_rule" "document_creation" {
  name           = "${var.environment}-ssm-document-creation"
  description    = "Trigger document creation Lambda on approval"
  event_bus_name = data.aws_cloudwatch_event_bus.default.name

  event_pattern = jsonencode({
    source      = ["ssm-access-manager"]
    detail-type = ["Request Approved"]
  })

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_cloudwatch_event_target" "document_creation" {
  rule           = aws_cloudwatch_event_rule.document_creation.name
  target_id      = "DocumentCreatorLambda"
  arn            = aws_lambda_function.document_creator.arn
  event_bus_name = data.aws_cloudwatch_event_bus.default.name
}

resource "aws_lambda_permission" "eventbridge_document_creator" {
  statement_id  = "AllowEventBridgeInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.document_creator.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.document_creation.arn
}
