package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/ssm-access-manager/internal/models"
)

// AccountRepository handles DynamoDB operations for AWS accounts
type AccountRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(client *dynamodb.Client, tableName string) *AccountRepository {
	return &AccountRepository{
		client:    client,
		tableName: tableName,
	}
}

// SaveAccount saves an account to DynamoDB
func (r *AccountRepository) SaveAccount(ctx context.Context, account *models.Account) error {
	if err := account.Validate(); err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	item, err := attributevalue.MarshalMap(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	}

	_, err = r.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}

	return nil
}

// GetAccountByID retrieves an account by ID
func (r *AccountRepository) GetAccountByID(ctx context.Context, accountID string) (*models.Account, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"account_id": &types.AttributeValueMemberS{Value: accountID},
		},
	}

	result, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if result.Item == nil {
		return nil, nil // Account not found
	}

	var account models.Account
	err = attributevalue.UnmarshalMap(result.Item, &account)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	return &account, nil
}

// ListAllAccounts retrieves all accounts
func (r *AccountRepository) ListAllAccounts(ctx context.Context) ([]*models.Account, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	}

	result, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	var accounts []*models.Account
	err = attributevalue.UnmarshalListOfMaps(result.Items, &accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	return accounts, nil
}

// ListActiveAccounts retrieves all active accounts
func (r *AccountRepository) ListActiveAccounts(ctx context.Context) ([]*models.Account, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("status-account_name-index"),
		KeyConditionExpression: aws.String("#status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(models.AccountStatusActive)},
		},
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list active accounts: %w", err)
	}

	var accounts []*models.Account
	err = attributevalue.UnmarshalListOfMaps(result.Items, &accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	return accounts, nil
}

// DeleteAccount deletes an account
func (r *AccountRepository) DeleteAccount(ctx context.Context, accountID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"account_id": &types.AttributeValueMemberS{Value: accountID},
		},
	}

	_, err := r.client.DeleteItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

// UpdateAccountStatus updates an account's status
func (r *AccountRepository) UpdateAccountStatus(ctx context.Context, accountID string, status models.AccountStatus) error {
	if !status.IsValid() {
		return fmt.Errorf("invalid status: %s", status)
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"account_id": &types.AttributeValueMemberS{Value: accountID},
		},
		UpdateExpression: aws.String("SET #status = :status, updated_at = :updated_at"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":     &types.AttributeValueMemberS{Value: string(status)},
			":updated_at": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
	}

	_, err := r.client.UpdateItem(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update account status: %w", err)
	}

	return nil
}

// AccountExists checks if an account exists
func (r *AccountRepository) AccountExists(ctx context.Context, accountID string) (bool, error) {
	account, err := r.GetAccountByID(ctx, accountID)
	if err != nil {
		return false, err
	}
	return account != nil, nil
}

// AccountOption represents an account option for dropdown selection
type AccountOption struct {
	Text  string // Display name (account name)
	Value string // Account ID
}

// ListActiveAccountsForDropdown retrieves active accounts formatted for dropdown selection
func (r *AccountRepository) ListActiveAccountsForDropdown(ctx context.Context) ([]AccountOption, error) {
	accounts, err := r.ListActiveAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active accounts: %w", err)
	}

	options := make([]AccountOption, len(accounts))
	for i, account := range accounts {
		options[i] = AccountOption{
			Text:  account.AccountName,
			Value: account.AccountID,
		}
	}

	return options, nil
}
