# SSM Access Manager Documentation

Complete documentation for SSM Access Manager.

## Quick Links

- **[Getting Started](../QUICKSTART.md)** - Complete setup and deployment guide
- **[User Guide](USER_GUIDE.md)** - How to request and use access
- **[Administrator Guide](ADMIN_GUIDE.md)** - Managing the system
- **[Operations Guide](OPERATIONS.md)** - Monitoring and troubleshooting

---

## Documentation Files

### For Everyone

**[../QUICKSTART.md](../QUICKSTART.md)** - Complete setup and deployment guide (~1 hour)
- Deploy infrastructure with Terraform
- Configure Slack app and endpoints
- Set up approval groups
- Add administrators
- Configure target AWS accounts
- Test the complete workflow

### For End Users

**[USER_GUIDE.md](USER_GUIDE.md)** - How to request and use access
- Requesting access with `/ssm-access`
- Understanding the two-tier approval workflow
- Using SSM documents for port forwarding
- Checking request status

### For Administrators

**[ADMIN_GUIDE.md](ADMIN_GUIDE.md)** - Complete administrator guide
- Managing approval groups (security and manager groups)
- Managing administrators
- Managing AWS target accounts
- Managing access requests
- Monitoring and operations
- Troubleshooting
- Best practices

### For Operations

**[OPERATIONS.md](OPERATIONS.md)** - Day-to-day operations
- Monitoring CloudWatch logs
- Handling errors
- Performance tuning
- Security best practices
- Common troubleshooting scenarios

---

## Architecture

```
Slack Users
    ↓
API Gateway (/slack/command, /slack/admin, /slack/interaction)
    ↓
Lambda Functions (Request, Approval, Admin, Document Creator, Cleanup)
    ↓
DynamoDB (Requests, Documents, Users, Accounts, Approval Groups)
    ↓
Target AWS Accounts (SSM Documents)
```

## User Roles

| Role | Permissions |
|------|-------------|
| **User** | Request access via `/ssm-access` |
| **Approval Group Member** | Approve/deny requests (via Slack user groups) |
| **Administrator** | Manage system via `/ssm-admin` |

## Slack Commands

### `/ssm-access` - Request Access (All Users)

Opens a modal form to request access:
- Select AWS account from dropdown
- Enter hostname and port
- Select manager group for approval
- Set expiration date (optional, defaults to 14 days)

### `/ssm-admin` - Manage System (Administrators Only)

```
# Approval Groups
/ssm-admin set-approval-group <type> <group_id>
/ssm-admin list-approval-groups
/ssm-admin remove-approval-group <group_id>

# Administrators
/ssm-admin add-admin @user
/ssm-admin remove-admin @user
/ssm-admin list-admins

# AWS Accounts
/ssm-admin add-account account_id=<id> account_name=<name> role_name=<role> regions=<regions>
/ssm-admin list-accounts

# Access Requests
/ssm-admin list-requests [pending|active|all]
/ssm-admin revoke-request <id> reason="<reason>"

# Help
/ssm-admin help
```

---

## Quick Reference by Task

### I want to...

**Deploy the system**
→ [QUICKSTART.md](../QUICKSTART.md)

**Request access to a resource**
→ [USER_GUIDE.md](USER_GUIDE.md)

**Approve an access request**
→ [USER_GUIDE.md](USER_GUIDE.md#approving-requests)

**Configure approval groups**
→ [ADMIN_GUIDE.md](ADMIN_GUIDE.md#managing-approval-groups)

**Add an administrator**
→ [ADMIN_GUIDE.md](ADMIN_GUIDE.md#managing-administrators)

**Add an AWS account**
→ [ADMIN_GUIDE.md](ADMIN_GUIDE.md#managing-aws-accounts)

**Revoke active access**
→ [ADMIN_GUIDE.md](ADMIN_GUIDE.md#revoking-active-access)

**Troubleshoot issues**
→ [OPERATIONS.md](OPERATIONS.md#troubleshooting) or [ADMIN_GUIDE.md](ADMIN_GUIDE.md#troubleshooting)

**Monitor the system**
→ [OPERATIONS.md](OPERATIONS.md) or [ADMIN_GUIDE.md](ADMIN_GUIDE.md#monitoring-and-operations)

---

## Getting Help

1. **Check the documentation** - Most questions are answered here
2. **Review CloudWatch logs** - Detailed error messages and stack traces
3. **Test with help commands** - `/ssm-admin help` shows all available commands
4. **Check GitHub Issues** - Known issues and solutions

---

## Contributing to Documentation

When adding new features:

1. Update [QUICKSTART.md](../QUICKSTART.md) if setup changes
2. Update [ADMIN_GUIDE.md](ADMIN_GUIDE.md) for admin features
3. Update [USER_GUIDE.md](USER_GUIDE.md) for user-facing features
4. Update this index (docs/README.md)
5. Update main [README.md](../README.md) if needed

---

## License

[Your License Here]

