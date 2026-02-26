# Lambda Function: Request Handler
resource "aws_lambda_function" "request_handler" {
  filename      = "${path.module}/../../bin/request-handler.zip"
  function_name = "${var.environment}-ssm-request-handler"
  role          = aws_iam_role.lambda_execution_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 30
  memory_size   = 256
  source_code_hash = filebase64sha256("${path.module}/../../bin/request-handler.zip")

  environment {
    variables = {
      ENVIRONMENT            = var.environment
      REQUESTS_TABLE         = aws_dynamodb_table.access_requests.name
      USERS_TABLE            = aws_dynamodb_table.users.name
      ACCOUNTS_TABLE         = aws_dynamodb_table.accounts.name
      APPROVAL_GROUPS_TABLE  = aws_dynamodb_table.approval_groups.name
      AUDIT_LOG_GROUP        = aws_cloudwatch_log_group.audit_logs.name
      SLACK_BOT_TOKEN        = var.slack_bot_token
      SLACK_SIGNING_SECRET   = var.slack_signing_secret
      SLACK_TEAM_ID          = var.slack_team_id
      DOCUMENT_PREFIX        = var.document_prefix
      ALLOW_SELF_APPROVAL    = var.allow_self_approval
    }
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Lambda Function: Approval Handler
resource "aws_lambda_function" "approval_handler" {
  filename      = "${path.module}/../../bin/approval-handler.zip"
  function_name = "${var.environment}-ssm-approval-handler"
  role          = aws_iam_role.lambda_execution_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 30
  memory_size   = 256
  source_code_hash = filebase64sha256("${path.module}/../../bin/approval-handler.zip")

  environment {
    variables = {
      ENVIRONMENT            = var.environment
      REQUESTS_TABLE         = aws_dynamodb_table.access_requests.name
      USERS_TABLE            = aws_dynamodb_table.users.name
      SLACK_BOT_TOKEN        = var.slack_bot_token
      SLACK_SIGNING_SECRET   = var.slack_signing_secret
      ALLOW_SELF_APPROVAL    = var.allow_self_approval
    }
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Lambda Function: Document Creator
resource "aws_lambda_function" "document_creator" {
  filename      = "${path.module}/../../bin/document-creator.zip"
  function_name = "${var.environment}-ssm-document-creator"
  role          = aws_iam_role.lambda_execution_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 60
  memory_size   = 512
  source_code_hash = filebase64sha256("${path.module}/../../bin/document-creator.zip")

  environment {
    variables = {
      ENVIRONMENT            = var.environment
      REQUESTS_TABLE         = aws_dynamodb_table.access_requests.name
      DOCUMENTS_TABLE        = aws_dynamodb_table.ssm_documents.name
      ACCOUNTS_TABLE         = aws_dynamodb_table.accounts.name
      SLACK_BOT_TOKEN        = var.slack_bot_token
      SLACK_SIGNING_SECRET   = var.slack_signing_secret
      DOCUMENT_PREFIX        = var.document_prefix
    }
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Lambda Function: Expiration Cleanup
resource "aws_lambda_function" "expiration_cleanup" {
  filename      = "${path.module}/../../bin/expiration-cleanup.zip"
  function_name = "${var.environment}-ssm-expiration-cleanup"
  role          = aws_iam_role.lambda_execution_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 300
  memory_size   = 512
  source_code_hash = filebase64sha256("${path.module}/../../bin/expiration-cleanup.zip")

  environment {
    variables = {
      ENVIRONMENT            = var.environment
      DOCUMENTS_TABLE        = aws_dynamodb_table.ssm_documents.name
      USERS_TABLE            = aws_dynamodb_table.users.name
      SLACK_BOT_TOKEN        = var.slack_bot_token
      SLACK_SIGNING_SECRET   = var.slack_signing_secret
    }
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Lambda Function: Admin Handler
resource "aws_lambda_function" "admin_handler" {
  filename      = "${path.module}/../../bin/admin-handler.zip"
  function_name = "${var.environment}-ssm-admin-handler"
  role          = aws_iam_role.lambda_execution_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 30
  memory_size   = 256
  source_code_hash = filebase64sha256("${path.module}/../../bin/admin-handler.zip")

  environment {
    variables = {
      ENVIRONMENT  = var.environment
      USERS_TABLE  = aws_dynamodb_table.users.name
      ACCOUNTS_TABLE = aws_dynamodb_table.accounts.name
    }
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "request_handler" {
  name              = "/aws/lambda/${aws_lambda_function.request_handler.function_name}"
  retention_in_days = 30

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_cloudwatch_log_group" "approval_handler" {
  name              = "/aws/lambda/${aws_lambda_function.approval_handler.function_name}"
  retention_in_days = 30

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_cloudwatch_log_group" "document_creator" {
  name              = "/aws/lambda/${aws_lambda_function.document_creator.function_name}"
  retention_in_days = 30

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_cloudwatch_log_group" "expiration_cleanup" {
  name              = "/aws/lambda/${aws_lambda_function.expiration_cleanup.function_name}"
  retention_in_days = 30

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_cloudwatch_log_group" "admin_handler" {
  name              = "/aws/lambda/${aws_lambda_function.admin_handler.function_name}"
  retention_in_days = 30

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Lambda Function: Admin Slack Handler
resource "aws_lambda_function" "admin_slack_handler" {
  filename      = "${path.module}/../../bin/admin-slack-handler.zip"
  function_name = "${var.environment}-ssm-admin-slack-handler"
  role          = aws_iam_role.lambda_execution_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 30
  memory_size   = 256
  source_code_hash = filebase64sha256("${path.module}/../../bin/admin-slack-handler.zip")

  environment {
    variables = {
      ENVIRONMENT          = var.environment
      USERS_TABLE          = aws_dynamodb_table.users.name
      ACCOUNTS_TABLE       = aws_dynamodb_table.accounts.name
      REQUESTS_TABLE       = aws_dynamodb_table.access_requests.name
      DOCUMENTS_TABLE      = aws_dynamodb_table.ssm_documents.name
      APPROVAL_GROUPS_TABLE = aws_dynamodb_table.approval_groups.name
      AUDIT_LOG_GROUP      = aws_cloudwatch_log_group.audit_logs.name
      SLACK_BOT_TOKEN      = var.slack_bot_token
      SLACK_SIGNING_SECRET = var.slack_signing_secret
      SLACK_TEAM_ID        = var.slack_team_id
      DOCUMENT_PREFIX      = var.document_prefix
    }
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# CloudWatch Log Group for Admin Slack Handler
resource "aws_cloudwatch_log_group" "admin_slack_handler" {
  name              = "/aws/lambda/${aws_lambda_function.admin_slack_handler.function_name}"
  retention_in_days = 30

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
