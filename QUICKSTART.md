# SSM Access Manager - Quick Start Guide

This guide will help you deploy SSM Access Manager to a test AWS account and set up the Slack bot.

## Prerequisites

Before you begin, ensure you have:

- ✅ AWS CLI installed and configured with admin credentials
- ✅ Terraform >= 1.0 installed
- ✅ Go >= 1.25 installed
- ✅ Access to a Slack workspace (admin or app installation permissions)
- ✅ At least one AWS account to use as a target (can be the same as deployment account)

## Part 1: Create Slack App (15 minutes)

### Step 1: Create the App

1. Go to https://api.slack.com/apps
2. Click **"Create New App"**
3. Choose **"From scratch"**
4. Enter:
   - **App Name**: `SSM Access Manager`
   - **Workspace**: Select your workspace
5. Click **"Create App"**

### Step 2: Configure OAuth & Permissions

1. In the left sidebar, click **"OAuth & Permissions"**
2. Scroll to **"Scopes"** → **"Bot Token Scopes"**
3. Add these scopes:
   - `chat:write` - Send messages
   - `users:read` - Read user information
   - `users:read.email` - Read user email addresses
   - `usergroups:read` - Read user groups and members (for approval groups)
4. Scroll to top and click **"Install to Workspace"**
5. Click **"Allow"**
6. **Copy the "Bot User OAuth Token"** (starts with `xoxb-`)
   - Save this - you'll need it for deployment

### Step 3: Get Signing Secret and Team ID

1. In the left sidebar, click **"Basic Information"**
2. Scroll to **"App Credentials"**
3. **Copy the "Signing Secret"**
   - Save this - you'll need it for deployment
4. **Get your Team ID**:
   - Open Slack in your browser
   - Your workspace URL is: `https://app.slack.com/client/T08REEQ88F9/...`
   - The part after `/client/` and before the next `/` is your Team ID (e.g., `T08REEQ88F9`)
   - Save this - you'll need it for deployment

### Step 4: Note Your Slack User ID

You'll need your Slack user ID to become the initial administrator:

1. Click on your profile picture in Slack
2. Click **"Profile"**
3. Click **"More"** (three dots)
4. Click **"Copy member ID"**
5. **Save this ID** (looks like `U01234ABCDE`)

**We'll configure the Slack endpoints after deployment.**

## Part 2: Deploy to AWS (20 minutes)

### Step 1: Clone and Prepare

```bash
# Navigate to your project directory
cd ssm-access-manager

# Verify Go is installed
go version

# Verify Terraform is installed
terraform version

# Verify AWS CLI is configured
aws sts get-caller-identity
```

### Step 2: Set Environment Variables

```bash
# Set your Slack credentials (from Part 1)
export TF_VAR_slack_bot_token="xoxb-your-bot-token-here"
export TF_VAR_slack_signing_secret="your-signing-secret-here"
export TF_VAR_slack_team_id="T08REEQ88F9"  # Your Team ID from Part 1

# Set deployment configuration
export TF_VAR_environment="test"
export TF_VAR_aws_region="us-east-1"  # or your preferred region

# Optional: Customize SSM document prefix (default: "PF")
# export TF_VAR_document_prefix="ACME"  # Use your organization name

# Optional: Allow self-approval for testing (default: false)
# WARNING: Only enable this in test/development environments!
# export TF_VAR_allow_self_approval="true"
```

**Document Prefix:** The default prefix is "PF" (PortForwarding). Documents will be named like `PF-username-host-port`. You can customize this to match your organization's naming convention.

**Self-Approval:** By default, users cannot approve their own access requests (enforces separation of duties). For testing purposes, you can set `ALLOW_SELF_APPROVAL=true` to bypass this check. **This should NEVER be enabled in production environments.**

### Step 3: Build Lambda Functions

```bash
# Build all Lambda functions
./scripts/build.sh
```

This creates ZIP files in the `bin/` directory.

### Step 4: Deploy Infrastructure

```bash
# Deploy with Terraform
./scripts/deploy.sh test us-east-1
```

When prompted "Do you want to apply these changes?", type `yes` and press Enter.

**Save the output!** You'll see:
- `slack_command_endpoint` - Use this for /ssm-access command
- `slack_interaction_endpoint` - Use this for interactive components
- `slack_admin_endpoint` - Use this for /ssm-admin command
- `admin_endpoint` - Use this for admin API operations

Example output:
```
slack_command_endpoint = "https://abc123.execute-api.us-east-1.amazonaws.com/test/slack/command"
slack_interaction_endpoint = "https://abc123.execute-api.us-east-1.amazonaws.com/test/slack/interaction"
slack_admin_endpoint = "https://abc123.execute-api.us-east-1.amazonaws.com/test/slack/admin"
admin_endpoint = "https://abc123.execute-api.us-east-1.amazonaws.com/test/admin"
```

### Step 5: Initialize Database

Create yourself as the initial administrator:

```bash
./scripts/init-db.sh test us-east-1 <YOUR_SLACK_USER_ID> <YOUR_EMAIL>
```

Example:
```bash
./scripts/init-db.sh test us-east-1 U01234ABCDE admin@example.com
```

## Part 3: Configure Slack App (15 minutes)

### Step 1: Add /ssm-access Slash Command

1. Go back to https://api.slack.com/apps
2. Select your **SSM Access Manager** app
3. In the left sidebar, click **"Slash Commands"**
4. Click **"Create New Command"**
5. Fill in:
   - **Command**: `/ssm-access`
   - **Request URL**: `<your_slack_command_endpoint>` (from deployment output)
   - **Short Description**: `Request SSM access to a resource`
   - **Usage Hint**: `host=db.example.com port=5432 account=123456789012 [expires=2026-12-31]`
   - Check **"Escape channels, users, and links sent to your app"**
6. Click **"Save"**

### Step 2: Add /ssm-admin Slash Command

1. Still in **"Slash Commands"**, click **"Create New Command"** again
2. Fill in:
   - **Command**: `/ssm-admin`
   - **Request URL**: `<your_slack_admin_endpoint>` (from deployment output)
   - **Short Description**: `Manage SSM Access Manager users`
   - **Usage Hint**: `add-approval-group group_id=<id> name=<name> type=<security|manager> | list-approval-groups | help`
   - Check **"Escape channels, users, and links sent to your app"**
3. Click **"Save"**

### Step 3: Enable Interactive Components

1. In the left sidebar, click **"Interactivity & Shortcuts"**
2. Toggle **"Interactivity"** to **ON**
3. **Request URL**: `<your_slack_interaction_endpoint>` (from deployment output)
4. Click **"Save Changes"**

### Step 4: Reinstall App

If you see a banner saying "Please reinstall your app":
1. Click **"reinstall your app"**
2. Click **"Allow"**

## Part 4: Configure Target AWS Account (15 minutes)

You need to create an IAM role in each AWS account where you want to create SSM documents.

### Step 1: Get Your Lambda Execution Role ARN

```bash
aws iam get-role \
  --role-name test-ssm-access-manager-lambda-role \
  --query 'Role.Arn' \
  --output text
```

Save this ARN (looks like `arn:aws:iam::123456789012:role/test-ssm-access-manager-lambda-role`)

### Step 2: Create IAM Role in Target Account

**If target account is the same as deployment account**, run this in your current account:

```bash
# Create the role
aws iam create-role \
  --role-name SSMDocumentManagerRole \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {
        "AWS": "<YOUR_LAMBDA_EXECUTION_ROLE_ARN>"
      },
      "Action": "sts:AssumeRole"
    }]
  }'

# Attach the policy
aws iam put-role-policy \
  --role-name SSMDocumentManagerRole \
  --policy-name SSMDocumentManagement \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Action": [
        "ssm:CreateDocument",
        "ssm:DeleteDocument",
        "ssm:DescribeDocument",
        "ssm:ListDocuments",
        "ssm:AddTagsToResource",
        "ssm:RemoveTagsFromResource",
        "ssm:ListTagsForResource"
      ],
      "Resource": "*"
    }]
  }'
```

**If target account is different**, you'll need to:
1. Log into the target account
2. Run the same commands above
3. Replace `<YOUR_LAMBDA_EXECUTION_ROLE_ARN>` with the ARN from Step 1

### Step 3: Add Target Account to System

```bash
# Get your AWS account ID
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Add the account
curl -X POST <YOUR_ADMIN_ENDPOINT> \
  -H "Content-Type: application/json" \
  -d "{
    \"action\": \"add_account\",
    \"admin_id\": \"<YOUR_SLACK_USER_ID>\",
    \"account_id\": \"$ACCOUNT_ID\",
    \"account_name\": \"Test Account\",
    \"role_name\": \"SSMDocumentManagerRole\",
    \"regions\": [\"us-east-1\"]
  }"
```

You should see: `{"message":"Account added successfully","account_id":"123456789012"}`

## Part 5: Configure Approval Groups (10 minutes)

The system uses a two-tier approval system requiring both security and manager approvals.

### Step 1: Create Slack User Groups

1. In Slack, go to **People & user groups** → **User groups**
2. Create a **Security Team** group:
   - Click **"Create User Group"**
   - **Name**: `Security Team`
   - **Handle**: `@ccr-sec` (or your preferred handle)
   - Add security team members
   - Click **"Create"**

3. Create one or more **Manager** groups:
   - Click **"Create User Group"**
   - **Name**: `SRE Cloud OPS` (or your team name)
   - **Handle**: `@ccr-cloudops` (or your preferred handle)
   - Add manager team members
   - Click **"Create"**

### Step 2: Get User Group IDs

You need the Group IDs to configure the system. Get them using the Slack API:

```bash
# Replace with your bot token
curl -H "Authorization: Bearer xoxb-your-bot-token" \
  "https://slack.com/api/usergroups.list" | jq '.usergroups[] | {id, name, handle}'
```

Example output:
```json
{
  "id": "S0AFPD6TLQ4",
  "name": "Security Team",
  "handle": "ccr-sec"
}
{
  "id": "S0AFYN85MLH",
  "name": "SRE Cloud OPS",
  "handle": "ccr-cloudops"
}
```

Save these Group IDs - you'll need them in the next step.

### Step 3: Configure Approval Groups in the System

Use the `/ssm-admin` command in Slack to configure the groups:

```
/ssm-admin add-approval-group group_id=S0AFPD6TLQ4 name="Security Team" type=security
/ssm-admin add-approval-group group_id=S0AFYN85MLH name="SRE Cloud OPS" type=manager
```

You should see confirmation messages for each group configured.

**Note:** You need exactly one security group and at least one manager group.

## Part 6: Add Administrators (5 minutes)

Administrators can manage users, accounts, and approval groups. Regular users just need to be members of the appropriate Slack user groups (security or manager groups) to approve requests.

### Method 1: Using Slack Commands (Easiest!)

Once you're an administrator, you can manage other administrators directly from Slack:

```
/ssm-admin add-admin @jane.admin
/ssm-admin list-admins
/ssm-admin remove-admin @john.doe
```

**Benefits:**
- No need to know user IDs or emails
- Just mention the user with @username
- Instant feedback in Slack
- Perfect for day-to-day management

**Example workflow:**
```
/ssm-admin add-admin @jane.admin
```

Response:
```
✅ Administrator Added

@jane.admin is now an administrator and can manage users, accounts, and approval groups.
```

### Method 2: Using Scripts (For Automation)

```bash
# Get Slack user ID:
# 1. In Slack, click on user's profile
# 2. Click "More" (three dots)
# 3. Click "Copy member ID"

# Add an administrator
./scripts/add-user.sh test U12345678 jane.admin jane@example.com administrator us-east-1

# List all users
./scripts/list-users.sh test us-east-1
```

### Method 3: Using the Admin API

```bash
curl -X POST <YOUR_ADMIN_ENDPOINT> \
  -H "Content-Type: application/json" \
  -d '{
    "action": "add_admin",
    "admin_id": "<YOUR_SLACK_USER_ID>",
    "user_id": "<NEW_ADMIN_SLACK_USER_ID>",
    "email": "admin@example.com"
  }'
```

### User Roles

- **Administrator**: Can manage users, accounts, approval groups, and approve/deny requests
- **Approval Group Members**: Users in Slack user groups (security/manager) can approve requests - no system role needed!

### Quick Reference: Slack Admin Commands

```bash
# Administrator Management
/ssm-admin add-admin @user
/ssm-admin remove-admin @user
/ssm-admin list-admins

# Approval Group Configuration
/ssm-admin add-approval-group group_id=S0AFPD6TLQ4 name="Security Team" type=security
/ssm-admin add-approval-group group_id=S0AFYN85MLH name="SRE Cloud OPS" type=manager

# Request Management
/ssm-admin list-requests
/ssm-admin approve-request <request_id>
/ssm-admin deny-request <request_id> <reason>
/ssm-admin cancel-request <request_id>

# Get help
/ssm-admin help
```

## Part 7: Test the System (10 minutes)

### Test 1: Test Admin Commands

In Slack, type:
```
/ssm-admin help
```

You should see the help message with all available commands.

Try listing approval groups:
```
/ssm-admin list-approval-groups
```

List all users:
```
/ssm-admin list-users
```

### Test 2: Submit an Access Request

In Slack, type:
```
/ssm-access
```

A modal will appear. Fill in:
- **Host**: `test.example.com`
- **Port**: `5432`
- **Account**: Select from dropdown
- **Manager Group**: Select from dropdown (e.g., "SRE Cloud OPS")
- **Expiration Date**: Leave blank for 14 days, or enter `2026-12-31`

Click **"Submit"**.

You should receive:
- ✅ Confirmation message that request was submitted
- ✅ All members of security and manager groups receive approval requests with buttons

### Test 3: Approve the Request (Two-Tier Approval)

The request requires both security and manager approvals:

**As a security team member:**
1. Click the **"Approve"** button in the Slack message
2. If you're in both groups, select **"Approve as Security Team"**
3. You should see: "✅ Security Approval Recorded"
4. All approvers' messages update to show partial approval status

**As a manager team member:**
1. Click the **"Approve"** button
2. If you're in both groups, select **"Approve as SRE Cloud OPS"**
3. You should see: "✅ Manager Approval Recorded - Request is now fully approved"
4. All approvers' messages update to show full approval
5. The requester receives an approval confirmation with both approvers listed

### Test 4: Verify Document Creation

Wait 1-2 minutes, then check CloudWatch logs:

```bash
aws logs tail /aws/lambda/test-ssm-document-creator --follow
```

You should see logs indicating the document was created.

### Test 5: List Documents

```bash
aws ssm list-documents \
  --filters Key=Owner,Values=Self \
  --query 'DocumentIdentifiers[?starts_with(Name, `PF-`)].Name'
```

You should see your document: `PF-<username>-test.example.com-5432`

**Note:** The document prefix is configurable via the `DOCUMENT_PREFIX` environment variable (default: "PF" for PortForwarding).

## Troubleshooting

### Slash Command Not Working

**Check:**
1. Verify the slash command endpoint in Slack matches deployment output
2. Check API Gateway logs:
   ```bash
   aws logs tail /aws/lambda/test-ssm-request-handler --follow
   ```
3. Test the endpoint:
   ```bash
   curl -X POST <slack_command_endpoint> -d "test=1"
   ```

### Approval Buttons Not Working

**Check:**
1. Verify the interaction endpoint in Slack matches deployment output
2. Check approval handler logs:
   ```bash
   aws logs tail /aws/lambda/test-ssm-approval-handler --follow
   ```

### Document Not Created

**Check:**
1. Verify the account was added successfully:
   ```bash
   curl -X POST <admin_endpoint> \
     -H "Content-Type: application/json" \
     -d '{"action":"list_accounts","admin_id":"<YOUR_SLACK_USER_ID>"}'
   ```
2. Check document creator logs:
   ```bash
   aws logs tail /aws/lambda/test-ssm-document-creator --follow
   ```
3. Verify IAM role exists:
   ```bash
   aws iam get-role --role-name SSMDocumentManagerRole
   ```

### "Unauthorized" Errors

**Check:**
1. Verify Slack signing secret is correct
2. Regenerate signing secret in Slack and update Terraform:
   ```bash
   export TF_VAR_slack_signing_secret="new-secret"
   cd infrastructure/terraform
   terraform apply
   ```

## Next Steps

Now that your test deployment is working:

1. **Add more administrators**: Use `/ssm-admin add-admin @teammate` in Slack
2. **Update approval groups**: Add/remove members in Slack user groups as needed
3. **Add production accounts**: Configure IAM roles in production accounts
4. **Set up monitoring**: Configure CloudWatch alarms to send notifications
4. **Review security**: Audit IAM policies and access controls
5. **Train users**: Share the user guide with your team

## Cleanup

To remove everything:

```bash
cd infrastructure/terraform
terraform destroy \
  -var="environment=test" \
  -var="aws_region=us-east-1"
```

Type `yes` when prompted.

## Getting Help

- **User Guide**: See `docs/USER_GUIDE.md`
- **Admin Guide**: See `docs/ADMIN_GUIDE.md`
- **Operations Guide**: See `docs/OPERATIONS.md`
- **CloudWatch Logs**: Check Lambda function logs for errors
- **GitHub Issues**: Report bugs or request features

## Summary

You've successfully:
- ✅ Created a Slack app with bot token, signing secret, and team ID
- ✅ Deployed SSM Access Manager to AWS
- ✅ Configured Slack slash commands (/ssm-access and /ssm-admin)
- ✅ Configured interactive components for approval buttons
- ✅ Created Slack user groups for security and manager teams
- ✅ Configured two-tier approval system with approval groups
- ✅ Set up target AWS account with IAM role
- ✅ Added yourself as administrator
- ✅ Tested the complete two-tier approval workflow

**Important:** Approvals are now handled through Slack user groups. Users don't need to be added as "managers" in the system - they just need to be members of the security or manager Slack user groups!

Your SSM Access Manager is now ready for use!
