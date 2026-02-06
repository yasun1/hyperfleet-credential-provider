package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/credentials"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

const (
	// aksResourceScope is the Azure resource scope for AKS
	// This is the standard scope for accessing AKS clusters
	aksResourceScope = "https://management.azure.com/.default"

	// defaultTokenDuration is the default duration for Azure AD tokens
	defaultTokenDuration = 1 * time.Hour
)

// TokenGenerator handles Azure AD token generation for AKS clusters
type TokenGenerator struct {
	config     *Config
	credLoader credentials.Loader
	logger     logger.Logger
}

// NewTokenGenerator creates a new Azure token generator
func NewTokenGenerator(config *Config, credLoader credentials.Loader, log logger.Logger) *TokenGenerator {
	return &TokenGenerator{
		config:     config,
		credLoader: credLoader,
		logger:     log,
	}
}

// GenerateToken generates an Azure AD token for AKS authentication
func (g *TokenGenerator) GenerateToken(ctx context.Context, opts provider.GetTokenOptions) (*provider.Token, error) {
	startTime := time.Now()

	g.logger.Debug("Starting Azure token generation",
		logger.String("cluster", opts.ClusterName),
		logger.String("subscription_id", opts.SubscriptionID),
		logger.String("tenant_id", opts.TenantID),
	)

	// Validate required parameters
	if opts.ClusterName == "" {
		return nil, errors.New(
			errors.ErrInvalidArgument,
			"cluster name is required",
		).WithField("provider", "azure")
	}

	// Step 1: Load Azure credentials
	azureCreds, err := g.loadAzureCredentials(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Step 2: Create Azure credential (service principal or managed identity)
	credential, err := g.createCredential(azureCreds)
	if err != nil {
		return nil, err
	}

	// Step 3: Get Azure AD access token
	accessToken, expiresOn, err := g.getAccessToken(ctx, credential)
	if err != nil {
		return nil, err
	}

	// Step 4: Create provider token
	token := &provider.Token{
		AccessToken: accessToken,
		ExpiresAt:   expiresOn,
		TokenType:   "Bearer",
	}

	duration := time.Since(startTime)
	g.logger.Info("Azure token generated successfully",
		logger.String("cluster", opts.ClusterName),
		logger.String("subscription_id", opts.SubscriptionID),
		logger.Duration("duration_ms", duration.Milliseconds()),
		logger.String("expires_at", token.ExpiresAt.Format(time.RFC3339)),
		logger.Duration("expires_in_seconds", int64(token.ExpiresIn().Seconds())),
	)

	return token, nil
}

// loadAzureCredentials loads Azure credentials from the credential loader
func (g *TokenGenerator) loadAzureCredentials(ctx context.Context, opts provider.GetTokenOptions) (*credentials.AzureCredentials, error) {
	// Determine tenant ID
	tenantID := opts.TenantID
	if tenantID == "" && g.config.TenantID != "" {
		tenantID = g.config.TenantID
	}

	// Load Azure credentials
	credOpts := credentials.AzureCredentialOptions{
		TenantID:       tenantID,
		UseEnvironment: true,
	}

	creds, err := g.credLoader.LoadAzure(ctx, credOpts)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialLoadFailed,
			err,
			"failed to load Azure credentials",
		).WithField("provider", "azure")
	}

	g.logger.Debug("Azure credentials loaded",
		logger.String("tenant_id", creds.TenantID),
		logger.Bool("has_client_secret", creds.ClientSecret != ""),
	)

	return creds, nil
}

// createCredential creates an Azure credential from service principal credentials
func (g *TokenGenerator) createCredential(creds *credentials.AzureCredentials) (azcore.TokenCredential, error) {
	// Create client secret credential for service principal authentication
	credential, err := azidentity.NewClientSecretCredential(
		creds.TenantID,
		creds.ClientID,
		creds.ClientSecret,
		&azidentity.ClientSecretCredentialOptions{
			ClientOptions: policy.ClientOptions{
				// Use default Azure cloud
			},
		},
	)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialInvalid,
			err,
			"failed to create Azure credential",
		).WithField("provider", "azure")
	}

	g.logger.Debug("Azure credential created",
		logger.String("credential_type", "ClientSecretCredential"),
	)

	return credential, nil
}

// getAccessToken retrieves an Azure AD access token using the credential
func (g *TokenGenerator) getAccessToken(ctx context.Context, credential azcore.TokenCredential) (string, time.Time, error) {
	// Request token with AKS resource scope
	tokenResult, err := credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{aksResourceScope},
	})
	if err != nil {
		return "", time.Time{}, errors.Wrap(
			errors.ErrTokenGenerationFailed,
			err,
			"failed to get Azure AD access token",
		).WithField("provider", "azure")
	}

	g.logger.Debug("Azure AD token retrieved",
		logger.String("token_length", fmt.Sprintf("%d", len(tokenResult.Token))),
		logger.String("expires_on", tokenResult.ExpiresOn.Format(time.RFC3339)),
	)

	return tokenResult.Token, tokenResult.ExpiresOn, nil
}

// getTokenDuration returns the configured token duration or default
func (g *TokenGenerator) getTokenDuration() time.Duration {
	if g.config.TokenDuration > 0 {
		return g.config.TokenDuration
	}
	return defaultTokenDuration
}

// ValidateToken validates that a token is valid and not expired
func (g *TokenGenerator) ValidateToken(token *provider.Token) error {
	if token == nil {
		return errors.New(
			errors.ErrTokenInvalid,
			"token is nil",
		).WithField("provider", "azure")
	}

	if token.AccessToken == "" {
		return errors.New(
			errors.ErrTokenInvalid,
			"access token is empty",
		).WithField("provider", "azure")
	}

	if token.IsExpired() {
		return errors.New(
			errors.ErrTokenExpired,
			"token has expired",
		).WithFields(map[string]interface{}{
			"provider":   "azure",
			"expires_at": token.ExpiresAt.Format(time.RFC3339),
		})
	}

	// Warn if token expires soon (less than 5 minutes)
	if token.ExpiresIn() < 5*time.Minute {
		g.logger.Warn("Token expires soon",
			logger.String("provider", "azure"),
			logger.Duration("expires_in_seconds", int64(token.ExpiresIn().Seconds())),
		)
	}

	return nil
}

// RefreshToken refreshes an expired or soon-to-expire token
func (g *TokenGenerator) RefreshToken(ctx context.Context, opts provider.GetTokenOptions, currentToken *provider.Token) (*provider.Token, error) {
	// Check if refresh is needed
	if currentToken != nil && !currentToken.IsExpired() {
		// For Azure, refresh if less than 5 minutes remaining
		if currentToken.ExpiresIn() > 5*time.Minute {
			g.logger.Debug("Token still valid, no refresh needed",
				logger.String("provider", "azure"),
				logger.Duration("expires_in_seconds", int64(currentToken.ExpiresIn().Seconds())),
			)
			return currentToken, nil
		}
	}

	g.logger.Info("Refreshing Azure token",
		logger.String("cluster", opts.ClusterName),
		logger.Bool("expired", currentToken == nil || currentToken.IsExpired()),
	)

	// Generate new token
	return g.GenerateToken(ctx, opts)
}
