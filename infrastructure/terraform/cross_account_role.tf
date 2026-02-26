# Cross-Account IAM Role for SSM Document Management
# This role should be created in each target AWS account that needs SSM document access
# The Lambda functions will assume this role to create/delete SSM documents

resource "aws_iam_role" "ssm_document_manager" {
  name        = "SSMDocumentManagerRole"
  description = "Role assumed by SSM Access Manager to manage SSM documents"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          AWS = aws_iam_role.lambda_execution_role.arn
        }
        Action = "sts:AssumeRole"
        Condition = {
          StringEquals = {
            "sts:ExternalId" = var.environment
          }
        }
      }
    ]
  })

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
    Purpose     = "SSM Document Management"
  }
}

# Policy for SSM Document Management
resource "aws_iam_role_policy" "ssm_document_manager_policy" {
  name = "SSMDocumentManagerPolicy"
  role = aws_iam_role.ssm_document_manager.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "SSMDocumentManagement"
        Effect = "Allow"
        Action = [
          "ssm:CreateDocument",
          "ssm:DeleteDocument",
          "ssm:DescribeDocument",
          "ssm:GetDocument",
          "ssm:ListDocuments",
          "ssm:UpdateDocument",
          "ssm:UpdateDocumentDefaultVersion",
          "ssm:AddTagsToResource",
          "ssm:RemoveTagsFromResource",
          "ssm:ListTagsForResource"
        ]
        Resource = [
          "arn:aws:ssm:*:*:document/${var.document_prefix}-*"
        ]
      },
      {
        Sid    = "SSMDocumentList"
        Effect = "Allow"
        Action = [
          "ssm:ListDocuments"
        ]
        Resource = "*"
      }
    ]
  })
}

# Output the role ARN for reference
output "ssm_document_manager_role_arn" {
  description = "ARN of the SSM Document Manager role"
  value       = aws_iam_role.ssm_document_manager.arn
}

output "ssm_document_manager_role_name" {
  description = "Name of the SSM Document Manager role"
  value       = aws_iam_role.ssm_document_manager.name
}
