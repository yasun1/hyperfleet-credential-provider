package azure

import (
	"context"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/credentials"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
)

// Provider implements the Azure token provider
type Provider struct {
	config         *Config
	logger         logger.Logger
	tokenGenerator *TokenGenerator
	credLoader     credentials.Loader
	azureCredOpts  credentials.AzureCredentialOptions
}

// NewProvider creates a new Azure provider
func NewProvider(config *Config, log logger.Logger) (*Provider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create credential loader
	credLoader := credentials.NewLoader(log)

	// Create token generator
	tokenGenerator := NewTokenGenerator(config, credLoader, log)

	// Setup Azure credential options
	azureCredOpts := credentials.AzureCredentialOptions{
		CredentialsFile: config.CredentialsFile, // Use config.CredentialsFile if provided
		UseEnvironment:  true,
	}

	return &Provider{
		config:         config,
		logger:         log,
		tokenGenerator: tokenGenerator,
		credLoader:     credLoader,
		azureCredOpts:  azureCredOpts,
	}, nil
}

// GetToken generates an AKS authentication token
func (p *Provider) GetToken(ctx context.Context, opts provider.GetTokenOptions) (*provider.Token, error) {
	p.logger.Info("Getting Azure token",
		logger.String("cluster", opts.ClusterName),
		logger.String("subscription", opts.SubscriptionID),
		logger.String("tenant", opts.TenantID),
	)

	// Generate token using token generator
	token, err := p.tokenGenerator.GenerateToken(ctx, opts)
	if err != nil {
		p.logger.Error("Failed to generate Azure token",
			logger.String("error", err.Error()),
		)
		return nil, err
	}

	// Validate token before returning
	if err := p.tokenGenerator.ValidateToken(token); err != nil {
		p.logger.Error("Generated token is invalid",
			logger.String("error", err.Error()),
		)
		return nil, err
	}

	return token, nil
}

// ValidateCredentials validates Azure credentials by attempting to generate a test token
func (p *Provider) ValidateCredentials(ctx context.Context) error {
	p.logger.Debug("Validating Azure credentials")

	// Try to generate a token with minimal options to validate credentials
	opts := provider.GetTokenOptions{
		ClusterName:    "validation-test",
		SubscriptionID: p.config.SubscriptionID,
		TenantID:       p.config.TenantID,
	}

	_, err := p.tokenGenerator.GenerateToken(ctx, opts)
	if err != nil {
		return errors.Wrap(
			errors.ErrCredentialValidationFailed,
			err,
			"credential validation failed",
		).WithField("provider", "azure")
	}

	p.logger.Info("Azure credentials validated successfully")
	return nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "azure"
}
