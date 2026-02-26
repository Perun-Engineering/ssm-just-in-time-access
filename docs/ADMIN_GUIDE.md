# Administrator Guide

Complete guide for managing SSM Access Manager as an administrator.

## Table of Contents

1. [Overview](#overview)
2. [Administrator Responsibilities](#administrator-responsibilities)
3. [Getting Started](#getting-started)
4. [Managing Approval Groups](#managing-approval-groups)
5. [Managing Administrators](#managing-administrators)
6. [Managing AWS Accounts](#managing-aws-accounts)
7. [Managing Access Requests](#managing-access-requests)
8. [Monitoring and Operations](#monitoring-and-operations)
9. [Troubleshooting](#troubleshooting)
10. [Best Practices](#best-practices)

## Overview

As an administrator, you manage the SSM Access Manager system including approval groups, other administrators, AWS accounts, and access requests. Most operations can be performed directly from Slack using the `/ssm-admin` command.

### Administrator Capabilities

- Configure approval groups (security and manager groups)
- Add/remove other administrators
- Add/remove AWS target accounts
- View and manage access requests
- Approve/deny requests (in addition to group-based approvals)
- Revoke active access
- View audit logs

## Administrator Responsibilities

### Daily Operations
- Monitor approval requests
- Respond to access issues
- Review audit logs

### Weekly Tasks
- Review active access requests
- Check for expired documents
- Verify approval group membership

### Monthly Tasks
- Audit administrator list
- Review AWS account configurations
- Update approval group configurations if needed
- Review security and compliance

## Getting Started

### Verify Your Administrator Access

Test that you have administrator privileges:

```
/ssm-admin help
```

You should see the complete command reference. If you get an "Unauthorized" error, you need to be added as an administrator first.

### Initial Setup Checklist

After deployment, complete these setup tasks:

- [ ] Configure security approval group
- [ ] Configure at least one manager approval group  
- [ ] Add other administrators
- [ ] Add target AWS accounts
- [ ] Test the approval workflow
- [ ] Review CloudWatch logs and alarms

## Managing Approval Groups

Approval groups are Slack user groups that control who can approve access requests. The system requires two-tier approval: security + manager.

### Understanding Approval Groups

**Security Group** (Required - exactly one)
- Members can provide security approval
- Typically your security or compliance team
- Required for all access requests

**Manager Groups** (Required - at least one)
- Members can provide manager approval
- Can have multiple manager groups (e.g., per team or service)
- Requesters select which manager group should approve their request

### Creating Slack User Groups

Before configuring approval groups in the system, create them in Slack:

1. In Slack, go to **People & user groups** → **User groups**
2. Click **Create User Group**
3. Enter name and handle (e.g., "Security Team" with handle "@security")
4. Add members
5. Save the group

### Getting Slack Group IDs

You need the Group ID to configure approval groups:

```bash
curl -H "Authorization: Bearer xoxb-your-bot-token" \
  "https://slack.com/api/usergroups.list" | jq '.usergroups[] | {id, name, handle}'
```

Example output:
```json
{
  "id": "S0AFPD6TLQ4",
  "name": "Security Team",
  "handle": "security"
}
```

### Configuring Approval Groups

#### Set Security Group

```
/ssm-admin set-approval-group security S0AFPD6TLQ4
```

Response:
```
✅ Approval Group Configured

Type: Security
Group ID: S0AFPD6TLQ4
Group Name: Security Team
```

#### Set Manager Groups

```
/ssm-admin set-approval-group manager S0AFYN85MLH
```

You can configure multiple manager groups by running the command multiple times with different group IDs.

#### List Approval Groups

```
/ssm-admin list-approval-groups
```

Response:
```
📋 Approval Groups

Security Group:
• Security Team (S0AFPD6TLQ4)

Manager Groups:
• SRE Cloud OPS (S0AFYN85MLH)
• Database Team (S0AFYN85ABC)
```

#### Remove Approval Group

```
/ssm-admin remove-approval-group <group_id>
```

**Warning:** Removing an approval group will prevent new requests from being approved by that group. Existing pending requests may fail.

### Managing Group Membership

Group membership is managed entirely in Slack:

1. Go to **People & user groups** → **User groups** in Slack
2. Select the group
3. Add or remove members
4. Changes take effect immediately (cached for 5 minutes)

**No system configuration needed** - the system automatically checks current Slack group membership.

## Managing Administrators

Administrators can manage the system via `/ssm-admin` commands.

### Adding Administrators

#### Via Slack (Recommended)

```
/ssm-admin add-admin @jane.doe
```

Response:
```
✅ Administrator Added

@jane.doe is now an administrator and can manage users, accounts, and approval groups.
```

The bot automatically fetches the user's email from Slack.

#### Via Script

```bash
# Get the user's Slack ID first (Profile → More → Copy member ID)
./scripts/add-user.sh test U12345678 jane.doe jane@example.com us-east-1
```

### Removing Administrators

```
/ssm-admin remove-admin @jane.doe
```

Response:
```
✅ Administrator Removed

@jane.doe is no longer an administrator.
```

**Note:** You cannot remove your own administrator privileges.

### Listing Administrators

```
/ssm-admin list-admins
```

Response:
```
📋 Administrators

• @alice.admin (alice@example.com)
• @bob.admin (bob@example.com)
• @charlie.admin (charlie@example.com)
```

### Administrator vs Approval Group Member

**Administrator:**
- Can manage system configuration
- Can add/remove other administrators
- Can configure approval groups
- Can manage AWS accounts
- Can approve/deny any request (override capability)

**Approval Group Member:**
- Can approve/deny requests (via group membership)
- No system management capabilities
- Managed through Slack user groups

Most users should be approval group members, not administrators.

## Managing AWS Accounts

Target AWS accounts must be configured before users can request access to resources in those accounts.

### Prerequisites

Each target account needs an IAM role that allows the SSM Access Manager to create documents. See the [Cross-Account Setup](#cross-account-setup) section below.

### Adding AWS Accounts

```
/ssm-admin add-account account_id=123456789012 account_name="Production" role_name=SSMDocumentManagerRole regions=us-east-1,us-west-2 bastion_host_id=i-1234567890abcdef0
```

Parameters:
- `account_id` - AWS account ID (required)
- `account_name` - Friendly name (required)
- `role_name` - IAM role name in target account (required)
- `regions` - Comma-separated list of regions (required)
- `bastion_host_id` - EC2 instance ID for port forwarding (optional)

Response:
```
✅ Account Added

Account ID: 123456789012
Account Name: Production
Role Name: SSMDocumentManagerRole
Regions: us-east-1, us-west-2
Bastion Host ID: i-1234567890abcdef0
```

### Updating AWS Accounts

```
/ssm-admin update-account account_id=123456789012 account_name="Production Updated" role_name=SSMDocumentManagerRole regions=us-east-1,us-west-2,eu-west-1 bastion_host_id=i-newinstance123
```

All parameters except `account_id` can be updated.

### Listing AWS Accounts

```
/ssm-admin list-accounts
```

Response:
```
📋 AWS Accounts

Production (123456789012)
• Role: SSMDocumentManagerRole
• Regions: us-east-1, us-west-2
• Bastion Host: i-1234567890abcdef0
• Status: active

Development (987654321098)
• Role: SSMDocumentManagerRole
• Regions: us-east-1
• Status: active
```

### Cross-Account Setup

Each target AWS account needs an IAM role with:

1. **Trust Policy** allowing the Lambda execution role to assume it:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": {
      "AWS": "arn:aws:iam::CENTRAL_ACCOUNT:role/ENV-ssm-access-manager-lambda-role"
    },
    "Action": "sts:AssumeRole"
  }]
}
```

2. **Permissions Policy** for SSM document management:

```json
{
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
    "Resource": "arn:aws:ssm:*:*:document/PF-*"
  }]
}
```

**Quick Setup Script:**

```bash
# In the target account
aws iam create-role \
  --role-name SSMDocumentManagerRole \
  --assume-role-policy-document file://trust-policy.json

aws iam put-role-policy \
  --role-name SSMDocumentManagerRole \
  --policy-name SSMDocumentManagement \
  --policy-document file://permissions-policy.json
```

## Managing Access Requests

### Listing Requests

```
/ssm-admin list-requests [pending|active|all]
```

- `pending` - Requests awaiting approval (default)
- `active` - Approved requests not yet expired
- `all` - All requests regardless of status

Response:
```
📋 Access Requests (pending)

Request ID: c75bcdd5-cd44-4048-9c6a-b42e18b8451f
• User: @john.doe
• Host: db.example.com:5432
• Account: 123456789012
• Expires: 2026-12-31
• Status: pending

Request ID: a1b2c3d4-e5f6-7890-abcd-ef1234567890
• User: @jane.smith
• Host: api.example.com:443
• Account: 987654321098
• Expires: 2026-12-25
• Status: partially_approved
```

### Approving Requests (Admin Override)

As an administrator, you can approve requests directly:

```
/ssm-admin approve-request c75bcdd5-cd44-4048-9c6a-b42e18b8451f
```

**Note:** This bypasses the normal two-tier approval workflow. Use sparingly and only for urgent situations.

### Denying Requests

```
/ssm-admin deny-request c75bcdd5-cd44-4048-9c6a-b42e18b8451f "Insufficient justification"
```

The reason is required and will be sent to the requester.

### Canceling Requests

```
/ssm-admin cancel-request c75bcdd5-cd44-4048-9c6a-b42e18b8451f
```

Cancels any request regardless of status. Use for requests that should not have been submitted.

### Revoking Active Access

```
/ssm-admin revoke-request c75bcdd5-cd44-4048-9c6a-b42e18b8451f reason="Security incident"
```

This:
- Marks the request as revoked
- Deletes the SSM document
- Notifies the requester
- Logs the revocation with reason

## Monitoring and Operations

### CloudWatch Logs

View logs for each Lambda function:

```bash
# Request handler (slash command)
aws logs tail /aws/lambda/ENV-ssm-request-handler --follow

# Approval handler (button clicks)
aws logs tail /aws/lambda/ENV-ssm-approval-handler --follow

# Admin handler (admin commands)
aws logs tail /aws/lambda/ENV-ssm-admin-slack-handler --follow

# Document creator (SSM document creation)
aws logs tail /aws/lambda/ENV-ssm-document-creator --follow

# Cleanup (expired document deletion)
aws logs tail /aws/lambda/ENV-ssm-expiration-cleanup --follow
```

### Audit Logs

Query audit logs in CloudWatch Logs Insights:

```
fields @timestamp, event_type, user_id, request_id, details
| filter event_type = "request_approved"
| sort @timestamp desc
| limit 100
```

Common event types:
- `request_created`
- `request_approved_security`
- `request_approved_manager`
- `request_denied`
- `request_revoked`
- `document_created`
- `document_deleted`
- `unauthorized_approval_attempt`

### Metrics and Alarms

CloudWatch alarms are automatically created for:
- Lambda errors
- Lambda duration
- DynamoDB throttling
- API Gateway errors

View them in CloudWatch → Alarms.

### Regular Maintenance

**Daily:**
- Check for failed document creations
- Review unauthorized approval attempts
- Monitor approval times

**Weekly:**
- Review active access requests
- Check for stuck requests
- Verify cleanup job is running

**Monthly:**
- Audit administrator list
- Review approval group membership
- Check AWS account configurations
- Review and update documentation

## Troubleshooting

### Users Can't Submit Requests

**Check:**
1. Verify `/ssm-access` command is configured in Slack
2. Check API Gateway endpoint is correct
3. View request handler logs for errors
4. Verify AWS accounts are configured

**Test:**
```bash
aws logs tail /aws/lambda/ENV-ssm-request-handler --follow
```

### Approval Buttons Not Working

**Check:**
1. Verify interactive components endpoint in Slack
2. Check approval handler logs
3. Verify approval groups are configured
4. Check user is member of approval group

**Test:**
```bash
aws logs tail /aws/lambda/ENV-ssm-approval-handler --follow
```

### Documents Not Created

**Check:**
1. Verify both approvals were granted
2. Check document creator logs
3. Verify IAM role exists in target account
4. Check role trust policy allows assumption

**Test:**
```bash
aws logs tail /aws/lambda/ENV-ssm-document-creator --follow

# Verify role can be assumed
aws sts assume-role \
  --role-arn "arn:aws:iam::TARGET_ACCOUNT:role/SSMDocumentManagerRole" \
  --role-session-name "test"
```

### Approval Group Members Not Receiving Notifications

**Check:**
1. Verify group is configured: `/ssm-admin list-approval-groups`
2. Check user is member of Slack user group
3. Verify bot has `usergroups:read` scope
4. Check group cache (5-minute TTL)

**Test:**
```bash
# List group members via Slack API
curl -H "Authorization: Bearer xoxb-token" \
  "https://slack.com/api/usergroups.users.list?usergroup=S0AFPD6TLQ4"
```

### "Unauthorized" Errors

**Check:**
1. Verify you're an administrator: `/ssm-admin list-admins`
2. Check Slack signing secret is correct
3. View admin handler logs for details

**Fix:**
```bash
# Add yourself as administrator
./scripts/add-user.sh ENV YOUR_USER_ID your.name your@email.com us-east-1
```

## Best Practices

### Security

1. **Limit Administrators** - Only give admin role to those who need it
2. **Regular Audits** - Review administrator list monthly
3. **Use Group-Based Approvals** - Don't use admin override unless necessary
4. **Monitor Audit Logs** - Review unauthorized attempts
5. **Rotate Slack Tokens** - Update tokens quarterly
6. **Enable MFA** - Require MFA for AWS console access

### Operational

1. **Document Changes** - Keep a log of configuration changes
2. **Test Before Production** - Test approval workflow in dev environment
3. **Monitor Metrics** - Set up CloudWatch alarms
4. **Regular Cleanup** - Review and revoke unnecessary access
5. **Keep Groups Updated** - Regularly review Slack group membership

### Approval Groups

1. **Clear Naming** - Use descriptive names (e.g., "Security Team", "SRE Cloud OPS")
2. **Appropriate Size** - Ensure enough members for coverage
3. **Regular Reviews** - Audit membership quarterly
4. **Document Responsibilities** - Clarify what each group approves
5. **Multiple Manager Groups** - Create groups per team or service

### AWS Accounts

1. **Consistent Naming** - Use clear, descriptive account names
2. **Document Bastion Hosts** - Keep bastion host IDs updated
3. **Multi-Region Support** - Configure all necessary regions
4. **Test Cross-Account Access** - Verify role assumption works
5. **Monitor Costs** - Track SSM document creation costs

## Command Reference

### Approval Groups
```
/ssm-admin set-approval-group <type> <group_id>
/ssm-admin list-approval-groups
/ssm-admin remove-approval-group <group_id>
```

### Administrators
```
/ssm-admin add-admin @user
/ssm-admin remove-admin @user
/ssm-admin list-admins
```

### AWS Accounts
```
/ssm-admin add-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]
/ssm-admin update-account account_id=<id> account_name=<name> role_name=<role> regions=<regions> [bastion_host_id=<id>]
/ssm-admin list-accounts
```

### Access Requests
```
/ssm-admin list-requests [pending|active|all]
/ssm-admin approve-request <request_id>
/ssm-admin deny-request <request_id> <reason>
/ssm-admin cancel-request <request_id>
/ssm-admin revoke-request <request_id> reason="<reason>"
```

### Help
```
/ssm-admin help
```

## Related Documentation

- [User Guide](USER_GUIDE.md) - For end users requesting access
- [Operations Guide](OPERATIONS.md) - Day-to-day operations and monitoring
- [Quick Start](../QUICKSTART.md) - Initial deployment and setup

## Getting Help

1. Check CloudWatch logs for errors
2. Review this guide for common issues
3. Test with `/ssm-admin help` to verify access
4. Check Slack app configuration
5. Verify AWS account and IAM role setup
