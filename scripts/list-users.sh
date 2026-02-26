#!/bin/bash
set -e

# List all users in the system

ENVIRONMENT=${1}
AWS_REGION=${2:-us-east-1}

if [ -z "$ENVIRONMENT" ]; then
  echo "Usage: $0 <environment> [aws_region]"
  echo ""
  echo "Arguments:"
  echo "  environment  - Environment name (e.g., test, prod)"
  echo "  aws_region   - AWS region (default: us-east-1)"
  echo ""
  echo "Example:"
  echo "  $0 test us-east-1"
  exit 1
fi

USERS_TABLE="${ENVIRONMENT}-ssm-users"

echo "=== Users in $USERS_TABLE ==="
echo ""

# List all users
aws dynamodb scan \
  --region "$AWS_REGION" \
  --table-name "$USERS_TABLE" \
  --query 'Items[*].[user_id.S, username.S, email.S, role.S]' \
  --output table

echo ""
echo "Total users: $(aws dynamodb scan --region "$AWS_REGION" --table-name "$USERS_TABLE" --select COUNT --query 'Count' --output text)"

