variable "aws_region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "slack_signing_secret" {
  description = "Slack signing secret for request verification"
  type        = string
  sensitive   = true
}

variable "slack_bot_token" {
  description = "Slack bot token"
  type        = string
  sensitive   = true
}

variable "slack_team_id" {
  description = "Slack team/workspace ID"
  type        = string
  sensitive   = true
}

variable "document_prefix" {
  description = "Prefix for SSM document names (e.g., PF, ACME, MYORG)"
  type        = string
  default     = "PF"
  
  validation {
    condition     = can(regex("^[A-Za-z0-9][A-Za-z0-9_-]*$", var.document_prefix))
    error_message = "Document prefix must start with alphanumeric character and contain only alphanumeric, hyphens, and underscores."
  }
  
  validation {
    condition     = length(var.document_prefix) <= 20
    error_message = "Document prefix must be 20 characters or less."
  }
}

variable "allow_self_approval" {
  description = "Allow users to approve their own requests (for testing only - should be false in production)"
  type        = string
  default     = "false"
  
  validation {
    condition     = contains(["true", "false"], var.allow_self_approval)
    error_message = "allow_self_approval must be either 'true' or 'false'."
  }
}
