# AccessRequests Table
resource "aws_dynamodb_table" "access_requests" {
  name           = "${var.environment}-ssm-access-requests"
  billing_mode   = "PAY_PER_REQUEST"
  
  hash_key = "request_id"

  attribute {
    name = "request_id"
    type = "S"
  }

  attribute {
    name = "username"
    type = "S"
  }

  attribute {
    name = "created_at"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  global_secondary_index {
    name            = "username-created_at-index"
    hash_key        = "username"
    range_key       = "created_at"
    projection_type = "ALL"
  }

  global_secondary_index {
    name            = "status-created_at-index"
    hash_key        = "status"
    range_key       = "created_at"
    projection_type = "ALL"
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# SSMDocuments Table
resource "aws_dynamodb_table" "ssm_documents" {
  name           = "${var.environment}-ssm-documents"
  billing_mode   = "PAY_PER_REQUEST"
  
  hash_key = "document_id"

  attribute {
    name = "document_id"
    type = "S"
  }

  attribute {
    name = "request_id"
    type = "S"
  }

  attribute {
    name = "account_id"
    type = "S"
  }

  attribute {
    name = "document_name"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  attribute {
    name = "expires_at"
    type = "S"
  }

  attribute {
    name = "username"
    type = "S"
  }

  attribute {
    name = "created_at"
    type = "S"
  }

  global_secondary_index {
    name            = "request_id-index"
    hash_key        = "request_id"
    projection_type = "ALL"
  }

  global_secondary_index {
    name            = "account_id-document_name-index"
    hash_key        = "account_id"
    range_key       = "document_name"
    projection_type = "ALL"
  }

  global_secondary_index {
    name            = "status-expires_at-index"
    hash_key        = "status"
    range_key       = "expires_at"
    projection_type = "ALL"
  }

  global_secondary_index {
    name            = "username-created_at-index"
    hash_key        = "username"
    range_key       = "created_at"
    projection_type = "ALL"
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Users Table
resource "aws_dynamodb_table" "users" {
  name           = "${var.environment}-ssm-users"
  billing_mode   = "PAY_PER_REQUEST"
  
  hash_key = "user_id"

  attribute {
    name = "user_id"
    type = "S"
  }

  attribute {
    name = "role"
    type = "S"
  }

  attribute {
    name = "username"
    type = "S"
  }

  global_secondary_index {
    name            = "role-username-index"
    hash_key        = "role"
    range_key       = "username"
    projection_type = "ALL"
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Accounts Table
resource "aws_dynamodb_table" "accounts" {
  name           = "${var.environment}-ssm-accounts"
  billing_mode   = "PAY_PER_REQUEST"
  
  hash_key = "account_id"

  attribute {
    name = "account_id"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  attribute {
    name = "account_name"
    type = "S"
  }

  global_secondary_index {
    name            = "status-account_name-index"
    hash_key        = "status"
    range_key       = "account_name"
    projection_type = "ALL"
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}


# Approval Groups Table
resource "aws_dynamodb_table" "approval_groups" {
  name           = "${var.environment}-ssm-approval-groups"
  billing_mode   = "PAY_PER_REQUEST"
  
  hash_key = "group_id"

  attribute {
    name = "group_id"
    type = "S"
  }

  attribute {
    name = "group_type"
    type = "S"
  }

  attribute {
    name = "group_name"
    type = "S"
  }

  global_secondary_index {
    name            = "group_type-group_name-index"
    hash_key        = "group_type"
    range_key       = "group_name"
    projection_type = "ALL"
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
