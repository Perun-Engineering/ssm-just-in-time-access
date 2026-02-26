#!/bin/bash
set -e

# Initialize DynamoDB with seed data

ENVIRONMENT=${1:-dev}
AWS_REGION=${2:-us-east-1}
ADMIN_USER_ID=${3}
ADMIN_EMAIL=${4}

if [ -z "$ADMIN_USER_ID" ] || [ -z "$ADMIN_EMAIL" ]; then
  echo "Usage: $0 <environment> <aws_region> <admin_user_id> <admin_email>"
  echo "Example: $0 dev us-east-1 U12345678 admin@example.com"
  exit 1
fi

echo "Initializing DynamoDB for environment: $ENVIRONMENT"
echo "Creating initial administrator: $ADMIN_USER_ID ($ADMIN_EMAIL)"

USERS_TABLE="${ENVIRONMENT}-ssm-users"

# Create initial administrator
aws dynamodb put-item \
  --region "$AWS_REGION" \
  --table-name "$USERS_TABLE" \
  --item "{
    \"user_id\": {\"S\": \"$ADMIN_USER_ID\"},
    \"username\": {\"S\": \"$ADMIN_USER_ID\"},
    \"role\": {\"S\": \"administrator\"},
    \"email\": {\"S\": \"$ADMIN_EMAIL\"},
    \"added_by\": {\"S\": \"system\"},
    \"added_at\": {\"S\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"},
    \"updated_at\": {\"S\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}
  }"

echo "Initial administrator created successfully!"
echo ""
echo "Next steps:"
echo "1. Configure Slack app with the API endpoints"
echo "2. Add target AWS accounts using the admin API"
echo "3. Add managers who can approve requests"
