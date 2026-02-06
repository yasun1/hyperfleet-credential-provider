package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/errors"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Validate validates the configuration
func Validate(config *Config) error {
	if config == nil {
		return errors.New(errors.ErrConfigInvalid, "configuration is nil")
	}

	// Validate struct tags
	if err := validate.Struct(config); err != nil {
		return formatValidationError(err)
	}

	// Provider-specific validation
	switch config.Provider.Name {
	case "gcp":
		if err := validateGCPConfig(config.Provider.GCP); err != nil {
			return err
		}
	case "aws":
		if err := validateAWSConfig(config.Provider.AWS); err != nil {
			return err
		}
	case "azure":
		if err := validateAzureConfig(config.Provider.Azure); err != nil {
			return err
		}
	default:
		return errors.New(
			errors.ErrProviderNotSupported,
			fmt.Sprintf("unsupported provider: %s", config.Provider.Name),
		).WithField("provider", config.Provider.Name)
	}

	return nil
}

// validateGCPConfig validates GCP-specific configuration
func validateGCPConfig(config *GCPConfig) error {
	if config == nil {
		return errors.New(
			errors.ErrConfigMissingField,
			"GCP configuration is required",
		).WithField("provider", "gcp")
	}

	if config.ProjectID == "" {
		return errors.New(
			errors.ErrConfigMissingField,
			"GCP project_id is required",
		).WithField("provider", "gcp")
	}

	return nil
}

// validateAWSConfig validates AWS-specific configuration
func validateAWSConfig(config *AWSConfig) error {
	if config == nil {
		return errors.New(
			errors.ErrConfigMissingField,
			"AWS configuration is required",
		).WithField("provider", "aws")
	}

	// AWS config is mostly optional as credentials can come from environment
	// and SDK chain

	return nil
}

// validateAzureConfig validates Azure-specific configuration
func validateAzureConfig(config *AzureConfig) error {
	if config == nil {
		return errors.New(
			errors.ErrConfigMissingField,
			"Azure configuration is required",
		).WithField("provider", "azure")
	}

	if config.SubscriptionID == "" {
		return errors.New(
			errors.ErrConfigMissingField,
			"Azure subscription_id is required",
		).WithField("provider", "azure")
	}

	if config.TenantID == "" {
		return errors.New(
			errors.ErrConfigMissingField,
			"Azure tenant_id is required",
		).WithField("provider", "azure")
	}

	return nil
}

// formatValidationError formats validator errors into application errors
func formatValidationError(err error) error {
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return errors.Wrap(
			errors.ErrValidationFailed,
			err,
			"validation failed",
		)
	}

	// Get the first validation error for simplicity
	if len(validationErrs) > 0 {
		fieldErr := validationErrs[0]
		return errors.New(
			errors.ErrValidationFailed,
			fmt.Sprintf("validation failed for field '%s'", fieldErr.Field()),
		).WithFields(map[string]interface{}{
			"field": fieldErr.Field(),
			"tag":   fieldErr.Tag(),
			"value": fieldErr.Value(),
		})
	}

	return errors.New(errors.ErrValidationFailed, "validation failed")
}

// ValidateProvider validates that a provider name is supported
func ValidateProvider(provider string) error {
	supportedProviders := []string{"gcp", "aws", "azure"}
	for _, supported := range supportedProviders {
		if provider == supported {
			return nil
		}
	}

	return errors.New(
		errors.ErrProviderNotSupported,
		fmt.Sprintf("provider '%s' is not supported", provider),
	).WithField("provider", provider).
		WithField("supported", supportedProviders)
}
