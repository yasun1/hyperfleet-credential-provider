package provider

import (
	"context"
	"time"
)

// Provider generates Kubernetes authentication tokens for a specific cloud platform
type Provider interface {
	// GetToken generates a short-lived authentication token
	GetToken(ctx context.Context, opts GetTokenOptions) (*Token, error)

	// ValidateCredentials verifies that credentials are valid
	ValidateCredentials(ctx context.Context) error

	// Name returns the provider name (gcp, aws, azure)
	Name() string
}

// GetTokenOptions contains parameters for token generation
type GetTokenOptions struct {
	// ClusterName is the Kubernetes cluster name
	ClusterName string

	// Region is the cloud region
	Region string

	// ProjectID is the GCP project ID (GCP only)
	ProjectID string

	// AccountID is the AWS account ID (AWS only, optional)
	AccountID string

	// SubscriptionID is the Azure subscription ID (Azure only)
	SubscriptionID string

	// TenantID is the Azure tenant ID (Azure only)
	TenantID string

	// ResourceGroup is the Azure resource group (Azure only, optional)
	ResourceGroup string
}

// Token represents a Kubernetes authentication token
type Token struct {
	// AccessToken is the bearer token for authentication
	AccessToken string

	// ExpiresAt is when the token expires
	ExpiresAt time.Time

	// TokenType is the token type (usually "Bearer")
	TokenType string
}

// IsExpired returns true if the token has expired
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// ExpiresIn returns the duration until the token expires
func (t *Token) ExpiresIn() time.Duration {
	return time.Until(t.ExpiresAt)
}

// ProviderName represents a cloud provider name
type ProviderName string

const (
	// ProviderGCP is Google Cloud Platform
	ProviderGCP ProviderName = "gcp"

	// ProviderAWS is Amazon Web Services
	ProviderAWS ProviderName = "aws"

	// ProviderAzure is Microsoft Azure
	ProviderAzure ProviderName = "azure"
)

// String returns the string representation of the provider name
func (p ProviderName) String() string {
	return string(p)
}

// IsValid returns true if the provider name is valid
func (p ProviderName) IsValid() bool {
	switch p {
	case ProviderGCP, ProviderAWS, ProviderAzure:
		return true
	default:
		return false
	}
}
