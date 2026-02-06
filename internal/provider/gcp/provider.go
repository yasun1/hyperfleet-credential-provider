package gcp

import (
	"context"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/credentials"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

type Provider struct{
	config         *Config
	logger         logger.Logger
	tokenGenerator *TokenGenerator
	credLoader     credentials.Loader
}

func NewProvider(config *Config, log logger.Logger) (*Provider, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.ProjectID == "" {
		return nil, errors.New(
			errors.ErrConfigMissingField,
			"GCP project_id is required",
		).WithField("provider", "gcp")
	}

	credLoader := credentials.NewLoader(log)
	tokenGenerator := NewTokenGenerator(config, credLoader, log)

	log.Debug("GCP provider initialized",
		logger.String("project_id", config.ProjectID),
		logger.Int("num_scopes", len(config.Scopes)),
	)

	return &Provider{
		config:         config,
		logger:         log,
		tokenGenerator: tokenGenerator,
		credLoader:     credLoader,
	}, nil
}

func (p *Provider) GetToken(ctx context.Context, opts provider.GetTokenOptions) (*provider.Token, error) {
	if opts.ClusterName == "" {
		return nil, errors.New(
			errors.ErrInvalidArgument,
			"cluster name is required",
		).WithField("provider", "gcp")
	}

	// Use project ID from options if provided, otherwise use config
	if opts.ProjectID == "" {
		opts.ProjectID = p.config.ProjectID
	}

	p.logger.Info("Generating GCP token",
		logger.String("cluster", opts.ClusterName),
		logger.String("project", opts.ProjectID),
		logger.String("region", opts.Region),
	)

	token, err := p.tokenGenerator.GenerateToken(ctx, opts)
	if err != nil {
		p.logger.Error("Failed to generate GCP token",
			logger.String("cluster", opts.ClusterName),
			logger.String("project", opts.ProjectID),
			logger.Error(err),
		)
		return nil, err
	}

	if err := p.tokenGenerator.ValidateToken(token); err != nil {
		return nil, err
	}

	return token, nil
}

func (p *Provider) ValidateCredentials(ctx context.Context) error {
	p.logger.Debug("Validating GCP credentials",
		logger.String("project_id", p.config.ProjectID),
	)

	creds, err := p.credLoader.LoadGCP(ctx, p.config.CredentialsFile)
	if err != nil {
		return errors.Wrap(
			errors.ErrCredentialValidationFailed,
			err,
			"failed to validate GCP credentials",
		).WithField("provider", "gcp")
	}

	if p.config.ProjectID != "" && creds.ProjectID != p.config.ProjectID {
		return errors.New(
			errors.ErrCredentialInvalid,
			"project ID mismatch between config and credentials",
		).WithFields(map[string]interface{}{
			"provider":       "gcp",
			"config_project": p.config.ProjectID,
			"creds_project":  creds.ProjectID,
		})
	}

	// Try to generate a test token to verify credentials work
	testOpts := provider.GetTokenOptions{
		ClusterName: "test-cluster",
		ProjectID:   p.config.ProjectID,
		Region:      "us-central1",
	}

	token, err := p.tokenGenerator.GenerateToken(ctx, testOpts)
	if err != nil {
		return errors.Wrap(
			errors.ErrCredentialValidationFailed,
			err,
			"credentials loaded but failed to generate test token",
		).WithField("provider", "gcp")
	}

	// Validate the test token
	if err := p.tokenGenerator.ValidateToken(token); err != nil {
		return errors.Wrap(
			errors.ErrCredentialValidationFailed,
			err,
			"test token validation failed",
		).WithField("provider", "gcp")
	}

	p.logger.Info("GCP credentials validated successfully",
		logger.String("project_id", creds.ProjectID),
		logger.String("client_email", creds.ClientEmail),
	)

	return nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "gcp"
}
