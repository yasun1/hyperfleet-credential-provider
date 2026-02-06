package credentials

import (
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/errors"
)

// validateGCPCredentials validates GCP service account credentials
func (l *DefaultLoader) validateGCPCredentials(creds *GCPCredentials) error {
	if creds.Type != "service_account" {
		return errors.New(
			errors.ErrCredentialInvalid,
			"invalid GCP credential type",
		).WithField("type", creds.Type).
			WithDetail("expected 'service_account'")
	}

	if creds.ProjectID == "" {
		return errors.New(
			errors.ErrCredentialMalformed,
			"GCP credentials missing project_id",
		)
	}

	if creds.PrivateKey == "" {
		return errors.New(
			errors.ErrCredentialMalformed,
			"GCP credentials missing private_key",
		)
	}

	if creds.ClientEmail == "" {
		return errors.New(
			errors.ErrCredentialMalformed,
			"GCP credentials missing client_email",
		)
	}

	return nil
}

// validateAWSCredentials validates AWS credentials
func (l *DefaultLoader) validateAWSCredentials(creds *AWSCredentials) error {
	if creds.AccessKeyID == "" {
		return errors.New(
			errors.ErrCredentialNotFound,
			"AWS access key ID not found",
		).WithDetail("set AWS_ACCESS_KEY_ID environment variable")
	}

	if creds.SecretAccessKey == "" {
		return errors.New(
			errors.ErrCredentialNotFound,
			"AWS secret access key not found",
		).WithDetail("set AWS_SECRET_ACCESS_KEY environment variable")
	}

	return nil
}

// validateAzureCredentials validates Azure credentials
func (l *DefaultLoader) validateAzureCredentials(creds *AzureCredentials) error {
	if creds.ClientID == "" {
		return errors.New(
			errors.ErrCredentialNotFound,
			"Azure client ID not found",
		).WithDetail("set AZURE_CLIENT_ID environment variable")
	}

	if creds.ClientSecret == "" {
		return errors.New(
			errors.ErrCredentialNotFound,
			"Azure client secret not found",
		).WithDetail("set AZURE_CLIENT_SECRET environment variable")
	}

	if creds.TenantID == "" {
		return errors.New(
			errors.ErrCredentialNotFound,
			"Azure tenant ID not found",
		).WithDetail("set AZURE_TENANT_ID environment variable")
	}

	return nil
}
