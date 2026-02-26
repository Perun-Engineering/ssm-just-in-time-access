# SSM Access Manager - Operations Guide

## Architecture Overview

The SSM Access Manager is a serverless application built on AWS with Slack integration:

- **Lambda Functions**: 5 functions handling different workflows
- **DynamoDB**: 4 tables for data persistence
- **API Gateway**: REST API for Slack webhooks
- **EventBridge**: Scheduled tasks and event-driven workflows
- **Secrets Manager**: Secure storage for Slack credentials
- **CloudWatch**: Logging and monitoring

## Monitoring

### CloudWatch Logs

Each Lambda function has its own log group, plus a dedicated audit log group:

```bash
# Audit logs (structured, immutable, 365-day retention)
aws logs tail /aws/ssm-access-manager/audit --follow --format json

# Request Handler logs
aws logs tail /aws/lambda/<environment>-ssm-request-handler --follow

# Approval Handler logs
aws logs tail /aws/lambda/<environment>-ssm-approval-handler --follow

# Document Creator logs
aws logs tail /aws/lambda/<environment>-ssm-document-creator --follow

# Expiration Cleanup logs
aws logs tail /aws/lambda/<environment>-ssm-expiration-cleanup --follow

# Admin Handler logs
aws logs tail /aws/lambda/<environment>-ssm-admin-handler --follow

# Admin Slack Handler logs
aws logs tail /aws/lambda/<environment>-ssm-admin-slack-handler --follow
```

**Note:** The audit log group contains structured JSON entries for all security-relevant events. Use CloudWatch Logs Insights for querying (see Security Operations section).

### CloudWatch Alarms

The following alarms are configured:

1. **Lambda Error Rate**: Triggers when error count > 5 in 5 minutes
2. **Lambda Duration**: Triggers when average duration > 10 seconds
3. **DynamoDB Throttle**: Triggers when throttle count > 10 in 5 minutes
4. **API Gateway 4XX**: Triggers when 4XX errors > 20 in 5 minutes
5. **API Gateway 5XX**: Triggers when 5XX errors > 5 in 5 minutes

### Key Metrics to Monitor

**Lambda Metrics:**
- Invocations
- Errors
- Duration
- Throttles
- Concurrent Executions

**DynamoDB Metrics:**
- ConsumedReadCapacityUnits
- ConsumedWriteCapacityUnits
- UserErrors (throttling)
- SystemErrors

**API Gateway Metrics:**
- Count (total requests)
- 4XXError
- 5XXError
- Latency

### Log Analysis Queries

**Application Logs (Lambda functions):**

Find all failed requests:
```
fields @timestamp, @message
| filter @message like /failed/ or @message like /error/
| sort @timestamp desc
| limit 100
```

Find slow operations:
```
fields @timestamp, @message
| filter @message like /duration/ and @message like /ms/
| parse @message /duration: (?<duration>\d+)ms/
| filter duration > 5000
| sort duration desc
```

**Audit Logs (Security events):**

See the "Security Operations > Audit Logging > Querying Audit Logs" section for comprehensive audit log queries, or use the Slack command:

```
/ssm-admin audit-logs [request_id=<id>] [user_id=<id>] [event_type=<type>]
```

## Incident Response

### High Error Rate

**Symptoms:**
- CloudWatch alarm triggered
- Users reporting failures
- High 5XX error rate in API Gateway

**Investigation:**
1. Check CloudWatch logs for error messages
2. Verify DynamoDB table status
3. Check Lambda function metrics
4. Verify Slack API status

**Common Causes:**
- DynamoDB throttling (increase capacity or use on-demand)
- Lambda timeout (increase timeout or optimize code)
- Slack API rate limiting (implement backoff)
- IAM permission issues (check Lambda execution role)

**Resolution:**
1. Identify root cause from logs
2. Apply appropriate fix (see common causes)
3. Monitor for improvement
4. Document incident and resolution

### Document Creation Failures

**Symptoms:**
- Users report approved requests but no document created
- Document creator logs show errors
- Slack notifications about creation failures

**Investigation:**
1. Check document-creator Lambda logs
2. Verify target account IAM role exists
3. Test role assumption manually
4. Check SSM API errors

**Common Causes:**
- IAM role doesn't exist in target account
- Trust policy doesn't allow Lambda role to assume it
- Insufficient SSM permissions on target role
- Network connectivity issues

**Resolution:**
1. Verify IAM role configuration in target account
2. Update trust policy if needed
3. Add missing SSM permissions
4. Retry failed document creation

### Expiration Cleanup Issues

**Symptoms:**
- Expired documents not being deleted
- Expiration cleanup Lambda errors
- Users report access still works after expiration

**Investigation:**
1. Check expiration-cleanup Lambda logs
2. Verify EventBridge rule is enabled
3. Check DynamoDB GSI for expired documents
4. Test role assumption for target accounts

**Common Causes:**
- EventBridge rule disabled
- Lambda timeout (too many documents to process)
- IAM role issues in target accounts
- DynamoDB query errors

**Resolution:**
1. Enable EventBridge rule if disabled
2. Increase Lambda timeout if needed
3. Fix IAM role issues
4. Manually delete stuck documents if necessary

### Slack Integration Issues

**Symptoms:**
- Slash commands not working
- Interactive buttons not responding
- 401 Unauthorized errors

**Investigation:**
1. Check API Gateway logs
2. Verify Slack signing secret is correct
3. Test Slack app configuration
4. Check Lambda function environment variables

**Common Causes:**
- Incorrect signing secret
- Slack app misconfigured
- API Gateway endpoint changed
- Lambda function not receiving requests

**Resolution:**
1. Verify Slack app configuration
2. Update signing secret if changed
3. Update Slack app endpoints
4. Test with Slack API tester

## Backup and Recovery

### DynamoDB Backups

Enable point-in-time recovery:

```bash
aws dynamodb update-continuous-backups \
  --table-name <environment>-ssm-access-requests \
  --point-in-time-recovery-specification PointInTimeRecoveryEnabled=true
```

Create on-demand backup:

```bash
aws dynamodb create-backup \
  --table-name <environment>-ssm-access-requests \
  --backup-name ssm-requests-backup-$(date +%Y%m%d)
```

### Restore from Backup

```bash
aws dynamodb restore-table-from-backup \
  --target-table-name <environment>-ssm-access-requests-restored \
  --backup-arn <backup-arn>
```

### Disaster Recovery

**RTO (Recovery Time Objective):** 1 hour
**RPO (Recovery Point Objective):** 5 minutes (with PITR)

**Recovery Steps:**
1. Restore DynamoDB tables from backup
2. Redeploy Lambda functions using Terraform
3. Verify API Gateway endpoints
4. Update Slack app if endpoints changed
5. Test end-to-end workflow
6. Notify users of service restoration

## Maintenance

### Updating Lambda Functions

```bash
# Build new versions
./scripts/build.sh

# Deploy updates
cd infrastructure/terraform
terraform apply -var="environment=<env>"
```

### Rotating Slack Credentials

1. Generate new token/secret in Slack app settings
2. Update Terraform variables:
   ```bash
   export TF_VAR_slack_bot_token="new-token"
   export TF_VAR_slack_signing_secret="new-secret"
   ```
3. Apply Terraform changes:
   ```bash
   terraform apply
   ```
4. Verify functionality

### Scaling Considerations

**DynamoDB:**
- Currently using PAY_PER_REQUEST (on-demand)
- No manual scaling needed
- Monitor for throttling

**Lambda:**
- Concurrent execution limit: 1000 (default)
- Increase if needed via AWS Support
- Monitor throttling metrics

**API Gateway:**
- Default limit: 10,000 requests/second
- Increase if needed via AWS Support

### Database Maintenance

**Clean up old data:**

```bash
# Delete requests older than 90 days
aws dynamodb scan \
  --table-name <environment>-ssm-access-requests \
  --filter-expression "created_at < :date" \
  --expression-attribute-values '{":date":{"S":"2024-01-01T00:00:00Z"}}' \
  | jq -r '.Items[].request_id.S' \
  | xargs -I {} aws dynamodb delete-item \
      --table-name <environment>-ssm-access-requests \
      --key '{"request_id":{"S":"{}"}}'
```

## Security Operations

### Audit Logging

All operations are logged to a dedicated CloudWatch Logs audit log group with structured JSON format. The audit log is immutable and provides a complete trail of all actions in the system.

**Audit Log Group:** `/aws/ssm-access-manager/audit` (or configured via `AUDIT_LOG_GROUP` environment variable)

**Retention:** 365 days (configured in Terraform)

#### Audit Log Structure

Each audit log entry contains:

```json
{
  "timestamp": "2024-03-15T10:30:00Z",
  "event_type": "request_created",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "actor": {
    "user_id": "U12345678",
    "username": "john.doe",
    "slack_team_id": "T08REEQ88F9"
  },
  "target": {
    "request_id": "req-123",
    "resource_type": "access_request"
  },
  "details": {
    "host": "db.example.com",
    "port": 5432,
    "account_id": "123456789012",
    "expiration_date": "2026-12-31T23:59:59Z",
    "manager_group_id": "S0AFYN85MLH",
    "manager_group_name": "SRE Cloud OPS"
  },
  "metadata": {
    "source_ip": "203.0.113.42",
    "user_agent": "Slackbot 1.0",
    "lambda_request_id": "abc-123-def-456"
  }
}
```

#### Audited Events

**Access Request Events:**
- `request_created` - New access request submitted
- `request_approved_security` - Security team approval granted
- `request_approved_manager` - Manager team approval granted
- `request_denied` - Request denied by approver
- `request_revoked` - Active request revoked by administrator
- `unauthorized_approval_attempt` - User attempted to approve without permission

**Document Events:**
- `document_created` - SSM document created in target account
- `document_deleted` - SSM document deleted (expiration or revocation)

**Approval Group Events:**
- `approval_group_added` - New approval group configured
- `approval_group_updated` - Approval group settings updated
- `approval_group_removed` - Approval group removed

**Account Events:**
- `account_added` - New AWS account added to system
- `account_updated` - Account configuration updated
- `account_removed` - Account removed from system

#### Querying Audit Logs

**Using Slack Command (Easiest):**

Generate CloudWatch Logs Insights queries directly from Slack:

```
/ssm-admin audit-logs request_id=c75bcdd5-cd44-4048-9c6a-b42e18b8451f
/ssm-admin audit-logs user_id=U12345678
/ssm-admin audit-logs event_type=request_approved_security
```

The command returns a ready-to-use CloudWatch Logs Insights query with instructions.

**Using CloudWatch Logs Insights:**

1. Go to AWS CloudWatch Console → Logs → Insights
2. Select log group: `/aws/ssm-access-manager/audit`
3. Use one of the queries below
4. Select time range and click "Run query"

**Common Audit Queries:**

Find all actions by a specific user:
```
fields @timestamp, event_type, target.request_id, details
| filter actor.user_id = "U12345678"
| sort @timestamp desc
```

Find all events for a specific request:
```
fields @timestamp, event_type, actor.username, details
| filter target.request_id = "c75bcdd5-cd44-4048-9c6a-b42e18b8451f"
| sort @timestamp desc
```

Find all denied requests:
```
fields @timestamp, actor.username, target.request_id, details.reason
| filter event_type = "request_denied"
| sort @timestamp desc
```

Find unauthorized approval attempts:
```
fields @timestamp, actor.user_id, actor.username, target.request_id
| filter event_type = "unauthorized_approval_attempt"
| sort @timestamp desc
```

Find all approval group changes:
```
fields @timestamp, event_type, actor.username, details
| filter event_type like /approval_group/
| sort @timestamp desc
```

Track a request through its lifecycle:
```
fields @timestamp, event_type, actor.username, details
| filter target.request_id = "req-123"
| sort @timestamp asc
```

Find all actions in the last 24 hours:
```
fields @timestamp, event_type, actor.username, target.request_id
| filter @timestamp > ago(24h)
| sort @timestamp desc
```

#### Audit Log Retention and Compliance

**Retention Policy:**
- Audit logs: 365 days (1 year)
- Lambda function logs: 30 days
- Can be extended via Terraform configuration

**Compliance Considerations:**
- All logs are encrypted at rest (CloudWatch default encryption)
- Logs are immutable once written
- Structured JSON format for easy parsing and analysis
- Contains complete actor, target, and action information
- Includes metadata (source IP, user agent, Lambda request ID)
- PII is sanitized in application logs but preserved in audit logs for accountability

**Export for Long-Term Storage:**

```bash
# Export audit logs to S3 for long-term retention
aws logs create-export-task \
  --log-group-name /aws/ssm-access-manager/audit \
  --from $(date -d '30 days ago' +%s)000 \
  --to $(date +%s)000 \
  --destination audit-logs-archive-bucket \
  --destination-prefix audit-logs/$(date +%Y/%m)
```

### Security Monitoring

**Monitor for:**
- Unauthorized access attempts (check `unauthorized_approval_attempt` events)
- Unusual request patterns (multiple requests from same user)
- Failed authentication (check Lambda logs for signature verification failures)
- Privilege escalation attempts (unauthorized admin actions)
- Suspicious document creation (unusual hosts, ports, or accounts)
- Approval group changes (monitor `approval_group_*` events)

**Query for security events:**
```
fields @timestamp, event_type, actor.user_id, actor.username, target.request_id
| filter event_type = "unauthorized_approval_attempt" or event_type like /approval_group/
| sort @timestamp desc
```

**Alert on suspicious activity:**
```
fields @timestamp, event_type, actor.user_id, count(*) as event_count
| filter event_type = "unauthorized_approval_attempt"
| stats count() by actor.user_id, bin(5m)
| filter event_count > 3
```

### Compliance

**Data Retention:**
- Audit logs: 365 days (CloudWatch Logs)
- Lambda function logs: 30 days (CloudWatch Logs)
- DynamoDB data: Indefinite (manual cleanup required)
- DynamoDB backups: 35 days (default PITR)

**Access Control:**
- Lambda execution role: Least privilege IAM policies
- API Gateway: Slack signature verification (no AWS authentication)
- DynamoDB: Encrypted at rest with AWS managed keys
- Secrets Manager: Encrypted Slack credentials
- Audit logs: Immutable, encrypted at rest

**Audit Trail:**
- All actions logged to dedicated audit log group
- Structured JSON format with actor, target, and action details
- 365-day retention for compliance requirements
- Can be exported to S3 for long-term archival

## Troubleshooting Guide

### Common Issues

**Issue: Slash command returns "Unauthorized"**
- Check Slack signing secret
- Verify API Gateway endpoint
- Check Lambda logs for signature verification errors

**Issue: Approval buttons don't work**
- Verify interaction endpoint in Slack app
- Check approval-handler Lambda logs
- Verify DynamoDB permissions

**Issue: Document not created after approval**
- Check document-creator Lambda logs
- Verify EventBridge rule is enabled
- Test IAM role assumption
- Check target account configuration

**Issue: Access not expiring**
- Check expiration-cleanup Lambda logs
- Verify EventBridge schedule rule
- Check DynamoDB GSI query
- Manually trigger cleanup Lambda

### Debug Mode

Enable debug logging:

```bash
# Update Lambda environment variable
aws lambda update-function-configuration \
  --function-name <environment>-ssm-request-handler \
  --environment Variables={LOG_LEVEL=debug}
```

### Manual Operations

**Manually trigger document creation:**
```bash
aws lambda invoke \
  --function-name <environment>-ssm-document-creator \
  --payload '{"detail":{"request_id":"req-123"}}' \
  response.json
```

**Manually trigger expiration cleanup:**
```bash
aws lambda invoke \
  --function-name <environment>-ssm-expiration-cleanup \
  response.json
```

## Performance Optimization

### Lambda Optimization
- Use provisioned concurrency for predictable workloads
- Optimize cold start time (currently ~200ms)
- Increase memory if CPU-bound operations

### DynamoDB Optimization
- Use sparse indexes for optional attributes
- Implement caching for frequently accessed data
- Use batch operations where possible

### API Gateway Optimization
- Enable caching for GET requests (if added)
- Use API Gateway throttling to protect backend
- Implement request validation

## Contact and Escalation

**On-Call Rotation:**
- Primary: [Team Lead]
- Secondary: [Senior Engineer]
- Escalation: [Engineering Manager]

**Slack Channels:**
- #ssm-access-manager-alerts
- #ssm-access-manager-support

**Runbook Location:**
- This document
- Internal wiki: [URL]
