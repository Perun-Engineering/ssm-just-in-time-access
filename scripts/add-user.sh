#!/bin/bash
set -e

# Add an administrator to the system
# Note: Approvals are now handled through Slack user groups.
# Users don't need to be added as "managers" - they just need to be
# members of the security or manager Slack user groups.

ENVIRONMENT=${1}
USER_ID=${2}
USERNAME=${3}
EMAIL=${4}
AWS_REGION=${5:-us-east-1}

if [ -z "$ENVIRONMENT" ] || [ -z "$USER_ID" ] || [ -z "$USERNAME" ] || [ -z "$EMAIL" ]; then
  echo "Usage: $0 <environment> <user_id> <username> <email> [aws_region]"
  echo ""
  echo "Arguments:"
  echo "  environment  - Environment name (e.g., test, prod)"
  echo "  user_id      - Slack user ID (e.g., U08RY2Y8QKW)"
  echo "  username     - User's display name"
  echo "  email        - User's email address"
  echo "  aws_region   - AWS region (default: us-east-1)"
  echo ""
  echo "Example:"
  echo "  $0 test U12345678 jane.admin jane@example.com"
  echo ""
  echo "Note: This script adds administrators only."
  echo "For approvals, add users to Slack user groups (security/manager groups)."
  echo ""
  echo "How to get Slack User ID:"
  echo "  1. In Slack, click on a user's profile"
  echo "  2. Click 'More' (three dots)"
  echo "  3. Click 'Copy member ID'"
  exit 1
fi

ROLE="administrator"
USERS_TABLE="${ENVIRONMENT}-ssm-users"

echo "Adding administrator to $USERS_TABLE:"
echo "  User ID:  $USER_ID"
echo "  Username: $USERNAME"
echo "  Email:    $EMAIL"
echo "  Role:     $ROLE"
echo ""

# Check if user already exists
EXISTING_USER=$(aws dynamodb get-item \
  --region "$AWS_REGION" \
  --table-name "$USERS_TABLE" \
  --key "{\"user_id\": {\"S\": \"$USER_ID\"}}" \
  --query 'Item' \
  --output text 2>/dev/null || echo "")

if [ -n "$EXISTING_USER" ]; then
  echo "⚠️  User $USER_ID already exists. Updating to administrator..."
fi

# Add or update user
aws dynamodb put-item \
  --region "$AWS_REGION" \
  --table-name "$USERS_TABLE" \
  --item "{
    \"user_id\": {\"S\": \"$USER_ID\"},
    \"username\": {\"S\": \"$USERNAME\"},
    \"role\": {\"S\": \"$ROLE\"},
    \"email\": {\"S\": \"$EMAIL\"},
    \"added_by\": {\"S\": \"admin-script\"},
    \"added_at\": {\"S\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"},
    \"updated_at\": {\"S\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}
  }"

echo "✅ Administrator added successfully!"
echo ""

# Show current administrators
echo "Current administrators:"
aws dynamodb scan \
  --region "$AWS_REGION" \
  --table-name "$USERS_TABLE" \
  --filter-expression "#role = :admin_role" \
  --expression-attribute-names '{"#role": "role"}' \
  --expression-attribute-values '{":admin_role": {"S": "administrator"}}' \
  --query 'Items[*].[user_id.S, username.S, email.S]' \
  --output table

echo ""
echo "Note: For approvals, add users to Slack user groups:"
echo "  - Security Team group for security approvals"
echo "  - Manager groups (e.g., SRE Cloud OPS) for manager approvals"
echo "  Use: /ssm-admin configure-group to set up approval groups"

