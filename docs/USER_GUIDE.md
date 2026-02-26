# SSM Access Manager - User Guide

## Overview

SSM Access Manager allows you to request temporary access to resources via AWS Systems Manager Session Manager with port forwarding. All requests are submitted through Slack and require manager approval.

## For End Users

### Requesting Access

You can request access in two ways:

#### Option 1: Using the Modal (Recommended)

Simply type `/ssm-access` in Slack without any parameters. A modal form will appear where you can:
- Select an account from a dropdown
- Enter the hostname or IP address
- Enter the port number
- Optionally set an expiration date (defaults to 14 days)

Click "Submit" to send your request.

#### Option 2: Using Command Parameters

Use the `/ssm-access` slash command in Slack with the following parameters:

```
/ssm-access host=<hostname> port=<port> account=<aws_account_id> [expires=<date>]
```

**Parameters:**
- `host`: The hostname or IP address you want to access (e.g., `db.rds.amazonaws.com` or `10.0.1.50`)
- `port`: The port number (e.g., `5432` for PostgreSQL, `3306` for MySQL)
- `account`: The AWS account ID where the resource is located (12-digit number)
- `expires`: (Optional) Expiration date in YYYY-MM-DD format. If not provided, defaults to 14 days from now. Maximum 90 days.

**Examples:**
```
/ssm-access host=db.rds.amazonaws.com port=3306 account=123456789012 expires=2026-03-31
```

```
/ssm-access host=10.0.1.50 port=5432 account=123456789012
```
(This will default to 14 days expiration)

### What Happens Next

1. You'll receive a confirmation message that your request was submitted
2. All managers will be notified and can approve or deny your request
3. Once approved, an SSM document will be created automatically
4. You'll receive a notification with the document name and details
5. You can now use the SSM document to establish a session

### Using the SSM Document

Once your request is approved and the document is created, use the AWS CLI to start a session:

```bash
aws ssm start-session \
  --target <instance-id> \
  --document-name PF-<username>-<host>-<port> \
  --parameters portNumber=<port>,localPortNumber=<port>
```

**Example:**
```bash
aws ssm start-session \
  --target i-1234567890abcdef0 \
  --document-name PF-john.doe-db.example.com-5432 \
  --parameters portNumber=5432,localPortNumber=5432
```

**Note:** The document prefix "PF" (PortForwarding) is configurable. If your organization uses a different prefix, replace "PF" with your configured prefix.

This creates a port forwarding tunnel. You can then connect to `localhost:5432` to access the remote resource.

### Access Expiration

- Your access automatically expires on the date you specified
- You'll receive a notification when your access expires
- The SSM document will be automatically deleted
- To extend access, submit a new request

### Troubleshooting

**"Missing Required Fields" Error:**
- Ensure host, port, and account parameters are provided (expires is optional)
- Check that the date format is YYYY-MM-DD if you provide expires
- Verify the account ID is exactly 12 digits
- Ensure the hostname is valid (e.g., `db.rds.amazonaws.com` or IP address)

**Request Denied:**
- Contact the manager who denied your request for the reason
- Address any concerns and submit a new request

**Can't Connect After Approval:**
- Wait a few minutes for the document to be created
- Verify you're using the correct document name
- Check that you have the necessary IAM permissions
- Ensure the target instance has SSM agent installed

## For Managers

### Approving Requests

When a user submits a request, you'll receive a Slack message with:
- User information
- Requested host and port
- AWS account
- Expiration date
- Request ID

**To approve:** Click the "Approve" button
**To deny:** Click the "Deny" button

### Best Practices

- Review the requested host and port for legitimacy
- Verify the expiration date is reasonable
- Check if the user should have access to the specified account
- Consider the principle of least privilege
- Deny requests that seem suspicious or unnecessary

### What Happens After Approval

1. The system automatically creates an SSM document in the target account
2. The user receives a notification with document details
3. The document is tagged with expiration date and user information
4. The document will be automatically deleted when it expires

## For Administrators

### Managing Managers

**Add a manager:**
```bash
curl -X POST <admin_endpoint> \
  -H "Content-Type: application/json" \
  -d '{
    "action": "add_manager",
    "admin_id": "<your_slack_user_id>",
    "user_id": "<manager_slack_user_id>",
    "email": "manager@example.com"
  }'
```

**Remove a manager:**
```bash
curl -X POST <admin_endpoint> \
  -H "Content-Type: application/json" \
  -d '{
    "action": "remove_manager",
    "admin_id": "<your_slack_user_id>",
    "user_id": "<manager_slack_user_id>"
  }'
```

**List all managers:**
```bash
curl -X POST <admin_endpoint> \
  -H "Content-Type: application/json" \
  -d '{
    "action": "list_managers",
    "admin_id": "<your_slack_user_id>"
  }'
```

### Managing Administrators

**Add an administrator:**
```bash
curl -X POST <admin_endpoint> \
  -H "Content-Type: application/json" \
  -d '{
    "action": "add_administrator",
    "admin_id": "<your_slack_user_id>",
    "user_id": "<new_admin_slack_user_id>",
    "email": "admin@example.com"
  }'
```

**Remove an administrator:**
```bash
curl -X POST <admin_endpoint> \
  -H "Content-Type: application/json" \
  -d '{
    "action": "remove_administrator",
    "admin_id": "<your_slack_user_id>",
    "user_id": "<admin_slack_user_id>"
  }'
```

**Note:** You cannot remove your own administrator privileges.

### Managing AWS Accounts

**Add an account:**
```bash
curl -X POST <admin_endpoint> \
  -H "Content-Type: application/json" \
  -d '{
    "action": "add_account",
    "admin_id": "<your_slack_user_id>",
    "account_id": "123456789012",
    "account_name": "Production",
    "role_name": "SSMDocumentManagerRole",
    "regions": ["us-east-1", "us-west-2"]
  }'
```

**Remove an account:**
```bash
curl -X POST <admin_endpoint> \
  -H "Content-Type: application/json" \
  -d '{
    "action": "remove_account",
    "admin_id": "<your_slack_user_id>",
    "account_id": "123456789012"
  }'
```

**List all accounts:**
```bash
curl -X POST <admin_endpoint> \
  -H "Content-Type: application/json" \
  -d '{
    "action": "list_accounts",
    "admin_id": "<your_slack_user_id>"
  }'
```

### Finding Slack User IDs

To find a user's Slack ID:
1. Click on their profile in Slack
2. Click "More" (three dots)
3. Click "Copy member ID"

Or use the Slack API:
```bash
curl -H "Authorization: Bearer <slack_bot_token>" \
  "https://slack.com/api/users.list"
```

## Security Considerations

### For Users
- Only request access when needed
- Use the shortest reasonable expiration date
- Don't share SSM document names with others
- Report any suspicious activity

### For Managers
- Review all requests carefully
- Deny requests that don't have a clear business justification
- Monitor for unusual patterns (e.g., same user requesting many resources)
- Report suspicious requests to administrators

### For Administrators
- Regularly review the list of managers and administrators
- Audit access logs in CloudWatch
- Remove inactive accounts promptly
- Ensure target account IAM roles follow least privilege
- Rotate Slack tokens periodically

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review CloudWatch logs for error details
3. Contact your system administrator
4. Refer to the operational documentation for advanced troubleshooting
