package aws

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// AssumedCredentials holds assumed role credentials with expiration
type AssumedCredentials struct {
	Config     aws.Config
	ExpiresAt  time.Time
	AccountID  string
	RoleName   string
	Region     string
}

// IsExpired checks if the credentials are expired or about to expire (within 5 minutes)
func (c *AssumedCredentials) IsExpired() bool {
	return time.Now().Add(5 * time.Minute).After(c.ExpiresAt)
}

// RoleAssumer handles AWS role assumption and credential caching
type RoleAssumer struct {
	BaseConfig aws.Config
	cache      map[string]*AssumedCredentials
	cacheMu    sync.RWMutex
	cacheTTL   time.Duration
}

// NewRoleAssumer creates a new role assumer
func NewRoleAssumer(ctx context.Context) (*RoleAssumer, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &RoleAssumer{
		BaseConfig: cfg,
		cache:      make(map[string]*AssumedCredentials),
		cacheTTL:   55 * time.Minute, // Cache for 55 minutes (credentials valid for 1 hour)
	}, nil
}

// AssumeRole assumes a role in the target account and returns AWS credentials
func (r *RoleAssumer) AssumeRole(ctx context.Context, accountID, roleName, region string) (aws.Config, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s:%s", accountID, roleName, region)
	
	r.cacheMu.RLock()
	cached, exists := r.cache[cacheKey]
	r.cacheMu.RUnlock()

	if exists && !cached.IsExpired() {
		return cached.Config, nil
	}

	// Assume the role
	roleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
	
	stsClient := sts.NewFromConfig(r.BaseConfig)
	
	// Get environment name for external ID
	externalID := getEnvironmentName()
	
	// Create credentials provider that assumes the role
	provider := stscreds.NewAssumeRoleProvider(stsClient, roleARN, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = fmt.Sprintf("ssm-access-manager-%d", time.Now().Unix())
		o.Duration = time.Hour // Request 1 hour credentials
		if externalID != "" {
			o.ExternalID = &externalID
		}
	})

	// Create config with assumed role credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(provider),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to assume role %s in account %s: %w", roleName, accountID, err)
	}

	// Cache the credentials
	assumedCreds := &AssumedCredentials{
		Config:    cfg,
		ExpiresAt: time.Now().Add(r.cacheTTL),
		AccountID: accountID,
		RoleName:  roleName,
		Region:    region,
	}

	r.cacheMu.Lock()
	r.cache[cacheKey] = assumedCreds
	r.cacheMu.Unlock()

	return cfg, nil
}

// ValidateRoleAssumption validates that the role can be assumed
func (r *RoleAssumer) ValidateRoleAssumption(ctx context.Context, accountID, roleName string) error {
	roleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
	
	stsClient := sts.NewFromConfig(r.BaseConfig)
	
	// Get environment name for external ID
	externalID := getEnvironmentName()
	
	// Try to get caller identity with assumed role
	provider := stscreds.NewAssumeRoleProvider(stsClient, roleARN, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = fmt.Sprintf("ssm-access-manager-validation-%d", time.Now().Unix())
		o.Duration = 15 * time.Minute // Short duration for validation
		if externalID != "" {
			o.ExternalID = &externalID
		}
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(provider),
	)
	if err != nil {
		return fmt.Errorf("failed to validate role assumption: %w", err)
	}

	// Try to get caller identity to verify the role works
	validationSTS := sts.NewFromConfig(cfg)
	_, err = validationSTS.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("role assumption validation failed: %w", err)
	}

	return nil
}

// GetSSMClient creates an SSM client with assumed role credentials
func (r *RoleAssumer) GetSSMClient(ctx context.Context, accountID, roleName, region string) (*ssm.Client, error) {
	cfg, err := r.AssumeRole(ctx, accountID, roleName, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSM client: %w", err)
	}

	return ssm.NewFromConfig(cfg), nil
}

// ClearCache clears the credential cache
func (r *RoleAssumer) ClearCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.cache = make(map[string]*AssumedCredentials)
}

// ClearCacheForAccount clears cached credentials for a specific account
func (r *RoleAssumer) ClearCacheForAccount(accountID string) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	
	for key := range r.cache {
		if r.cache[key].AccountID == accountID {
			delete(r.cache, key)
		}
	}
}

// getEnvironmentName returns the environment name from environment variable
func getEnvironmentName() string {
	return os.Getenv("ENVIRONMENT")
}
