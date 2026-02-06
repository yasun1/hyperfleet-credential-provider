package provider

import (
	"context"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/config"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
)

// Factory creates providers based on configuration
type Factory struct {
	registry *Registry
	logger   logger.Logger
}

// NewFactory creates a new provider factory
func NewFactory(registry *Registry, logger logger.Logger) *Factory {
	return &Factory{
		registry: registry,
		logger:   logger,
	}
}

// CreateFromConfig creates a provider based on the configuration
func (f *Factory) CreateFromConfig(ctx context.Context, cfg *config.Config) (Provider, error) {
	if cfg == nil {
		return nil, errors.New(
			errors.ErrConfigInvalid,
			"configuration is nil",
		)
	}

	providerName := ProviderName(cfg.Provider.Name)
	if !providerName.IsValid() {
		return nil, errors.New(
			errors.ErrProviderNotSupported,
			"invalid provider name",
		).WithField("provider", cfg.Provider.Name)
	}

	// Validate provider-specific configuration
	if err := f.validateProviderConfig(cfg); err != nil {
		return nil, err
	}

	// Create the provider
	provider, err := f.registry.Create(ctx, providerName)
	if err != nil {
		return nil, err
	}

	f.logger.Info("Provider created from config",
		logger.String("provider", providerName.String()),
		logger.String("cluster", cfg.Provider.ClusterName),
	)

	return provider, nil
}

// validateProviderConfig validates provider-specific configuration
func (f *Factory) validateProviderConfig(cfg *config.Config) error {
	switch ProviderName(cfg.Provider.Name) {
	case ProviderGCP:
		if cfg.Provider.GCP == nil {
			return errors.New(
				errors.ErrConfigMissingField,
				"GCP configuration is required",
			)
		}
		if cfg.Provider.GCP.ProjectID == "" {
			return errors.New(
				errors.ErrConfigMissingField,
				"GCP project_id is required",
			)
		}

	case ProviderAWS:
		if cfg.Provider.AWS == nil {
			return errors.New(
				errors.ErrConfigMissingField,
				"AWS configuration is required",
			)
		}

	case ProviderAzure:
		if cfg.Provider.Azure == nil {
			return errors.New(
				errors.ErrConfigMissingField,
				"Azure configuration is required",
			)
		}
		if cfg.Provider.Azure.SubscriptionID == "" {
			return errors.New(
				errors.ErrConfigMissingField,
				"Azure subscription_id is required",
			)
		}
		if cfg.Provider.Azure.TenantID == "" {
			return errors.New(
				errors.ErrConfigMissingField,
				"Azure tenant_id is required",
			)
		}

	default:
		return errors.New(
			errors.ErrProviderNotSupported,
			"unsupported provider",
		).WithField("provider", cfg.Provider.Name)
	}

	return nil
}

// Create creates a provider by name
func (f *Factory) Create(ctx context.Context, name ProviderName) (Provider, error) {
	return f.registry.Create(ctx, name)
}
