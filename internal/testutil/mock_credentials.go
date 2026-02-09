// Package testutil provides testing utilities for the credential provider
package testutil

import (
	"context"
	"time"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/credentials"
)

// MockCredLoader is a mock implementation of credentials.Loader for testing
// It can be configured to return specific credentials or errors for each provider type
type MockCredLoader struct {
	GCPCreds   *credentials.GCPCredentials
	GCPErr     error
	AWSCreds   *credentials.AWSCredentials
	AWSErr     error
	AzureCreds *credentials.AzureCredentials
	AzureErr   error
}

// NewMockCredLoader creates a new mock credential loader
func NewMockCredLoader() *MockCredLoader {
	return &MockCredLoader{}
}

// WithGCPCreds configures the mock to return specific GCP credentials
func (m *MockCredLoader) WithGCPCreds(creds *credentials.GCPCredentials) *MockCredLoader {
	m.GCPCreds = creds
	return m
}

// WithGCPError configures the mock to return an error for GCP credentials
func (m *MockCredLoader) WithGCPError(err error) *MockCredLoader {
	m.GCPErr = err
	return m
}

// WithAWSCreds configures the mock to return specific AWS credentials
func (m *MockCredLoader) WithAWSCreds(creds *credentials.AWSCredentials) *MockCredLoader {
	m.AWSCreds = creds
	return m
}

// WithAWSError configures the mock to return an error for AWS credentials
func (m *MockCredLoader) WithAWSError(err error) *MockCredLoader {
	m.AWSErr = err
	return m
}

// WithAzureCreds configures the mock to return specific Azure credentials
func (m *MockCredLoader) WithAzureCreds(creds *credentials.AzureCredentials) *MockCredLoader {
	m.AzureCreds = creds
	return m
}

// WithAzureError configures the mock to return an error for Azure credentials
func (m *MockCredLoader) WithAzureError(err error) *MockCredLoader {
	m.AzureErr = err
	return m
}

// LoadGCP implements credentials.Loader interface
func (m *MockCredLoader) LoadGCP(ctx context.Context, path string) (*credentials.GCPCredentials, error) {
	if m.GCPErr != nil {
		return nil, m.GCPErr
	}
	return m.GCPCreds, nil
}

// LoadAWS implements credentials.Loader interface
func (m *MockCredLoader) LoadAWS(ctx context.Context, opts credentials.AWSCredentialOptions) (*credentials.AWSCredentials, error) {
	if m.AWSErr != nil {
		return nil, m.AWSErr
	}
	return m.AWSCreds, nil
}

// LoadAzure implements credentials.Loader interface
func (m *MockCredLoader) LoadAzure(ctx context.Context, opts credentials.AzureCredentialOptions) (*credentials.AzureCredentials, error) {
	if m.AzureErr != nil {
		return nil, m.AzureErr
	}
	return m.AzureCreds, nil
}

// --- GCP Credential Helpers ---

// CreateValidGCPCredentials creates valid-looking (but fake) GCP credentials for testing
// These credentials will NOT work with real Google APIs but are valid in structure
func CreateValidGCPCredentials() *credentials.GCPCredentials {
	return &credentials.GCPCredentials{
		Type:        "service_account",
		ProjectID:   "test-project-12345",
		PrivateKeyID: "abcdef1234567890",
		PrivateKey:  "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7W8jlH1234567\n-----END PRIVATE KEY-----\n",
		ClientEmail: "test-sa@test-project-12345.iam.gserviceaccount.com",
		ClientID:    "123456789012345678901",
		AuthURI:     "https://accounts.google.com/o/oauth2/auth",
		TokenURI:    "https://oauth2.googleapis.com/token",
		AuthProviderX509CertURL: "https://www.googleapis.com/oauth2/v1/certs",
		ClientX509CertURL:       "https://www.googleapis.com/robot/v1/metadata/x509/test-sa%40test-project-12345.iam.gserviceaccount.com",
	}
}

// CreateInvalidGCPCredentials creates invalid GCP credentials for error testing
func CreateInvalidGCPCredentials() *credentials.GCPCredentials {
	return &credentials.GCPCredentials{
		Type:       "service_account",
		ProjectID:  "", // Invalid - empty
		PrivateKey: "", // Invalid - empty
	}
}

// --- AWS Credential Helpers ---

// CreateValidAWSCredentials creates valid-looking (but fake) AWS credentials for testing
// These credentials will NOT work with real AWS APIs but are valid in structure
func CreateValidAWSCredentials() *credentials.AWSCredentials {
	return &credentials.AWSCredentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Region:          "us-east-1",
	}
}

// CreateValidAWSCredentialsWithSessionToken creates AWS credentials with session token
func CreateValidAWSCredentialsWithSessionToken() *credentials.AWSCredentials {
	creds := CreateValidAWSCredentials()
	creds.SessionToken = "FwoGZXIvYXdzEBYaDH...TestSessionToken"
	return creds
}

// CreateAWSCredentialsWithRegion creates AWS credentials with specific region
func CreateAWSCredentialsWithRegion(region string) *credentials.AWSCredentials {
	creds := CreateValidAWSCredentials()
	creds.Region = region
	return creds
}

// CreateInvalidAWSCredentials creates invalid AWS credentials for error testing
func CreateInvalidAWSCredentials() *credentials.AWSCredentials {
	return &credentials.AWSCredentials{
		AccessKeyID:     "", // Invalid - empty
		SecretAccessKey: "",
	}
}

// --- Azure Credential Helpers ---

// CreateValidAzureCredentials creates valid-looking (but fake) Azure credentials for testing
// These credentials will NOT work with real Azure APIs but are valid in structure
func CreateValidAzureCredentials() *credentials.AzureCredentials {
	return &credentials.AzureCredentials{
		ClientID:     "11111111-1111-1111-1111-111111111111",
		ClientSecret: "test-client-secret-value-12345",
		TenantID:     "22222222-2222-2222-2222-222222222222",
	}
}

// CreateValidAzureCredentialsWithTenant creates Azure credentials with specific tenant
func CreateValidAzureCredentialsWithTenant(tenantID string) *credentials.AzureCredentials {
	creds := CreateValidAzureCredentials()
	creds.TenantID = tenantID
	return creds
}

// CreateInvalidAzureCredentials creates invalid Azure credentials for error testing
func CreateInvalidAzureCredentials() *credentials.AzureCredentials {
	return &credentials.AzureCredentials{
		ClientID:     "", // Invalid - empty
		ClientSecret: "",
		TenantID:     "",
	}
}

// --- Time Mock ---

// MockTime is a mock time provider for testing time-dependent logic
type MockTime struct {
	CurrentTime time.Time
}

// NewMockTime creates a new mock time with the specified current time
func NewMockTime(now time.Time) *MockTime {
	return &MockTime{CurrentTime: now}
}

// Now returns the mocked current time
func (m *MockTime) Now() time.Time {
	return m.CurrentTime
}
