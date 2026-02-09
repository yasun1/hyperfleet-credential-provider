package aws

import (
	"context"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/credentials"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

// Provider implements the AWS token provider
type Provider struct {
	config         *Config
	logger         logger.Logger
	tokenGenerator *TokenGenerator
	credLoader     credentials.Loader
	awsCredOpts    credentials.AWSCredentialOptions
}

// NewProvider creates a new AWS provider
func NewProvider(config *Config, log logger.Logger) (*Provider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Note: For AWS, region is optional and can be provided at token generation time
	// Unlike GCP which requires project_id, AWS can work with just credentials

	credLoader := credentials.NewLoader(log)

	tokenGenerator := NewTokenGenerator(config, credLoader, log)

	// Setup AWS credential options
	awsCredOpts := credentials.AWSCredentialOptions{
		CredentialsFile: config.CredentialsFile, // Use config.CredentialsFile if provided
		UseEnvironment:  true,
	}

	log.Debug("AWS provider initialized",
		logger.String("region", config.Region),
		logger.Duration("token_duration_seconds", int64(config.TokenDuration.Seconds())),
	)

	return &Provider{
		config:         config,
		logger:         log,
		tokenGenerator: tokenGenerator,
		credLoader:     credLoader,
		awsCredOpts:    awsCredOpts,
	}, nil
}

// GetToken generates an EKS authentication token
func (p *Provider) GetToken(ctx context.Context, opts provider.GetTokenOptions) (*provider.Token, error) {
	if opts.ClusterName == "" {
		return nil, errors.New(
			errors.ErrInvalidArgument,
			"cluster name is required",
		).WithField("provider", "aws")
	}

	if opts.Region == "" && p.config.Region != "" {
		opts.Region = p.config.Region
	}

	// Region is still optional for AWS (can use default region from env)
	if opts.Region == "" {
		p.logger.Debug("No region specified, will use AWS default region")
	}

	p.logger.Info("Generating AWS token",
		logger.String("cluster", opts.ClusterName),
		logger.String("region", opts.Region),
		logger.String("account_id", opts.AccountID),
	)

	// Generate token using token generator
	token, err := p.tokenGenerator.GenerateToken(ctx, opts)
	if err != nil {
		p.logger.Error("Failed to generate AWS token",
			logger.String("cluster", opts.ClusterName),
			logger.String("region", opts.Region),
			logger.Error(err),
		)
		return nil, err
	}

	if err := p.tokenGenerator.ValidateToken(token); err != nil {
		return nil, err
	}

	return token, nil
}

// ValidateCredentials validates AWS credentials
func (p *Provider) ValidateCredentials(ctx context.Context) error {
	p.logger.Debug("Validating AWS credentials",
		logger.String("region", p.config.Region),
	)

	// Try to load credentials
	credOpts := credentials.AWSCredentialOptions{
		Region:         p.config.Region,
		UseEnvironment: true,
	}

	creds, err := p.credLoader.LoadAWS(ctx, credOpts)
	if err != nil {
		return errors.Wrap(
			errors.ErrCredentialValidationFailed,
			err,
			"failed to validate AWS credentials",
		).WithField("provider", "aws")
	}

	// Try to generate a test token to verify credentials work
	testOpts := provider.GetTokenOptions{
		ClusterName: "test-cluster",
		Region:      creds.Region,
	}

	token, err := p.tokenGenerator.GenerateToken(ctx, testOpts)
	if err != nil {
		return errors.Wrap(
			errors.ErrCredentialValidationFailed,
			err,
			"credentials loaded but failed to generate test token",
		).WithField("provider", "aws")
	}

	if err := p.tokenGenerator.ValidateToken(token); err != nil {
		return errors.Wrap(
			errors.ErrCredentialValidationFailed,
			err,
			"test token validation failed",
		).WithField("provider", "aws")
	}

	p.logger.Info("AWS credentials validated successfully",
		logger.String("region", creds.Region),
		logger.Bool("has_session_token", creds.SessionToken != ""),
	)

	return nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "aws"
}
