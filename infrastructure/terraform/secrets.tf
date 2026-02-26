# Slack Bot Token Secret
resource "aws_secretsmanager_secret" "slack_bot_token" {
  name        = "${var.environment}/ssm-access-manager/slack-bot-token"
  description = "Slack bot token for SSM Access Manager"

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_secretsmanager_secret_version" "slack_bot_token" {
  secret_id     = aws_secretsmanager_secret.slack_bot_token.id
  secret_string = var.slack_bot_token
}

# Slack Signing Secret
resource "aws_secretsmanager_secret" "slack_signing_secret" {
  name        = "${var.environment}/ssm-access-manager/slack-signing-secret"
  description = "Slack signing secret for request verification"

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_secretsmanager_secret_version" "slack_signing_secret" {
  secret_id     = aws_secretsmanager_secret.slack_signing_secret.id
  secret_string = var.slack_signing_secret
}
