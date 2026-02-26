#!/bin/bash
set -e

ENVIRONMENT=${1:-test}
AWS_REGION=${2:-us-east-1}

echo "=== Slack Token Verification ==="
echo ""

# Get token from Lambda
echo "1. Token stored in Lambda:"
LAMBDA_TOKEN=$(aws lambda get-function-configuration \
  --function-name "${ENVIRONMENT}-ssm-request-handler" \
  --region "$AWS_REGION" \
  --query 'Environment.Variables.SLACK_BOT_TOKEN' \
  --output text)

echo "   ${LAMBDA_TOKEN:0:20}...${LAMBDA_TOKEN: -10}"
echo ""

# Test the token with Slack API
echo "2. Testing token with Slack API..."
RESPONSE=$(curl -s -X POST https://slack.com/api/auth.test \
  -H "Authorization: Bearer $LAMBDA_TOKEN" \
  -H "Content-Type: application/json")

echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
echo ""

# Check if token is valid
if echo "$RESPONSE" | grep -q '"ok":true'; then
  echo "✅ Token is VALID"
else
  echo "❌ Token is INVALID"
  echo ""
  echo "Possible reasons:"
  echo "  1. Token was regenerated in Slack app settings"
  echo "  2. App was uninstalled/reinstalled (generates new token)"
  echo "  3. Token was revoked"
  echo ""
  echo "To fix:"
  echo "  1. Go to https://api.slack.com/apps"
  echo "  2. Select your SSM Access Manager app"
  echo "  3. Click 'OAuth & Permissions'"
  echo "  4. Copy the 'Bot User OAuth Token' (starts with xoxb-)"
  echo "  5. Run: ./scripts/update-slack-credentials.sh $ENVIRONMENT $AWS_REGION 'NEW_TOKEN' 'SIGNING_SECRET'"
fi
echo ""
